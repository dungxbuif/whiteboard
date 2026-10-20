package usecase

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/url"
	"strings"
	"testing"
	"time"

	"devtool/internal/domain"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type testCipher struct{}

func (testCipher) Seal(v string) ([]byte, error) { return []byte(v), nil }
func (testCipher) Open(v []byte) (string, error) { return string(v), nil }

type ssoRepoStub struct {
	connection domain.SSOConnection
	tx         map[string]domain.SSOTransaction
	exchanges  map[string]domain.ExchangeCode
	identities map[string]domain.ExternalIdentity
	users      map[uuid.UUID]*domain.User
	emails     map[string]*domain.User
}

func newSSORepoStub() *ssoRepoStub {
	return &ssoRepoStub{tx: map[string]domain.SSOTransaction{}, exchanges: map[string]domain.ExchangeCode{}, identities: map[string]domain.ExternalIdentity{}, users: map[uuid.UUID]*domain.User{}, emails: map[string]*domain.User{}}
}
func (r *ssoRepoStub) GetConnection(_ context.Context, workspace uuid.UUID) (domain.SSOConnection, error) {
	if r.connection.ID == uuid.Nil || r.connection.WorkspaceID != workspace {
		return domain.SSOConnection{}, domain.ErrSSONotConfigured
	}
	return r.connection, nil
}
func (r *ssoRepoStub) GetConnectionByID(_ context.Context, id uuid.UUID) (domain.SSOConnection, error) {
	if id != r.connection.ID {
		return domain.SSOConnection{}, domain.ErrSSONotConfigured
	}
	return r.connection, nil
}
func (r *ssoRepoStub) UpsertConnection(_ context.Context, c domain.SSOConnection) (domain.SSOConnection, error) {
	r.connection = c
	return c, nil
}
func (r *ssoRepoStub) CreateTransaction(_ context.Context, v domain.SSOTransaction) error {
	r.tx[hex.EncodeToString(v.StateHash)] = v
	return nil
}
func (r *ssoRepoStub) ConsumeTransaction(_ context.Context, state, browser []byte, now time.Time) (domain.SSOTransaction, error) {
	key := hex.EncodeToString(state)
	v, ok := r.tx[key]
	if !ok {
		return v, domain.ErrSSOInvalid
	}
	if v.ConsumedAt != nil {
		return v, domain.ErrSSOReplay
	}
	if !now.Before(v.ExpiresAt) {
		return v, domain.ErrSSOExpired
	}
	if !equalHash(browser, v.BrowserBindingHash) {
		return v, domain.ErrSSOInvalid
	}
	v.ConsumedAt = &now
	r.tx[key] = v
	return v, nil
}
func (r *ssoRepoStub) FindIdentity(_ context.Context, issuer, subject string) (domain.ExternalIdentity, error) {
	v, ok := r.identities[issuer+"\x00"+subject]
	if !ok {
		return v, gorm.ErrRecordNotFound
	}
	return v, nil
}
func (r *ssoRepoStub) FindIdentityForUser(ctx context.Context, user uuid.UUID, issuer, subject string) (domain.ExternalIdentity, error) {
	v, e := r.FindIdentity(ctx, issuer, subject)
	if e != nil || v.UserID != user {
		return domain.ExternalIdentity{}, gorm.ErrRecordNotFound
	}
	return v, nil
}
func (r *ssoRepoStub) CreateIdentity(_ context.Context, v domain.ExternalIdentity) error {
	r.identities[v.Issuer+"\x00"+v.Subject] = v
	return nil
}
func (r *ssoRepoStub) CreateExchange(_ context.Context, v domain.ExchangeCode) error {
	r.exchanges[hex.EncodeToString(v.CodeHash)] = v
	return nil
}
func (r *ssoRepoStub) ConsumeExchange(_ context.Context, code, browser []byte, now time.Time) (domain.ExchangeCode, error) {
	k := hex.EncodeToString(code)
	v, ok := r.exchanges[k]
	if !ok {
		return v, domain.ErrSSOInvalid
	}
	if v.ConsumedAt != nil {
		return v, domain.ErrSSOReplay
	}
	if !equalHash(v.BrowserBindingHash, browser) {
		return v, domain.ErrSSOInvalid
	}
	v.ConsumedAt = &now
	r.exchanges[k] = v
	return v, nil
}
func (r *ssoRepoStub) FindUser(_ context.Context, id uuid.UUID) (*domain.User, error) {
	v, ok := r.users[id]
	if !ok {
		return nil, gorm.ErrRecordNotFound
	}
	return v, nil
}
func (r *ssoRepoStub) FindUserByEmail(_ context.Context, email string) (*domain.User, error) {
	v, ok := r.emails[email]
	if !ok {
		return nil, gorm.ErrRecordNotFound
	}
	return v, nil
}
func (r *ssoRepoStub) CreateUserWithDefaultWorkspace(_ context.Context, u *domain.User) error {
	r.users[u.ID] = u
	r.emails[u.Email] = u
	return nil
}
func (r *ssoRepoStub) CreateUserIdentityWithDefaultWorkspace(ctx context.Context, u *domain.User, i domain.ExternalIdentity) error {
	if e := r.CreateUserWithDefaultWorkspace(ctx, u); e != nil {
		return e
	}
	return r.CreateIdentity(ctx, i)
}
func (r *ssoRepoStub) WithMutation(ctx context.Context, _, _, _ uuid.UUID, _ string, _ []byte, fn func(context.Context) (domain.StoredFeatureResponse, error)) (domain.StoredFeatureResponse, error) {
	return fn(ctx)
}

