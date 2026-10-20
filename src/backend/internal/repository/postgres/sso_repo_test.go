package postgres

import (
	"context"
	"crypto/sha256"
	"errors"
	"os"
	"testing"
	"time"

	"devtool/internal/domain"

	"github.com/google/uuid"
	pg "gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func ssoTestHash(value string) []byte { sum := sha256.Sum256([]byte(value)); return sum[:] }

func TestSSOMigrationPreservesLegacyAuthSessions(t *testing.T) {
	db := accessTestDB(t, 25)
	for _, table := range []string{"auth_sessions", "enterprise_login_sessions", "workspace_sso_connections", "sso_flow_transactions", "sso_external_identities", "sso_exchange_codes", "sso_mutation_receipts", "workspace_break_glass"} {
		var value *string
		if e := db.Raw("SELECT to_regclass(?)", table).Scan(&value).Error; e != nil || value == nil {
			t.Fatalf("missing %s: %v", table, e)
		}
	}
}
func TestSSOMigrationDownLeavesLegacyAuthSessions(t *testing.T) {
	db := accessTestDB(t, 24)
	up, e := os.ReadFile("../../../migrations/000025_workspace_sso.up.sql")
	if e != nil {
		t.Fatal(e)
	}
	if e = db.Exec(string(up)).Error; e != nil {
		t.Fatal(e)
	}
	down, e := os.ReadFile("../../../migrations/000025_workspace_sso.down.sql")
	if e != nil {
		t.Fatal(e)
	}
	if e = db.Exec(string(down)).Error; e != nil {
		t.Fatal(e)
	}
	var legacy *string
	if e = db.Raw("SELECT to_regclass('auth_sessions')").Scan(&legacy).Error; e != nil || legacy == nil {
		t.Fatalf("legacy sessions removed: %v", e)
	}
	var enterprise *string
	if e = db.Raw("SELECT to_regclass('enterprise_login_sessions')").Scan(&enterprise).Error; e != nil || enterprise != nil {
		t.Fatalf("enterprise sessions remain: %v %v", enterprise, e)
	}
}

func TestSSOSessionRepositoryRevocation(t *testing.T) {
	db := accessTestDB(t, 25)
	owner, workspace := domainRepositoryWorkspace(t, db, "SSO")
	connection := uuid.New()
	mustExec(t, db, "INSERT INTO workspace_sso_connections(id,workspace_id,issuer,client_id,client_secret_cipher,mode,public_links_enabled) VALUES (?,?,?,?,?,'enforced',false)", connection, workspace, "https://idp.example", "client", []byte("cipher"))
	repo := NewAuthSessionRepository(db)
	now := time.Now().UTC()
	row := domain.AuthSession{ID: uuid.New(), UserID: owner, Method: domain.AuthMethodOIDC, ConnectionID: &connection, WorkspaceID: &workspace, AuthenticatedAt: now, ExpiresAt: now.Add(time.Hour)}
	if e := repo.CreateSession(context.Background(), row); e != nil {
		t.Fatal(e)
	}
	got, e := repo.GetSession(context.Background(), row.ID)
	if e != nil || got.UserID != owner || got.ConnectionID == nil || *got.ConnectionID != connection || got.WorkspaceID == nil || *got.WorkspaceID != workspace {
		t.Fatalf("got=%+v err=%v", got, e)
	}
	if e = repo.RevokeWorkspaceSessions(context.Background(), workspace, owner, &connection, now.Add(time.Minute)); e != nil {
		t.Fatal(e)
	}
	got, e = repo.GetSession(context.Background(), row.ID)
	if e != nil || got.RevokedAt == nil {
		t.Fatalf("revoked=%+v err=%v", got, e)
	}
}

func TestSSOProviderRotationSerializesWithSessionAuthority(t *testing.T) {
	db := accessTestDB(t, 25)
	owner, workspace := domainRepositoryWorkspace(t, db, "SSO lock")
	connection := uuid.New()
	mustExec(t, db, "INSERT INTO workspace_sso_connections(id,workspace_id,issuer,client_id,client_secret_cipher,mode,public_links_enabled) VALUES (?,?,?,?,?,'enforced',false)", connection, workspace, "https://idp.example", "client", []byte("cipher"))
	now := time.Now().UTC()
	session := domain.AuthSession{ID: uuid.New(), UserID: owner, Method: domain.AuthMethodOIDC, ConnectionID: &connection, WorkspaceID: &workspace, AuthenticatedAt: now, ExpiresAt: now.Add(time.Hour)}
	if err := NewAuthSessionRepository(db).CreateSession(context.Background(), session); err != nil {
		t.Fatal(err)
	}
	var schema string
	if err := db.Raw("SELECT current_schema()").Scan(&schema).Error; err != nil {
		t.Fatal(err)
	}
	parallelDB, err := gorm.Open(pg.Open(os.Getenv("ACCESS_TEST_DATABASE_URL")+"&search_path="+schema), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	parallelSQL, err := parallelDB.DB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = parallelSQL.Close() })

	locked, release, authorityDone := make(chan struct{}), make(chan struct{}), make(chan error, 1)
	go func() {
		authorityDone <- NewAccessRepository(db).WithWorkspaceLock(context.Background(), workspace, owner, func(lockCtx context.Context) error {
			close(locked)
			<-release
			current, lookupErr := NewAuthSessionRepository(db).GetSession(lockCtx, session.ID)
			if lookupErr != nil || current.RevokedAt != nil {
				return errors.New("session authority changed inside the existing workspace lock")
			}
			return nil
		})
	}()
	<-locked
	revokeDone := make(chan error, 1)
	go func() {
		revokeDone <- NewAccessRepository(parallelDB).WithWorkspaceLock(context.Background(), workspace, owner, func(lockCtx context.Context) error {
			return NewAuthSessionRepository(parallelDB).RevokeConnectionSessions(lockCtx, connection, now.Add(time.Minute))
		})
	}()
	select {
	case err := <-revokeDone:
		t.Fatalf("provider rotation bypassed workspace lock: %v", err)
	case <-time.After(150 * time.Millisecond):
	}
	close(release)
	if err := <-authorityDone; err != nil {
		t.Fatal(err)
	}
	if err := <-revokeDone; err != nil {
		t.Fatal(err)
	}
	got, err := NewAuthSessionRepository(db).GetSession(context.Background(), session.ID)
	if err != nil || got.RevokedAt == nil {
		t.Fatalf("rotation did not revoke session: %+v err=%v", got, err)
	}
}

