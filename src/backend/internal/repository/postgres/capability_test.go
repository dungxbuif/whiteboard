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
	pg "gorm.io/driver/postgres"
	"gorm.io/gorm"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"
)

func TestSceneAssetIsolation(t *testing.T) {
	db := accessTestDB(t, 8)
	owner, _, w := accessFixture(t, db)
	a, b := uuid.New(), uuid.New()
	mustExec(t, db, "INSERT INTO scenes(id,workspace_id,title) VALUES (?,?,'A'),(?,?,'B')", a, w, b, w)
	mustExec(t, db, "INSERT INTO scene_assets(workspace_id,scene_id,uploaded_by,asset_kind,file_id,storage_bucket,storage_key,mime_type,byte_size) VALUES (?,?,?,'excalidraw_file','foreign','bucket','key','image/png',1)", w, b, owner)
	assets, e := NewSceneAssetRepository(db).GetByFileIDs(context.Background(), w, a, []string{"foreign"})
	if e != nil {
		t.Fatal(e)
	}
	if len(assets) != 0 {
		t.Fatalf("repository returned another scene's assets: %+v", assets)
	}

}

type capabilityStorage struct {
	domain.S3Repository
	fail bool
}

func (s capabilityStorage) PresignUploadConstrained(_ domain.Context, b, k, m string, n int64, ttl time.Duration) (string, map[string]string, error) {
	if s.fail {
		return "", nil, errors.New("storage unavailable")
	}
	return "https://storage.invalid/" + k, map[string]string{"Content-Type": m}, nil
}
func TestPublicUploadCapability(t *testing.T) {
	repo, links, rooms, access, owner, ref := publicLinkFixture(t)
	migration, e := os.ReadFile("../../../migrations/000012_scene_asset_capabilities.up.sql")
	if e != nil {
		t.Fatal(e)
	}
	mustExec(t, repo.db, string(migration))
	assets := usecase.NewAssetUsecase(NewSceneRepository(repo.db), NewSceneAssetRepository(repo.db), capabilityStorage{}, "assets", time.Minute)
	caps, e := usecase.NewCapabilityService(access, repo, assets, []byte(strings.Repeat("g", 32)))
	if e != nil {
		t.Fatal(e)
	}
	ctx := context.Background()
	editor, e := links.Create(ctx, owner, ref, domain.CreateShareLink{Access: "edit"})
	if e != nil {
		t.Fatal(e)
	}
	view, e := links.Create(ctx, owner, ref, domain.CreateShareLink{Access: "view"})
	if e != nil {
		t.Fatal(e)
	}
	identity := func(token string) domain.Principal {
		t.Helper()
		s, e := rooms.ExchangePublicSession(ctx, token, "", "Guest")
		if e != nil {
			t.Fatal(e)
		}
		p, e := rooms.ParseSceneSession(s.Token)
		if e != nil {
			t.Fatal(e)
		}
		return p.Principal
	}
	p := identity(editor.Token)
	viewer := identity(view.Token)
	req := []domain.AssetUploadRequest{{FileID: "file", MimeType: "image/png", ByteSize: 7}}
	if g, e := caps.CreateAssetCapability(ctx, viewer, ref, req); e == nil || len(g.URLs) != 0 {
		t.Fatal("view grant uploaded")
	}
	foreign := ref
	foreign.SceneID = uuid.New()
	mustExec(t, repo.db, "INSERT INTO scenes(id,workspace_id,title) VALUES (?,?,'Foreign')", foreign.SceneID, ref.WorkspaceID)
	if _, e := caps.CreateAssetCapability(ctx, p, foreign, req); e == nil {
		t.Fatal("guest uploaded outside target scene")
	}
	expired := p
	expired.CredentialExpiresAt = time.Now().Add(-time.Second)
	if _, e := caps.CreateAssetCapability(ctx, expired, ref, req); e == nil {
		t.Fatal("expired guest uploaded")
	}
	p.CredentialExpiresAt = time.Now().Add(3 * time.Second).Truncate(time.Second)
	g, e := caps.CreateAssetCapability(ctx, p, ref, req)
	if e != nil {
		t.Fatal(e)
	}
	if len(g.URLs) != 1 || g.ExpiresAt.After(p.CredentialExpiresAt) {
		t.Fatalf("invalid scoped upload grant %+v", g)
	}
	var row domain.SceneAsset
	if e = repo.db.First(&row, "id=?", g.URLs["file"].AssetID).Error; e != nil {
		t.Fatal(e)
	}
	if row.UploadedBy != nil || row.UploadedPrincipalID != p.ID || row.UploadedPrincipalType != "public" || row.SceneID == nil || *row.SceneID != ref.SceneID {
		t.Fatalf("lost guest audit: %+v", row)
	}
	if e = links.Revoke(ctx, owner, ref, editor.ID); e != nil {
		t.Fatal(e)
	}
	if _, e = caps.CreateAssetCapability(ctx, p, ref, []domain.AssetUploadRequest{{FileID: "after-revoke", MimeType: "image/png", ByteSize: 7}}); e == nil {
		t.Fatal("revoked guest uploaded")
	}
	user := domain.UserPrincipal(owner)
	user.CredentialExpiresAt = time.Now().Add(time.Minute)
	if _, e = caps.CreateAssetCapability(ctx, user, foreign, req); e != nil {
		t.Fatalf("same file ID in different scene rejected: %v", e)
	}
	down, e := os.ReadFile("../../../migrations/000012_scene_asset_capabilities.down.sql")
	if e != nil {
		t.Fatal(e)
	}
	if e = repo.db.Exec(string(down)).Error; e == nil {
		t.Fatal("lossy rollback accepted colliding scene file IDs")
	}
	var count int64
	repo.db.Table("scene_assets").Count(&count)
	if count != 2 {
		t.Fatal("rollback removed assets")
	}
	failing := usecase.NewAssetUsecase(NewSceneRepository(repo.db), NewSceneAssetRepository(repo.db), capabilityStorage{fail: true}, "assets", time.Minute)
	broken, _ := usecase.NewCapabilityService(access, repo, failing, []byte(strings.Repeat("g", 32)))
	if grant, e := broken.CreateAssetCapability(ctx, user, ref, []domain.AssetUploadRequest{{FileID: "failure", MimeType: "image/png", ByteSize: 7}}); e == nil || len(grant.URLs) != 0 {
		t.Fatal("storage error returned upload grant")
	}
}
func secondCapabilityDB(t *testing.T, db *gorm.DB) *gorm.DB {
	t.Helper()
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
	pool.SetMaxOpenConns(1)
	t.Cleanup(func() { pool.Close() })
	return other
}
func TestRevokeRace(t *testing.T) {
	db := accessTestDB(t, 8)
	owner, member, w := accessFixture(t, db)
	scene := uuid.New()
	mustExec(t, db, "INSERT INTO scenes(id,workspace_id,title) VALUES (?,?,'Race')", scene, w)
	mustExec(t, db, "INSERT INTO workspace_members(workspace_id,user_id,role) VALUES (?,?,'editor')", w, member)
	access, _ := usecase.NewAccessService(NewAccessRepository(db))
	other := secondCapabilityDB(t, db)
	second, _ := usecase.NewAccessService(NewAccessRepository(other))
	caps, _ := usecase.NewCapabilityService(second, nil, nil, []byte(strings.Repeat("g", 32)))
	p := domain.UserPrincipal(member)
	p.CredentialExpiresAt = time.Now().Add(time.Minute)
	m := domain.SceneMutation{WorkspaceID: w.String(), SceneID: scene.String(), Action: domain.ActionSceneUpdate, RequestID: "race", PayloadHash: strings.Repeat("a", 64)}
	old, e := caps.AuthorizeCommand(context.Background(), p, m)
	if e != nil {
		t.Fatal(e)
	}
	claims := &domain.MutationClaims{}
	_, e = jwt.ParseWithClaims(old.Token, claims, func(*jwt.Token) (any, error) { return []byte(strings.Repeat("g", 32)), nil })
	if e != nil {
		t.Fatal(e)
	}
	started := make(chan struct{})
	result := make(chan error, 1)
	e = access.WithWorkspaceLock(context.Background(), w, owner, func(ctx domain.Context) error {
		if e := access.SetMember(ctx, owner, w, member, nil); e != nil {
			return e
		}
		go func() {
			close(started)
			g, e := caps.AuthorizeCommand(context.Background(), p, m)
			if e == nil || g.Token != "" {
				result <- errors.New("revoked principal received grant")
				return
			}
			result <- nil
		}()
		<-started
		select {
		case e := <-result:
			t.Fatalf("authority bypassed revocation transaction lock: %v", e)
		case <-time.After(100 * time.Millisecond):
		}
		return nil
	})
	if e != nil {
		t.Fatal(e)
	}
	select {
	case e := <-result:
		if e != nil {
			t.Fatal(e)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("authority deadlocked")
	}
	var current string
	other.Raw("SELECT acl_version::text FROM workspaces WHERE id=?", w).Scan(&current)
	if current == claims.ACLVersion {
		t.Fatal("pre-revoke grant still has current durable ACL version")
	}
}

func TestPublicUploadCapabilityHTTP(t *testing.T) {
	repo, links, rooms, access, owner, ref := publicLinkFixture(t)
	up, e := os.ReadFile("../../../migrations/000012_scene_asset_capabilities.up.sql")
	if e != nil {
		t.Fatal(e)
	}
	mustExec(t, repo.db, string(up))
	ctx := context.Background()
	link, e := links.Create(ctx, owner, ref, domain.CreateShareLink{Access: "edit"})
	if e != nil {
		t.Fatal(e)
	}
	session, e := rooms.ExchangePublicSession(ctx, link.Token, "", "Guest")
	if e != nil {
		t.Fatal(e)
	}
	identity, e := rooms.ParseSceneSession(session.Token)
	if e != nil {
		t.Fatal(e)
	}
	room, e := rooms.MintRoomToken(ctx, identity.Principal, ref)
	if e != nil {
		t.Fatal(e)
	}
	assets := usecase.NewAssetUsecase(NewSceneRepository(repo.db), NewSceneAssetRepository(repo.db), capabilityStorage{}, "assets", time.Minute)
	caps, _ := usecase.NewCapabilityService(access, repo, assets, []byte(strings.Repeat("g", 32)))
	h, _ := handler.NewCapabilityHandler(caps, rooms, access, assets, []byte(strings.Repeat("s", 32)))
	if e = h.SetTURNConfig("turn:turn.example.test:3478", strings.Repeat("t", 32)); e != nil {
		t.Fatal(e)
	}
	ah := handler.NewAssetHandler(assets)
	ah.SetCapabilityService(caps)
	ah.Capabilities = h
	auth := middleware.AuthMiddleware([]byte(strings.Repeat("l", 32)))
	api := httptest.NewServer(delivery.NewRouterWithAccess(nil, nil, nil, nil, ah, auth, access, nil, nil, nil, nil, nil, nil))
	defer api.Close()
	private := httptest.NewServer(h.AuthorityHandler())
	defer private.Close()
	request := func(base, method, path, token string, body any) (int, []byte) {
		t.Helper()
		b, _ := json.Marshal(body)
		r, _ := http.NewRequest(method, base+path, bytes.NewReader(b))
		r.Header.Set("Authorization", "Bearer "+token)
		r.Header.Set("X-Workspace-ID", ref.WorkspaceID.String())
		response, e := http.DefaultClient.Do(r)
		if e != nil {
			t.Fatal(e)
		}
		defer response.Body.Close()
		raw, _ := io.ReadAll(response.Body)
		return response.StatusCode, raw
	}
	code, raw := request(api.URL, "POST", "/api/v1/scenes/"+ref.SceneID.String()+"/assets/upload-capability", session.Token, map[string]any{"assets": []domain.AssetUploadRequest{{FileID: "http-file", MimeType: "image/png", ByteSize: 7}}})
	if code != 200 {
		t.Fatalf("public upload %d %s", code, raw)
	}
	login, _ := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{"sub": owner.String(), "exp": time.Now().Add(time.Minute).Unix()}).SignedString([]byte(strings.Repeat("l", 32)))
	code, raw = request(api.URL, "POST", "/api/v1/scenes/"+ref.SceneID.String()+"/assets/upload-capability", login, map[string]any{"assets": []domain.AssetUploadRequest{{FileID: "login-file", MimeType: "image/jpeg", ByteSize: 7}}})
	if code != 200 {
		t.Fatalf("login upload %d %s", code, raw)
	}
	code, raw = request(api.URL, "GET", "/api/v1/realtime/config", room.Token, nil)
	if code != 200 || !bytes.Contains(raw, []byte("credential")) || bytes.Contains(raw, []byte(strings.Repeat("t", 32))) {
		t.Fatalf("scoped TURN %d %s", code, raw)
	}
	code, _ = request(api.URL, "GET", "/api/v1/realtime/config", login, nil)
	if code != 401 {
		t.Fatalf("unscoped TURN admitted %d", code)
	}
	m := map[string]any{"roomToken": room.Token, "workspaceId": ref.WorkspaceID.String(), "sceneId": ref.SceneID.String(), "action": "scene:update", "requestId": "http", "payloadHash": strings.Repeat("a", 64)}
	code, raw = request(private.URL, "POST", "/internal/authority/decisions", strings.Repeat("s", 32), m)
	if code != 200 || !bytes.Contains(raw, []byte("token")) {
		t.Fatalf("private decision %d %s", code, raw)
	}
	code, _ = request(api.URL, "POST", "/internal/authority/decisions", strings.Repeat("s", 32), m)
	if code != 404 {
		t.Fatal("private authority publicly routed")
	}
	if e = links.Revoke(ctx, owner, ref, link.ID); e != nil {
		t.Fatal(e)
	}
	code, raw = request(private.URL, "POST", "/internal/authority/decisions", strings.Repeat("s", 32), m)
	if code < 400 || bytes.Contains(raw, []byte(`"token"`)) {
		t.Fatalf("revoked decision returned %d %s", code, raw)
	}
	code, _ = request(api.URL, "GET", "/api/v1/realtime/config", room.Token, nil)
	if code < 400 {
		t.Fatal("revoked TURN admitted")
	}
}