type sessionRepoStub struct {
	rows       map[uuid.UUID]domain.AuthSession
	connection domain.SSOConnection
	breakGlass bool
}

func (s *sessionRepoStub) CreateSession(_ context.Context, v domain.AuthSession) error {
	if s.rows == nil {
		s.rows = map[uuid.UUID]domain.AuthSession{}
	}
	s.rows[v.ID] = v
	return nil
}
func (s *sessionRepoStub) GetSession(_ context.Context, id uuid.UUID) (domain.AuthSession, error) {
	v, ok := s.rows[id]
	if !ok {
		return v, domain.ErrSessionRevoked
	}
	return v, nil
}
func (s *sessionRepoStub) RevokeWorkspaceSessions(_ context.Context, _, user uuid.UUID, connection *uuid.UUID, now time.Time) error {
	for id, v := range s.rows {
		if v.UserID == user && (connection == nil || v.ConnectionID != nil && *v.ConnectionID == *connection) {
			v.RevokedAt = &now
			s.rows[id] = v
		}
	}
	return nil
}
func (s *sessionRepoStub) RevokeConnectionSessions(_ context.Context, connection uuid.UUID, now time.Time) error {
	for id, value := range s.rows {
		if value.ConnectionID != nil && *value.ConnectionID == connection {
			value.RevokedAt = &now
			s.rows[id] = value
		}
	}
	return nil
}
func (s *sessionRepoStub) HasTestedBreakGlass(context.Context, uuid.UUID, uuid.UUID, time.Time) (bool, error) {
	return s.breakGlass, nil
}

type ssoAccessStub struct {
	owner     uuid.UUID
	deniedNow *bool
}

func (a ssoAccessStub) Resolve(context.Context, domain.Principal, domain.SceneRef) (domain.AccessDecision, error) {
	return domain.AccessDecision{Access: domain.AccessEdit, Owner: true}, nil
}
func (a ssoAccessStub) ResolveWorkspace(context.Context, domain.Principal, uuid.UUID) (domain.AccessDecision, error) {
	if a.deniedNow != nil && *a.deniedNow {
		return domain.AccessDecision{}, domain.ErrAccessDenied
	}
	return domain.AccessDecision{Access: domain.AccessEdit, Owner: true}, nil
}
func (a ssoAccessStub) RequireOwner(_ context.Context, user, _ uuid.UUID) error {
	if user != a.owner {
		return domain.ErrAccessDenied
	}
	return nil
}
func (ssoAccessStub) RequireFolderEdit(context.Context, domain.Principal, uuid.UUID, uuid.UUID) error {
	return nil
}
func (ssoAccessStub) ResolveWorkspaceID(context.Context, string) (uuid.UUID, error) {
	return uuid.Nil, nil
}
func (ssoAccessStub) ResolveSceneID(context.Context, uuid.UUID, string) (uuid.UUID, error) {
	return uuid.Nil, nil
}
func (ssoAccessStub) WithWorkspaceLock(ctx context.Context, _, _ uuid.UUID, fn func(context.Context) error) error {
	return fn(ctx)
}
func (ssoAccessStub) ListMembers(context.Context, uuid.UUID, uuid.UUID) ([]domain.WorkspaceMember, error) {
	return nil, nil
}
func (ssoAccessStub) SetMember(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, *domain.WorkspaceRole) error {
	return nil
}
func (ssoAccessStub) SetOverride(context.Context, uuid.UUID, domain.SceneRef, uuid.UUID, *domain.WorkspaceRole) error {
	return nil
}

