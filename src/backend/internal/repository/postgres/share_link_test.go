package postgres

import (
	"bytes"
	"context"
	delivery "devtool/internal/delivery/http"
	"devtool/internal/delivery/http/handler"
	"devtool/internal/delivery/http/middleware"
	"devtool/internal/domain"
	"devtool/internal/usecase"
	"encoding/json"
	"errors"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"log"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

func publicLinkFixture(t *testing.T) (*ShareLinkRepository, *usecase.ShareLinkService, *usecase.RoomService, *usecase.AccessService, uuid.UUID, domain.SceneRef) {
	t.Helper()
	db := accessTestDB(t, 8)
	owner, _, w := accessFixture(t, db)
	scene := uuid.New()
	mustExec(t, db, "INSERT INTO scenes(id,workspace_id,title) VALUES (?,?,'Shared')", scene, w)
	repo := NewShareLinkRepository(db)
	access, e := usecase.NewAccessService(NewAccessRepository(db))
	if e != nil {
		t.Fatal(e)
	}
	links := usecase.NewShareLinkService(repo, access)
	rooms, e := usecase.NewRoomService(repo, access, []byte("0123456789abcdef0123456789abcdef"))
	if e != nil {
		t.Fatal(e)
	}
	return repo, links, rooms, access, owner, domain.SceneRef{WorkspaceID: w, SceneID: scene}
}
func TestRevokeActiveGrant(t *testing.T) {
	repo, links, rooms, access, owner, ref := publicLinkFixture(t)
	ctx := context.Background()
	first, e := links.Create(ctx, owner, ref, domain.CreateShareLink{Access: "edit", Password: "long-password"})
	if e != nil {
		t.Fatal(e)
	}
	second, e := links.Create(ctx, owner, ref, domain.CreateShareLink{Access: "view"})
	if e != nil {
		t.Fatal(e)
	}
	if first.Token == second.Token || first.ID == second.ID {
		t.Fatal("links not independent")
	}
	list, e := links.List(ctx, owner, ref)
	if e != nil || len(list) != 2 {
		t.Fatalf("list %v %v", list, e)
	}
	metadata, _ := json.Marshal(list)
	if strings.Contains(string(metadata), first.Token) || strings.Contains(string(metadata), "Hash") || strings.Contains(string(metadata), "password") {
		t.Fatal("metadata exposes credentials")
	}
	if _, e = rooms.ExchangePublicSession(ctx, first.Token, "wrong", "Guest"); !errors.Is(e, domain.ErrSharePassword) {
		t.Fatalf("password bypass: %v", e)
	}
	session, e := rooms.ExchangePublicSession(ctx, first.Token, "long-password", "Guest")
	if e != nil {
		t.Fatal(e)
	}
	identity, e := rooms.ParseSceneSession(session.Token)
	if e != nil {
		t.Fatal(e)
	}
	before, e := access.Resolve(ctx, identity.Principal, ref)
	if e != nil || before.Access != "edit" {
		t.Fatalf("edit link: %+v %v", before, e)
	}
	var auditsBefore int64
	repo.db.Table("acl_audit_events").Count(&auditsBefore)
	abort := errors.New("rollback fixture")
	e = access.WithWorkspaceLock(ctx, ref.WorkspaceID, owner, func(ctx context.Context) error {
		if e := links.Revoke(ctx, owner, ref, first.ID); e != nil {
			return e
		}
		return abort
	})
	if !errors.Is(e, abort) {
		t.Fatal(e)
	}
	after, e := access.Resolve(ctx, identity.Principal, ref)
	if e != nil || after.ACLVersion != before.ACLVersion {
		t.Fatalf("rollback changed grant/version: %+v %v", after, e)
	}
	var auditsAfter int64
	repo.db.Table("acl_audit_events").Count(&auditsAfter)
	if auditsAfter != auditsBefore {
		t.Fatal("audit escaped rollback")
	}
	if e = links.Revoke(ctx, owner, ref, first.ID); e != nil {
		t.Fatal(e)
	}
	if _, e = rooms.MintRoomToken(ctx, identity.Principal, ref); e == nil {
		t.Fatal("revoked active session refreshed")
	}
	if _, e = rooms.ExchangePublicSession(ctx, first.Token, "long-password", "Guest"); e == nil {
		t.Fatal("revoked link exchanged")
	}
	if _, e = rooms.ExchangePublicSession(ctx, second.Token, "", ""); e != nil {
		t.Fatalf("independent view link revoked: %v", e)
	}
	current, e := access.Resolve(ctx, domain.UserPrincipal(owner), ref)
	if e != nil || current.ACLVersion != before.ACLVersion+1 {
		t.Fatalf("version not bumped exactly once: %+v %v", current, e)
	}
	repo.db.Table("acl_audit_events").Where("actor_id=? AND event_type='share_link_update'", owner).Count(&auditsAfter)
	if auditsAfter != 1 {
		t.Fatalf("missing revoke audit: %d", auditsAfter)
	}
	down, e := os.ReadFile("../../../migrations/000008_share_links.down.sql")
	if e != nil {
		t.Fatal(e)
	}
	if e = repo.db.Exec(string(down)).Error; e == nil {
		t.Fatal("lossy rollback accepted")
	}
	if !repo.db.Migrator().HasTable("share_links") {
		t.Fatal("refused rollback dropped links")
	}
}
func TestPublicLinkAuthorityAndExpiry(t *testing.T) {
	repo, links, rooms, access, owner, ref := publicLinkFixture(t)
	ctx := context.Background()
	outsider := uuid.New()
	if _, e := links.Create(ctx, outsider, ref, domain.CreateShareLink{Access: "view"}); e == nil {
		t.Fatal("outsider created link")
	}
	if _, e := links.List(ctx, outsider, ref); e == nil {
		t.Fatal("outsider listed links")
	}
	for _, in := range []domain.CreateShareLink{{Access: "owner"}, {Access: "edit", Password: "short"}, {Access: "view", ExpiresAt: ptrPublicTime(time.Now().Add(-time.Minute))}} {
		if _, e := links.Create(ctx, owner, ref, in); !errors.Is(e, domain.ErrInvalidShareInput) {
			t.Fatalf("invalid input accepted: %+v %v", in, e)
		}
	}
	expires := time.Now().Add(time.Minute)
	link, e := links.Create(ctx, owner, ref, domain.CreateShareLink{Access: "view", ExpiresAt: &expires})
	if e != nil {
		t.Fatal(e)
	}
	session, e := rooms.ExchangePublicSession(ctx, link.Token, "", "")
	if e != nil || session.ExpiresAt.After(expires) {
		t.Fatalf("expiry exceeds grant: %v", e)
	}
	identity, e := rooms.ParseSceneSession(session.Token)
	if e != nil {
		t.Fatal(e)
	}
	mustExec(t, repo.db, "UPDATE share_links SET expires_at=now()-interval '1 second' WHERE id=?", link.ID)
	if _, e = rooms.MintRoomToken(ctx, identity.Principal, ref); e == nil {
		t.Fatal("expired link refreshed")
	}
	mustExec(t, repo.db, "UPDATE share_links SET expires_at=NULL WHERE id=?", link.ID)
	mustExec(t, repo.db, "UPDATE scenes SET deleted_at=now() WHERE id=?", ref.SceneID)
	if _, e = access.Resolve(ctx, identity.Principal, ref); e == nil {
		t.Fatal("deleted scene accessible")
	}
}
func ptrPublicTime(t time.Time) *time.Time { return &t }

func TestPublicSessionHTTP(t *testing.T) {
	repo, links, rooms, access, owner, ref := publicLinkFixture(t)
	viewer := uuid.New()
	mustExec(t, repo.db, "INSERT INTO users(id,email,name) VALUES (?,?,'Viewer')", viewer, viewer.String()+"@test.example")
	mustExec(t, repo.db, "INSERT INTO workspace_members(workspace_id,user_id,role) VALUES (?,?,'viewer')", ref.WorkspaceID, viewer)
	rh := handler.NewRoomHandler(rooms, access)
	auth := middleware.AuthMiddleware([]byte("login-key"))
	legacy := usecase.NewScenePermissionUsecase(NewScenePermissionRepository(repo.db), NewSceneRepository(repo.db))
	router := delivery.NewRouterWithAccess(nil, nil, nil, nil, handler.NewAssetHandler(nil), auth, access, nil, handler.NewSharingHandler(legacy, links), handler.NewPublicHandler(legacy, rh), nil, nil, nil)
	login := func(id uuid.UUID) string {
		raw, e := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{"sub": id.String(), "exp": time.Now().Add(time.Hour).Unix()}).SignedString([]byte("login-key"))
		if e != nil {
			t.Fatal(e)
		}
		return raw
	}
	request := func(method, path, body, bearer string, want int) *httptest.ResponseRecorder {
		t.Helper()
		req := httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set(domain.HeaderWorkspaceID, ref.WorkspaceID.String())
		if bearer != "" {
			req.Header.Set("Authorization", "Bearer "+bearer)
		}
		rr := httptest.NewRecorder()
		router.ServeHTTP(accessMemoryResponse{rr}, req)
		if rr.Code != want {
			t.Fatalf("%s %s: %d want %d: %s", method, path, rr.Code, want, rr.Body.String())
		}
		if want >= 400 {
			var payload map[string]any
			if e := json.Unmarshal(rr.Body.Bytes(), &payload); e != nil || payload["code"] == nil || payload["message"] == nil || payload["requestId"] == "" {
				t.Fatalf("invalid error envelope: %s", rr.Body.String())
			}
		}
		return rr
	}
	base := "/api/v1/scenes/" + ref.SceneID.String()
	request("POST", base+"/share-links", `{"access":"edit"}`, login(viewer), 403)
	request("POST", base+"/share-links", `{"access":"owner"}`, login(owner), 400)
	created := request("POST", base+"/share-links", `{"access":"edit"}`, login(owner), 201)
	var link domain.CreatedShareLink
	if e := json.Unmarshal(created.Body.Bytes(), &link); e != nil {
		t.Fatal(e)
	}
	request("POST", "/api/v1/public/scene-session", `{"token":"`+link.Token+`"}`, "", 400)
	exchanged := request("POST", "/api/v1/public/scene-session", `{"token":"`+link.Token+`","displayName":"Guest"}`, "", 200)
	var session domain.SceneSession
	if e := json.Unmarshal(exchanged.Body.Bytes(), &session); e != nil {
		t.Fatal(e)
	}
	request("POST", base+"/room-token", "", session.Token, 200)
	request("POST", base+"/room-token", "", login(viewer), 200)
	request("POST", "/api/v1/scenes/"+uuid.NewString()+"/room-token", "", session.Token, 404)
	request("DELETE", base+"/share-links/"+link.ID.String(), "", login(owner), 204)
	request("POST", base+"/room-token", "", session.Token, 404)
	request("POST", "/api/v1/public/scene-session", `{"token":"x"} {}`, "", 400)
	request("POST", "/api/v1/public/scene-session", `{"token":"`+strings.Repeat("x", 5000)+`"}`, "", 413)
}

