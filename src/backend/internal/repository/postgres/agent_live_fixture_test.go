package postgres

import (
	"context"
	"devtool/internal/delivery/http/handler"
	"devtool/internal/domain"
	collabrepo "devtool/internal/repository/collab"
	"devtool/internal/usecase"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestAgentLiveFixture is an opt-in test fixture process used by the MCP live
// stack test. Its database is guarded and isolated by the standard test helper.
func TestAgentLiveFixture(t *testing.T) {
	ready := os.Getenv("D3_FIXTURE_READY")
	if ready == "" {
		return
	}
	r, s, a, owner, ref := agentFixture(t)
	migration, e := os.ReadFile("../../../migrations/000009_collab_documents.up.sql")
	if e != nil {
		t.Fatal(e)
	}
	mustExec(t, r.db, string(migration))
	mustExec(t, r.db, `UPDATE scenes SET content='{"elements":[],"appState":{},"files":{}}' WHERE id=?`, ref.SceneID)
	rooms, _ := usecase.NewRoomService(NewShareLinkRepository(r.db), a, []byte(strings.Repeat("r", 32)))
	caps, _ := usecase.NewCapabilityService(a, nil, nil, []byte(strings.Repeat("g", 32)))
	remote, e := collabrepo.NewAgentClient(os.Getenv("D3_COLLAB_URL"), strings.Repeat("i", 32))
	if e != nil {
		t.Fatal(e)
	}
	h := handler.NewAgentHandler(s, rooms, a)
	h.SetBridge(usecase.NewAgentBridge(s, rooms, caps, remote))
	capHandler, e := handler.NewCapabilityHandler(caps, rooms, a, nil, []byte(strings.Repeat("s", 32)))
	if e != nil {
		t.Fatal(e)
	}
	mux := http.NewServeMux()
	mux.Handle("/api/v1/agent/", http.StripPrefix("/api/v1/agent", h.Routes()))
	mux.Handle("/internal/", capHandler.AuthorityHandler())
	api := httptest.NewServer(mux)
	defer api.Close()
	keys := map[string]string{}
	ids := map[string]string{}
	for label, scopes := range map[string][]string{"full": {domain.ScopeSceneRead, domain.ScopeSceneWrite, domain.ScopeSceneCreate}, "write": {domain.ScopeSceneWrite}, "read": {domain.ScopeSceneRead}, "export": {domain.ScopeSceneExport}} {
		key, e := s.CreateAgentKey(context.Background(), owner, scopes, domain.AgentTargets{WorkspaceID: ref.WorkspaceID})
		if e != nil {
			t.Fatal(e)
		}
		keys[label] = key.Key
		ids[label] = key.ID.String()
	}
	room, e := rooms.MintRoomToken(context.Background(), domain.UserPrincipal(owner), ref)
	if e != nil {
		t.Fatal(e)
	}
	var schema string
	r.db.Raw("SELECT current_schema()").Scan(&schema)
	encoded, _ := json.Marshal(map[string]any{"api": api.URL, "schema": schema, "ref": map[string]string{"workspaceId": ref.WorkspaceID.String(), "sceneId": ref.SceneID.String()}, "keys": keys, "keyIds": ids, "roomToken": room.Token})
	if e = os.WriteFile(ready, encoded, 0600); e != nil {
		t.Fatal(e)
	}
	stop := filepath.Join(filepath.Dir(ready), "stop")
	deadline := time.Now().Add(90 * time.Second)
	for time.Now().Before(deadline) {
		if _, e = os.Stat(stop); e == nil {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("live fixture timed out")
}