type providerStub struct {
	identity        domain.OIDCIdentity
	verifier, nonce string
}

func (p *providerStub) AuthorizationURL(state, nonce, challenge string) string {
	return "https://idp.example/authorize?state=" + url.QueryEscape(state) + "&nonce=" + url.QueryEscape(nonce) + "&code_challenge=" + url.QueryEscape(challenge)
}
func (p *providerStub) ExchangeAndVerify(_ context.Context, _, verifier, nonce string) (domain.OIDCIdentity, error) {
	p.verifier = verifier
	p.nonce = nonce
	return p.identity, nil
}

type providerFactoryStub struct{ p *providerStub }

func (f providerFactoryStub) Provider(context.Context, domain.SSOConnection, string) (domain.OIDCProvider, error) {
	return f.p, nil
}

func TestOIDCStateNoncePKCEReplay(t *testing.T) {
	now := time.Date(2026, 10, 20, 1, 0, 0, 0, time.UTC)
	owner := uuid.New()
	workspace := uuid.New()
	connection := domain.SSOConnection{ID: uuid.New(), WorkspaceID: workspace, Issuer: "https://idp.example", ClientID: "client", ClientSecretCipher: []byte("secret"), Mode: domain.SSOModeOptional}
	repo := newSSORepoStub()
	repo.connection = connection
	sessionRepo := &sessionRepoStub{rows: map[uuid.UUID]domain.AuthSession{}}
	sessions, _ := NewSessionService(sessionRepo, repo, []byte("01234567890123456789012345678901"), func() time.Time { return now })
	provider := &providerStub{identity: domain.OIDCIdentity{Issuer: connection.Issuer, Subject: "subject", Email: "new@example.test", EmailVerified: true, Name: "New"}}
	service, _ := NewSSOService(repo, ssoAccessStub{owner: owner}, sessions, providerFactoryStub{provider}, testCipher{}, "https://app.example/api/v1/sso/callback", func() time.Time { return now })
	binding := "browser-binding-01234567890123456789"
	ctx := context.WithValue(context.Background(), domain.ContextKeyBrowserBinding, binding)
	redirect, e := service.Begin(ctx, connection.ID, "/w/demo")
	if e != nil {
		t.Fatal(e)
	}
	parsed, _ := url.Parse(redirect.URL)
	state := parsed.Query().Get("state")
	if state == "" || parsed.Query().Get("nonce") == "" || parsed.Query().Get("code_challenge") == "" {
		t.Fatalf("redirect=%s", redirect.URL)
	}
	exchange, e := service.Callback(context.Background(), state, "code", binding)
	if e != nil {
		t.Fatal(e)
	}
	if exchange.Code == "" || provider.verifier == "" || provider.nonce == "" {
		t.Fatalf("exchange=%+v verifier=%q nonce=%q", exchange, provider.verifier, provider.nonce)
	}
	if _, e = service.Callback(context.Background(), state, "code", binding); !errors.Is(e, domain.ErrSSOReplay) {
		t.Fatalf("replay=%v", e)
	}
	login, e := service.Exchange(context.Background(), exchange.Code, binding)
	if e != nil || login.Token == "" {
		t.Fatalf("login=%+v err=%v", login, e)
	}
	if _, e = service.Exchange(context.Background(), exchange.Code, binding); !errors.Is(e, domain.ErrSSOReplay) {
		t.Fatalf("exchange replay=%v", e)
	}
}

