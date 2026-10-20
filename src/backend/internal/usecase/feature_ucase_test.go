package usecase

import (
	"context"
	"devtool/internal/domain"
	"errors"
	"testing"

	"github.com/google/uuid"
)

type featureAccessStub struct {
	decision domain.AccessDecision
	err      error
}

func (s featureAccessStub) Resolve(context.Context, domain.Principal, domain.SceneRef) (domain.AccessDecision, error) {
	return s.decision, s.err
}
func (s featureAccessStub) ResolveWorkspace(context.Context, domain.Principal, uuid.UUID) (domain.AccessDecision, error) {
	return s.decision, s.err
}
func (s featureAccessStub) RequireOwner(context.Context, uuid.UUID, uuid.UUID) error {
	if s.decision.Owner {
		return nil
	}
	return domain.ErrAccessDenied
}
func (featureAccessStub) RequireFolderEdit(context.Context, domain.Principal, uuid.UUID, uuid.UUID) error {
	return domain.ErrAccessDenied
}
func (featureAccessStub) ResolveWorkspaceID(context.Context, string) (uuid.UUID, error) {
	return uuid.Nil, domain.ErrInaccessible
}
func (featureAccessStub) ResolveSceneID(context.Context, uuid.UUID, string) (uuid.UUID, error) {
	return uuid.Nil, domain.ErrInaccessible
}
func (featureAccessStub) WithWorkspaceLock(ctx context.Context, _ uuid.UUID, _ uuid.UUID, fn func(context.Context) error) error {
	return fn(ctx)
}
func (featureAccessStub) ListMembers(context.Context, uuid.UUID, uuid.UUID) ([]domain.WorkspaceMember, error) {
	return nil, nil
}
func (featureAccessStub) SetMember(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, *domain.WorkspaceRole) error {
	return nil
}
func (featureAccessStub) SetOverride(context.Context, uuid.UUID, domain.SceneRef, uuid.UUID, *domain.WorkspaceRole) error {
	return nil
}

type featureRepositoryStub struct {
	policy        domain.WorkspaceFeaturePolicy
	publicComment bool
}

func (s featureRepositoryStub) Policy(context.Context, uuid.UUID) (domain.WorkspaceFeaturePolicy, error) {
	return s.policy, nil
}
func (s featureRepositoryStub) PublicCommentsAllowed(context.Context, domain.Principal, domain.SceneRef) (bool, error) {
	return s.publicComment, nil
}
func (featureRepositoryStub) UpdatePolicy(context.Context, uuid.UUID, int64, domain.WorkspaceFeaturePolicyPatch) (domain.WorkspaceFeaturePolicy, error) {
	return domain.WorkspaceFeaturePolicy{}, nil
}
func (featureRepositoryStub) WithMutation(context.Context, domain.Principal, domain.SceneRef, domain.FeatureAction, uuid.UUID, string, func(context.Context) (domain.StoredFeatureResponse, error)) (domain.StoredFeatureResponse, error) {
	return domain.StoredFeatureResponse{}, errors.New("not implemented")
}

func TestFeaturePolicyMatrix(t *testing.T) {
	ref := domain.SceneRef{WorkspaceID: uuid.New(), SceneID: uuid.New()}
	policy := domain.DefaultWorkspaceFeaturePolicy(ref.WorkspaceID)
	policy.CommentsEnabled = true
	policy.VersionsEnabled = true
	policy.LibrariesEnabled = true
	policy.PresentationEnabled = true
	policy.ScreenShareEnabled = true

	cases := []struct {
		name      string
		principal domain.Principal
		decision  domain.AccessDecision
		action    domain.FeatureAction
		allowed   bool
	}{
		{"owner manages enterprise", domain.UserPrincipal(uuid.New()), domain.AccessDecision{Access: domain.AccessEdit, Owner: true}, domain.FeatureEnterpriseManage, true},
		{"editor restores version", domain.UserPrincipal(uuid.New()), domain.AccessDecision{Access: domain.AccessEdit}, domain.FeatureVersionRestore, true},
		{"viewer reads history", domain.UserPrincipal(uuid.New()), domain.AccessDecision{Access: domain.AccessView}, domain.FeatureVersionRead, true},
		{"viewer cannot restore", domain.UserPrincipal(uuid.New()), domain.AccessDecision{Access: domain.AccessView}, domain.FeatureVersionRestore, false},
		{"viewer comments by policy", domain.UserPrincipal(uuid.New()), domain.AccessDecision{Access: domain.AccessView}, domain.FeatureCommentWrite, true},
		{"public history denied", domain.Principal{Kind: domain.PrincipalPublic, ID: "guest:one", GrantID: uuid.NewString(), GrantVersion: "1"}, domain.AccessDecision{Access: domain.AccessView}, domain.FeatureVersionRead, false},
		{"agent gets no implicit comment scope", domain.Principal{Kind: domain.PrincipalAgent, ID: "agent:one"}, domain.AccessDecision{Access: domain.AccessEdit}, domain.FeatureCommentWrite, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			repo := featureRepositoryStub{policy: policy, publicComment: true}
			service := NewFeatureService(repo, featureAccessStub{decision: tc.decision})
			err := service.Require(context.Background(), tc.principal, ref, tc.action)
			if (err == nil) != tc.allowed {
				t.Fatalf("allowed=%v err=%v", tc.allowed, err)
			}
		})
	}
}

func TestCommentDoesNotGrantEdit(t *testing.T) {
	ref := domain.SceneRef{WorkspaceID: uuid.New(), SceneID: uuid.New()}
	policy := domain.DefaultWorkspaceFeaturePolicy(ref.WorkspaceID)
	policy.CommentsEnabled = true
	principal := domain.Principal{Kind: domain.PrincipalPublic, ID: "guest:commenter", GrantID: uuid.NewString(), GrantVersion: "1"}
	service := NewFeatureService(featureRepositoryStub{policy: policy, publicComment: true}, featureAccessStub{decision: domain.AccessDecision{Access: domain.AccessView}})
	if err := service.Require(context.Background(), principal, ref, domain.FeatureCommentWrite); err != nil {
		t.Fatalf("comment permission denied: %v", err)
	}
	if err := service.Require(context.Background(), principal, ref, domain.FeatureVersionRestore); !errors.Is(err, domain.ErrAccessDenied) {
		t.Fatalf("comment permission escalated: %v", err)
	}
}
