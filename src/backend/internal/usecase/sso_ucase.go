package usecase

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/url"
	"strings"
	"time"

	"devtool/internal/domain"
	oidcrepo "devtool/internal/repository/oidc"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type SSOService struct {
	repo        domain.SSORepository
	access      domain.AccessService
	sessions    *SessionService
	providers   domain.OIDCProviderFactory
	cipher      domain.SecretCipher
	callbackURL string
	now         func() time.Time
}

func NewSSOService(repo domain.SSORepository, access domain.AccessService, sessions *SessionService, providers domain.OIDCProviderFactory, cipher domain.SecretCipher, callbackURL string, clocks ...func() time.Time) (*SSOService, error) {
	u, e := url.Parse(callbackURL)
	if repo == nil || access == nil || sessions == nil || providers == nil || cipher == nil || e != nil || u.Scheme != "https" || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
		return nil, domain.ErrSSOInvalid
	}
	now := time.Now
	if len(clocks) > 0 && clocks[0] != nil {
		now = clocks[0]
	}
	return &SSOService{repo: repo, access: access, sessions: sessions, providers: providers, cipher: cipher, callbackURL: u.String(), now: now}, nil
}
func randomOpaque() (string, error) {
	var b [32]byte
	if _, e := rand.Read(b[:]); e != nil {
		return "", e
	}
	return base64.RawURLEncoding.EncodeToString(b[:]), nil
}
func opaqueHash(raw string) []byte { sum := sha256.Sum256([]byte(raw)); return sum[:] }
func bindingFromContext(ctx domain.Context) (string, error) {
	value, ok := ctx.Value(domain.ContextKeyBrowserBinding).(string)
	if !ok || len(value) < 32 || len(value) > 256 {
		return "", domain.ErrSSOInvalid
	}
	return value, nil
}
func safeReturnPath(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "/"
	}
	u, e := url.Parse(value)
	if e != nil || u.IsAbs() || u.Host != "" || !strings.HasPrefix(u.Path, "/") || strings.HasPrefix(value, "//") {
		return "/"
	}
	return u.RequestURI()
}

type verifierBundle struct {
	Verifier string `json:"verifier"`
	Nonce    string `json:"nonce"`
}

func (s *SSOService) Begin(ctx domain.Context, connectionID uuid.UUID, returnPath string) (domain.AuthorizationRedirect, error) {
	return s.begin(ctx, connectionID, returnPath, nil, nil)
}

func (s *SSOService) BeginLink(ctx domain.Context, principal domain.Principal, connectionID uuid.UUID, returnPath string, requestID uuid.UUID) (domain.AuthorizationRedirect, error) {
	if principal.UserID == nil || *principal.UserID == uuid.Nil || requestID == uuid.Nil {
		return domain.AuthorizationRedirect{}, domain.ErrSSOInvalid
	}
	connection, err := s.repo.GetConnectionByID(ctx, connectionID)
	if err != nil {
		return domain.AuthorizationRedirect{}, err
	}
	principal = principalSession(ctx, principal)
	if err = s.sessions.RequireWorkspaceAssurance(ctx, principal, connection.WorkspaceID); err != nil {
		return domain.AuthorizationRedirect{}, err
	}
	if _, err = s.access.ResolveWorkspace(ctx, principal, connection.WorkspaceID); err != nil {
		return domain.AuthorizationRedirect{}, err
	}
	return s.begin(ctx, connectionID, returnPath, principal.UserID, &requestID)
}

func (s *SSOService) begin(ctx domain.Context, connectionID uuid.UUID, returnPath string, linkUserID, linkRequestID *uuid.UUID) (domain.AuthorizationRedirect, error) {
	binding, e := bindingFromContext(ctx)
	if e != nil {
		return domain.AuthorizationRedirect{}, e
	}
	connection, e := s.repo.GetConnectionByID(ctx, connectionID)
	if e != nil {
		return domain.AuthorizationRedirect{}, e
	}
	state, e := randomOpaque()
	if e != nil {
		return domain.AuthorizationRedirect{}, e
	}
	nonce, e := randomOpaque()
	if e != nil {
		return domain.AuthorizationRedirect{}, e
	}
	verifier, e := randomOpaque()
	if e != nil {
		return domain.AuthorizationRedirect{}, e
	}
	bundle, _ := json.Marshal(verifierBundle{Verifier: verifier, Nonce: nonce})
	encrypted, e := s.cipher.Seal(string(bundle))
	if e != nil {
		return domain.AuthorizationRedirect{}, e
	}
	provider, e := s.providers.Provider(ctx, connection, s.callbackURL)
	if e != nil {
		return domain.AuthorizationRedirect{}, e
	}
	now := s.now().UTC()
	expires := now.Add(domain.SSOTransactionLifetime)
	transaction := domain.SSOTransaction{StateHash: opaqueHash(state), ConnectionID: connection.ID, WorkspaceID: connection.WorkspaceID, BrowserBindingHash: opaqueHash(binding), NonceHash: opaqueHash(nonce), PKCEVerifierCipher: encrypted, ReturnPath: safeReturnPath(returnPath), ExpiresAt: expires}
	transaction.LinkUserID = linkUserID
	transaction.LinkRequestID = linkRequestID
	if e = s.repo.CreateTransaction(ctx, transaction); e != nil {
		return domain.AuthorizationRedirect{}, e
	}
	return domain.AuthorizationRedirect{URL: provider.AuthorizationURL(state, nonce, oidcrepo.S256Challenge(verifier)), ExpiresAt: expires}, nil
}

