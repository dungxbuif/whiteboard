package usecase

import (
	"context"
	"crypto/sha256"
	"devtool/internal/domain"
	"encoding/hex"
	"encoding/json"
	"strconv"

	"github.com/google/uuid"
)

type FeatureService struct {
	repo   domain.FeatureRepository
	access domain.AccessService
}

type storedFeaturePolicy struct {
	Policy   domain.WorkspaceFeaturePolicy `json:"policy"`
	Revision string                        `json:"revision"`
}

func NewFeatureService(repo domain.FeatureRepository, access domain.AccessService) *FeatureService {
	return &FeatureService{repo: repo, access: access}
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
	decision, err := s.access.Resolve(ctx, principal, ref)
	if err != nil {
		return err
	}
	policy, err := s.repo.Policy(ctx, ref.WorkspaceID)
	if err != nil {
		return err
	}
	if principal.Kind == domain.PrincipalAgent {
		return domain.ErrAccessDenied
	}
	if principal.Kind == domain.PrincipalPublic {
		if action != domain.FeatureCommentRead && action != domain.FeatureCommentWrite {
			return domain.ErrAccessDenied
		}
		if !policy.CommentsEnabled {
			return domain.ErrFeatureDisabled
		}
		allowed, err := s.repo.PublicCommentsAllowed(ctx, principal, ref)
		if err != nil {
			return err
		}
		if !allowed {
			return domain.ErrAccessDenied
		}
		return nil
	}
	if principal.Kind != domain.PrincipalUser {
		return domain.ErrAccessDenied
	}

	canEdit := decision.Access == domain.AccessEdit
	switch action {
	case domain.FeatureCommentRead:
		return requireEnabled(policy.CommentsEnabled)
	case domain.FeatureCommentWrite:
		if err := requireEnabled(policy.CommentsEnabled); err != nil {
			return err
		}
		if canEdit || policy.MemberComments {
			return nil
		}
	case domain.FeatureCommentModerate:
		if err := requireEnabled(policy.CommentsEnabled); err != nil {
			return err
		}
		if decision.Owner {
			return nil
		}
	case domain.FeatureActivityRead:
		return requireEnabled(policy.ActivityEnabled)
	case domain.FeatureVersionRead:
		if err := requireEnabled(policy.VersionsEnabled); err != nil {
			return err
		}
		if canEdit || policy.MemberHistory {
			return nil
		}
	case domain.FeatureVersionCreate, domain.FeatureVersionRestore:
		if err := requireEnabled(policy.VersionsEnabled); err != nil {
			return err
		}
		if canEdit {
			return nil
		}
	case domain.FeatureLibraryRead:
		return requireEnabled(policy.LibrariesEnabled)
	case domain.FeatureLibraryWrite:
		if err := requireEnabled(policy.LibrariesEnabled); err != nil {
			return err
		}
		if decision.Owner {
			return nil
		}
	case domain.FeatureCollectionManage:
		if err := requireEnabled(policy.CollectionsEnabled); err != nil {
			return err
		}
		if canEdit {
			return nil
		}
	case domain.FeaturePresentationHost:
		if err := requireEnabled(policy.PresentationEnabled); err != nil {
			return err
		}
		if canEdit {
			return nil
		}
	case domain.FeatureScreenPublish:
		if err := requireEnabled(policy.ScreenShareEnabled); err != nil {
			return err
		}
		if canEdit {
			return nil
		}
	case domain.FeatureEnterpriseManage:
		if decision.Owner {
			return nil
		}
	default:
		return domain.ErrAccessDenied
	}
	return domain.ErrAccessDenied
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
