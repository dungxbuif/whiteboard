package usecase

import (
	"context"
	"devtool/internal/domain"
	"errors"
	"testing"

	"github.com/google/uuid"
)

type collectionAccessStub struct {
	workspace domain.AccessDecision
	scenes    map[uuid.UUID]error
}

func (s collectionAccessStub) Resolve(_ domain.Context, _ domain.Principal, ref domain.SceneRef) (domain.AccessDecision, error) {
	if err := s.scenes[ref.SceneID]; err != nil {
		return domain.AccessDecision{}, err
	}
	return domain.AccessDecision{Access: domain.AccessView}, nil
}
func (s collectionAccessStub) ResolveWorkspace(domain.Context, domain.Principal, uuid.UUID) (domain.AccessDecision, error) {
	return s.workspace, nil
}
func (collectionAccessStub) WithWorkspaceLock(ctx domain.Context, _ uuid.UUID, _ uuid.UUID, fn func(domain.Context) error) error {
	return fn(ctx)
}

type collectionFeatureStub struct{ policy domain.WorkspaceFeaturePolicy }

func (s collectionFeatureStub) Policy(domain.Context, domain.Principal, uuid.UUID) (domain.WorkspaceFeaturePolicy, error) {
	return s.policy, nil
}

type collectionRepoStub struct {
	value        domain.Collection
	mutated      bool
	addCalled    bool
	removeCalled bool
	reorderErr   error
	deleted      bool
	deleteOwner  bool
}

func (s *collectionRepoStub) WithMutation(ctx domain.Context, _ domain.Principal, _ uuid.UUID, _ uuid.UUID, _ []byte, fn func(domain.Context) (domain.StoredFeatureResponse, error)) (domain.StoredFeatureResponse, error) {
	s.mutated = true
	return fn(ctx)
}
func (s *collectionRepoStub) Create(domain.Context, domain.Collection) (domain.Collection, error) {
	return s.value, nil
}
func (s *collectionRepoStub) Update(domain.Context, uuid.UUID, uuid.UUID, uuid.UUID, int64, string, domain.CollectionVisibility) (domain.Collection, error) {
	return s.value, nil
}
func (s *collectionRepoStub) Delete(_ domain.Context, _, _, _ uuid.UUID, _ int64, owner bool) error {
	s.deleted, s.deleteOwner = true, owner
	return nil
}
func (s *collectionRepoStub) Get(domain.Context, uuid.UUID, uuid.UUID, uuid.UUID, bool) (domain.Collection, error) {
	return s.value, nil
}
func (s *collectionRepoStub) List(domain.Context, uuid.UUID, uuid.UUID, bool, *uuid.UUID, int) ([]domain.Collection, error) {
	return []domain.Collection{s.value}, nil
}
func (s *collectionRepoStub) AddScene(domain.Context, uuid.UUID, uuid.UUID, uuid.UUID, uuid.UUID, int64) (domain.Collection, error) {
	s.addCalled = true
	return s.value, nil
}
func (s *collectionRepoStub) RemoveScene(domain.Context, uuid.UUID, uuid.UUID, uuid.UUID, uuid.UUID, int64) (domain.Collection, error) {
	s.removeCalled = true
	return s.value, nil
}
func (s *collectionRepoStub) Reorder(domain.Context, uuid.UUID, uuid.UUID, uuid.UUID, int64, []uuid.UUID) (domain.Collection, error) {
	return s.value, s.reorderErr
}

func enabledCollectionPolicy(workspaceID uuid.UUID) collectionFeatureStub {
	return collectionFeatureStub{policy: domain.WorkspaceFeaturePolicy{WorkspaceID: workspaceID, CollectionsEnabled: true}}
}

func TestPrivateCollectionDeniedToOtherMembers(t *testing.T) {
	workspaceID, creatorID, memberID := uuid.New(), uuid.New(), uuid.New()
	value := domain.Collection{ID: uuid.New(), WorkspaceID: workspaceID, CreatorID: creatorID, Visibility: domain.CollectionPrivate, Revision: 1}
	repo := &collectionRepoStub{value: value}
	service := NewCollectionService(repo, enabledCollectionPolicy(workspaceID), collectionAccessStub{workspace: domain.AccessDecision{Access: domain.AccessEdit}})
	if _, err := service.Get(context.Background(), domain.UserPrincipal(memberID), workspaceID, value.ID); !errors.Is(err, domain.ErrInaccessible) {
		t.Fatalf("private collection read error=%v", err)
	}
	ownerService := NewCollectionService(repo, enabledCollectionPolicy(workspaceID), collectionAccessStub{workspace: domain.AccessDecision{Access: domain.AccessEdit, Owner: true}})
	if _, err := ownerService.Get(context.Background(), domain.UserPrincipal(memberID), workspaceID, value.ID); !errors.Is(err, domain.ErrInaccessible) {
		t.Fatalf("owner learned private collection: %v", err)
	}
	if err := ownerService.Delete(context.Background(), domain.UserPrincipal(memberID), workspaceID, value.ID, domain.CollectionRevisionInput{RequestID: uuid.New(), ExpectedRevision: 1}); err != nil {
		t.Fatal(err)
	}
	if !repo.deleted || !repo.deleteOwner {
		t.Fatal("owner moderation delete was not executed")
	}
}

