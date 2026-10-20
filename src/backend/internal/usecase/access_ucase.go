package usecase

import (
	"devtool/internal/domain"
	"github.com/casbin/casbin/v3"
	"github.com/casbin/casbin/v3/model"
	"github.com/google/uuid"
	"strings"
	"time"
)

type AccessService struct {
	repo      domain.AccessRepository
	policy    *casbin.Enforcer
	assurance domain.SessionAssurance
}

const accessModel = `[request_definition]
r = sub, role, act
[policy_definition]
p = role, act
[policy_effect]
e = some(where (p.eft == allow))
[matchers]
m = r.sub != "" && r.role == p.role && r.act == p.act
`

func NewAccessService(repo domain.AccessRepository, modelPaths ...string) (*AccessService, error) {
	return NewAccessServiceWithAssurance(repo, nil, modelPaths...)
}
func NewAccessServiceWithAssurance(repo domain.AccessRepository, assurance domain.SessionAssurance, modelPaths ...string) (*AccessService, error) {
	m, e := model.NewModelFromString(accessModel)
	if len(modelPaths) > 0 {
		m, e = model.NewModelFromFile(modelPaths[0])
	}
	if e != nil {
		return nil, e
	}
	policy, e := casbin.NewEnforcer(m)
	if e != nil {
		return nil, e
	}
	for _, p := range [][]string{{"owner", "view"}, {"owner", "edit"}, {"owner", "manage"}, {"editor", "view"}, {"editor", "edit"}, {"viewer", "view"}} {
		if _, e = policy.AddPolicy(p); e != nil {
			return nil, e
		}
	}
	return &AccessService{repo: repo, policy: policy, assurance: assurance}, nil
}
func principalSession(ctx domain.Context, p domain.Principal) domain.Principal {
	if id, ok := ctx.Value(domain.ContextKeySessionID).(uuid.UUID); ok {
		p.SessionID = id
	}
	if method, ok := ctx.Value(domain.ContextKeyAuthMethod).(string); ok {
		p.AuthMethod = method
	}
	if at, ok := ctx.Value(domain.ContextKeyAuthTime).(time.Time); ok {
		p.AuthenticatedAt = at
	}
	return p
}
func (s *AccessService) requireAssurance(ctx domain.Context, p domain.Principal, w uuid.UUID) error {
	if s.assurance == nil || p.Kind != domain.PrincipalUser {
		return nil
	}
	return s.assurance.RequireWorkspaceAssurance(ctx, principalSession(ctx, p), w)
}
func normalizedUser(p domain.Principal) (uuid.UUID, error) {
	if p.Kind != domain.PrincipalUser || p.UserID == nil || *p.UserID == uuid.Nil || p.ID != p.UserID.String() {
		return uuid.Nil, domain.ErrAccessDenied
	}
	return *p.UserID, nil
}
func (s *AccessService) decision(p domain.Principal, state domain.AccessState) (domain.AccessDecision, error) {
	role := state.Role
	if role != domain.WorkspaceRoleOwner && state.Override != "" {
		role = state.Override
	}
	allow, err := s.policy.Enforce(p.ID, string(role), domain.AccessView)
	if err != nil {
		return domain.AccessDecision{}, err
	}
	if !allow {
		return domain.AccessDecision{}, domain.ErrInaccessible
	}
	edit, err := s.policy.Enforce(p.ID, string(role), domain.AccessEdit)
	if err != nil {
		return domain.AccessDecision{}, err
	}
	access := domain.AccessView
	if edit {
		access = domain.AccessEdit
	}
	return domain.AccessDecision{Access: access, Owner: state.Role == domain.WorkspaceRoleOwner, ACLVersion: state.ACLVersion}, nil
}
func (s *AccessService) Resolve(ctx domain.Context, p domain.Principal, ref domain.SceneRef) (domain.AccessDecision, error) {
	if !p.CredentialExpiresAt.IsZero() && !time.Now().Before(p.CredentialExpiresAt) {
		return domain.AccessDecision{}, domain.ErrInvalidSceneSession
	}
	if p.Kind == domain.PrincipalAgent {
		repo, ok := s.repo.(domain.AgentAccessRepository)
		if !ok {
			return domain.AccessDecision{}, domain.ErrAccessDenied
		}
		d, e := repo.AgentSceneState(ctx, p, ref)
		if e != nil {
			return domain.AccessDecision{}, e
		}
		if d.Owner || (d.Access != domain.AccessView && d.Access != domain.AccessEdit) {
			return domain.AccessDecision{}, domain.ErrAccessDenied
		}
		return d, nil
	}
	if p.Kind == domain.PrincipalPublic {
		if s.assurance != nil {
			if e := s.assurance.RequirePublicLinkAccess(ctx, ref.WorkspaceID); e != nil {
				return domain.AccessDecision{}, e
			}
		}
		repo, ok := s.repo.(domain.PublicAccessRepository)
		if !ok || p.GrantVersion == "" || p.UserID != nil || !strings.HasPrefix(p.ID, "guest:") || len(p.ID) <= len("guest:") {
			return domain.AccessDecision{}, domain.ErrAccessDenied
		}
		if _, err := uuid.Parse(p.GrantID); err != nil {
			return domain.AccessDecision{}, domain.ErrAccessDenied
		}
		d, e := repo.PublicSceneState(ctx, p, ref)
		if e != nil {
			return domain.AccessDecision{}, e
		}
		if d.Owner || (d.Access != domain.AccessView && d.Access != domain.AccessEdit) {
			return domain.AccessDecision{}, domain.ErrAccessDenied
		}
		return d, nil
	}
	id, e := normalizedUser(p)
	if e != nil {
		return domain.AccessDecision{}, e
	}
	if e = s.requireAssurance(ctx, p, ref.WorkspaceID); e != nil {
		return domain.AccessDecision{}, e
	}
	state, e := s.repo.SceneState(ctx, id, ref)
	if e != nil {
		return domain.AccessDecision{}, e
	}
	return s.decision(p, state)
}
func (s *AccessService) ResolveWorkspace(ctx domain.Context, p domain.Principal, w uuid.UUID) (domain.AccessDecision, error) {
	id, e := normalizedUser(p)
	if e != nil {
		return domain.AccessDecision{}, e
	}
	if e = s.requireAssurance(ctx, p, w); e != nil {
		return domain.AccessDecision{}, e
	}
	state, e := s.repo.WorkspaceState(ctx, id, w)
	if e != nil {
		return domain.AccessDecision{}, e
	}
	return s.decision(p, state)
}
func (s *AccessService) RequireOwner(ctx domain.Context, u, w uuid.UUID) error {
	state, e := s.repo.WorkspaceState(ctx, u, w)
	if e != nil {
		return e
	}
	ok, e := s.policy.Enforce(u.String(), string(state.Role), domain.AccessManage)
	if e != nil {
		return e
	}
	if !ok {
		return domain.ErrAccessDenied
	}
	return nil
}
func (s *AccessService) ResolveWorkspaceID(ctx domain.Context, id string) (uuid.UUID, error) {
	return s.repo.ResolveWorkspaceID(ctx, id)
}
func (s *AccessService) ResolveSceneID(ctx domain.Context, w uuid.UUID, id string) (uuid.UUID, error) {
	return s.repo.ResolveSceneID(ctx, w, id)
}
func (s *AccessService) WithWorkspaceLock(ctx domain.Context, w, u uuid.UUID, fn func(domain.Context) error) error {
	return s.repo.WithWorkspaceLock(ctx, w, u, fn)
}
func (s *AccessService) RequireFolderEdit(ctx domain.Context, p domain.Principal, w, f uuid.UUID) error {
	d, e := s.ResolveWorkspace(ctx, p, w)
	if e != nil {
		return e
	}
	if d.Access != domain.AccessEdit {
		return domain.ErrAccessDenied
	}
	ids, e := s.repo.DescendantScenes(ctx, w, f)
	if e != nil {
		return e
	}
	for _, id := range ids {
		d, e := s.Resolve(ctx, p, domain.SceneRef{WorkspaceID: w, SceneID: id})
		if e != nil {
			return e
		}
		if d.Access != domain.AccessEdit {
			return domain.ErrAccessDenied
		}
	}
	return nil
}
func (s *AccessService) ListMembers(ctx domain.Context, u, w uuid.UUID) ([]domain.WorkspaceMember, error) {
	if e := s.RequireOwner(ctx, u, w); e != nil {
		return nil, e
	}
	return s.repo.ListMembers(ctx, w)
}
func validMemberRole(role *domain.WorkspaceRole) bool {
	return role == nil || *role == domain.WorkspaceRoleEditor || *role == domain.WorkspaceRoleViewer
}
func (s *AccessService) SetMember(ctx domain.Context, actor, w, target uuid.UUID, role *domain.WorkspaceRole) error {
	if !validMemberRole(role) {
		return domain.ErrAccessDenied
	}
	return s.WithWorkspaceLock(ctx, w, actor, func(ctx domain.Context) error {
		if e := s.RequireOwner(ctx, actor, w); e != nil {
			return e
		}
		state, e := s.repo.WorkspaceState(ctx, target, w)
		if e != nil {
			return e
		}
		if state.Role == domain.WorkspaceRoleOwner || actor == target {
			return domain.ErrOwnerProtected
		}
		return s.repo.ChangeMember(ctx, w, target, role)
	})
}
func (s *AccessService) SetOverride(ctx domain.Context, actor uuid.UUID, ref domain.SceneRef, target uuid.UUID, role *domain.WorkspaceRole) error {
	if !validMemberRole(role) {
		return domain.ErrAccessDenied
	}
	return s.WithWorkspaceLock(ctx, ref.WorkspaceID, actor, func(ctx domain.Context) error {
		if e := s.RequireOwner(ctx, actor, ref.WorkspaceID); e != nil {
			return e
		}
		state, e := s.repo.SceneState(ctx, target, ref)
		if e != nil {
			return e
		}
		if state.Role == domain.WorkspaceRoleOwner {
			return domain.ErrOwnerProtected
		}
		return s.repo.ChangeOverride(ctx, ref, target, role)
	})
}

func (s *AccessService) RequireAgentScope(ctx domain.Context, p domain.Principal, ref domain.SceneRef, scope string) error {
	if p.Kind != domain.PrincipalAgent {
		return domain.ErrAccessDenied
	}
	if !p.CredentialExpiresAt.IsZero() && !time.Now().Before(p.CredentialExpiresAt) {
		return domain.ErrInvalidSceneSession
	}
	repo, ok := s.repo.(domain.AgentScopeRepository)
	if !ok {
		return domain.ErrAccessDenied
	}
	return repo.RequireAgentScope(ctx, p, ref, scope)
}
