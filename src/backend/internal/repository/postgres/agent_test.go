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

func agentFixture(t *testing.T) (*AgentRepository, *usecase.AgentService, *usecase.AccessService, uuid.UUID, domain.SceneRef) {
	t.Helper()
	db := accessTestDB(t, 8)
	owner, _, w := accessFixture(t, db)
	scene := uuid.New()
	mustExec(t, db, "INSERT INTO scenes(id,workspace_id,title) VALUES (?,?,'Agent scene')", scene, w)
	// Migration is loaded explicitly so other workers can add non-contiguous migrations.
	if b, e := os.ReadFile("../../../migrations/000010_agent_keys.up.sql"); e == nil {
		mustExec(t, db, string(b))
	}
	a, e := usecase.NewAccessService(NewAccessRepository(db))
	if e != nil {
		t.Fatal(e)
	}
	r := NewAgentRepository(db)
	return r, usecase.NewAgentService(r, a), a, owner, domain.SceneRef{WorkspaceID: w, SceneID: scene}
}
func TestAgentScopeIntersection(t *testing.T) {
	r, s, a, owner, ref := agentFixture(t)
	ctx := context.Background()
	k, e := s.CreateAgentKey(ctx, owner, []string{domain.ScopeWorkspaceRead, domain.ScopeSceneRead}, domain.AgentTargets{WorkspaceID: ref.WorkspaceID, SceneIDs: []uuid.UUID{ref.SceneID}})
	if e != nil {
		t.Fatalf("owner could not create scoped key: %v", e)
	}
	p, e := s.AuthenticateAgent(ctx, k.Key)
	if e != nil || p.UserID != nil || p.Kind != domain.PrincipalAgent {
		t.Fatalf("identity: %+v %v", p, e)
	}
	d, e := a.Resolve(ctx, p, ref)
	if e != nil || d.Access != "view" || d.Owner {
		t.Fatalf("read-only key exceeded scope: %+v %v", d, e)
	}
	if _, e = a.Resolve(ctx, p, domain.SceneRef{WorkspaceID: uuid.New(), SceneID: ref.SceneID}); e == nil {
		t.Fatal("key escaped workspace")
	}
	other := uuid.New()
	mustExec(t, r.db, "INSERT INTO scenes(id,workspace_id,title) VALUES (?,?,'Other')", other, ref.WorkspaceID)
	if _, e = a.Resolve(ctx, p, domain.SceneRef{WorkspaceID: ref.WorkspaceID, SceneID: other}); e == nil {
		t.Fatal("key escaped scene target")
	}
	var stored string
	r.db.Table("agent_keys").Select("token_hash").Where("id=?", k.ID).Scan(&stored)
	if len(stored) != 64 || strings.Contains(stored, k.Key) {
		t.Fatal("key not hash-only")
	}
	rows, e := r.List(ctx, ref.WorkspaceID)
	b, _ := json.Marshal(rows)
	if e != nil || strings.Contains(string(b), k.Key) || strings.Contains(string(b), stored) {
		t.Fatal("list leaks credential")
	}
	mustExec(t, r.db, "UPDATE users SET deleted_at=now() WHERE id=?", owner)
	if _, e = s.AuthenticateAgent(ctx, k.Key); e == nil {
		t.Fatal("inactive owner authenticated")
	}
	if _, e = a.Resolve(ctx, p, ref); e == nil {
		t.Fatal("inactive owner room retained access")
	}
	mustExec(t, r.db, "UPDATE users SET deleted_at=NULL WHERE id=?", owner)
	if e = r.Revoke(ctx, ref.WorkspaceID, k.ID); e != nil {
		t.Fatal(e)
	}
	if _, e = s.AuthenticateAgent(ctx, k.Key); e == nil {
		t.Fatal("revoked key authenticated")
	}
	if _, e = a.Resolve(ctx, p, ref); e == nil {
		t.Fatal("revoked key room retained access")
	}
}