func TestSameEmailCannotLinkAccount(t *testing.T) {
	now := time.Now()
	connection := domain.SSOConnection{ID: uuid.New(), WorkspaceID: uuid.New(), Issuer: "https://idp.example", ClientID: "client", ClientSecretCipher: []byte("secret"), Mode: domain.SSOModeOptional}
	repo := newSSORepoStub()
	repo.connection = connection
	existing := &domain.User{ID: uuid.New(), Email: "same@example.test", Name: "Existing"}
	repo.users[existing.ID] = existing
	repo.emails[existing.Email] = existing
	sessionRepo := &sessionRepoStub{rows: map[uuid.UUID]domain.AuthSession{}}
	sessions, _ := NewSessionService(sessionRepo, repo, []byte("01234567890123456789012345678901"), func() time.Time { return now })
	provider := &providerStub{identity: domain.OIDCIdentity{Issuer: connection.Issuer, Subject: "new-sub", Email: existing.Email, EmailVerified: true}}
	service, _ := NewSSOService(repo, ssoAccessStub{}, sessions, providerFactoryStub{provider}, testCipher{}, "https://app.example/callback", func() time.Time { return now })
	binding := "browser-binding-01234567890123456789"
	ctx := context.WithValue(context.Background(), domain.ContextKeyBrowserBinding, binding)
	redirect, _ := service.Begin(ctx, connection.ID, "/")
	u, _ := url.Parse(redirect.URL)
	_, e := service.Callback(context.Background(), u.Query().Get("state"), "code", binding)
	if !errors.Is(e, domain.ErrSSOEmailLinkDenied) {
		t.Fatalf("error=%v", e)
	}
}

func TestAuthenticatedLinkFlowBindsVerifiedIdentity(t *testing.T) {
	now := time.Date(2026, 10, 20, 2, 0, 0, 0, time.UTC)
	userID := uuid.New()
	connection := domain.SSOConnection{ID: uuid.New(), WorkspaceID: uuid.New(), Issuer: "https://idp.example", ClientID: "client", ClientSecretCipher: []byte("secret"), Mode: domain.SSOModeOptional}
	repo := newSSORepoStub()
	repo.connection = connection
	user := &domain.User{ID: userID, Email: "existing@example.test", Name: "Existing"}
	repo.users[userID] = user
	repo.emails[user.Email] = user
	sessions, _ := NewSessionService(&sessionRepoStub{rows: map[uuid.UUID]domain.AuthSession{}}, repo, []byte("01234567890123456789012345678901"), func() time.Time { return now })
	provider := &providerStub{identity: domain.OIDCIdentity{Issuer: connection.Issuer, Subject: "linked-subject", Email: user.Email, EmailVerified: true}}
	service, _ := NewSSOService(repo, ssoAccessStub{}, sessions, providerFactoryStub{provider}, testCipher{}, "https://app.example/callback", func() time.Time { return now })
	binding := "browser-binding-01234567890123456789"
	ctx := context.WithValue(context.Background(), domain.ContextKeyBrowserBinding, binding)
	source, _ := sessions.IssuePasswordSession(context.Background(), userID)
	principal := domain.UserPrincipal(userID)
	principal.SessionID = source
	redirect, err := service.BeginLink(ctx, principal, connection.ID, "/w/demo", uuid.New())
	if err != nil {
		t.Fatal(err)
	}
	parsed, _ := url.Parse(redirect.URL)
	exchange, err := service.Callback(context.Background(), parsed.Query().Get("state"), "code", binding)
	if err != nil || exchange.Code == "" {
		t.Fatalf("exchange=%+v err=%v", exchange, err)
	}
	linked, err := repo.FindIdentity(context.Background(), connection.Issuer, "linked-subject")
	if err != nil || linked.UserID != userID {
		t.Fatalf("identity=%+v err=%v", linked, err)
	}
}