func TestSSOTransactionAndExchangeConsumeOnce(t *testing.T) {
	db := accessTestDB(t, 25)
	_, workspace := domainRepositoryWorkspace(t, db, "Flow")
	connection := uuid.New()
	mustExec(t, db, "INSERT INTO workspace_sso_connections(id,workspace_id,issuer,client_id,client_secret_cipher,mode,public_links_enabled) VALUES (?,?,?,?,?,'optional',false)", connection, workspace, "https://idp.example", "client", []byte("cipher"))
	repo := NewSSORepository(db)
	now := time.Now().UTC()
	state, browser, nonce := ssoTestHash("state"), ssoTestHash("browser"), ssoTestHash("nonce")
	if e := repo.CreateTransaction(context.Background(), domain.SSOTransaction{StateHash: state, ConnectionID: connection, WorkspaceID: workspace, BrowserBindingHash: browser, NonceHash: nonce, PKCEVerifierCipher: []byte("cipher"), ReturnPath: "/", ExpiresAt: now.Add(time.Minute)}); e != nil {
		t.Fatal(e)
	}
	if _, e := repo.ConsumeTransaction(context.Background(), state, browser, now); e != nil {
		t.Fatal(e)
	}
	if _, e := repo.ConsumeTransaction(context.Background(), state, browser, now); e == nil {
		t.Fatal("transaction replay accepted")
	}
	owner := uuid.New()
	mustExec(t, db, "INSERT INTO users(id,email,name) VALUES (?,?,'Owner')", owner, owner.String()+"@example.test")
	sessionRepo := NewAuthSessionRepository(db)
	session := domain.AuthSession{ID: uuid.New(), UserID: owner, Method: domain.AuthMethodOIDC, ConnectionID: &connection, WorkspaceID: &workspace, AuthenticatedAt: now, ExpiresAt: now.Add(time.Hour)}
	if e := sessionRepo.CreateSession(context.Background(), session); e != nil {
		t.Fatal(e)
	}
	linkState := ssoTestHash("link-state")
	requestID := uuid.New()
	if e := repo.CreateTransaction(context.Background(), domain.SSOTransaction{StateHash: linkState, ConnectionID: connection, WorkspaceID: workspace, BrowserBindingHash: browser, NonceHash: nonce, PKCEVerifierCipher: []byte("cipher"), ReturnPath: "/", LinkUserID: &owner, LinkRequestID: &requestID, SourceSessionID: &session.ID, ExpiresAt: now.Add(time.Minute)}); e != nil {
		t.Fatal(e)
	}
	linked, e := repo.ConsumeTransaction(context.Background(), linkState, browser, now)
	if e != nil || linked.SourceSessionID == nil || *linked.SourceSessionID != session.ID || linked.LinkUserID == nil || *linked.LinkUserID != owner {
		t.Fatalf("link transaction=%+v err=%v", linked, e)
	}
	code := ssoTestHash("code")
	if e := repo.CreateExchange(context.Background(), domain.ExchangeCode{CodeHash: code, SessionID: session.ID, BrowserBindingHash: browser, ReturnPath: "/", ExpiresAt: now.Add(time.Minute)}); e != nil {
		t.Fatal(e)
	}
	if _, e := repo.ConsumeExchange(context.Background(), code, browser, now); e != nil {
		t.Fatal(e)
	}
	if _, e := repo.ConsumeExchange(context.Background(), code, browser, now); e == nil {
		t.Fatal("exchange replay accepted")
	}
}

func TestSSOMutationReceiptExactRetryAndReuse(t *testing.T) {
	db := accessTestDB(t, 25)
	owner, workspace := domainRepositoryWorkspace(t, db, "Receipt")
	repo := NewSSORepository(db)
	requestID := uuid.New()
	calls := 0
	fn := func(context.Context) (domain.StoredFeatureResponse, error) {
		calls++
		return domain.StoredFeatureResponse{Status: 200, Body: []byte(`{"ok":true}`)}, nil
	}
	first, e := repo.WithMutation(context.Background(), workspace, requestID, owner, "configure", []byte(`{"mode":"optional"}`), fn)
	if e != nil {
		t.Fatal(e)
	}
	second, e := repo.WithMutation(context.Background(), workspace, requestID, owner, "configure", []byte(`{"mode":"optional"}`), fn)
	if e != nil || calls != 1 || string(first.Body) != string(second.Body) {
		t.Fatalf("retry calls=%d first=%s second=%s err=%v", calls, first.Body, second.Body, e)
	}
	if _, e = repo.WithMutation(context.Background(), workspace, requestID, owner, "link", []byte(`{"mode":"optional"}`), fn); !errors.Is(e, domain.ErrRequestIDReused) {
		t.Fatalf("cross-action=%v", e)
	}
	if _, e = repo.WithMutation(context.Background(), workspace, requestID, owner, "configure", []byte(`{"mode":"enforced"}`), fn); !errors.Is(e, domain.ErrRequestIDReused) {
		t.Fatalf("cross-body=%v", e)
	}
}