func TestAgentRoomAndExactScopes(t *testing.T) {
	r, s, a, owner, ref := agentFixture(t)
	ctx := context.Background()
	k, e := s.CreateAgentKey(ctx, owner, []string{domain.ScopeSceneWrite}, domain.AgentTargets{WorkspaceID: ref.WorkspaceID})
	if e != nil {
		t.Fatal(e)
	}
	p, e := s.AuthenticateAgent(ctx, k.Key)
	if e != nil {
		t.Fatal(e)
	}
	if e = s.RequireScope(ctx, p, ref, domain.ScopeSceneRead); e == nil {
		t.Fatal("write implied read")
	}
	if _, e = s.CreateScene(ctx, p, ref.WorkspaceID, "Denied"); e == nil {
		t.Fatal("write implied create")
	}
	rooms, e := usecase.NewRoomService(NewShareLinkRepository(r.db), a, []byte("0123456789abcdef0123456789abcdef"))
	if e != nil {
		t.Fatal(e)
	}
	token, e := rooms.MintRoomToken(ctx, p, ref)
	if e != nil {
		t.Fatalf("agent room exchange denied: %v", e)
	}
	identity, e := rooms.VerifyRoomToken(token.Token)
	if e != nil || identity.Principal.Kind != domain.PrincipalAgent || identity.Principal.UserID != nil {
		t.Fatalf("invalid room identity: %+v %v", identity, e)
	}
	if _, e = rooms.ParseSceneSession(token.Token); e == nil {
		t.Fatal("room token used as public session")
	}
	if e = s.RevokeKey(ctx, owner, ref.WorkspaceID, k.ID); e != nil {
		t.Fatal(e)
	}
	if _, e = rooms.MintRoomToken(ctx, identity.Principal, ref); e == nil {
		t.Fatal("revoked agent room renewed")
	}
}
func TestAgentDiscoveryCreateProjection(t *testing.T) {
	r, s, _, owner, ref := agentFixture(t)
	ctx := context.Background()
	k, e := s.CreateAgentKey(ctx, owner, []string{domain.ScopeWorkspaceRead, domain.ScopeSceneRead, domain.ScopeSceneCreate}, domain.AgentTargets{WorkspaceID: ref.WorkspaceID})
	if e != nil {
		t.Fatal(e)
	}
	p, _ := s.AuthenticateAgent(ctx, k.Key)
	other := uuid.New()
	mustExec(t, r.db, "INSERT INTO workspaces(id,name,owner_id) VALUES (?,'Hidden',?)", other, owner)
	ws, e := s.ListWorkspaces(ctx, p, "", 50)
	if e != nil || len(ws.Items) != 1 || ws.Items[0].ID != ref.WorkspaceID {
		t.Fatalf("workspace leak: %+v %v", ws, e)
	}
	if _, e = s.ListScenes(ctx, p, other, "", 50); e == nil {
		t.Fatal("enumerated other workspace")
	}
	created, e := s.CreateScene(ctx, p, ref.WorkspaceID, "Created")
	if e != nil {
		t.Fatal(e)
	}
	if !strings.Contains(string(created.Content), `"elements":[]`) {
		t.Fatal("invalid empty content")
	}
	page, e := s.ListScenes(ctx, p, ref.WorkspaceID, "", 1)
	if e != nil || len(page.Items) != 1 || page.NextCursor == "" {
		t.Fatalf("pagination: %+v %v", page, e)
	}
	next, e := s.ListScenes(ctx, p, ref.WorkspaceID, page.NextCursor, 1)
	if e != nil || len(next.Items) != 1 || next.Items[0].ID == page.Items[0].ID || next.NextCursor != "" {
		t.Fatalf("cursor failed: %+v %v", next, e)
	}
	if _, e = s.ListScenes(ctx, p, ref.WorkspaceID, "", 51); e == nil {
		t.Fatal("unbounded page accepted")
	}
	elements := make([]map[string]string, 501)
	for i := range elements {
		elements[i] = map[string]string{"id": uuid.NewString()}
	}
	b, _ := json.Marshal(map[string]any{"elements": elements})
	mustExec(t, r.db, "UPDATE scenes SET content=? WHERE id=?", string(b), ref.SceneID)
	if _, e = s.GetScene(ctx, p, ref); e != domain.ErrAgentProjection {
		t.Fatalf("unbounded projection: %v", e)
	}
}

