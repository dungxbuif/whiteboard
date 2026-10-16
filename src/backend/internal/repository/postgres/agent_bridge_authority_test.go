package postgres

import (
	"context"
	"devtool/internal/domain"
	"devtool/internal/usecase"
	"errors"
	"strings"
	"testing"
)

func TestAgentExportScopeIndependent(t *testing.T) {
	r, s, a, owner, ref := agentFixture(t)
	ctx := context.Background()
	rooms, _ := usecase.NewRoomService(NewShareLinkRepository(r.db), a, []byte(strings.Repeat("r", 32)))
	caps, _ := usecase.NewCapabilityService(a, nil, nil, []byte(strings.Repeat("g", 32)))
	for _, scope := range []string{domain.ScopeSceneRead, domain.ScopeSceneWrite, domain.ScopeSceneExport} {
		key, e := s.CreateAgentKey(ctx, owner, []string{scope}, domain.AgentTargets{WorkspaceID: ref.WorkspaceID})
		if e != nil {
			t.Fatal(e)
		}
		p, e := s.AuthenticateAgent(ctx, key.Key)
		if e != nil {
			t.Fatal(e)
		}
		room, e := rooms.MintRoomToken(ctx, p, ref)
		if e != nil {
			t.Fatal(e)
		}
		identity, e := rooms.VerifyRoomToken(room.Token)
		if e != nil {
			t.Fatal(e)
		}
		for _, action := range []string{domain.ActionSceneRead, domain.ActionSceneCommand, domain.ActionSceneExport} {
			digest := "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
			grant, e := caps.AuthorizeCommand(ctx, identity.Principal, domain.SceneMutation{WorkspaceID: ref.WorkspaceID.String(), SceneID: ref.SceneID.String(), Action: action, RequestID: "scope-test", PayloadHash: digest})
			allowed := scope == action || (scope == domain.ScopeSceneWrite && action == domain.ActionSceneCommand)
			if allowed && (e != nil || grant.Token == "") {
				t.Fatalf("%s cannot %s: %v", scope, action, e)
			}
			if !allowed && (e == nil || grant.Token != "") {
				t.Fatalf("%s escalated to %s", scope, action)
			}
		}
	}
}
func TestAgentRemoteAuditUncertainty(t *testing.T) {
	r, s, _, owner, ref := agentFixture(t)
	ctx := context.Background()
	key, e := s.CreateAgentKey(ctx, owner, []string{domain.ScopeSceneWrite}, domain.AgentTargets{WorkspaceID: ref.WorkspaceID})
	if e != nil {
		t.Fatal(e)
	}
	p, e := s.AuthenticateAgent(ctx, key.Key)
	if e != nil {
		t.Fatal(e)
	}
	mustExec(t, r.db, "CREATE FUNCTION fail_remote_audit() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'audit unavailable'; END $$; CREATE TRIGGER fail_remote_audit BEFORE INSERT ON agent_operation_audit FOR EACH ROW EXECUTE FUNCTION fail_remote_audit()")
	called := false
	_, e = s.RunAuditedRemoteOperation(ctx, p, "patch_elements", ref, func(domain.Context) (any, error) { called = true; return map[string]string{"sequence": "1"}, nil })
	if !called || !errors.Is(e, domain.ErrAgentAuditCompletionUncertain) {
		t.Fatalf("lost uncertain completed remote outcome: %v", e)
	}
}