func TestPasswordSessionCannotEnterEnforcedWorkspace(t *testing.T) {
	now := time.Now()
	workspace := uuid.New()
	user := uuid.New()
	repo := newSSORepoStub()
	repo.connection = domain.SSOConnection{ID: uuid.New(), WorkspaceID: workspace, Mode: domain.SSOModeEnforced}
	sessionsRepo := &sessionRepoStub{rows: map[uuid.UUID]domain.AuthSession{}}
	sessions, _ := NewSessionService(sessionsRepo, repo, []byte("01234567890123456789012345678901"), func() time.Time { return now })
	sid, _ := sessions.IssuePasswordSession(context.Background(), user)
	p := domain.UserPrincipal(user)
	p.SessionID = sid
	if e := sessions.RequireWorkspaceAssurance(context.Background(), p, workspace); !errors.Is(e, domain.ErrWorkspaceAssurance) {
		t.Fatalf("error=%v", e)
	}
}
func TestSSORevokedSessionCannotRenewRoomOrAssets(t *testing.T) {
	now := time.Now()
	workspace := uuid.New()
	user := uuid.New()
	repo := newSSORepoStub()
	repo.connection = domain.SSOConnection{ID: uuid.New(), WorkspaceID: workspace, Mode: domain.SSOModeOptional}
	sessionsRepo := &sessionRepoStub{rows: map[uuid.UUID]domain.AuthSession{}}
	sessions, _ := NewSessionService(sessionsRepo, repo, []byte("01234567890123456789012345678901"), func() time.Time { return now })
	row, _ := sessions.IssueOIDCSession(context.Background(), user, repo.connection.ID)
	p := domain.UserPrincipal(user)
	p.SessionID = row.ID
	if e := sessions.RevokeWorkspaceSessions(context.Background(), workspace, user, &repo.connection.ID); e != nil {
		t.Fatal(e)
	}
	if e := sessions.RequireWorkspaceAssurance(context.Background(), p, workspace); !errors.Is(e, domain.ErrSessionRevoked) {
		t.Fatalf("error=%v", e)
	}
}
func TestBreakGlassOwnerRecovery(t *testing.T) {
	now := time.Now()
	workspace := uuid.New()
	user := uuid.New()
	repo := newSSORepoStub()
	repo.connection = domain.SSOConnection{ID: uuid.New(), WorkspaceID: workspace, Mode: domain.SSOModeEnforced}
	sessionsRepo := &sessionRepoStub{rows: map[uuid.UUID]domain.AuthSession{}, breakGlass: true}
	sessions, _ := NewSessionService(sessionsRepo, repo, []byte("01234567890123456789012345678901"), func() time.Time { return now })
	row, e := sessions.IssueBreakGlassSession(context.Background(), user, workspace)
	if e != nil {
		t.Fatal(e)
	}
	p := domain.UserPrincipal(user)
	p.SessionID = row.ID
	p.RecoveryWorkspaceID = workspace
	if row.WorkspaceID == nil || *row.WorkspaceID != workspace || row.ExpiresAt.Sub(row.AuthenticatedAt) != domain.BreakGlassSessionLifetime {
		t.Fatalf("unscoped recovery session=%+v", row)
	}
	if e = sessions.RequireWorkspaceAssurance(context.Background(), p, workspace); e != nil {
		t.Fatal(e)
	}
	repo.connection.Mode = domain.SSOModeOptional
	if e = sessions.RequireWorkspaceAssurance(context.Background(), p, workspace); !errors.Is(e, domain.ErrWorkspaceAssurance) {
		t.Fatalf("optional workspace accepted recovery session: %v", e)
	}
	repo.connection = domain.SSOConnection{}
	if e = sessions.RequireWorkspaceAssurance(context.Background(), p, workspace); !errors.Is(e, domain.ErrWorkspaceAssurance) {
		t.Fatalf("unconfigured workspace accepted recovery session: %v", e)
	}
}

func TestRevokedSessionCannotExchange(t *testing.T) {
	now := time.Date(2026, 10, 20, 3, 0, 0, 0, time.UTC)
	workspace, userID := uuid.New(), uuid.New()
	repo := newSSORepoStub()
	repo.connection = domain.SSOConnection{ID: uuid.New(), WorkspaceID: workspace, Issuer: "https://idp.example", Mode: domain.SSOModeOptional}
	user := &domain.User{ID: userID, Email: "user@example.test"}
	repo.users[userID] = user
	sessionRepo := &sessionRepoStub{rows: map[uuid.UUID]domain.AuthSession{}}
	sessions, _ := NewSessionService(sessionRepo, repo, []byte("01234567890123456789012345678901"), func() time.Time { return now })
	service, _ := NewSSOService(repo, ssoAccessStub{}, sessions, providerFactoryStub{&providerStub{}}, testCipher{}, "https://app.example/callback", func() time.Time { return now })
	row, _ := sessions.IssueOIDCSession(context.Background(), userID, repo.connection.ID)
	code, binding := strings.Repeat("c", 43), "browser-binding-01234567890123456789"
	repo.exchanges[hex.EncodeToString(opaqueHash(code))] = domain.ExchangeCode{CodeHash: opaqueHash(code), SessionID: row.ID, BrowserBindingHash: opaqueHash(binding), ExpiresAt: now.Add(time.Minute)}
	revoked := now
	stored := sessionRepo.rows[row.ID]
	stored.RevokedAt = &revoked
	sessionRepo.rows[row.ID] = stored
	if _, err := service.Exchange(context.Background(), code, binding); !errors.Is(err, domain.ErrSessionRevoked) {
		t.Fatalf("revoked exchange err=%v", err)
	}
}