func TestAgentAuthorityScopes(t *testing.T) {
	r, s, a, owner, ref := agentFixture(t)
	ctx := context.Background()
	caps, _ := usecase.NewCapabilityService(a, nil, nil, []byte(strings.Repeat("m", 32)))
	rooms, _ := usecase.NewRoomService(NewShareLinkRepository(r.db), a, []byte(strings.Repeat("r", 32)))
	for _, scope := range []string{domain.ScopeSceneRead, domain.ScopeSceneWrite} {
		k, e := s.CreateAgentKey(ctx, owner, []string{scope}, domain.AgentTargets{WorkspaceID: ref.WorkspaceID})
		if e != nil {
			t.Fatal(e)
		}
		p, _ := s.AuthenticateAgent(ctx, k.Key)
		room, e := rooms.MintRoomToken(ctx, p, ref)
		if e != nil {
			t.Fatal(e)
		}
		identity, e := rooms.VerifyRoomToken(room.Token)
		if e != nil {
			t.Fatal(e)
		}
		for _, action := range []string{domain.ActionSceneRead, domain.ActionSceneUpdate, domain.ActionSceneCommand} {
			hash := strings.Repeat("a", 64)
			if action == domain.ActionSceneRead {
				hash = "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
			}
			m := domain.SceneMutation{WorkspaceID: ref.WorkspaceID.String(), SceneID: ref.SceneID.String(), Action: action, RequestID: "agent-proof", PayloadHash: hash}
			grant, e := caps.AuthorizeCommand(ctx, identity.Principal, m)
			allowed := (scope == domain.ScopeSceneRead) == (action == domain.ActionSceneRead)
			if allowed && (e != nil || grant.Token == "") {
				t.Fatalf("allowed %s %s denied: %v", scope, action, e)
			}
			if !allowed && (e == nil || grant.Token != "") {
				t.Fatalf("scope escalation %s %s", scope, action)
			}
		}
		if e = s.RevokeKey(ctx, owner, ref.WorkspaceID, k.ID); e != nil {
			t.Fatal(e)
		}
		m := domain.SceneMutation{WorkspaceID: ref.WorkspaceID.String(), SceneID: ref.SceneID.String(), Action: domain.ActionSceneCommand, RequestID: "revoked", PayloadHash: strings.Repeat("a", 64)}
		if grant, e := caps.AuthorizeCommand(ctx, identity.Principal, m); e == nil || grant.Token != "" {
			t.Fatal("revoked key got authority grant")
		}
	}
}
func TestAgentHTTP(t *testing.T) {
	r, s, a, owner, ref := agentFixture(t)
	rooms, _ := usecase.NewRoomService(NewShareLinkRepository(r.db), a, []byte(strings.Repeat("r", 32)))
	h := handler.NewAgentHandler(s, rooms, a)
	auth := middleware.AuthMiddleware([]byte("login"))
	router := delivery.NewRouterWithAgents(nil, nil, nil, nil, handler.NewAssetHandler(nil), auth, a, nil, nil, nil, nil, nil, nil, nil, h)
	login, _ := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{"sub": owner.String(), "exp": time.Now().Add(time.Hour).Unix()}).SignedString([]byte("login"))
	request := func(method, path, body, token string, want int) *httptest.ResponseRecorder {
		t.Helper()
		req := httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set("Authorization", "Bearer "+token)
		rr := httptest.NewRecorder()
		router.ServeHTTP(accessMemoryResponse{rr}, req)
		if rr.Code != want {
			t.Fatalf("%s %s: %d want %d %s", method, path, rr.Code, want, rr.Body.String())
		}
		return rr
	}
	base := "/api/v1/workspaces/" + ref.WorkspaceID.String() + "/agent-keys"
	created := request("POST", base, `{"scopes":["workspace:read","scene:read"]}`, login, 201)
	var k domain.KeyOnce
	if e := json.Unmarshal(created.Body.Bytes(), &k); e != nil {
		t.Fatal(e)
	}
	request("GET", base, "", k.Key, 401)
	request("GET", "/api/v1/users/me", "", k.Key, 401)
	request("GET", "/api/v1/agent/auth", "", login, 401)
	request("GET", "/api/v1/agent/workspaces", "", k.Key, 200)
	request("POST", "/api/v1/agent/workspaces/"+ref.WorkspaceID.String()+"/scenes", `{"title":"Denied"}`, k.Key, 403)
	request("GET", base, "", login, 200)
	request("DELETE", base+"/"+k.ID.String(), "", login, 204)
	request("GET", "/api/v1/agent/auth", "", k.Key, 401)
}

