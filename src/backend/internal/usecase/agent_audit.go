package usecase

import (
	"context"
	"devtool/internal/domain"
	"errors"
	"github.com/google/uuid"
	"slices"
	"time"
)

var agentAuditTools = []string{"list_workspaces", "list_scenes", "get_scene", "create_scene", "generate_diagram", "patch_elements", "delete_elements", "get_scene_activity", "list_diagram_types", "get_diagram_template", "validate_diagram", "compile_mermaid", "export_scene"}

func agentAuditOutcome(err error) (string, string) {
	switch {
	case err == nil:
		return "success", ""
	case errors.Is(err, domain.ErrAccessDenied), errors.Is(err, domain.ErrOwnerProtected):
		return "denied", "ACCESS_DENIED"
	case errors.Is(err, domain.ErrAgentCredential), errors.Is(err, domain.ErrInvalidSceneSession):
		return "denied", "UNAUTHENTICATED"
	case errors.Is(err, domain.ErrInaccessible):
		return "denied", "NOT_FOUND"
	case errors.Is(err, domain.ErrAgentInput), errors.Is(err, domain.ErrInvalidShareInput):
		return "invalid", "INVALID_INPUT"
	case errors.Is(err, domain.ErrAgentProjection):
		return "invalid", "PROJECTION_TOO_LARGE"
	default:
		return "failure", "OPERATION_UNAVAILABLE"
	}
}

// RunAuditedOperation is a trusted server-side boundary, not an audit ingestion
// API. The principal must come from credential verification. Metadata never
// authorizes the callback; it must perform its normal current scope checks.
// D3 can reuse this for its Go-side operations while retaining B2 receipts.
func (s *AgentService) RunAuditedOperation(ctx domain.Context, p domain.Principal, tool string, ref domain.SceneRef, operation func(domain.Context) (any, error)) (any, error) {
	started := time.Now()
	repo, ok := s.repo.(domain.AgentAuditRepository)
	if !ok {
		return nil, domain.ErrAgentAuditUnavailable
	}
	id, e := uuid.Parse(p.GrantID)
	if e != nil || p.Kind != domain.PrincipalAgent || p.UserID != nil || p.ID != "agent:"+id.String() {
		return nil, domain.ErrAgentCredential
	}
	if !slices.Contains(agentAuditTools, tool) {
		return nil, domain.ErrAgentInput
	}
	key, e := repo.FindAuditKey(ctx, id)
	if e != nil {
		return nil, e
	}
	correlation, ok := ctx.Value(domain.AgentAuditCorrelationKey{}).(uuid.UUID)
	if !ok || correlation == uuid.Nil {
		correlation = uuid.New()
	}
	event := domain.AgentOperationAudit{ID: uuid.New(), CorrelationID: correlation, KeyID: key.ID, PrincipalID: p.ID, WorkspaceID: key.WorkspaceID, Tool: tool, Scope: agentAuditScope(tool), Provenance: "go_operation"}
	if ref.WorkspaceID != uuid.Nil {
		event.RequestedWorkspaceID = &ref.WorkspaceID
	}
	if ref.SceneID != uuid.Nil {
		event.SceneID = &ref.SceneID
	}
	finish := func(err error) {
		event.Outcome, event.FailureClass = agentAuditOutcome(err)
		event.LatencyMs = min(time.Since(started).Milliseconds(), 86400000)
		event.CreatedAt = time.Now()
	}
	var result any
	e = s.access.WithWorkspaceLock(ctx, key.WorkspaceID, uuid.Nil, func(locked domain.Context) error {
		var err error
		result, err = operation(locked)
		if err != nil {
			return err
		}
		if created, ok := result.(domain.AgentScene); ok {
			event.SceneID = &created.ID
		}
		finish(nil)
		if err = repo.RecordOperation(locked, &event); err != nil {
			return domain.ErrAgentAuditUnavailable
		}
		return nil
	})
	if e == nil {
		return result, nil
	}
	// A failed workspace transaction must not erase the denial/failure event.
	// Detach cancellation only for this short, bounded metadata write.
	finish(e)
	auditCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 3*time.Second)
	defer cancel()
	if err := repo.RecordOperationFailure(auditCtx, &event); err != nil {
		return nil, domain.ErrAgentAuditUnavailable
	}
	return nil, e
}

func auditedAgentResult[T any](s *AgentService, ctx domain.Context, p domain.Principal, tool string, ref domain.SceneRef, operation func(domain.Context) (T, error)) (T, error) {
	var zero T
	result, e := s.RunAuditedOperation(ctx, p, tool, ref, func(ctx domain.Context) (any, error) { return operation(ctx) })
	if e != nil {
		return zero, e
	}
	return result.(T), nil
}
func (s *AgentService) ListWorkspaces(ctx domain.Context, p domain.Principal, cursor string, limit int) (domain.AgentPage[domain.AgentWorkspace], error) {
	return auditedAgentResult(s, ctx, p, "list_workspaces", domain.SceneRef{}, func(ctx domain.Context) (domain.AgentPage[domain.AgentWorkspace], error) {
		return s.listWorkspaces(ctx, p, cursor, limit)
	})
}
func (s *AgentService) ListScenes(ctx domain.Context, p domain.Principal, w uuid.UUID, cursor string, limit int) (domain.AgentPage[domain.AgentScene], error) {
	return auditedAgentResult(s, ctx, p, "list_scenes", domain.SceneRef{WorkspaceID: w}, func(ctx domain.Context) (domain.AgentPage[domain.AgentScene], error) {
		return s.listScenes(ctx, p, w, cursor, limit)
	})
}
func (s *AgentService) GetScene(ctx domain.Context, p domain.Principal, ref domain.SceneRef) (domain.AgentScene, error) {
	return auditedAgentResult(s, ctx, p, "get_scene", ref, func(ctx domain.Context) (domain.AgentScene, error) { return s.getScene(ctx, p, ref) })
}
func (s *AgentService) CreateScene(ctx domain.Context, p domain.Principal, w uuid.UUID, title string) (domain.AgentScene, error) {
	return auditedAgentResult(s, ctx, p, "create_scene", domain.SceneRef{WorkspaceID: w}, func(ctx domain.Context) (domain.AgentScene, error) { return s.createScene(ctx, p, w, title) })
}

func agentAuditScope(tool string) string {
	switch tool {
	case "list_workspaces":
		return domain.ScopeWorkspaceRead
	case "list_scenes", "get_scene", "get_scene_activity":
		return domain.ScopeSceneRead
	case "create_scene":
		return domain.ScopeSceneCreate
	case "generate_diagram", "patch_elements", "delete_elements":
		return domain.ScopeSceneWrite
	case "export_scene":
		return domain.ScopeSceneExport
	default:
		return ""
	}
}