func TestLinkCallbackRechecksSourceSession(t *testing.T) {
	now := time.Date(2026, 10, 20, 4, 0, 0, 0, time.UTC)
	workspace, userID := uuid.New(), uuid.New()
	repo := newSSORepoStub()
	repo.connection = domain.SSOConnection{ID: uuid.New(), WorkspaceID: workspace, Issuer: "https://idp.example", ClientID: "client", ClientSecretCipher: []byte("secret"), Mode: domain.SSOModeOptional}
	user := &domain.User{ID: userID, Email: "user@example.test"}
	repo.users[userID], repo.emails[user.Email] = user, user
	sessionRepo := &sessionRepoStub{rows: map[uuid.UUID]domain.AuthSession{}}
	sessions, _ := NewSessionService(sessionRepo, repo, []byte("01234567890123456789012345678901"), func() time.Time { return now })
	provider := &providerStub{identity: domain.OIDCIdentity{Issuer: repo.connection.Issuer, Subject: "subject", Email: user.Email, EmailVerified: true}}
	service, _ := NewSSOService(repo, ssoAccessStub{}, sessions, providerFactoryStub{provider}, testCipher{}, "https://app.example/callback", func() time.Time { return now })
	source, _ := sessions.create(context.Background(), userID, domain.AuthMethodPassword, nil, nil, domain.DefaultSessionDuration)
	p := domain.UserPrincipal(userID)
	p.SessionID = source.ID
	binding := "browser-binding-01234567890123456789"
	ctx := context.WithValue(context.Background(), domain.ContextKeyBrowserBinding, binding)
	redirect, err := service.BeginLink(ctx, p, repo.connection.ID, "/", uuid.New())
	if err != nil {
		t.Fatal(err)
	}
	stored := sessionRepo.rows[source.ID]
	revoked := now
	stored.RevokedAt = &revoked
	sessionRepo.rows[source.ID] = stored
	parsed, _ := url.Parse(redirect.URL)
	if _, err = service.Callback(context.Background(), parsed.Query().Get("state"), "code", binding); !errors.Is(err, domain.ErrSessionRevoked) {
		t.Fatalf("revoked link source err=%v", err)
	}
}

func TestLinkCallbackRechecksCurrentMembership(t *testing.T) {
	now := time.Date(2026, 10, 20, 5, 0, 0, 0, time.UTC)
	workspace, userID := uuid.New(), uuid.New()
	repo := newSSORepoStub()
	repo.connection = domain.SSOConnection{ID: uuid.New(), WorkspaceID: workspace, Issuer: "https://idp.example", ClientID: "client", ClientSecretCipher: []byte("secret"), Mode: domain.SSOModeOptional}
	user := &domain.User{ID: userID, Email: "user@example.test"}
	repo.users[userID], repo.emails[user.Email] = user, user
	sessionRepo := &sessionRepoStub{rows: map[uuid.UUID]domain.AuthSession{}}
	sessions, _ := NewSessionService(sessionRepo, repo, []byte("01234567890123456789012345678901"), func() time.Time { return now })
	provider := &providerStub{identity: domain.OIDCIdentity{Issuer: repo.connection.Issuer, Subject: "subject", Email: user.Email, EmailVerified: true}}
	denied := false
	service, _ := NewSSOService(repo, ssoAccessStub{deniedNow: &denied}, sessions, providerFactoryStub{provider}, testCipher{}, "https://app.example/callback", func() time.Time { return now })
	source, _ := sessions.IssuePasswordSession(context.Background(), userID)
	principal := domain.UserPrincipal(userID)
	principal.SessionID = source
	binding := "browser-binding-01234567890123456789"
	ctx := context.WithValue(context.Background(), domain.ContextKeyBrowserBinding, binding)
	redirect, err := service.BeginLink(ctx, principal, repo.connection.ID, "/", uuid.New())
	if err != nil {
		t.Fatal(err)
	}
	denied = true
	parsed, _ := url.Parse(redirect.URL)
	if _, err = service.Callback(context.Background(), parsed.Query().Get("state"), "code", binding); !errors.Is(err, domain.ErrAccessDenied) {
		t.Fatalf("removed member link err=%v", err)
	}
}