func TestAgentManagementAndAudit(t *testing.T) {
	r, s, a, owner, ref := agentFixture(t)
	ctx := context.Background()
	outsider := uuid.New()
	if _, e := s.CreateAgentKey(ctx, outsider, []string{domain.ScopeSceneRead}, domain.AgentTargets{WorkspaceID: ref.WorkspaceID}); e == nil {
		t.Fatal("nonowner created key")
	}
	for _, scopes := range [][]string{{}, {"manage"}, {domain.ScopeSceneRead, domain.ScopeSceneRead}} {
		if _, e := s.CreateAgentKey(ctx, owner, scopes, domain.AgentTargets{WorkspaceID: ref.WorkspaceID}); e == nil {
			t.Fatal("invalid scopes accepted")
		}
	}
	if _, e := s.CreateAgentKey(ctx, owner, []string{domain.ScopeSceneCreate}, domain.AgentTargets{WorkspaceID: ref.WorkspaceID, SceneIDs: []uuid.UUID{ref.SceneID}}); e == nil {
		t.Fatal("targeted create key accepted")
	}
	k, e := s.CreateAgentKey(ctx, owner, []string{domain.ScopeSceneRead}, domain.AgentTargets{WorkspaceID: ref.WorkspaceID})
	if e != nil {
		t.Fatal(e)
	}
	if _, e = s.ListKeys(ctx, outsider, ref.WorkspaceID); e == nil {
		t.Fatal("nonowner listed keys")
	}
	var before int64
	r.db.Table("workspaces").Select("acl_version").Where("id=?", ref.WorkspaceID).Scan(&before)
	abort := context.Canceled
	e = a.WithWorkspaceLock(ctx, ref.WorkspaceID, owner, func(ctx domain.Context) error {
		if e := s.RevokeKey(ctx, owner, ref.WorkspaceID, k.ID); e != nil {
			return e
		}
		return abort
	})
	if e != abort {
		t.Fatal(e)
	}
	if _, e = s.AuthenticateAgent(ctx, k.Key); e != nil {
		t.Fatal("rollback lost key")
	}
	if e = s.RevokeKey(ctx, owner, ref.WorkspaceID, k.ID); e != nil {
		t.Fatal(e)
	}
	var after int64
	r.db.Table("workspaces").Select("acl_version").Where("id=?", ref.WorkspaceID).Scan(&after)
	if after != before+1 {
		t.Fatalf("key revoke ACL bump %d -> %d", before, after)
	}
	var count int64
	r.db.Table("acl_audit_events").Where("subject_id=? AND actor_id=? AND event_type='agent_key_update'", k.ID, owner).Count(&count)
	if count != 1 {
		t.Fatal("missing key-ID-only audit")
	}
	var logs bytes.Buffer
	db := r.db.Session(&gorm.Session{Logger: logger.New(log.New(&logs, "", 0), logger.Config{LogLevel: logger.Info})})
	logged := NewAgentRepository(db)
	_, _ = logged.FindHash(ctx, "private-credential-hash")
	k.TokenHash = "private-credential-hash"
	k.ID = uuid.New()
	_ = logged.Create(ctx, &k.AgentKey)
	if strings.Contains(logs.String(), "private-credential-hash") {
		t.Fatal("credential SQL logged")
	}
}

func TestAgentListHonorsOwnerRevocationLock(t *testing.T) {
	r, s, a, owner, ref := agentFixture(t)
	ctx := context.Background()
	if _, e := s.CreateAgentKey(ctx, owner, []string{domain.ScopeSceneRead}, domain.AgentTargets{WorkspaceID: ref.WorkspaceID}); e != nil {
		t.Fatal(e)
	}
	other := secondCapabilityDB(t, r.db)
	otherAccess, _ := usecase.NewAccessService(NewAccessRepository(other))
	second := usecase.NewAgentService(NewAgentRepository(other), otherAccess)
	result := make(chan error, 1)
	e := a.WithWorkspaceLock(ctx, ref.WorkspaceID, owner, func(ctx domain.Context) error {
		if e := accessDB(ctx, r.db).Exec("UPDATE users SET deleted_at=now() WHERE id=?", owner).Error; e != nil {
			return e
		}
		go func() { _, e := second.ListKeys(context.Background(), owner, ref.WorkspaceID); result <- e }()
		select {
		case e := <-result:
			t.Fatalf("key list bypassed authority transaction: %v", e)
		case <-time.After(50 * time.Millisecond):
		}
		return nil
	})
	if e != nil {
		t.Fatal(e)
	}
	select {
	case e := <-result:
		if e == nil {
			t.Fatal("revoked owner listed keys")
		}
	case <-time.After(time.Second):
		t.Fatal("list blocked after commit")
	}
}