func TestPublicSessionRenewalAndPasswordChange(t *testing.T) {
	repo, links, rooms, _, owner, ref := publicLinkFixture(t)
	ctx := context.Background()
	link, e := links.Create(ctx, owner, ref, domain.CreateShareLink{Access: "edit", Password: "old-password"})
	if e != nil {
		t.Fatal(e)
	}
	session, e := rooms.ExchangePublicSession(ctx, link.Token, "old-password", "Guest")
	if e != nil {
		t.Fatal(e)
	}
	token, _, e := jwt.NewParser().ParseUnverified(session.Token, jwt.MapClaims{})
	if e != nil {
		t.Fatal(e)
	}
	claims := token.Claims.(jwt.MapClaims)
	claims["iat"] = time.Now().Add(-9 * time.Minute).Unix()
	claims["exp"] = time.Now().Add(time.Minute).Unix()
	nearlyExpired, e := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte("0123456789abcdef0123456789abcdef"))
	if e != nil {
		t.Fatal(e)
	}
	identity, e := rooms.ParseSceneSession(nearlyExpired)
	if e != nil {
		t.Fatal(e)
	}
	room, e := rooms.MintRoomToken(ctx, identity.Principal, ref)
	if e != nil {
		t.Fatal(e)
	}
	encoded, _ := json.Marshal(room)
	var response struct {
		SceneSession *domain.SceneSession `json:"sceneSession"`
	}
	if e = json.Unmarshal(encoded, &response); e != nil {
		t.Fatal(e)
	}
	if response.SceneSession == nil || !response.SceneSession.ExpiresAt.After(time.Now().Add(8*time.Minute)) || response.SceneSession.PrincipalID != session.PrincipalID {
		t.Fatalf("valid guest cannot renew session: %s", encoded)
	}
	renewed, e := rooms.ParseSceneSession(response.SceneSession.Token)
	if e != nil {
		t.Fatal(e)
	}
	mustExec(t, repo.db, "UPDATE share_links SET password_hash='changed-hash' WHERE id=?", link.ID)
	if _, e = rooms.MintRoomToken(ctx, renewed.Principal, ref); e == nil {
		t.Fatal("password change did not invalidate existing guest session")
	}
}

