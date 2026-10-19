package postgres

import (
	"bufio"
	"context"
	delivery "devtool/internal/delivery/http"
	"devtool/internal/delivery/http/handler"
	"devtool/internal/domain"
	"devtool/internal/usecase"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/google/uuid"
	pg "gorm.io/driver/postgres"
	"gorm.io/gorm"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"
)

func accessTestDB(t *testing.T, through int) *gorm.DB {
	t.Helper()
	raw := os.Getenv("ACCESS_TEST_DATABASE_URL")
	if raw == "" {
		t.Skip("ACCESS_TEST_DATABASE_URL required for disposable PostgreSQL integration")
	}
	u, err := url.Parse(raw)
	if err != nil || (u.Hostname() != "127.0.0.1" && u.Hostname() != "localhost") || !strings.HasSuffix(u.Path, "_test") {
		t.Fatal("refusing non-local/non-test database")
	}
	db, err := gorm.Open(pg.Open(raw), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	schema := "access_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	if err = db.Exec("CREATE SCHEMA " + schema).Error; err != nil {
		t.Fatal(err)
	}
	sqlDB, _ := db.DB()
	sqlDB.SetMaxOpenConns(1)
	if err = db.Exec("SET search_path TO " + schema + ", public").Error; err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Exec("DROP SCHEMA " + schema + " CASCADE"); sqlDB.Close() })
	files, _ := filepath.Glob("../../../migrations/*.up.sql")
	sort.Strings(files)
	for i, f := range files {
		if i >= through {
			break
		}
		b, e := os.ReadFile(f)
		if e != nil {
			t.Fatal(e)
		}
		if e = db.Exec(string(b)).Error; e != nil {
			t.Fatalf("%s: %v", f, e)
		}
	}
	return db
}
func accessFixture(t *testing.T, db *gorm.DB) (uuid.UUID, uuid.UUID, uuid.UUID) {
	t.Helper()
	owner, member, workspace := uuid.New(), uuid.New(), uuid.New()
	for _, id := range []uuid.UUID{owner, member} {
		if e := db.Exec("INSERT INTO users(id,email,name) VALUES (?,?,'Test')", id, id.String()+"@example.test").Error; e != nil {
			t.Fatal(e)
		}
	}
	if e := db.Exec("INSERT INTO workspaces(id,name,owner_id) VALUES (?,'Test',?)", workspace, owner).Error; e != nil {
		t.Fatal(e)
	}
	return owner, member, workspace
}
func TestAccessMigration(t *testing.T) {
	db := accessTestDB(t, 6)
	owner, member, w := accessFixture(t, db)
	if e := db.Exec("INSERT INTO workspace_members(workspace_id,user_id,role) VALUES (?,?,'member')", w, member).Error; e != nil {
		t.Fatal(e)
	}
	files, _ := filepath.Glob("../../../migrations/000007*.up.sql")
	for _, f := range files {
		b, _ := os.ReadFile(f)
		if e := db.Exec(string(b)).Error; e != nil {
			t.Fatal(e)
		}
	}
	var role string
	db.Raw("SELECT role FROM workspace_members WHERE workspace_id=? AND user_id=?", w, member).Scan(&role)
	if role != "editor" {
		t.Errorf("legacy member must become editor, got %q", role)
	}
	var count int64
	db.Table("workspace_members").Where("workspace_id=? AND user_id=? AND role='owner'", w, owner).Count(&count)
	if count != 1 {
		t.Errorf("owner membership not backfilled: %d", count)
	}
	found, e := NewWorkspaceRepository(db).FindByID(context.Background(), w)
	if e != nil || found.ID != w {
		t.Fatalf("UUID link not retained: %v", e)
	}
	repo := NewAccessRepository(db)
	alias, err := domain.NewPublicID()
	if err != nil {
		t.Fatal(err)
	}
	if len(alias) != 21 || strings.ContainsAny(alias, " /+=") {
		t.Fatalf("invalid alias: %q", alias)
	}
	mustExec(t, db, "UPDATE workspaces SET public_id=? WHERE id=?", alias, w)
	resolved, err := repo.ResolveWorkspaceID(context.Background(), alias)
	if err != nil || resolved != w {
		t.Fatalf("alias lookup %s %v", resolved, err)
	}
	if e := db.Exec("INSERT INTO workspaces(name,owner_id,public_id) VALUES ('Collision',?,?)", owner, alias).Error; e == nil {
		t.Fatal("duplicate public ID accepted")
	}
	scene := uuid.New()
	mustExec(t, db, "INSERT INTO scenes(id,workspace_id,title,public_id) VALUES (?,?,'Alias scene',?)", scene, w, alias)
	resolved, err = repo.ResolveSceneID(context.Background(), w, alias)
	if err != nil || resolved != scene {
		t.Fatalf("scene alias lookup %s %v", resolved, err)
	}
	resolved, err = repo.ResolveSceneID(context.Background(), w, scene.String())
	if err != nil || resolved != scene {
		t.Fatalf("scene UUID lookup %s %v", resolved, err)
	}
	if e := db.Exec("INSERT INTO scenes(workspace_id,title,public_id) VALUES (?,'Collision',?)", w, alias).Error; e == nil {
		t.Fatal("duplicate scene alias accepted")
	}
	if e := db.Exec("INSERT INTO scene_member_overrides(workspace_id,scene_id,user_id,role) VALUES (?,?,?,'owner')", w, scene, member).Error; e == nil {
		t.Fatal("invalid override accepted")
	}
	orphan := uuid.New()
	mustExec(t, db, "INSERT INTO users(id,email,name) VALUES (?,?,'Orphan')", orphan, orphan.String()+"@example.test")
	if e := db.Exec("INSERT INTO scene_member_overrides(workspace_id,scene_id,user_id,role) VALUES (?,?,?,'editor')", w, scene, orphan).Error; e == nil {
		t.Fatal("nonmember override accepted")
	}
	service, _ := usecase.NewAccessService(repo)
	beforeStatus, _ := service.ResolveWorkspace(context.Background(), domain.UserPrincipal(owner), w)
	mustExec(t, db, "UPDATE users SET deleted_at=now() WHERE id=?", member)
	afterStatus, _ := service.ResolveWorkspace(context.Background(), domain.UserPrincipal(owner), w)
	if afterStatus.ACLVersion <= beforeStatus.ACLVersion {
		t.Fatal("inactive user must invalidate existing scene grants")
	}
	if _, e := service.Resolve(context.Background(), domain.UserPrincipal(member), domain.SceneRef{WorkspaceID: w, SceneID: scene}); e == nil {
		t.Fatal("inactive user admitted")
	}
	down, e := os.ReadFile("../../../migrations/000007_collab_access.down.sql")
	if e != nil {
		t.Fatal(e)
	}
	mustExec(t, db, string(down))
	var legacyRole string
	db.Raw("SELECT role FROM workspace_members WHERE workspace_id=? AND user_id=?", w, member).Scan(&legacyRole)
	if legacyRole != "member" {
		t.Fatalf("rollback role: %s", legacyRole)
	}

}
func TestWorkspaceListAndSearchFilter(t *testing.T) {
	db := accessTestDB(t, 7)
	owner, editor, w := accessFixture(t, db)
	viewer, outsider := uuid.New(), uuid.New()
	for _, id := range []uuid.UUID{viewer, outsider} {
		mustExec(t, db, "INSERT INTO users(id,email,name) VALUES (?,?,'User')", id, id.String()+"@example.test")
	}
	mustExec(t, db, "INSERT INTO workspace_members(workspace_id,user_id,role) VALUES (?,?,'editor'),(?,?,'viewer')", w, editor, w, viewer)
	other := uuid.New()
	mustExec(t, db, "INSERT INTO workspaces(id,name,owner_id) VALUES (?,'Other',?)", other, outsider)
	folder, child := uuid.New(), uuid.New()
	mustExec(t, db, "INSERT INTO folders(id,workspace_id,name,path,depth) VALUES (?,?,'Folder','/root/',0),(?,?,'Child','/root/child/',1)", folder, w, child, w)
	scene, foreign, deleted := uuid.New(), uuid.New(), uuid.New()
	mustExec(t, db, "INSERT INTO scenes(id,workspace_id,folder_id,title) VALUES (?,?,?,'Protected'),(?,?,NULL,'Foreign'),(?,?,NULL,'Deleted')", scene, w, child, foreign, other, deleted, w)
	mustExec(t, db, "UPDATE scenes SET deleted_at=now() WHERE id=?", deleted)
	mustExec(t, db, "INSERT INTO scene_member_overrides(workspace_id,scene_id,user_id,role) VALUES (?,?,?,'viewer'),(?,?,?,'editor')", w, scene, editor, w, scene, viewer)
	service, e := usecase.NewAccessService(NewAccessRepository(db))
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
	folders := usecase.NewFolderUsecase(NewFolderRepository(db))
	scenes := usecase.NewSceneUsecase(NewSceneRepository(db), NewFolderRepository(db))
	router := delivery.NewRouterWithAccess(nil, handler.NewWorkspaceHandler(usecase.NewWorkspaceUsecase(NewWorkspaceRepository(db))), handler.NewFolderHandler(folders), handler.NewSceneHandler(scenes), handler.NewAssetHandler(nil), auth, service, handler.NewSearchHandler(usecase.NewSearchUsecase(NewSearchRepository(db))), nil, nil, nil, nil, nil)
	request := func(user uuid.UUID, method, path, body string, workspace string) *httptest.ResponseRecorder {
		t.Helper()
		r := httptest.NewRequest(method, "/api/v1"+path, strings.NewReader(body))
		r.Header.Set("Test-User", user.String())
		r.Header.Set(domain.HeaderWorkspaceID, workspace)
		rr := httptest.NewRecorder()
		router.ServeHTTP(accessMemoryResponse{rr}, r)
		return rr
	}
	check := func(user uuid.UUID, method, path, body string, want int) *httptest.ResponseRecorder {
		t.Helper()
		rr := request(user, method, path, body, w.String())
		if rr.Code != want {
			t.Fatalf("%s %s: got %d want %d: %s", method, path, rr.Code, want, rr.Body.String())
		}
		return rr
	}
	listed := check(editor, "GET", "/workspaces/", "", 200)
	var workspaces []domain.Workspace
	if e = json.Unmarshal(listed.Body.Bytes(), &workspaces); e != nil {
		t.Fatal(e)
	}
	var workspaceDTOs []map[string]any
	_ = json.Unmarshal(listed.Body.Bytes(), &workspaceDTOs)
	if len(workspaceDTOs) != 1 || workspaceDTOs[0]["role"] != "editor" {
		t.Fatalf("missing computed workspace role: %s", listed.Body.String())
	}
	if len(workspaces) != 1 || workspaces[0].ID != w {
		t.Fatalf("list leak/missing: %s", listed.Body.String())
	}
	check(outsider, "GET", "/search?q=Protected", "", 404)
	search := check(editor, "GET", "/search?q=", "", 200)
	if strings.Contains(search.Body.String(), "Foreign") || strings.Contains(search.Body.String(), "Deleted") {
		t.Fatalf("search leaked: %s", search.Body.String())
	}
	for _, tc := range []struct {
		user   uuid.UUID
		access string
		owner  bool
	}{{owner, "edit", true}, {editor, "view", false}, {viewer, "edit", false}} {
		rr := check(tc.user, "GET", "/scenes/"+scene.String(), "", 200)
		var dto map[string]any
		_ = json.Unmarshal(rr.Body.Bytes(), &dto)
		if dto["access"] != tc.access || dto["owner"] != tc.owner {
			t.Fatalf("missing effective scene access: %s", rr.Body.String())
		}
	}
	check(editor, "GET", "/scenes/"+foreign.String(), "", 404)
	check(editor, "GET", "/scenes/"+deleted.String(), "", 404)
	check(viewer, "POST", "/scenes/", `{"title":"No","content":{}}`, 403)
	check(editor, "PATCH", "/scenes/"+scene.String(), `{"title":"No"}`, 403)
	check(viewer, "PATCH", "/scenes/"+scene.String(), `{"title":"Allowed"}`, 200)
	for _, tc := range []struct{ method, path, body string }{{"DELETE", "/folders/" + folder.String(), ""}, {"PATCH", "/folders/" + folder.String() + "/move", `{"newParentId":null}`}} {
		check(editor, tc.method, tc.path, tc.body, 403)
	}
	var intact domain.Scene
	if e = db.First(&intact, "id=?", scene).Error; e != nil {
		t.Fatalf("protected descendant changed: %v", e)
	}
	check(editor, "GET", "/workspaces/"+w.String()+"/members", "", 403)
	check(owner, "PATCH", "/workspaces/"+w.String()+"/members/"+owner.String(), `{"role":"viewer"}`, 403)
	check(owner, "DELETE", "/workspaces/"+w.String()+"/members/"+owner.String(), "", 403)
	check(owner, "GET", "/workspaces/"+w.String()+"/members", "", 200)

	check(owner, "PUT", "/scenes/"+scene.String()+"/members/"+viewer.String(), `{"role":"viewer"}`, 204)
	check(viewer, "PATCH", "/scenes/"+scene.String(), `{"title":"Denied"}`, 403)
	check(owner, "DELETE", "/scenes/"+scene.String()+"/members/"+viewer.String(), "", 204)
	check(owner, "PUT", "/scenes/"+scene.String()+"/members/"+viewer.String(), `{"role":"editor"}`, 204)
	check(editor, "PUT", "/scenes/"+scene.String()+"/members/"+viewer.String(), `{"role":"editor"}`, 403)
	check(owner, "PUT", "/scenes/"+scene.String()+"/members/"+viewer.String(), `{"role":"owner"}`, 400)
	check(owner, "PATCH", "/workspaces/"+w.String()+"/members/"+editor.String(), `{"role":"viewer"}`, 204)
	check(owner, "PATCH", "/workspaces/"+w.String()+"/members/"+editor.String(), `{"role":"editor"}`, 204)
	alias, _ := domain.NewPublicID()
	wsAlias, _ := domain.NewPublicID()
	mustExec(t, db, "UPDATE workspaces SET public_id=? WHERE id=?", wsAlias, w)
	mustExec(t, db, "UPDATE scenes SET public_id=? WHERE id=?", alias, scene)
	aliased := request(owner, "GET", "/scenes/"+alias, "", wsAlias)
	if aliased.Code != 200 || !strings.Contains(aliased.Body.String(), scene.String()) {
		t.Fatalf("alias round trip: %d %s", aliased.Code, aliased.Body.String())
	}
	before, e := service.Resolve(context.Background(), domain.UserPrincipal(editor), domain.SceneRef{WorkspaceID: w, SceneID: scene})
	if e != nil {
		t.Fatal(e)
	}
	check(owner, "DELETE", "/workspaces/"+w.String()+"/members/"+editor.String(), "", 204)
	check(editor, "GET", "/scenes/"+scene.String(), "", 404)
	var n int64
	db.Table("scene_member_overrides").Where("user_id=?", editor).Count(&n)
	if n != 0 {
		t.Fatal("removed member kept override")
	}
	after, e := service.Resolve(context.Background(), domain.UserPrincipal(owner), domain.SceneRef{WorkspaceID: w, SceneID: scene})
	if e != nil || after.ACLVersion <= before.ACLVersion {
		t.Fatalf("ACL version did not advance: %+v %v", after, e)
	}
	db.Table("acl_audit_events").Where("actor_id=? AND subject_id=?", owner, editor).Count(&n)
	if n == 0 {
		t.Fatal("owner mutation audit absent")
	}
	check(owner, "GET", "/scenes/?trash=true", "", 200)
	check(viewer, "GET", "/search?view=trash", "", 403)
	mustExec(t, db, "UPDATE workspaces SET deleted_at=now() WHERE id=?", w)
	check(owner, "GET", "/scenes/"+scene.String(), "", 404)
}
func mustExec(t *testing.T, db *gorm.DB, sql string, args ...any) {
	t.Helper()
	if e := db.Exec(sql, args...).Error; e != nil {
		t.Fatal(e)
	}
}

