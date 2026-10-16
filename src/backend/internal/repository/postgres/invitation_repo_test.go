package postgres

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	delivery "devtool/internal/delivery/http"
	"devtool/internal/delivery/http/handler"
	"devtool/internal/domain"
	"devtool/internal/usecase"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	pg "gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func inviteFixture(t *testing.T) (*gorm.DB, domain.InvitationUsecase, uuid.UUID, uuid.UUID, uuid.UUID, *time.Time) {
	t.Helper()
	db := accessTestDB(t, 7)
	b, e := os.ReadFile("../../../migrations/000011_workspace_invitations.up.sql")
	if e != nil {
		t.Fatal(e)
	}
	mustExec(t, db, string(b))
	owner, user, w := accessFixture(t, db)
	access, e := usecase.NewAccessService(NewAccessRepository(db))
	if e != nil {
		t.Fatal(e)
	}
	now := time.Date(2026, 10, 16, 3, 0, 0, 0, time.UTC)
	return db, usecase.NewInvitationUsecase(NewInvitationRepository(db), access, "https://whiteboard.example.test", func() time.Time { return now }), owner, user, w, &now
}
func inviteToken(t *testing.T, result domain.InviteResult) string {
	t.Helper()
	u, e := url.Parse(result.InviteURL)
	if e != nil {
		t.Fatal(e)
	}
	if u.Scheme != "https" || u.Host != "whiteboard.example.test" || u.RawQuery != "" || u.Fragment != "" || !strings.HasPrefix(u.Path, "/invite/") {
		t.Fatalf("invite URL must use approved frontend route: %s", result.InviteURL)
	}
	token := strings.TrimPrefix(u.Path, "/invite/")
	if token == "" {
		t.Fatal("creation must return copyable token URL")
	}
	return token
}
func inviteFor(t *testing.T, uc domain.InvitationUsecase, owner, w uuid.UUID, email string, role domain.WorkspaceRole) domain.InviteResult {
	t.Helper()
	i, e := uc.CreateInvitation(context.Background(), owner, w, email, role)
	if e != nil {
		t.Fatalf("create invitation: %v", e)
	}
	return i
}
func TestInviteExpiryAndEmail(t *testing.T) {
	db, uc, owner, user, w, now := inviteFixture(t)
	email := user.String() + "@example.test"
	result := inviteFor(t, uc, owner, w, "  "+strings.ToUpper(email)+"  ", domain.WorkspaceRoleEditor)
	if result.Invitation.Email != email {
		t.Fatalf("email not normalized: %s", result.Invitation.Email)
	}
	if result.Invitation.ExpiresAt.Sub(result.Invitation.CreatedAt) != 7*24*time.Hour {
		t.Fatal("expiry must be exactly seven days")
	}
	token := inviteToken(t, result)
	hash := sha256.Sum256([]byte(token))
	var stored domain.Invitation
	if e := db.First(&stored, "id=?", result.Invitation.ID).Error; e != nil {
		t.Fatal(e)
	}
	if stored.TokenHash != hex.EncodeToString(hash[:]) || strings.Contains(stored.TokenHash, token) {
		t.Fatal("raw token persisted or hash mismatch")
	}
	if _, e := uc.AcceptInvitation(context.Background(), owner, token); !errors.Is(e, domain.ErrInvitationEmail) {
		t.Fatalf("wrong email admitted: %v", e)
	}
	*now = result.Invitation.ExpiresAt
	if _, e := uc.AcceptInvitation(context.Background(), user, token); !errors.Is(e, domain.ErrInvitationExpired) {
		t.Fatalf("exact expiry accepted: %v", e)
	}
	*now = now.Add(-time.Nanosecond)
	if _, e := uc.AcceptInvitation(context.Background(), user, token); e != nil {
		t.Fatalf("valid boundary refused: %v", e)
	}
	if _, e := uc.AcceptInvitation(context.Background(), user, token); !errors.Is(e, domain.ErrInvitationInvalid) {
		t.Fatalf("consumed link reusable: %v", e)
	}
}
func TestConcurrentInviteAccept(t *testing.T) {
	db, uc, owner, user, w, _ := inviteFixture(t)
	result := inviteFor(t, uc, owner, w, user.String()+"@example.test", domain.WorkspaceRoleEditor)
	token := inviteToken(t, result)
	var schema string
	db.Raw("SELECT current_schema()").Scan(&schema)
	u, _ := url.Parse(os.Getenv("ACCESS_TEST_DATABASE_URL"))
	q := u.Query()
	q.Set("search_path", schema+",public")
	u.RawQuery = q.Encode()
	other, e := gorm.Open(pg.Open(u.String()), &gorm.Config{})
	if e != nil {
		t.Fatal(e)
	}
	pool, _ := other.DB()
	defer pool.Close()
	access, e := usecase.NewAccessService(NewAccessRepository(other))
	if e != nil {
		t.Fatal(e)
	}
	second := usecase.NewInvitationUsecase(NewInvitationRepository(other), access, "https://whiteboard.example.test")
	var successes atomic.Int32
	var wg sync.WaitGroup
	start := make(chan struct{})
	for _, service := range []domain.InvitationUsecase{uc, second} {
		wg.Add(1)
		go func(service domain.InvitationUsecase) {
			defer wg.Done()
			<-start
			if _, e := service.AcceptInvitation(context.Background(), user, token); e == nil {
				successes.Add(1)
			} else if !errors.Is(e, domain.ErrInvitationInvalid) {
				t.Errorf("unexpected concurrent result: %v", e)
			}
		}(service)
	}
	close(start)
	wg.Wait()
	if successes.Load() != 1 {
		t.Fatalf("successful accepts: %d, want 1", successes.Load())
	}
}
func TestRegisterInviteAtomic(t *testing.T) {
	db, uc, owner, existing, w, _ := inviteFixture(t)
	input := domain.RegistrationInput{Email: "new@example.test", Name: "New", Password: "valid-password"}
	result := inviteFor(t, uc, owner, w, input.Email, domain.WorkspaceRoleEditor)
	token := inviteToken(t, result)
	mustExec(t, db, `CREATE FUNCTION reject_personal_fixture() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN IF NEW.name='My Workspace' THEN RAISE EXCEPTION 'provision fixture failure'; END IF; RETURN NEW; END $$; CREATE TRIGGER reject_personal_fixture BEFORE INSERT ON workspaces FOR EACH ROW EXECUTE FUNCTION reject_personal_fixture()`)
	if _, e := uc.RegisterWithInvitation(context.Background(), token, input); e == nil {
		t.Fatal("broken provisioning committed")
	}
	var n int64
	db.Table("users").Where("email=?", input.Email).Count(&n)
	if n != 0 {
		t.Fatal("registration escaped rollback")
	}
	var invite domain.Invitation
	db.First(&invite, "id=?", result.Invitation.ID)
	if invite.AcceptedAt != nil {
		t.Fatal("failed registration consumed invite")
	}
	mustExec(t, db, "DROP TRIGGER reject_personal_fixture ON workspaces; DROP FUNCTION reject_personal_fixture()")
	user, e := uc.RegisterWithInvitation(context.Background(), token, input)
	if e != nil {
		t.Fatal(e)
	}
	if user.PasswordHash == nil || bcrypt.CompareHashAndPassword([]byte(*user.PasswordHash), []byte(input.Password)) != nil {
		t.Fatal("password policy/hash mismatch")
	}
	db.Table("workspaces").Where("owner_id=? AND name='My Workspace'", user.ID).Count(&n)
	if n != 1 {
		t.Fatalf("personal workspace count: %d", n)
	}
	provision := NewUserProvisioningRepository(db)
	for i := 0; i < 2; i++ {
		if err := usecase.EnsureDefaultWorkspace(context.Background(), provision, user.ID); err != nil {
			t.Fatal(err)
		}
	}
	var personal domain.Workspace
	if err := db.Where("owner_id=?", user.ID).First(&personal).Error; err != nil {
		t.Fatal(err)
	}
	if personal.PublicID == nil || len(*personal.PublicID) != 21 {
		t.Fatal("personal workspace needs public alias")
	}
	db.Table("workspaces").Where("owner_id=?", user.ID).Count(&n)
	if n != 1 {
		t.Fatal("default workspace provisioning is not idempotent")
	}
	db.Table("workspace_members").Where("user_id=?", user.ID).Count(&n)
	if n != 2 {
		t.Fatalf("personal owner + invited membership count: %d", n)
	}
	original, _ := bcrypt.GenerateFromPassword([]byte("original-password"), bcrypt.DefaultCost)
	mustExec(t, db, "UPDATE users SET password_hash=? WHERE id=?", string(original), existing)
	existingInvite := inviteFor(t, uc, owner, w, existing.String()+"@example.test", domain.WorkspaceRoleViewer)
	input.Email = existingInvite.Invitation.Email
	if _, e = uc.RegisterWithInvitation(context.Background(), inviteToken(t, existingInvite), input); !errors.Is(e, domain.ErrRegistrationRejected) {
		t.Fatalf("existing email must log in: %v", e)
	}
	var reloaded domain.User
	db.First(&reloaded, "id=?", existing)
	if reloaded.PasswordHash == nil || *reloaded.PasswordHash != string(original) {
		t.Fatal("existing password changed")
	}
	if _, e = uc.AcceptInvitation(context.Background(), existing, inviteToken(t, existingInvite)); e != nil {
		t.Fatalf("existing invite wrongly consumed: %v", e)
	}
}
func TestInviteManagementAndNoDowngrade(t *testing.T) {
	db, uc, owner, user, w, now := inviteFixture(t)
	email := user.String() + "@example.test"
	if _, e := uc.CreateInvitation(context.Background(), user, w, email, domain.WorkspaceRoleEditor); e == nil {
		t.Fatal("nonowner create allowed")
	}
	if _, e := uc.CreateInvitation(context.Background(), owner, w, email, domain.WorkspaceRoleOwner); !errors.Is(e, domain.ErrInvitationInput) {
		t.Fatalf("owner role invite allowed: %v", e)
	}
	result := inviteFor(t, uc, owner, w, email, domain.WorkspaceRoleViewer)
	list, e := uc.ListInvitations(context.Background(), owner, w)
	if e != nil || len(list) != 1 || list[0].TokenHash != "" {
		t.Fatalf("pending metadata: %+v %v", list, e)
	}
	if e = uc.RevokeInvitation(context.Background(), user, w, result.Invitation.ID); e == nil {
		t.Fatal("nonowner revoke allowed")
	}
	if e = uc.RevokeInvitation(context.Background(), owner, w, result.Invitation.ID); e != nil {
		t.Fatal(e)
	}
	if _, e = uc.AcceptInvitation(context.Background(), user, inviteToken(t, result)); !errors.Is(e, domain.ErrInvitationInvalid) {
		t.Fatalf("revoked accepted: %v", e)
	}
	result = inviteFor(t, uc, owner, w, email, domain.WorkspaceRoleViewer)
	mustExec(t, db, "INSERT INTO workspace_members(workspace_id,user_id,role) VALUES (?,?,'editor')", w, user)
	m, e := uc.AcceptInvitation(context.Background(), user, inviteToken(t, result))
	if e != nil || m.Role != domain.WorkspaceRoleEditor {
		t.Fatalf("existing editor downgraded: %+v %v", m, e)
	}
	result = inviteFor(t, uc, owner, w, email, domain.WorkspaceRoleEditor)
	*now = result.Invitation.ExpiresAt
	list, e = uc.ListInvitations(context.Background(), owner, w)
	if e != nil || len(list) != 0 {
		t.Fatalf("inactive invitation listed: %+v %v", list, e)
	}
	down, e := os.ReadFile("../../../migrations/000011_workspace_invitations.down.sql")
	if e != nil {
		t.Fatal(e)
	}
	mustExec(t, db, string(down))
	if db.Migrator().HasTable(&domain.Invitation{}) {
		t.Fatal("invitation rollback retained table")
	}
}