func TestPublicLegacyCompatibility(t *testing.T) {
	repo, links, rooms, access, owner, ref := publicLinkFixture(t)
	ctx := context.Background()
	legacy := usecase.NewScenePermissionUsecase(NewScenePermissionRepository(repo.db), NewSceneRepository(repo.db))
	var permission *domain.ScenePermission
	e := access.WithWorkspaceLock(ctx, ref.WorkspaceID, owner, func(ctx context.Context) error {
		var e error
		permission, e = legacy.Share(ctx, ref.WorkspaceID, ref.SceneID, true, "legacy-password")
		return e
	})
	if e != nil {
		t.Fatal(e)
	}
	session, e := rooms.ExchangePublicSession(ctx, permission.Token, "legacy-password", "")
	if e != nil || session.Access != "view" {
		t.Fatalf("legacy view link cannot exchange: %v", e)
	}
	if e = links.Revoke(ctx, owner, ref, permission.ID); e != nil {
		t.Fatal(e)
	}
	if _, e = legacy.GetPublicScene(ctx, permission.Token, "legacy-password"); e == nil {
		t.Fatal("legacy lookup survives migrated-link revoke")
	}
	e = access.WithWorkspaceLock(ctx, ref.WorkspaceID, owner, func(ctx context.Context) error {
		_, e := legacy.Share(ctx, ref.WorkspaceID, ref.SceneID, true, "new-password")
		return e
	})
	if e != nil {
		t.Fatal(e)
	}
	if _, e = rooms.ExchangePublicSession(ctx, permission.Token, "legacy-password", ""); e == nil {
		t.Fatal("old password survives legacy change")
	}
	if _, e = rooms.ExchangePublicSession(ctx, permission.Token, "new-password", ""); e != nil {
		t.Fatal(e)
	}
	// Production legacy public reader must also enforce current workspace status.
	rh := handler.NewRoomHandler(rooms, access)
	router := delivery.NewRouterWithAccess(nil, nil, nil, nil, handler.NewAssetHandler(nil), middleware.AuthMiddleware([]byte("login-key")), access, nil, handler.NewSharingHandler(legacy, links), handler.NewPublicHandler(legacy, rh), nil, nil, nil)
	mustExec(t, repo.db, "UPDATE workspaces SET deleted_at=now() WHERE id=?", ref.WorkspaceID)
	req := httptest.NewRequest("POST", "/api/v1/public/scenes/"+permission.Token, strings.NewReader(`{"password":"new-password"}`))
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != 404 {
		t.Fatalf("legacy read bypassed inactive workspace: %d", rr.Code)
	}
}