func TestProviderSecretRotationRevokesConnectionSessions(t *testing.T) {
	now := time.Date(2026, 10, 20, 6, 0, 0, 0, time.UTC)
	workspace, owner := uuid.New(), uuid.New()
	repo := newSSORepoStub()
	repo.connection = domain.SSOConnection{ID: uuid.New(), WorkspaceID: workspace, Issuer: "https://idp.example", ClientID: "client", ClientSecretCipher: []byte("old-secret"), Mode: domain.SSOModeOptional, Revision: 1}
	repo.users[owner] = &domain.User{ID: owner, Email: "owner@example.test"}
	sessionRepo := &sessionRepoStub{rows: map[uuid.UUID]domain.AuthSession{}}
	sessions, _ := NewSessionService(sessionRepo, repo, []byte("01234567890123456789012345678901"), func() time.Time { return now })
	row, _ := sessions.IssueOIDCSession(context.Background(), owner, repo.connection.ID)
	service, _ := NewSSOService(repo, ssoAccessStub{owner: owner}, sessions, providerFactoryStub{&providerStub{}}, testCipher{}, "https://app.example/callback", func() time.Time { return now })
	principal := domain.UserPrincipal(owner)
	principal.SessionID, principal.AuthMethod, principal.AuthenticatedAt = row.ID, domain.AuthMethodOIDC, now
	publicLinks := false
	_, err := service.Configure(context.Background(), principal, workspace, domain.SSOConnectionInput{Issuer: repo.connection.Issuer, ClientID: repo.connection.ClientID, ClientSecret: "rotated-secret", Mode: domain.SSOModeOptional, PublicLinksEnabled: &publicLinks, RequestID: uuid.New(), ExpectedRevision: 1})
	if err != nil {
		t.Fatal(err)
	}
	if sessionRepo.rows[row.ID].RevokedAt == nil {
		t.Fatal("provider secret rotation left old connection session active")
	}
}

func TestDirectLinkRequiresCurrentSourceSession(t *testing.T) {
	userID := uuid.New()
	repo := newSSORepoStub()
	repo.connection = domain.SSOConnection{ID: uuid.New(), WorkspaceID: uuid.New(), Issuer: "https://idp.example", Mode: domain.SSOModeOptional}
	repo.users[userID] = &domain.User{ID: userID, Email: "user@example.test"}
	sessions, _ := NewSessionService(&sessionRepoStub{rows: map[uuid.UUID]domain.AuthSession{}}, repo, []byte("01234567890123456789012345678901"))
	service, _ := NewSSOService(repo, ssoAccessStub{}, sessions, providerFactoryStub{&providerStub{}}, testCipher{}, "https://app.example/callback")
	err := service.Link(context.Background(), userID, domain.OIDCIdentity{ConnectionID: repo.connection.ID, Issuer: repo.connection.Issuer, Subject: "subject", Email: "user@example.test", EmailVerified: true}, uuid.New())
	if !errors.Is(err, domain.ErrSessionRevoked) {
		t.Fatalf("link without source session err=%v", err)
	}
}

func TestPublicLinksFollowExplicitWorkspaceSSOPolicy(t *testing.T) {
	repo := newSSORepoStub()
	workspace := uuid.New()
	repo.connection = domain.SSOConnection{ID: uuid.New(), WorkspaceID: workspace, Mode: domain.SSOModeOptional, PublicLinksEnabled: false}
	sessions, _ := NewSessionService(&sessionRepoStub{}, repo, []byte("01234567890123456789012345678901"))
	if err := sessions.RequirePublicLinkAccess(context.Background(), workspace); !errors.Is(err, domain.ErrAccessDenied) {
		t.Fatalf("disabled public links err=%v", err)
	}
	repo.connection.PublicLinksEnabled = true
	if err := sessions.RequirePublicLinkAccess(context.Background(), workspace); err != nil {
		t.Fatalf("enabled public links err=%v", err)
	}
}

