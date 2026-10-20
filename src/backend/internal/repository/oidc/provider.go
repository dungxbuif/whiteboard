package oidc

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"devtool/internal/domain"

	coreoidc "github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
)

type AESGCMCipher struct{ key []byte }

func NewAESGCMCipher(key []byte) (*AESGCMCipher, error) {
	if len(key) != 32 {
		return nil, errors.New("SSO_ENCRYPTION_KEY must contain exactly 32 bytes")
	}
	return &AESGCMCipher{key: append([]byte(nil), key...)}, nil
}
func (c *AESGCMCipher) Seal(value string) ([]byte, error) {
	block, e := aes.NewCipher(c.key)
	if e != nil {
		return nil, e
	}
	gcm, e := cipher.NewGCM(block)
	if e != nil {
		return nil, e
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, e = io.ReadFull(rand.Reader, nonce); e != nil {
		return nil, e
	}
	return gcm.Seal(nonce, nonce, []byte(value), nil), nil
}
func (c *AESGCMCipher) Open(value []byte) (string, error) {
	block, e := aes.NewCipher(c.key)
	if e != nil {
		return "", e
	}
	gcm, e := cipher.NewGCM(block)
	if e != nil {
		return "", e
	}
	if len(value) < gcm.NonceSize() {
		return "", domain.ErrSSOInvalid
	}
	plain, e := gcm.Open(nil, value[:gcm.NonceSize()], value[gcm.NonceSize():], nil)
	if e != nil {
		return "", domain.ErrSSOInvalid
	}
	return string(plain), nil
}

type Factory struct {
	cipher  domain.SecretCipher
	timeout time.Duration
}

func NewFactory(secretCipher domain.SecretCipher, timeout time.Duration) *Factory {
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	return &Factory{cipher: secretCipher, timeout: timeout}
}

func publicHTTPS(raw string) (*url.URL, error) {
	u, e := url.Parse(raw)
	if e != nil || u.Scheme != "https" || u.Hostname() == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
		return nil, domain.ErrSSOInvalid
	}
	if u.Path != "" && u.Path != "/" {
		u.Path = strings.TrimSuffix(u.Path, "/")
	}
	return u, nil
}
func forbiddenIP(ip net.IP) bool {
	return ip == nil || ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsMulticast() || ip.IsUnspecified()
}

func safeHTTPClient(timeout time.Duration) *http.Client {
	dialer := &net.Dialer{Timeout: timeout}
	transport := &http.Transport{TLSClientConfig: &tls.Config{MinVersion: tls.VersionTLS12}, ResponseHeaderTimeout: timeout, DialContext: func(ctx context.Context, network, address string) (net.Conn, error) {
		host, port, e := net.SplitHostPort(address)
		if e != nil {
			return nil, e
		}
		ips, e := net.DefaultResolver.LookupIP(ctx, "ip", host)
		if e != nil || len(ips) == 0 {
			return nil, domain.ErrSSOInvalid
		}
		for _, ip := range ips {
			if forbiddenIP(ip) {
				return nil, domain.ErrSSOInvalid
			}
		}
		return dialer.DialContext(ctx, network, net.JoinHostPort(ips[0].String(), port))
	}}
	return &http.Client{Transport: transport, Timeout: timeout, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
}

type provider struct {
	oauth    oauth2.Config
	verifier *coreoidc.IDTokenVerifier
	issuer   string
	client   *http.Client
}

func (p *provider) AuthorizationURL(state, nonce, challenge string) string {
	return p.oauth.AuthCodeURL(state, oauth2.SetAuthURLParam("nonce", nonce), oauth2.SetAuthURLParam("code_challenge", challenge), oauth2.SetAuthURLParam("code_challenge_method", "S256"))
}
func (p *provider) ExchangeAndVerify(ctx domain.Context, code, verifier, nonce string) (domain.OIDCIdentity, error) {
	ctx = coreoidc.ClientContext(ctx, p.client)
	token, e := p.oauth.Exchange(ctx, code, oauth2.SetAuthURLParam("code_verifier", verifier))
	if e != nil {
		return domain.OIDCIdentity{}, domain.ErrSSOInvalid
	}
	raw, ok := token.Extra("id_token").(string)
	if !ok || raw == "" {
		return domain.OIDCIdentity{}, domain.ErrSSOInvalid
	}
	id, e := p.verifier.Verify(ctx, raw)
	if e != nil || id.Issuer != p.issuer || id.Subject == "" || id.Nonce != nonce {
		return domain.OIDCIdentity{}, domain.ErrSSOInvalid
	}
	var claims struct {
		Email         string `json:"email"`
		EmailVerified bool   `json:"email_verified"`
		Name          string `json:"name"`
		Picture       string `json:"picture"`
	}
	if e = id.Claims(&claims); e != nil || claims.Email == "" || !claims.EmailVerified {
		return domain.OIDCIdentity{}, domain.ErrSSOInvalid
	}
	return domain.OIDCIdentity{Issuer: id.Issuer, Subject: id.Subject, Email: strings.ToLower(strings.TrimSpace(claims.Email)), EmailVerified: claims.EmailVerified, Name: claims.Name, Picture: claims.Picture}, nil
}

func sameOrigin(base *url.URL, raw string) bool {
	u, e := publicHTTPS(raw)
	return e == nil && strings.EqualFold(u.Scheme, base.Scheme) && strings.EqualFold(u.Host, base.Host)
}

func (f *Factory) Provider(ctx domain.Context, connection domain.SSOConnection, redirectURL string) (domain.OIDCProvider, error) {
	issuer, e := publicHTTPS(connection.Issuer)
	if e != nil {
		return nil, e
	}
	redirect, e := url.Parse(redirectURL)
	if e != nil || redirect.Scheme != "https" || redirect.Host == "" {
		return nil, domain.ErrSSOInvalid
	}
	client := safeHTTPClient(f.timeout)
	ctx = coreoidc.ClientContext(ctx, client)
	discovered, e := coreoidc.NewProvider(ctx, strings.TrimSuffix(issuer.String(), "/"))
	if e != nil {
		return nil, fmt.Errorf("oidc discovery: %w", domain.ErrSSOInvalid)
	}
	var metadata struct {
		Issuer                string `json:"issuer"`
		AuthorizationEndpoint string `json:"authorization_endpoint"`
		TokenEndpoint         string `json:"token_endpoint"`
		JWKSURI               string `json:"jwks_uri"`
	}
	if e = discovered.Claims(&metadata); e != nil || metadata.Issuer != strings.TrimSuffix(issuer.String(), "/") || !sameOrigin(issuer, metadata.AuthorizationEndpoint) || !sameOrigin(issuer, metadata.TokenEndpoint) || !sameOrigin(issuer, metadata.JWKSURI) {
		return nil, domain.ErrSSOInvalid
	}
	secret, e := f.cipher.Open(connection.ClientSecretCipher)
	if e != nil {
		return nil, e
	}
	conf := oauth2.Config{ClientID: connection.ClientID, ClientSecret: secret, Endpoint: discovered.Endpoint(), RedirectURL: redirect.String(), Scopes: []string{coreoidc.ScopeOpenID, "email", "profile"}}
	return &provider{oauth: conf, verifier: discovered.VerifierContext(ctx, &coreoidc.Config{ClientID: connection.ClientID}), issuer: connection.Issuer, client: client}, nil
}

func S256Challenge(verifier string) string {
	sum := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}