func TestPublicRepositoryDoesNotLogSecrets(t *testing.T) {
	repo, _, _, _, _, ref := publicLinkFixture(t)
	var logs bytes.Buffer
	db := repo.db.Session(&gorm.Session{Logger: logger.New(log.New(&logs, "", 0), logger.Config{LogLevel: logger.Info})})
	legacy := NewScenePermissionRepository(db)
	_, _ = legacy.FindActiveByToken(context.Background(), "private-link-token")
	_ = legacy.Save(context.Background(), &domain.ScenePermission{ID: uuid.New(), WorkspaceID: ref.WorkspaceID, SceneID: uuid.New(), Token: "private-link-token", PasswordHash: "private-password-hash", Active: true})
	if strings.Contains(logs.String(), "private-link-token") || strings.Contains(logs.String(), "private-password-hash") {
		t.Fatal("public repository logged a credential")
	}
}

func TestRoomTokenCurrentMembershipAndRevision(t *testing.T) {
	repo, links, rooms, access, owner, ref := publicLinkFixture(t)
	ctx := context.Background()
	editor := uuid.New()
	mustExec(t, repo.db, "INSERT INTO users(id,email,name) VALUES (?,?,'Editor')", editor, editor.String()+"@test.example")
	mustExec(t, repo.db, "INSERT INTO workspace_members(workspace_id,user_id,role) VALUES (?,?,'editor')", ref.WorkspaceID, editor)
	mustExec(t, repo.db, "UPDATE workspaces SET acl_version=9007199254740993 WHERE id=?", ref.WorkspaceID)
	room, e := rooms.MintRoomToken(ctx, domain.UserPrincipal(editor), ref)
	if e != nil {
		t.Fatal(e)
	}
	identity, e := rooms.VerifyRoomToken(room.Token)
	if e != nil || identity.Principal.Kind != domain.PrincipalUser || identity.Principal.UserID == nil || *identity.Principal.UserID != editor {
		t.Fatalf("user principal lost: %+v %v", identity, e)
	}
	parsed, _, e := jwt.NewParser().ParseUnverified(room.Token, jwt.MapClaims{})
	if e != nil {
		t.Fatal(e)
	}
	if parsed.Claims.(jwt.MapClaims)["aclVersion"] != "9007199254740993" {
		t.Fatal("ACL version lost bigint precision")
	}
	viewer := domain.WorkspaceRoleViewer
	if e = access.SetMember(ctx, owner, ref.WorkspaceID, editor, &viewer); e != nil {
		t.Fatal(e)
	}
	room, e = rooms.MintRoomToken(ctx, domain.UserPrincipal(editor), ref)
	if e != nil {
		t.Fatal(e)
	}
	parsed, _, _ = jwt.NewParser().ParseUnverified(room.Token, jwt.MapClaims{})
	if parsed.Claims.(jwt.MapClaims)["access"] != "view" {
		t.Fatal("downgraded member refreshed edit token")
	}
	if e = access.SetMember(ctx, owner, ref.WorkspaceID, editor, nil); e != nil {
		t.Fatal(e)
	}
	if _, e = rooms.MintRoomToken(ctx, domain.UserPrincipal(editor), ref); e == nil {
		t.Fatal("removed member refreshed token")
	}
	link, e := links.Create(ctx, owner, ref, domain.CreateShareLink{Access: "view"})
	if e != nil {
		t.Fatal(e)
	}
	session, e := rooms.ExchangePublicSession(ctx, link.Token, "", "")
	if e != nil {
		t.Fatal(e)
	}
	guest, e := rooms.ParseSceneSession(session.Token)
	if e != nil {
		t.Fatal(e)
	}
	mustExec(t, repo.db, "UPDATE workspaces SET acl_version=acl_version+1 WHERE id=?", ref.WorkspaceID)
	if _, e = rooms.MintRoomToken(ctx, guest.Principal, ref); e != nil {
		t.Fatalf("unrelated ACL change invalidated guest session: %v", e)
	}
	mustExec(t, repo.db, "UPDATE share_links SET password_hash='first' WHERE id=?", link.ID)
	mustExec(t, repo.db, "UPDATE share_links SET password_hash='second' WHERE id=?", link.ID)
	var revision int64
	repo.db.Table("share_links").Select("revision").Where("id=?", link.ID).Scan(&revision)
	if revision != 3 {
		t.Fatalf("rapid mutations did not increment revision: %d", revision)
	}
	if _, e = rooms.MintRoomToken(ctx, guest.Principal, ref); e == nil {
		t.Fatal("stale guest revision accepted")
	}
}

func TestPublicLinkExpiryInsideTransaction(t *testing.T) {
	repo, links, _, access, owner, ref := publicLinkFixture(t)
	ctx := context.Background()
	var expires time.Time
	link, e := links.Create(ctx, owner, ref, domain.CreateShareLink{Access: "view"})
	if e != nil {
		t.Fatal(e)
	}
	e = access.WithWorkspaceLock(ctx, ref.WorkspaceID, owner, func(ctx context.Context) error {
		if e := accessDB(ctx, repo.db).Raw("UPDATE share_links SET expires_at=clock_timestamp()+interval '100 milliseconds' WHERE id=? RETURNING expires_at", link.ID).Scan(&expires).Error; e != nil {
			return e
		}
		time.Sleep(time.Until(expires) + 10*time.Millisecond)
		_, e := repo.FindByID(ctx, link.ID)
		return e
	})
	if !errors.Is(e, domain.ErrInaccessible) {
		t.Fatalf("transaction-start timestamp kept an expired link alive: %v", e)
	}
}
