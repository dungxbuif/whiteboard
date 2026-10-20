package usecase

import (
	"context"
	"crypto/sha256"
	"devtool/internal/domain"
	"encoding/hex"
	"encoding/json"
	"strconv"

	"github.com/casbin/casbin/v3"
	"github.com/casbin/casbin/v3/model"
	"github.com/google/uuid"
)

type FeatureService struct {
	repo         domain.FeatureRepository
	access       domain.AccessService
	actionPolicy *casbin.Enforcer
	policyErr    error
}

type storedFeaturePolicy struct {
	Policy   domain.WorkspaceFeaturePolicy `json:"policy"`
	Revision string                        `json:"revision"`
}

func NewFeatureService(repo domain.FeatureRepository, access domain.AccessService) *FeatureService {
	actionPolicy, err := newFeatureActionPolicy()
	return &FeatureService{repo: repo, access: access, actionPolicy: actionPolicy, policyErr: err}
}

const featureActionModel = `[request_definition]
r = role, action
[policy_definition]
p = role, action
[policy_effect]
e = some(where (p.eft == allow))
[matchers]
m = r.role == p.role && r.action == p.action
`

var featureActionRules = map[string][]domain.FeatureAction{
	"owner": {
		domain.FeatureCommentRead, domain.FeatureCommentWrite, domain.FeatureCommentModerate,
		domain.FeatureActivityRead, domain.FeatureVersionRead, domain.FeatureVersionCreate,
		domain.FeatureVersionRestore, domain.FeatureLibraryRead, domain.FeatureLibraryWrite,
		domain.FeatureCollectionManage, domain.FeaturePresentationHost, domain.FeatureScreenPublish,
		domain.FeatureEnterpriseManage,
	},
	"editor": {
		domain.FeatureCommentRead, domain.FeatureCommentWrite, domain.FeatureActivityRead,
		domain.FeatureVersionRead, domain.FeatureVersionCreate, domain.FeatureVersionRestore,
		domain.FeatureLibraryRead, domain.FeatureCollectionManage, domain.FeaturePresentationHost,
		domain.FeatureScreenPublish,
	},
	"viewer": {
		domain.FeatureCommentRead, domain.FeatureCommentWrite, domain.FeatureActivityRead,
		domain.FeatureVersionRead, domain.FeatureLibraryRead,
	},
	"public": {domain.FeatureCommentRead, domain.FeatureCommentWrite},
}

type featureRequirement struct {
	enabled       func(domain.WorkspaceFeaturePolicy) bool
	viewerAllowed func(domain.WorkspaceFeaturePolicy) bool
}

var featureRequirements = map[domain.FeatureAction]featureRequirement{
	domain.FeatureCommentRead:      {enabled: func(p domain.WorkspaceFeaturePolicy) bool { return p.CommentsEnabled }},
	domain.FeatureCommentWrite:     {enabled: func(p domain.WorkspaceFeaturePolicy) bool { return p.CommentsEnabled }, viewerAllowed: func(p domain.WorkspaceFeaturePolicy) bool { return p.MemberComments }},
	domain.FeatureCommentModerate:  {enabled: func(p domain.WorkspaceFeaturePolicy) bool { return p.CommentsEnabled }},
	domain.FeatureActivityRead:     {enabled: func(p domain.WorkspaceFeaturePolicy) bool { return p.ActivityEnabled }},
	domain.FeatureVersionRead:      {enabled: func(p domain.WorkspaceFeaturePolicy) bool { return p.VersionsEnabled }, viewerAllowed: func(p domain.WorkspaceFeaturePolicy) bool { return p.MemberHistory }},
	domain.FeatureVersionCreate:    {enabled: func(p domain.WorkspaceFeaturePolicy) bool { return p.VersionsEnabled }},
	domain.FeatureVersionRestore:   {enabled: func(p domain.WorkspaceFeaturePolicy) bool { return p.VersionsEnabled }},
	domain.FeatureLibraryRead:      {enabled: func(p domain.WorkspaceFeaturePolicy) bool { return p.LibrariesEnabled }},
	domain.FeatureLibraryWrite:     {enabled: func(p domain.WorkspaceFeaturePolicy) bool { return p.LibrariesEnabled }},
	domain.FeatureCollectionManage: {enabled: func(p domain.WorkspaceFeaturePolicy) bool { return p.CollectionsEnabled }},
	domain.FeaturePresentationHost: {enabled: func(p domain.WorkspaceFeaturePolicy) bool { return p.PresentationEnabled }},
	domain.FeatureScreenPublish:    {enabled: func(p domain.WorkspaceFeaturePolicy) bool { return p.ScreenShareEnabled }},
	domain.FeatureEnterpriseManage: {enabled: func(domain.WorkspaceFeaturePolicy) bool { return true }},
}

func newFeatureActionPolicy() (*casbin.Enforcer, error) {
	m, err := model.NewModelFromString(featureActionModel)
	if err != nil {
		return nil, err
	}
	enforcer, err := casbin.NewEnforcer(m)
	if err != nil {
		return nil, err
	}
	for role, actions := range featureActionRules {
		for _, action := range actions {
			if _, err = enforcer.AddPolicy(role, string(action)); err != nil {
				return nil, err
			}
		}
	}
	return enforcer, nil
}

func featureRole(principal domain.Principal, decision domain.AccessDecision) string {
	switch principal.Kind {
	case domain.PrincipalPublic:
		return "public"
	case domain.PrincipalAgent:
		return "agent"
	case domain.PrincipalUser:
		if decision.Owner {
			return "owner"
		}
		if decision.Access == domain.AccessEdit {
			return "editor"
		}
		if decision.Access == domain.AccessView {
			return "viewer"
		}
	}
	return ""
}

