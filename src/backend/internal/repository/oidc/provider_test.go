package oidc

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"devtool/internal/domain"

	coreoidc "github.com/coreos/go-oidc/v3/oidc"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/oauth2"
)

func TestIssuerDiscoverySSRFAndRedirect(t *testing.T) {
	for _, raw := range []string{"http://idp.example", "https://127.0.0.1", "https://[::1]", "https://user@idp.example", "https://idp.example?next=x"} {
		u, e := publicHTTPS(raw)
		if e == nil && (u.Hostname() == "127.0.0.1" || u.Hostname() == "::1") {
			ips, _ := net.LookupIP(u.Hostname())
			if len(ips) == 0 || !forbiddenIP(ips[0]) {
				t.Fatalf("private host accepted: %s", raw)
			}
		} else if e == nil && raw != "https://127.0.0.1" && raw != "https://[::1]" {
			t.Fatalf("unsafe issuer accepted: %s", raw)
		}
	}
	base, _ := publicHTTPS("https://idp.example")
	if sameOrigin(base, "https://evil.example/token") || sameOrigin(base, "https://idp.example.evil/token") {
		t.Fatal("cross-origin endpoint accepted")
	}
	factory := NewFactory(testBox{}, time.Second)
	_, e := factory.Provider(context.Background(), domain.SSOConnection{Issuer: "https://127.0.0.1", ClientID: "client", ClientSecretCipher: []byte("secret")}, "https://app.example/callback")
	if e == nil {
		t.Fatal("loopback discovery accepted")
	}
}

type testBox struct{}

func (testBox) Seal(v string) ([]byte, error) { return []byte(v), nil }
func (testBox) Open(v []byte) (string, error) { return string(v), nil }

func TestOIDCWrongIssuerAudienceExpiry(t *testing.T) {
	key, e := rsa.GenerateKey(rand.Reader, 2048)
	if e != nil {
		t.Fatal(e)
	}
	issuer := "https://issuer.example"
	nonce := "nonce"
	clientID := "client"
	var rawToken string
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"access_token": "access", "token_type": "Bearer", "expires_in": 60, "id_token": rawToken})
	}))
	defer server.Close()
	verifier := coreoidc.NewVerifier(issuer, &coreoidc.StaticKeySet{PublicKeys: []crypto.PublicKey{&key.PublicKey}}, &coreoidc.Config{ClientID: clientID})
	p := &provider{oauth: oauth2.Config{ClientID: clientID, ClientSecret: "secret", RedirectURL: "https://app.example/callback", Endpoint: oauth2.Endpoint{TokenURL: server.URL}}, verifier: verifier, issuer: issuer, client: server.Client()}
	ctx := context.WithValue(context.Background(), oauth2.HTTPClient, server.Client())
	cases := []struct {
		name, iss, aud string
		exp            time.Time
	}{{"wrong issuer", "https://wrong.example", clientID, time.Now().Add(time.Minute)}, {"wrong audience", issuer, "other", time.Now().Add(time.Minute)}, {"expired", issuer, clientID, time.Now().Add(-time.Minute)}}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			claims := jwt.MapClaims{"iss": tc.iss, "sub": "subject", "aud": tc.aud, "exp": tc.exp.Unix(), "iat": time.Now().Add(-time.Minute).Unix(), "nonce": nonce, "email": "user@example.test", "email_verified": true}
			rawToken, e = jwt.NewWithClaims(jwt.SigningMethodRS256, claims).SignedString(key)
			if e != nil {
				t.Fatal(e)
			}
			if _, e = p.ExchangeAndVerify(ctx, "code", "verifier", nonce); e == nil {
				t.Fatal("invalid ID token accepted")
			}
		})
	}
}