func TestInviteHTTPRoutes(t *testing.T) {
	db, uc, owner, user, w, _ := inviteFixture(t)
	access, e := usecase.NewAccessService(NewAccessRepository(db))
	if e != nil {
		t.Fatal(e)
	}
	auth := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			id, e := uuid.Parse(r.Header.Get("Test-User"))
			if e != nil {
				w.WriteHeader(401)
				return
			}
			next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), domain.ContextKeyUserID, id)))
		})
	}
	router := delivery.NewRouterWithAccess(nil, nil, nil, nil, handler.NewAssetHandler(nil), auth, access, nil, nil, nil, nil, nil, nil, handler.NewInvitationHandler(uc))
	request := func(actor uuid.UUID, method, path, body string) *httptest.ResponseRecorder {
		t.Helper()
		r := httptest.NewRequest(method, "/api/v1"+path, strings.NewReader(body))
		if actor != uuid.Nil {
			r.Header.Set("Test-User", actor.String())
		}
		rr := httptest.NewRecorder()
		router.ServeHTTP(accessMemoryResponse{rr}, r)
		return rr
	}
	alias, _ := domain.NewPublicID()
	mustExec(t, db, "UPDATE workspaces SET public_id=? WHERE id=?", alias, w)
	mustExec(t, db, "INSERT INTO workspace_members(workspace_id,user_id,role) VALUES (?,?,'viewer')", w, user)
	path := "/workspaces/" + alias + "/invitations"
	body := `{"email":"` + user.String() + `@example.test","role":"editor"}`
	denied := request(user, "POST", path, body)
	if denied.Code != 403 {
		t.Fatalf("owner enforcement: %d %s", denied.Code, denied.Body.String())
	}
	created := request(owner, "POST", path, body)
	if created.Code != 201 {
		t.Fatalf("create route: %d %s", created.Code, created.Body.String())
	}
	var result domain.InviteResult
	if e = json.Unmarshal(created.Body.Bytes(), &result); e != nil {
		t.Fatal(e)
	}
	if strings.Contains(created.Body.String(), "tokenHash") {
		t.Fatal("hash leaked")
	}
	listed := request(owner, "GET", path, "")
	if listed.Code != 200 || strings.Contains(listed.Body.String(), "inviteUrl") || strings.Contains(listed.Body.String(), inviteToken(t, result)) {
		t.Fatalf("list metadata: %d %s", listed.Code, listed.Body.String())
	}
	accepted := request(user, "POST", "/invitations/accept", `{"token":"`+inviteToken(t, result)+`"}`)
	if accepted.Code != 200 {
		t.Fatalf("accept route: %d %s", accepted.Code, accepted.Body.String())
	}
	invalid := request(user, "POST", "/invitations/accept", `{"token":"`+inviteToken(t, result)+`"}`)
	if invalid.Code != 400 || !strings.Contains(invalid.Body.String(), "requestId") {
		t.Fatalf("error contract: %d %s", invalid.Code, invalid.Body.String())
	}
	result = inviteFor(t, uc, owner, w, "route-new@example.test", domain.WorkspaceRoleViewer)
	registered := request(uuid.Nil, "POST", "/invitations/register", `{"token":"`+inviteToken(t, result)+`","email":"  ROUTE-NEW@EXAMPLE.TEST  ","name":"Route user","password":"valid-password"}`)
	if registered.Code != 201 || strings.Contains(registered.Body.String(), "password") {
		t.Fatalf("public registration: %d %s", registered.Code, registered.Body.String())
	}
	result = inviteFor(t, uc, owner, w, user.String()+"@example.test", domain.WorkspaceRoleViewer)
	revoked := request(owner, "DELETE", path+"/"+result.Invitation.ID.String(), "")
	if revoked.Code != 204 {
		t.Fatalf("revoke route: %d %s", revoked.Code, revoked.Body.String())
	}
	oversized := request(owner, "POST", path, strings.Repeat("x", (1<<20)+1))
	if oversized.Code != 413 {
		t.Fatalf("unbounded owner body: %d", oversized.Code)
	}
}

func TestRegisterInviteDuplicateEmailError(t *testing.T) {
	db, _, _, user, _, _ := inviteFixture(t)
	duplicate := domain.User{ID: uuid.New(), Email: user.String() + "@example.test", Name: "Duplicate"}
	if err := NewInvitationRepository(db).CreateUser(context.Background(), &duplicate); !errors.Is(err, domain.ErrRegistrationRejected) {
		t.Fatalf("unique registration race must use generic rejection: %v", err)
	}
}