func (s *FeatureService) Policy(ctx context.Context, principal domain.Principal, workspaceID uuid.UUID) (domain.WorkspaceFeaturePolicy, error) {
	if principal.Kind != domain.PrincipalUser {
		return domain.WorkspaceFeaturePolicy{}, domain.ErrAccessDenied
	}
	if _, err := s.access.ResolveWorkspace(ctx, principal, workspaceID); err != nil {
		return domain.WorkspaceFeaturePolicy{}, err
	}
	return s.repo.Policy(ctx, workspaceID)
}

func (s *FeatureService) Require(ctx context.Context, principal domain.Principal, ref domain.SceneRef, action domain.FeatureAction) error {
	if s.policyErr != nil {
		return s.policyErr
	}
	decision, err := s.access.Resolve(ctx, principal, ref)
	if err != nil {
		return err
	}
	policy, err := s.repo.Policy(ctx, ref.WorkspaceID)
	if err != nil {
		return err
	}
	role := featureRole(principal, decision)
	allowed, err := s.actionPolicy.Enforce(role, string(action))
	if err != nil {
		return err
	}
	if !allowed {
		return domain.ErrAccessDenied
	}
	requirement, ok := featureRequirements[action]
	if !ok {
		return domain.ErrAccessDenied
	}
	if err := requireEnabled(requirement.enabled(policy)); err != nil {
		return err
	}
	if principal.Kind == domain.PrincipalPublic {
		allowed, err := s.repo.PublicCommentsAllowed(ctx, principal, ref)
		if err != nil {
			return err
		}
		if !allowed {
			return domain.ErrAccessDenied
		}
		return nil
	}
	if role == "viewer" && requirement.viewerAllowed != nil && !requirement.viewerAllowed(policy) {
		return domain.ErrAccessDenied
	}
	return nil
}

func requireEnabled(enabled bool) error {
	if !enabled {
		return domain.ErrFeatureDisabled
	}
	return nil
}

func (s *FeatureService) Mutate(ctx context.Context, principal domain.Principal, ref domain.SceneRef, action domain.FeatureAction, requestID uuid.UUID, payload []byte, fn func(context.Context) (domain.StoredFeatureResponse, error)) (domain.StoredFeatureResponse, error) {
	if requestID == uuid.Nil {
		return domain.StoredFeatureResponse{}, domain.ErrInvalidFeatureInput
	}
	actor := uuid.Nil
	if principal.Kind == domain.PrincipalUser && principal.UserID != nil {
		actor = *principal.UserID
	}
	digest := sha256.Sum256(payload)
	var response domain.StoredFeatureResponse
	err := s.access.WithWorkspaceLock(ctx, ref.WorkspaceID, actor, func(txCtx context.Context) error {
		if err := s.Require(txCtx, principal, ref, action); err != nil {
			return err
		}
		var err error
		response, err = s.repo.WithMutation(txCtx, principal, ref, action, requestID, hex.EncodeToString(digest[:]), fn)
		return err
	})
	return response, err
}

func (s *FeatureService) UpdatePolicy(ctx context.Context, principal domain.Principal, workspaceID uuid.UUID, expectedRevision int64, requestID uuid.UUID, patch domain.WorkspaceFeaturePolicyPatch) (domain.WorkspaceFeaturePolicy, error) {
	if principal.Kind != domain.PrincipalUser || principal.UserID == nil || requestID == uuid.Nil || expectedRevision < 1 || patch.Empty() {
		return domain.WorkspaceFeaturePolicy{}, domain.ErrInvalidFeatureInput
	}
	payload, err := json.Marshal(struct {
		ExpectedRevision int64                              `json:"expectedRevision"`
		Patch            domain.WorkspaceFeaturePolicyPatch `json:"patch"`
	}{ExpectedRevision: expectedRevision, Patch: patch})
	if err != nil {
		return domain.WorkspaceFeaturePolicy{}, err
	}
	ref := domain.SceneRef{WorkspaceID: workspaceID}
	digest := sha256.Sum256(payload)
	var policy domain.WorkspaceFeaturePolicy
	err = s.access.WithWorkspaceLock(ctx, workspaceID, *principal.UserID, func(txCtx context.Context) error {
		if err := s.access.RequireOwner(txCtx, *principal.UserID, workspaceID); err != nil {
			return err
		}
		stored, err := s.repo.WithMutation(txCtx, principal, ref, domain.FeatureEnterpriseManage, requestID, hex.EncodeToString(digest[:]), func(callCtx context.Context) (domain.StoredFeatureResponse, error) {
			updated, err := s.repo.UpdatePolicy(callCtx, workspaceID, expectedRevision, patch)
			if err != nil {
				return domain.StoredFeatureResponse{}, err
			}
			body, err := json.Marshal(storedFeaturePolicy{Policy: updated, Revision: strconv.FormatInt(updated.Revision, 10)})
			if err != nil {
				return domain.StoredFeatureResponse{}, err
			}
			return domain.StoredFeatureResponse{Status: 200, Body: body}, nil
		})
		if err != nil {
			return err
		}
		var persisted storedFeaturePolicy
		if err := json.Unmarshal(stored.Body, &persisted); err != nil {
			return err
		}
		revision, err := strconv.ParseInt(persisted.Revision, 10, 64)
		if err != nil || revision < 1 || strconv.FormatInt(revision, 10) != persisted.Revision {
			return domain.ErrAccessDenied
		}
		policy = persisted.Policy
		policy.Revision = revision
		return nil
	})
	return policy, err
}