func (s *SSOService) Callback(ctx domain.Context, state, code, browserBinding string) (domain.OneTimeExchange, error) {
	if len(state) != 43 || code == "" || len(browserBinding) < 32 {
		return domain.OneTimeExchange{}, domain.ErrSSOInvalid
	}
	now := s.now().UTC()
	transaction, e := s.repo.ConsumeTransaction(ctx, opaqueHash(state), opaqueHash(browserBinding), now)
	if e != nil {
		return domain.OneTimeExchange{}, e
	}
	raw, e := s.cipher.Open(transaction.PKCEVerifierCipher)
	if e != nil {
		return domain.OneTimeExchange{}, e
	}
	var bundle verifierBundle
	if json.Unmarshal([]byte(raw), &bundle) != nil || !equalHash(opaqueHash(bundle.Nonce), transaction.NonceHash) {
		return domain.OneTimeExchange{}, domain.ErrSSOInvalid
	}
	connection, e := s.repo.GetConnectionByID(ctx, transaction.ConnectionID)
	if e != nil {
		return domain.OneTimeExchange{}, e
	}
	provider, e := s.providers.Provider(ctx, connection, s.callbackURL)
	if e != nil {
		return domain.OneTimeExchange{}, e
	}
	identity, e := provider.ExchangeAndVerify(ctx, code, bundle.Verifier, bundle.Nonce)
	if e != nil {
		return domain.OneTimeExchange{}, e
	}
	identity.ConnectionID = connection.ID
	if transaction.LinkUserID != nil {
		if transaction.LinkRequestID == nil {
			return domain.OneTimeExchange{}, domain.ErrSSOInvalid
		}
		if e = s.Link(ctx, *transaction.LinkUserID, identity, *transaction.LinkRequestID); e != nil {
			return domain.OneTimeExchange{}, e
		}
		user, e := s.repo.FindUser(ctx, *transaction.LinkUserID)
		if e != nil {
			return domain.OneTimeExchange{}, e
		}
		return s.createExchange(ctx, user, connection, browserBinding, transaction.ReturnPath, now)
	}
	linked, e := s.repo.FindIdentity(ctx, identity.Issuer, identity.Subject)
	var user *domain.User
	if e == nil {
		user, e = s.repo.FindUser(ctx, linked.UserID)
	} else if errors.Is(e, gorm.ErrRecordNotFound) {
		if existing, findErr := s.repo.FindUserByEmail(ctx, identity.Email); findErr == nil && existing != nil {
			return domain.OneTimeExchange{}, domain.ErrSSOEmailLinkDenied
		} else if findErr != nil && !errors.Is(findErr, gorm.ErrRecordNotFound) {
			return domain.OneTimeExchange{}, findErr
		}
		name := strings.TrimSpace(identity.Name)
		if name == "" {
			name = identity.Email
		}
		user = &domain.User{ID: uuid.New(), Email: identity.Email, Name: name, AvatarURL: identity.Picture}
		row := domain.ExternalIdentity{ID: uuid.New(), ConnectionID: connection.ID, Issuer: identity.Issuer, Subject: identity.Subject, Email: identity.Email, Name: name, Picture: identity.Picture, UserID: user.ID}
		if e = s.repo.CreateUserIdentityWithDefaultWorkspace(ctx, user, row); e != nil {
			return domain.OneTimeExchange{}, e
		}
	} else {
		return domain.OneTimeExchange{}, e
	}
	if e != nil {
		return domain.OneTimeExchange{}, e
	}
	return s.createExchange(ctx, user, connection, browserBinding, transaction.ReturnPath, now)
}