func TestAccessAuthorityLockAndRollback(t *testing.T) {
	db := accessTestDB(t, 7)
	owner, editor, w := accessFixture(t, db)
	mustExec(t, db, "INSERT INTO workspace_members(workspace_id,user_id,role) VALUES (?,?,'editor')", w, editor)
	scene := uuid.New()
	mustExec(t, db, "INSERT INTO scenes(id,workspace_id,title) VALUES (?,?,'Original')", scene, w)
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
	sqlDB, _ := other.DB()
	sqlDB.SetMaxOpenConns(1)
	defer sqlDB.Close()
	mustExec(t, other, "SET lock_timeout='100ms'")
	access, _ := usecase.NewAccessService(NewAccessRepository(db))
	second, _ := usecase.NewAccessService(NewAccessRepository(other))
	abort := errors.New("rollback fixture")
	err := access.WithWorkspaceLock(context.Background(), w, editor, func(ctx context.Context) error {
		if _, e := access.Resolve(ctx, domain.UserPrincipal(editor), domain.SceneRef{WorkspaceID: w, SceneID: scene}); e != nil {
			t.Fatal(e)
		}
		if e := second.SetMember(context.Background(), owner, w, editor, nil); e == nil || !strings.Contains(e.Error(), "lock timeout") {
			t.Fatalf("revoke must wait for authorized writer: %v", e)
		}
		repo := NewSceneRepository(db)
		record, e := repo.FindByID(ctx, w, scene)
		if e != nil {
			t.Fatal(e)
		}
		record.Title = "Rolled back"
		if e = repo.Update(ctx, record); e != nil {
			t.Fatal(e)
		}
		return abort
	})
	if !errors.Is(err, abort) {
		t.Fatalf("transaction error: %v", err)
	}
	var title string
	db.Raw("SELECT title FROM scenes WHERE id=?", scene).Scan(&title)
	if title != "Original" {
		t.Fatalf("authorized write escaped transaction: %s", title)
	}
	if e := second.SetMember(context.Background(), owner, w, editor, nil); e != nil {
		t.Fatal(e)
	}
	if _, e := access.Resolve(context.Background(), domain.UserPrincipal(editor), domain.SceneRef{WorkspaceID: w, SceneID: scene}); e == nil {
		t.Fatal("revoked principal retained write access")
	}
}

