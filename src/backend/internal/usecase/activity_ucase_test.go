package usecase

import (
	"context"
	"devtool/internal/domain"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
)

type activityRepoStub struct {
	items    []domain.ActivityItem
	retained time.Time
	owner    bool
	sceneID  *uuid.UUID
}

func (s *activityRepoStub) ListActivity(_ domain.Context, _ uuid.UUID, _ uuid.UUID, sceneID *uuid.UUID, owner bool, _ *domain.ActivityCursor, _ int) ([]domain.ActivityItem, error) {
	s.owner, s.sceneID = owner, sceneID
	return append([]domain.ActivityItem(nil), s.items...), nil
}

func (s *activityRepoStub) ActivityRetainedSince(domain.Context, uuid.UUID, uuid.UUID, *uuid.UUID, bool) (time.Time, error) {
	return s.retained, nil
}

type activityFeatureStub struct {
	policy domain.WorkspaceFeaturePolicy
	err    error
}

func (s activityFeatureStub) Require(domain.Context, domain.Principal, domain.SceneRef, domain.FeatureAction) error {
	return s.err
}
func (s activityFeatureStub) Policy(domain.Context, domain.Principal, uuid.UUID) (domain.WorkspaceFeaturePolicy, error) {
	return s.policy, s.err
}
func (activityFeatureStub) UpdatePolicy(domain.Context, domain.Principal, uuid.UUID, int64, uuid.UUID, domain.WorkspaceFeaturePolicyPatch) (domain.WorkspaceFeaturePolicy, error) {
	return domain.WorkspaceFeaturePolicy{}, nil
}
func (activityFeatureStub) Mutate(domain.Context, domain.Principal, domain.SceneRef, domain.FeatureAction, uuid.UUID, []byte, func(domain.Context) (domain.StoredFeatureResponse, error)) (domain.StoredFeatureResponse, error) {
	return domain.StoredFeatureResponse{}, nil
}

type activityAccessStub struct {
	decision domain.AccessDecision
	err      error
}

func (s activityAccessStub) ResolveWorkspace(domain.Context, domain.Principal, uuid.UUID) (domain.AccessDecision, error) {
	return s.decision, s.err
}
func (s activityAccessStub) Resolve(domain.Context, domain.Principal, domain.SceneRef) (domain.AccessDecision, error) {
	return s.decision, s.err
}

func TestActivityListCarriesOwnerAndSceneScope(t *testing.T) {
	workspaceID, sceneID, userID := uuid.New(), uuid.New(), uuid.New()
	now := time.Now().UTC()
	repo := &activityRepoStub{items: []domain.ActivityItem{{ID: "2", OccurredAt: now}, {ID: "1", OccurredAt: now.Add(-time.Second)}}, retained: now.Add(-time.Hour)}
	service := NewActivityService(repo, activityFeatureStub{policy: domain.WorkspaceFeaturePolicy{ActivityEnabled: true}}, activityAccessStub{decision: domain.AccessDecision{Access: domain.AccessEdit, Owner: true}})
	page, err := service.List(context.Background(), domain.UserPrincipal(userID), workspaceID, &sceneID, "", 1)
	if err != nil {
		t.Fatal(err)
	}
	if !repo.owner || repo.sceneID == nil || *repo.sceneID != sceneID || len(page.Items) != 1 || page.NextCursor == "" || page.RetainedSince == nil || !page.RetainedSince.Equal(repo.retained) {
		t.Fatalf("page=%+v owner=%v scene=%v", page, repo.owner, repo.sceneID)
	}
}

func TestActivityListRejectsDisabledAndPublicPrincipals(t *testing.T) {
	workspaceID := uuid.New()
	service := NewActivityService(&activityRepoStub{}, activityFeatureStub{policy: domain.WorkspaceFeaturePolicy{}}, activityAccessStub{decision: domain.AccessDecision{Access: domain.AccessView}})
	if _, err := service.List(context.Background(), domain.UserPrincipal(uuid.New()), workspaceID, nil, "", 50); !errors.Is(err, domain.ErrFeatureDisabled) {
		t.Fatalf("disabled activity err=%v", err)
	}
	public := domain.Principal{Kind: domain.PrincipalPublic, ID: "guest:test"}
	if _, err := service.List(context.Background(), public, workspaceID, nil, "", 50); !errors.Is(err, domain.ErrAccessDenied) {
		t.Fatalf("public activity err=%v", err)
	}
}

func TestActivityCursorRejectsTampering(t *testing.T) {
	service := NewActivityService(&activityRepoStub{}, activityFeatureStub{policy: domain.WorkspaceFeaturePolicy{ActivityEnabled: true}}, activityAccessStub{decision: domain.AccessDecision{Access: domain.AccessView}})
	if _, err := service.List(context.Background(), domain.UserPrincipal(uuid.New()), uuid.New(), nil, "not-a-cursor", 50); !errors.Is(err, domain.ErrInvalidActivity) {
		t.Fatalf("tampered cursor err=%v", err)
	}
}

func TestActivityListRejectsExpiredCredential(t *testing.T) {
	principal := domain.UserPrincipal(uuid.New())
	principal.CredentialExpiresAt = time.Now().Add(-time.Minute)
	service := NewActivityService(&activityRepoStub{}, activityFeatureStub{policy: domain.WorkspaceFeaturePolicy{ActivityEnabled: true}}, activityAccessStub{decision: domain.AccessDecision{Access: domain.AccessView}})
	if _, err := service.List(context.Background(), principal, uuid.New(), nil, "", 50); !errors.Is(err, domain.ErrInvalidSceneSession) {
		t.Fatalf("expired credential err=%v", err)
	}
}