func (s *SSOService) createExchange(ctx domain.Context, user *domain.User, connection domain.SSOConnection, browserBinding, returnPath string, now time.Time) (domain.OneTimeExchange, error) {
	session, e := s.sessions.IssueOIDCSession(ctx, user.ID, connection.ID)
	if e != nil {
		return domain.OneTimeExchange{}, e
	}
	exchange, e := randomOpaque()
	if e != nil {
		return domain.OneTimeExchange{}, e
	}
	expires := now.Add(domain.SSOExchangeLifetime)
	if e = s.repo.CreateExchange(ctx, domain.ExchangeCode{CodeHash: opaqueHash(exchange), SessionID: session.ID, BrowserBindingHash: opaqueHash(browserBinding), ReturnPath: returnPath, ExpiresAt: expires}); e != nil {
		return domain.OneTimeExchange{}, e
	}
	return domain.OneTimeExchange{Code: exchange, ReturnPath: returnPath, ExpiresAt: expires}, nil
}
func equalHash(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	var v byte
	for i := range a {
		v |= a[i] ^ b[i]
	}
	return v == 0
}

func (s *SSOService) Exchange(ctx domain.Context, code, browserBinding string) (domain.LoginResult, error) {
	if len(code) != 43 || len(browserBinding) < 32 {
		return domain.LoginResult{}, domain.ErrSSOInvalid
	}
	row, e := s.repo.ConsumeExchange(ctx, opaqueHash(code), opaqueHash(browserBinding), s.now().UTC())
	if e != nil {
		return domain.LoginResult{}, e
	}
	session, e := s.sessions.repo.GetSession(ctx, row.SessionID)
	if e != nil {
		return domain.LoginResult{}, e
	}
	user, e := s.repo.FindUser(ctx, session.UserID)
	if e != nil {
		return domain.LoginResult{}, e
	}
	token, e := s.sessions.IssueLoginToken(user, session)
	if e != nil {
		return domain.LoginResult{}, e
	}
	return domain.LoginResult{Token: token, User: *user}, nil
}

func (s *SSOService) Link(ctx domain.Context, user uuid.UUID, identity domain.OIDCIdentity, requestID uuid.UUID) error {
	if !identity.EmailVerified || identity.ConnectionID == uuid.Nil || identity.Issuer == "" || identity.Subject == "" || requestID == uuid.Nil || user == uuid.Nil {
		return domain.ErrSSOInvalid
	}
	connection, e := s.repo.GetConnectionByID(ctx, identity.ConnectionID)
	if e != nil || connection.Issuer != identity.Issuer {
		return domain.ErrSSOInvalid
	}
	payload, _ := json.Marshal(struct {
		ConnectionID    uuid.UUID
		Issuer, Subject string
	}{identity.ConnectionID, identity.Issuer, identity.Subject})
	return s.access.WithWorkspaceLock(ctx, connection.WorkspaceID, user, func(ctx domain.Context) error {
		response, e := s.repo.WithMutation(ctx, connection.WorkspaceID, requestID, user, "link", payload, func(ctx domain.Context) (domain.StoredFeatureResponse, error) {
			if _, e := s.repo.FindUser(ctx, user); e != nil {
				return domain.StoredFeatureResponse{}, e
			}
			existing, e := s.repo.FindIdentity(ctx, identity.Issuer, identity.Subject)
			if e == nil {
				if existing.UserID != user {
					return domain.StoredFeatureResponse{}, domain.ErrSSOIdentityConflict
				}
				body, _ := json.Marshal(existing)
				return domain.StoredFeatureResponse{Status: 200, Body: body}, nil
			}
			if !errors.Is(e, gorm.ErrRecordNotFound) {
				return domain.StoredFeatureResponse{}, e
			}
			row := domain.ExternalIdentity{ID: uuid.New(), ConnectionID: connection.ID, Issuer: identity.Issuer, Subject: identity.Subject, Email: strings.ToLower(strings.TrimSpace(identity.Email)), Name: identity.Name, Picture: identity.Picture, UserID: user}
			if e = s.repo.CreateIdentity(ctx, row); e != nil {
				return domain.StoredFeatureResponse{}, e
			}
			body, _ := json.Marshal(row)
			return domain.StoredFeatureResponse{Status: 200, Body: body}, nil
		})
		_ = response
		return e
	})
}