type observedAccessBody struct {
	io.ReadCloser
	once    sync.Once
	started chan struct{}
}

func (b *observedAccessBody) Read(p []byte) (int, error) {
	b.once.Do(func() { close(b.started) })
	return b.ReadCloser.Read(p)
}

func TestAccessStalledBodyDoesNotBlockRevocation(t *testing.T) {
	db := accessTestDB(t, 7)
	owner, viewer, workspace := accessFixture(t, db)
	mustExec(t, db, "INSERT INTO workspace_members(workspace_id,user_id,role) VALUES (?,?,'viewer')", workspace, viewer)
	access, err := usecase.NewAccessService(NewAccessRepository(db))
	if err != nil {
		t.Fatal(err)
	}
	started := make(chan struct{})
	auth := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			id, e := uuid.Parse(r.Header.Get("Test-User"))
			if e != nil {
				w.WriteHeader(401)
				return
			}
			if r.URL.Path == "/api/v1/assets/presign-view" {
				r.Body = &observedAccessBody{ReadCloser: r.Body, started: started}
			}
			next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), domain.ContextKeyUserID, id)))
		})
	}
	router := delivery.NewRouterWithAccess(nil, nil, nil, nil, handler.NewAssetHandler(nil), auth, access, nil, nil, nil, nil, nil, nil)
	server := httptest.NewServer(router)
	defer server.Close()
	addr := strings.TrimPrefix(server.URL, "http://")
	stalled, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatal(err)
	}
	defer stalled.Close()
	_, err = fmt.Fprintf(stalled, "POST /api/v1/assets/presign-view HTTP/1.1\r\nHost: %s\r\nTest-User: %s\r\nX-Workspace-ID: %s\r\nContent-Length: 1000\r\nConnection: close\r\n\r\n{", addr, viewer, workspace)
	if err != nil {
		t.Fatal(err)
	}
	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("stalled body never reached server")
	}
	request, _ := http.NewRequest(http.MethodDelete, server.URL+"/api/v1/workspaces/"+workspace.String()+"/members/"+viewer.String(), nil)
	request.Header.Set("Test-User", owner.String())
	client := &http.Client{Timeout: time.Second}
	response, err := client.Do(request)
	if err != nil {
		t.Fatalf("stalled viewer body blocks owner revocation: %v", err)
	}
	response.Body.Close()
	if response.StatusCode != http.StatusNoContent {
		t.Fatalf("revocation status %d", response.StatusCode)
	}
	if _, err = access.ResolveWorkspace(context.Background(), domain.UserPrincipal(viewer), workspace); err == nil {
		t.Fatal("viewer membership not revoked")
	}
	stalled.SetReadDeadline(time.Now().Add(12 * time.Second))
	timeoutResponse, err := http.ReadResponse(bufio.NewReader(stalled), nil)
	if err != nil {
		t.Fatalf("stalled request not terminated with bounded HTTP response: %v", err)
	}
	defer timeoutResponse.Body.Close()
	if timeoutResponse.StatusCode != http.StatusRequestTimeout {
		t.Fatalf("stalled request status %d, want 408", timeoutResponse.StatusCode)
	}
}

// In-memory fixture bodies cannot stall on network I/O. Supply the server's
// deadline capability while exercising the same production gateway.
type accessMemoryResponse struct{ *httptest.ResponseRecorder }

func (accessMemoryResponse) SetReadDeadline(time.Time) error { return nil }
