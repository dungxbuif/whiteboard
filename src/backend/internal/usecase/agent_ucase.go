package usecase

import (
	"devtool/internal/domain"
	"encoding/json"
	"github.com/google/uuid"
	"slices"
	"strings"
	"time"
	"unicode/utf8"
)

type AgentService struct {
	repo   domain.AgentRepository
	access domain.AccessService
}

func NewAgentService(repo domain.AgentRepository, access domain.AccessService) *AgentService {
	return &AgentService{repo, access}
}
func (s *AgentService) CreateAgentKey(ctx domain.Context, owner uuid.UUID, scopes []string, targets domain.AgentTargets) (domain.KeyOnce, error) {
	var out domain.KeyOnce
	valid := []string{domain.ScopeWorkspaceRead, domain.ScopeSceneRead, domain.ScopeSceneCreate, domain.ScopeSceneWrite, domain.ScopeSceneMove, domain.ScopeSceneTrash, domain.ScopeSceneExport}
	if len(scopes) == 0 || len(scopes) > len(valid) || len(targets.SceneIDs) > 500 || targets.WorkspaceID == uuid.Nil {
		return out, domain.ErrAgentInput
	}
	seen := map[string]bool{}
	for _, v := range scopes {
		if !slices.Contains(valid, v) || seen[v] {
			return out, domain.ErrAgentInput
		}
		seen[v] = true
	}
	if len(targets.SceneIDs) > 0 && slices.Contains(scopes, domain.ScopeSceneCreate) {
		return out, domain.ErrAgentInput
	}
	e := s.access.WithWorkspaceLock(ctx, targets.WorkspaceID, owner, func(ctx domain.Context) error {
		if e := s.access.RequireOwner(ctx, owner, targets.WorkspaceID); e != nil {
			return e
		}
		ids := map[uuid.UUID]bool{}
		for _, id := range targets.SceneIDs {
			if id == uuid.Nil || ids[id] {
				return domain.ErrAgentInput
			}
			ids[id] = true
			if _, e := s.access.Resolve(ctx, domain.UserPrincipal(owner), domain.SceneRef{WorkspaceID: targets.WorkspaceID, SceneID: id}); e != nil {
				return e
			}
		}
		raw, e := randomShareSecret()
		if e != nil {
			return e
		}
		if targets.SceneIDs == nil {
			targets.SceneIDs = []uuid.UUID{}
		}
		out = domain.KeyOnce{AgentKey: domain.AgentKey{ID: uuid.New(), WorkspaceID: targets.WorkspaceID, OwnerID: owner, Scopes: scopes, SceneIDs: targets.SceneIDs, TokenHash: shareTokenHash("wb_agent_" + raw), CreatedAt: time.Now()}, Key: "wb_agent_" + raw}
		return s.repo.Create(ctx, &out.AgentKey)
	})
	if e != nil {
		return domain.KeyOnce{}, e
	}
	return out, nil
}
func (s *AgentService) AuthenticateAgent(ctx domain.Context, bearer string) (domain.Principal, error) {
	if !strings.HasPrefix(bearer, "wb_agent_") || len(bearer) != 52 {
		return domain.Principal{}, domain.ErrAgentCredential
	}
	k, e := s.repo.FindHash(ctx, shareTokenHash(bearer))
	if e != nil {
		return domain.Principal{}, e
	}
	return domain.Principal{Kind: domain.PrincipalAgent, ID: "agent:" + k.ID.String(), GrantID: k.ID.String(), DisplayName: "Agent"}, nil
}
func (s *AgentService) key(ctx domain.Context, p domain.Principal, w uuid.UUID, scope string) (domain.AgentKey, error) {
	if p.Kind != domain.PrincipalAgent || p.UserID != nil || (!p.CredentialExpiresAt.IsZero() && !time.Now().Before(p.CredentialExpiresAt)) {
		return domain.AgentKey{}, domain.ErrAgentCredential
	}
	id, e := uuid.Parse(p.GrantID)
	if e != nil || p.ID != "agent:"+id.String() {
		return domain.AgentKey{}, domain.ErrAgentCredential
	}
	k, e := s.repo.FindActive(ctx, id)
	if e != nil {
		return k, e
	}
	if w != uuid.Nil && k.WorkspaceID != w {
		return k, domain.ErrInaccessible
	}
	if !slices.Contains(k.Scopes, scope) {
		return k, domain.ErrAccessDenied
	}
	return k, nil
}
func (s *AgentService) ListKeys(ctx domain.Context, owner, w uuid.UUID) ([]domain.AgentKey, error) {
	var out []domain.AgentKey
	err := s.access.WithWorkspaceLock(ctx, w, owner, func(ctx domain.Context) error {
		if e := s.access.RequireOwner(ctx, owner, w); e != nil {
			return e
		}
		var e error
		out, e = s.repo.List(ctx, w)
		return e
	})
	return out, err
}
func (s *AgentService) RevokeKey(ctx domain.Context, owner, w, id uuid.UUID) error {
	return s.access.WithWorkspaceLock(ctx, w, owner, func(ctx domain.Context) error {
		if e := s.access.RequireOwner(ctx, owner, w); e != nil {
			return e
		}
		return s.repo.Revoke(ctx, w, id)
	})
}
func agentCursor(raw string) (uuid.UUID, error) {
	if raw == "" {
		return uuid.Nil, nil
	}
	id, e := uuid.Parse(raw)
	if e != nil || id == uuid.Nil {
		return uuid.Nil, domain.ErrAgentInput
	}
	return id, nil
}
func (s *AgentService) ListWorkspaces(ctx domain.Context, p domain.Principal, cursor string, limit int) (domain.AgentPage[domain.AgentWorkspace], error) {
	out := domain.AgentPage[domain.AgentWorkspace]{Items: []domain.AgentWorkspace{}}
	c, e := agentCursor(cursor)
	if e != nil || limit < 1 || limit > 50 {
		return out, domain.ErrAgentInput
	}
	k, e := s.key(ctx, p, uuid.Nil, domain.ScopeWorkspaceRead)
	if e != nil {
		return out, e
	}
	e = s.access.WithWorkspaceLock(ctx, k.WorkspaceID, uuid.Nil, func(ctx domain.Context) error {
		k, e = s.key(ctx, p, k.WorkspaceID, domain.ScopeWorkspaceRead)
		if e != nil {
			return e
		}
		if c != uuid.Nil && strings.Compare(k.WorkspaceID.String(), c.String()) <= 0 {
			return nil
		}
		w, e := s.repo.Workspace(ctx, k.WorkspaceID)
		if e == nil {
			out.Items = append(out.Items, w)
		}
		return e
	})
	return out, e
}
func (s *AgentService) ListScenes(ctx domain.Context, p domain.Principal, w uuid.UUID, cursor string, limit int) (domain.AgentPage[domain.AgentScene], error) {
	out := domain.AgentPage[domain.AgentScene]{Items: []domain.AgentScene{}}
	c, e := agentCursor(cursor)
	if e != nil || limit < 1 || limit > 50 {
		return out, domain.ErrAgentInput
	}
	e = s.access.WithWorkspaceLock(ctx, w, uuid.Nil, func(ctx domain.Context) error {
		k, e := s.key(ctx, p, w, domain.ScopeSceneRead)
		if e != nil {
			return e
		}
		rows, e := s.repo.Scenes(ctx, k, c, limit+1)
		if e != nil {
			return e
		}
		if len(rows) > limit {
			out.NextCursor = rows[limit-1].ID.String()
			rows = rows[:limit]
		}
		out.Items = rows
		return nil
	})
	return out, e
}
func (s *AgentService) RequireScope(ctx domain.Context, p domain.Principal, ref domain.SceneRef, scope string) error {
	k, e := s.key(ctx, p, ref.WorkspaceID, scope)
	if e != nil {
		return e
	}
	if len(k.SceneIDs) > 0 && !slices.Contains(k.SceneIDs, ref.SceneID) {
		return domain.ErrInaccessible
	}
	_, e = s.access.Resolve(ctx, domain.UserPrincipal(k.OwnerID), ref)
	return e
}
func (s *AgentService) GetScene(ctx domain.Context, p domain.Principal, ref domain.SceneRef) (domain.AgentScene, error) {
	var out domain.AgentScene
	e := s.access.WithWorkspaceLock(ctx, ref.WorkspaceID, uuid.Nil, func(ctx domain.Context) error {
		if e := s.RequireScope(ctx, p, ref, domain.ScopeSceneRead); e != nil {
			return e
		}
		var e error
		out, e = s.repo.Scene(ctx, ref)
		if e != nil {
			return e
		}
		var projection struct {
			Elements []json.RawMessage `json:"elements"`
		}
		if len(out.Content) > 256*1024 || json.Unmarshal(out.Content, &projection) != nil || len(projection.Elements) > 500 {
			return domain.ErrAgentProjection
		}
		return nil
	})
	return out, e
}
func (s *AgentService) CreateScene(ctx domain.Context, p domain.Principal, w uuid.UUID, title string) (domain.AgentScene, error) {
	var out domain.AgentScene
	title = strings.TrimSpace(title)
	if title == "" || utf8.RuneCountInString(title) > 255 {
		return out, domain.ErrAgentInput
	}
	e := s.access.WithWorkspaceLock(ctx, w, uuid.Nil, func(ctx domain.Context) error {
		k, e := s.key(ctx, p, w, domain.ScopeSceneCreate)
		if e != nil {
			return e
		}
		if len(k.SceneIDs) > 0 {
			return domain.ErrAccessDenied
		}
		d, e := s.access.ResolveWorkspace(ctx, domain.UserPrincipal(k.OwnerID), w)
		if e != nil {
			return e
		}
		if d.Access != domain.AccessEdit {
			return domain.ErrAccessDenied
		}
		alias, e := domain.NewPublicID()
		if e != nil {
			return e
		}
		scene := domain.Scene{WriterMode: "legacy", ID: uuid.New(), PublicID: &alias, WorkspaceID: w, Title: title, Content: json.RawMessage(`{"elements":[],"appState":{},"files":{}}`), ContentVersion: 1}
		if e = s.repo.CreateScene(ctx, &scene); e != nil {
			return e
		}
		out = domain.AgentScene{ID: scene.ID, WorkspaceID: w, Title: title, Content: scene.Content, ContentVersion: 1}
		return nil
	})
	return out, e
}