func TestSSOConfigureCanonicalReceiptPayloadOmitsSecret(t *testing.T) {
	input := domain.SSOConnectionInput{Issuer: "https://idp.example", ClientID: "client", ClientSecret: "top-secret", Mode: domain.SSOModeOptional, RequestID: uuid.New()}
	payload, _ := json.Marshal(struct {
		Issuer, ClientID   string
		Mode               domain.SSOConnectionMode
		PublicLinksEnabled *bool
		ExpectedRevision   int64
	}{input.Issuer, input.ClientID, input.Mode, input.PublicLinksEnabled, input.ExpectedRevision})
	if strings.Contains(string(payload), input.ClientSecret) {
		t.Fatal("secret entered receipt payload")
	}
}

type assuranceProbe struct {
	principal domain.Principal
	err       error
}

func (a *assuranceProbe) RequireWorkspaceAssurance(_ context.Context, p domain.Principal, _ uuid.UUID) error {
	a.principal = p
	return a.err
}
func (a *assuranceProbe) RequirePublicLinkAccess(context.Context, uuid.UUID) error { return a.err }
func (a *assuranceProbe) RevokeWorkspaceSessions(context.Context, uuid.UUID, uuid.UUID, *uuid.UUID) error {
	return nil
}
func TestAccessChecksCurrentSessionBeforeWorkspaceDecision(t *testing.T) {
	user, sid := uuid.New(), uuid.New()
	probe := &assuranceProbe{err: domain.ErrSessionRevoked}
	service, e := NewAccessServiceWithAssurance(accessStateRepo{state: domain.AccessState{Role: domain.WorkspaceRoleOwner, ACLVersion: 1}}, probe)
	if e != nil {
		t.Fatal(e)
	}
	ctx := context.WithValue(context.Background(), domain.ContextKeySessionID, sid)
	ctx = context.WithValue(ctx, domain.ContextKeyAuthMethod, domain.AuthMethodOIDC)
	if _, e = service.ResolveWorkspace(ctx, domain.UserPrincipal(user), uuid.New()); !errors.Is(e, domain.ErrSessionRevoked) {
		t.Fatalf("error=%v", e)
	}
	if probe.principal.SessionID != sid || probe.principal.AuthMethod != domain.AuthMethodOIDC {
		t.Fatalf("principal=%+v", probe.principal)
	}
}

type deniedAssetUploader struct{ domain.PrincipalAssetUploader }

func TestRevokedSessionCannotMintCurrentRoomOrAssetAuthority(t *testing.T) {
	user, sid := uuid.New(), uuid.New()
	ref := domain.SceneRef{WorkspaceID: uuid.New(), SceneID: uuid.New()}
	probe := &assuranceProbe{err: domain.ErrSessionRevoked}
	access, err := NewAccessServiceWithAssurance(accessStateRepo{state: domain.AccessState{Role: domain.WorkspaceRoleEditor, ACLVersion: 1}}, probe)
	if err != nil {
		t.Fatal(err)
	}
	principal := domain.UserPrincipal(user)
	principal.SessionID = sid
	principal.AuthMethod = domain.AuthMethodOIDC
	principal.CredentialExpiresAt = time.Now().Add(time.Hour)
	rooms, _ := NewRoomService(nil, access, []byte("0123456789abcdef0123456789abcdef"))
	if _, err = rooms.MintRoomToken(context.Background(), principal, ref); !errors.Is(err, domain.ErrSessionRevoked) {
		t.Fatalf("room authority err=%v", err)
	}
	capabilities, _ := NewCapabilityService(access, nil, deniedAssetUploader{}, []byte("0123456789abcdef0123456789abcdef"))
	request := []domain.AssetUploadRequest{{FileID: "file-1", MimeType: "image/png", ByteSize: 1}}
	if _, err = capabilities.CreateAssetCapability(context.Background(), principal, ref, request); !errors.Is(err, domain.ErrSessionRevoked) {
		t.Fatalf("asset authority err=%v", err)
	}
}