func TestCollectionNeverGrantsSceneAccess(t *testing.T) {
	workspaceID, userID, sceneID := uuid.New(), uuid.New(), uuid.New()
	repo := &collectionRepoStub{value: domain.Collection{ID: uuid.New(), WorkspaceID: workspaceID, CreatorID: userID, Visibility: domain.CollectionWorkspace, Revision: 1}}
	service := NewCollectionService(repo, enabledCollectionPolicy(workspaceID), collectionAccessStub{workspace: domain.AccessDecision{Access: domain.AccessEdit}, scenes: map[uuid.UUID]error{sceneID: domain.ErrInaccessible}})
	_, err := service.AddScene(context.Background(), domain.UserPrincipal(userID), workspaceID, repo.value.ID, domain.CollectionSceneInput{RequestID: uuid.New(), ExpectedRevision: 1, SceneID: sceneID})
	if !errors.Is(err, domain.ErrInaccessible) || repo.addCalled || repo.mutated {
		t.Fatalf("inaccessible scene add err=%v mutated=%v add=%v", err, repo.mutated, repo.addCalled)
	}
}

func TestCollectionRemoveDoesNotRevealInaccessibleMembership(t *testing.T) {
	workspaceID, userID, sceneID := uuid.New(), uuid.New(), uuid.New()
	repo := &collectionRepoStub{value: domain.Collection{ID: uuid.New(), WorkspaceID: workspaceID, CreatorID: userID, Visibility: domain.CollectionWorkspace, Revision: 1}}
	service := NewCollectionService(repo, enabledCollectionPolicy(workspaceID), collectionAccessStub{workspace: domain.AccessDecision{Access: domain.AccessEdit}, scenes: map[uuid.UUID]error{sceneID: domain.ErrInaccessible}})
	_, err := service.RemoveScene(context.Background(), domain.UserPrincipal(userID), workspaceID, repo.value.ID, domain.CollectionSceneInput{RequestID: uuid.New(), ExpectedRevision: 1, SceneID: sceneID})
	if !errors.Is(err, domain.ErrInaccessible) || repo.removeCalled || repo.mutated {
		t.Fatalf("inaccessible scene remove err=%v mutated=%v remove=%v", err, repo.mutated, repo.removeCalled)
	}
}

func TestCollectionForeignWorkspaceRejected(t *testing.T) {
	workspaceID, otherWorkspace, userID := uuid.New(), uuid.New(), uuid.New()
	repo := &collectionRepoStub{value: domain.Collection{ID: uuid.New(), WorkspaceID: otherWorkspace, CreatorID: userID, Visibility: domain.CollectionWorkspace, Revision: 1}}
	service := NewCollectionService(repo, enabledCollectionPolicy(workspaceID), collectionAccessStub{workspace: domain.AccessDecision{Access: domain.AccessEdit}})
	if _, err := service.Get(context.Background(), domain.UserPrincipal(userID), workspaceID, repo.value.ID); !errors.Is(err, domain.ErrInaccessible) {
		t.Fatalf("foreign collection error=%v", err)
	}
}

func TestCollectionOrderConflict(t *testing.T) {
	workspaceID, userID := uuid.New(), uuid.New()
	repo := &collectionRepoStub{value: domain.Collection{ID: uuid.New(), WorkspaceID: workspaceID, CreatorID: userID, Visibility: domain.CollectionWorkspace, Revision: 3}, reorderErr: domain.ErrRevisionConflict}
	service := NewCollectionService(repo, enabledCollectionPolicy(workspaceID), collectionAccessStub{workspace: domain.AccessDecision{Access: domain.AccessEdit}})
	_, err := service.Reorder(context.Background(), domain.UserPrincipal(userID), workspaceID, repo.value.ID, domain.CollectionReorderInput{RequestID: uuid.New(), ExpectedRevision: 2, SceneIDs: []uuid.UUID{uuid.New()}})
	if !errors.Is(err, domain.ErrRevisionConflict) {
		t.Fatalf("reorder error=%v", err)
	}
}