func (s *SSOService) Configure(ctx domain.Context, p domain.Principal, workspace uuid.UUID, input domain.SSOConnectionInput) (domain.SSOConnection, error) {
	if p.UserID == nil || *p.UserID == uuid.Nil || input.RequestID == uuid.Nil || input.ClientID == "" || (input.Mode != domain.SSOModeOptional && input.Mode != domain.SSOModeEnforced) {
		return domain.SSOConnection{}, domain.ErrSSOInvalid
	}
	issuer, err := url.Parse(strings.TrimSuffix(strings.TrimSpace(input.Issuer), "/"))
	if err != nil || issuer.Scheme != "https" || issuer.Host == "" || issuer.User != nil {
		return domain.SSOConnection{}, domain.ErrSSOInvalid
	}
	secretDigest := sha256.Sum256([]byte(input.ClientSecret))
	payload, _ := json.Marshal(struct {
		Issuer, ClientID, ClientSecretHash string
		Mode                               domain.SSOConnectionMode
		PublicLinksEnabled                 *bool
		ExpectedRevision                   int64
	}{issuer.String(), input.ClientID, hex.EncodeToString(secretDigest[:]), input.Mode, input.PublicLinksEnabled, input.ExpectedRevision})
	var result domain.SSOConnection
	err = s.access.WithWorkspaceLock(ctx, workspace, *p.UserID, func(ctx domain.Context) error {
		p = principalSession(ctx, p)
		if e := s.sessions.RequireWorkspaceAssurance(ctx, p, workspace); e != nil {
			return e
		}
		if e := s.access.RequireOwner(ctx, *p.UserID, workspace); e != nil {
			return e
		}
		response, e := s.repo.WithMutation(ctx, workspace, input.RequestID, *p.UserID, "configure", payload, func(ctx domain.Context) (domain.StoredFeatureResponse, error) {
			current, e := s.repo.GetConnection(ctx, workspace)
			if errors.Is(e, domain.ErrSSONotConfigured) {
				current = domain.SSOConnection{WorkspaceID: workspace, Mode: input.Mode}
			} else if e != nil {
				return domain.StoredFeatureResponse{}, e
			} else if input.ExpectedRevision != current.Revision {
				return domain.StoredFeatureResponse{}, domain.ErrRevisionConflict
			}
			if input.ClientSecret != "" {
				current.ClientSecretCipher, e = s.cipher.Seal(input.ClientSecret)
				if e != nil {
					return domain.StoredFeatureResponse{}, e
				}
			} else if current.ID == uuid.Nil {
				return domain.StoredFeatureResponse{}, domain.ErrSSOInvalid
			}
			current.Issuer = issuer.String()
			current.ClientID = input.ClientID
			current.Mode = input.Mode
			if input.PublicLinksEnabled == nil {
				return domain.StoredFeatureResponse{}, domain.ErrSSOInvalid
			}
			current.PublicLinksEnabled = *input.PublicLinksEnabled
			if _, e = s.providers.Provider(ctx, current, s.callbackURL); e != nil {
				return domain.StoredFeatureResponse{}, e
			}
			if input.Mode == domain.SSOModeEnforced {
				if p.AuthenticatedAt.IsZero() || s.now().Sub(p.AuthenticatedAt) > domain.RecentAuthLifetime {
					return domain.StoredFeatureResponse{}, domain.ErrRecentAuthRequired
				}
				ok, e := s.sessions.repo.HasTestedBreakGlass(ctx, workspace, *p.UserID, s.now().Add(-30*24*time.Hour))
				if e != nil {
					return domain.StoredFeatureResponse{}, e
				}
				if !ok {
					return domain.StoredFeatureResponse{}, domain.ErrBreakGlassRequired
				}
			}
			current, e = s.repo.UpsertConnection(ctx, current)
			if e != nil {
				return domain.StoredFeatureResponse{}, e
			}
			body, _ := json.Marshal(current)
			return domain.StoredFeatureResponse{Status: 200, Body: body}, nil
		})
		if e != nil {
			return e
		}
		return json.Unmarshal(response.Body, &result)
	})
	return result, err
}

func (s *SSOService) GetConnection(ctx domain.Context, p domain.Principal, workspace uuid.UUID) (domain.SSOConnection, error) {
	if p.UserID == nil {
		return domain.SSOConnection{}, domain.ErrAccessDenied
	}
	p = principalSession(ctx, p)
	if e := s.sessions.RequireWorkspaceAssurance(ctx, p, workspace); e != nil {
		return domain.SSOConnection{}, e
	}
	if e := s.access.RequireOwner(ctx, *p.UserID, workspace); e != nil {
		return domain.SSOConnection{}, e
	}
	value, e := s.repo.GetConnection(ctx, workspace)
	value.ClientSecretCipher = nil
	return value, e
}

func canonicalIdentity(i domain.OIDCIdentity) string {
	sum := sha256.Sum256([]byte(i.Issuer + "\x00" + i.Subject))
	return hex.EncodeToString(sum[:])
}
