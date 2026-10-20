package usecase

import (
	"devtool/internal/domain"
	"encoding/json"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"
)

type collectionAccess interface {
	Resolve(domain.Context, domain.Principal, domain.SceneRef) (domain.AccessDecision, error)
	ResolveWorkspace(domain.Context, domain.Principal, uuid.UUID) (domain.AccessDecision, error)
	WithWorkspaceLock(domain.Context, uuid.UUID, uuid.UUID, func(domain.Context) error) error
}

type collectionFeatures interface {
	Policy(domain.Context, domain.Principal, uuid.UUID) (domain.WorkspaceFeaturePolicy, error)
}

type collectionEvents interface {
	AppendEvent(domain.Context, domain.EventInput) (domain.WorkspaceEvent, error)
}

type CollectionService struct {
	repo     domain.CollectionRepository
	features collectionFeatures
	access   collectionAccess
	events   collectionEvents
}

func NewCollectionService(repo domain.CollectionRepository, features collectionFeatures, access collectionAccess) *CollectionService {
	return &CollectionService{repo: repo, features: features, access: access}
}

func (s *CollectionService) SetEventService(events collectionEvents) { s.events = events }

func validCollectionName(value string) bool {
	return utf8.ValidString(value) && strings.TrimSpace(value) == value && utf8.RuneCountInString(value) >= 1 && utf8.RuneCountInString(value) <= 100
}

func validCollectionVisibility(value domain.CollectionVisibility) bool {
	return value == domain.CollectionWorkspace || value == domain.CollectionPrivate
}

func collectionUser(principal domain.Principal) (uuid.UUID, error) {
	if principal.Kind != domain.PrincipalUser || principal.UserID == nil || *principal.UserID == uuid.Nil || principal.ID != principal.UserID.String() {
		return uuid.Nil, domain.ErrAccessDenied
	}
	return *principal.UserID, nil
}

func (s *CollectionService) authorize(ctx domain.Context, principal domain.Principal, workspaceID uuid.UUID, write bool) (uuid.UUID, domain.AccessDecision, error) {
	userID, err := collectionUser(principal)
	if err != nil || workspaceID == uuid.Nil || s.repo == nil || s.features == nil || s.access == nil {
		return uuid.Nil, domain.AccessDecision{}, domain.ErrAccessDenied
	}
	decision, err := s.access.ResolveWorkspace(ctx, principal, workspaceID)
	if err != nil {
		return uuid.Nil, domain.AccessDecision{}, err
	}
	policy, err := s.features.Policy(ctx, principal, workspaceID)
	if err != nil {
		return uuid.Nil, domain.AccessDecision{}, err
	}
	if !policy.CollectionsEnabled {
		return uuid.Nil, domain.AccessDecision{}, domain.ErrFeatureDisabled
	}
	if write && decision.Access != domain.AccessEdit {
		return uuid.Nil, domain.AccessDecision{}, domain.ErrAccessDenied
	}
	return userID, decision, nil
}

func decodeCollection(stored domain.StoredFeatureResponse) (domain.Collection, error) {
	var value domain.Collection
	if stored.Status < 200 || stored.Status > 299 || json.Unmarshal(stored.Body, &value) != nil || value.ID == uuid.Nil {
		return domain.Collection{}, domain.ErrInvalidCollection
	}
	return value, nil
}

func storedCollection(value domain.Collection) (domain.StoredFeatureResponse, error) {
	body, err := json.Marshal(value)
	return domain.StoredFeatureResponse{Status: 200, Body: body}, err
}

func (s *CollectionService) appendEvent(ctx domain.Context, principal domain.Principal, workspaceID uuid.UUID, collectionID uuid.UUID, requestID uuid.UUID, kind string, revision int64) error {
	if s.events == nil {
		return nil
	}
	_, err := s.events.AppendEvent(ctx, domain.EventInput{WorkspaceID: workspaceID, ActorKind: domain.EventActorUser, ActorID: principal.ID, Kind: kind, SubjectID: collectionID.String(), SafeMetadata: map[string]any{"collectionId": collectionID.String(), "requestId": requestID.String(), "revision": revision}, Source: "collection", SourceKey: collectionID.String() + ":" + requestID.String()})
	return err
}

func (s *CollectionService) mutate(ctx domain.Context, principal domain.Principal, workspaceID, requestID uuid.UUID, payload any, fn func(domain.Context, uuid.UUID, domain.AccessDecision) (domain.StoredFeatureResponse, error)) (domain.StoredFeatureResponse, error) {
	if requestID == uuid.Nil {
		return domain.StoredFeatureResponse{}, domain.ErrInvalidCollection
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return domain.StoredFeatureResponse{}, err
	}
	userID, _, err := s.authorize(ctx, principal, workspaceID, true)
	if err != nil {
		return domain.StoredFeatureResponse{}, err
	}
	var result domain.StoredFeatureResponse
	err = s.access.WithWorkspaceLock(ctx, workspaceID, userID, func(txCtx domain.Context) error {
		_, decision, authorizeErr := s.authorize(txCtx, principal, workspaceID, true)
		if authorizeErr != nil {
			return authorizeErr
		}
		var mutationErr error
		result, mutationErr = s.repo.WithMutation(txCtx, principal, workspaceID, requestID, encoded, func(callCtx domain.Context) (domain.StoredFeatureResponse, error) {
			return fn(callCtx, userID, decision)
		})
		return mutationErr
	})
	return result, err
}

func (s *CollectionService) Create(ctx domain.Context, principal domain.Principal, workspaceID uuid.UUID, input domain.CollectionCreateInput) (domain.Collection, error) {
	if !validCollectionName(input.Name) || !validCollectionVisibility(input.Visibility) {
		return domain.Collection{}, domain.ErrInvalidCollection
	}
	stored, err := s.mutate(ctx, principal, workspaceID, input.RequestID, input, func(txCtx domain.Context, userID uuid.UUID, _ domain.AccessDecision) (domain.StoredFeatureResponse, error) {
		created, createErr := s.repo.Create(txCtx, domain.Collection{ID: uuid.New(), WorkspaceID: workspaceID, CreatorID: userID, Name: input.Name, Visibility: input.Visibility, Revision: 1, Scenes: []domain.CollectionScene{}})
		if createErr == nil {
			createErr = s.appendEvent(txCtx, principal, workspaceID, created.ID, input.RequestID, "collection.created", created.Revision)
		}
		if createErr != nil {
			return domain.StoredFeatureResponse{}, createErr
		}
		return storedCollection(created)
	})
	if err != nil {
		return domain.Collection{}, err
	}
	return decodeCollection(stored)
}

func (s *CollectionService) Update(ctx domain.Context, principal domain.Principal, workspaceID, collectionID uuid.UUID, input domain.CollectionUpdateInput) (domain.Collection, error) {
	if collectionID == uuid.Nil || input.ExpectedRevision < 1 || !validCollectionName(input.Name) || !validCollectionVisibility(input.Visibility) {
		return domain.Collection{}, domain.ErrInvalidCollection
	}
	stored, err := s.mutate(ctx, principal, workspaceID, input.RequestID, struct {
		ID    uuid.UUID
		Input domain.CollectionUpdateInput
	}{collectionID, input}, func(txCtx domain.Context, userID uuid.UUID, decision domain.AccessDecision) (domain.StoredFeatureResponse, error) {
		if _, getErr := s.repo.Get(txCtx, workspaceID, collectionID, userID, false); getErr != nil {
			return domain.StoredFeatureResponse{}, getErr
		}
		updated, updateErr := s.repo.Update(txCtx, workspaceID, collectionID, userID, input.ExpectedRevision, input.Name, input.Visibility)
		if updateErr == nil {
			updateErr = s.appendEvent(txCtx, principal, workspaceID, collectionID, input.RequestID, "collection.updated", updated.Revision)
		}
		if updateErr != nil {
			return domain.StoredFeatureResponse{}, updateErr
		}
		_ = decision
		return storedCollection(updated)
	})
	if err != nil {
		return domain.Collection{}, err
	}
	return decodeCollection(stored)
}

func (s *CollectionService) Delete(ctx domain.Context, principal domain.Principal, workspaceID, collectionID uuid.UUID, input domain.CollectionRevisionInput) error {
	if collectionID == uuid.Nil || input.ExpectedRevision < 1 {
		return domain.ErrInvalidCollection
	}
	_, err := s.mutate(ctx, principal, workspaceID, input.RequestID, struct {
		ID    uuid.UUID
		Input domain.CollectionRevisionInput
	}{collectionID, input}, func(txCtx domain.Context, userID uuid.UUID, decision domain.AccessDecision) (domain.StoredFeatureResponse, error) {
		if deleteErr := s.repo.Delete(txCtx, workspaceID, collectionID, userID, input.ExpectedRevision, decision.Owner); deleteErr != nil {
			return domain.StoredFeatureResponse{}, deleteErr
		}
		if eventErr := s.appendEvent(txCtx, principal, workspaceID, collectionID, input.RequestID, "collection.deleted", input.ExpectedRevision+1); eventErr != nil {
			return domain.StoredFeatureResponse{}, eventErr
		}
		return domain.StoredFeatureResponse{Status: 204, Body: json.RawMessage(`{"deleted":true}`)}, nil
	})
	return err
}

func (s *CollectionService) Get(ctx domain.Context, principal domain.Principal, workspaceID, collectionID uuid.UUID) (domain.Collection, error) {
	userID, decision, err := s.authorize(ctx, principal, workspaceID, false)
	if err != nil {
		return domain.Collection{}, err
	}
	if collectionID == uuid.Nil {
		return domain.Collection{}, domain.ErrInvalidCollection
	}
	value, err := s.repo.Get(ctx, workspaceID, collectionID, userID, decision.Owner)
	if err != nil {
		return domain.Collection{}, err
	}
	if value.WorkspaceID != workspaceID || value.Visibility == domain.CollectionPrivate && value.CreatorID != userID {
		return domain.Collection{}, domain.ErrInaccessible
	}
	return value, nil
}

func collectionCursor(raw string) (*uuid.UUID, error) {
	if raw == "" {
		return nil, nil
	}
	id, err := uuid.Parse(raw)
	if err != nil || id == uuid.Nil {
		return nil, domain.ErrInvalidCollection
	}
	return &id, nil
}

func (s *CollectionService) List(ctx domain.Context, principal domain.Principal, workspaceID uuid.UUID, cursor string, limit int) (domain.CollectionPage, error) {
	page := domain.CollectionPage{Items: []domain.Collection{}}
	userID, decision, err := s.authorize(ctx, principal, workspaceID, false)
	if err != nil || limit < 1 || limit > 50 {
		if err == nil {
			err = domain.ErrInvalidCollection
		}
		return page, err
	}
	parsed, err := collectionCursor(cursor)
	if err != nil {
		return page, err
	}
	rows, err := s.repo.List(ctx, workspaceID, userID, decision.Owner, parsed, limit+1)
	if err != nil {
		return page, err
	}
	if len(rows) > limit {
		page.NextCursor = rows[limit-1].ID.String()
		rows = rows[:limit]
	}
	page.Items = rows
	return page, nil
}

func (s *CollectionService) AddScene(ctx domain.Context, principal domain.Principal, workspaceID, collectionID uuid.UUID, input domain.CollectionSceneInput) (domain.Collection, error) {
	if collectionID == uuid.Nil || input.SceneID == uuid.Nil || input.ExpectedRevision < 1 {
		return domain.Collection{}, domain.ErrInvalidCollection
	}
	if _, err := s.access.Resolve(ctx, principal, domain.SceneRef{WorkspaceID: workspaceID, SceneID: input.SceneID}); err != nil {
		return domain.Collection{}, err
	}
	stored, err := s.mutate(ctx, principal, workspaceID, input.RequestID, struct {
		ID    uuid.UUID
		Input domain.CollectionSceneInput
	}{collectionID, input}, func(txCtx domain.Context, userID uuid.UUID, _ domain.AccessDecision) (domain.StoredFeatureResponse, error) {
		if _, accessErr := s.access.Resolve(txCtx, principal, domain.SceneRef{WorkspaceID: workspaceID, SceneID: input.SceneID}); accessErr != nil {
			return domain.StoredFeatureResponse{}, accessErr
		}
		if _, getErr := s.repo.Get(txCtx, workspaceID, collectionID, userID, false); getErr != nil {
			return domain.StoredFeatureResponse{}, getErr
		}
		updated, addErr := s.repo.AddScene(txCtx, workspaceID, collectionID, userID, input.SceneID, input.ExpectedRevision)
		if addErr == nil {
			addErr = s.appendEvent(txCtx, principal, workspaceID, collectionID, input.RequestID, "collection.scene_added", updated.Revision)
		}
		if addErr != nil {
			return domain.StoredFeatureResponse{}, addErr
		}
		return storedCollection(updated)
	})
	if err != nil {
		return domain.Collection{}, err
	}
	return decodeCollection(stored)
}

func (s *CollectionService) RemoveScene(ctx domain.Context, principal domain.Principal, workspaceID, collectionID uuid.UUID, input domain.CollectionSceneInput) (domain.Collection, error) {
	if collectionID == uuid.Nil || input.SceneID == uuid.Nil || input.ExpectedRevision < 1 {
		return domain.Collection{}, domain.ErrInvalidCollection
	}
	stored, err := s.mutate(ctx, principal, workspaceID, input.RequestID, struct {
		ID    uuid.UUID
		Input domain.CollectionSceneInput
	}{collectionID, input}, func(txCtx domain.Context, userID uuid.UUID, _ domain.AccessDecision) (domain.StoredFeatureResponse, error) {
		if _, getErr := s.repo.Get(txCtx, workspaceID, collectionID, userID, false); getErr != nil {
			return domain.StoredFeatureResponse{}, getErr
		}
		updated, removeErr := s.repo.RemoveScene(txCtx, workspaceID, collectionID, userID, input.SceneID, input.ExpectedRevision)
		if removeErr == nil {
			removeErr = s.appendEvent(txCtx, principal, workspaceID, collectionID, input.RequestID, "collection.scene_removed", updated.Revision)
		}
		if removeErr != nil {
			return domain.StoredFeatureResponse{}, removeErr
		}
		return storedCollection(updated)
	})
	if err != nil {
		return domain.Collection{}, err
	}
	return decodeCollection(stored)
}

func (s *CollectionService) Reorder(ctx domain.Context, principal domain.Principal, workspaceID, collectionID uuid.UUID, input domain.CollectionReorderInput) (domain.Collection, error) {
	if collectionID == uuid.Nil || input.ExpectedRevision < 1 || len(input.SceneIDs) > 500 {
		return domain.Collection{}, domain.ErrInvalidCollection
	}
	seen := map[uuid.UUID]bool{}
	for _, sceneID := range input.SceneIDs {
		if sceneID == uuid.Nil || seen[sceneID] {
			return domain.Collection{}, domain.ErrInvalidCollection
		}
		seen[sceneID] = true
	}
	stored, err := s.mutate(ctx, principal, workspaceID, input.RequestID, struct {
		ID    uuid.UUID
		Input domain.CollectionReorderInput
	}{collectionID, input}, func(txCtx domain.Context, userID uuid.UUID, _ domain.AccessDecision) (domain.StoredFeatureResponse, error) {
		if _, getErr := s.repo.Get(txCtx, workspaceID, collectionID, userID, false); getErr != nil {
			return domain.StoredFeatureResponse{}, getErr
		}
		for _, sceneID := range input.SceneIDs {
			if _, accessErr := s.access.Resolve(txCtx, principal, domain.SceneRef{WorkspaceID: workspaceID, SceneID: sceneID}); accessErr != nil {
				return domain.StoredFeatureResponse{}, accessErr
			}
		}
		updated, reorderErr := s.repo.Reorder(txCtx, workspaceID, collectionID, userID, input.ExpectedRevision, input.SceneIDs)
		if reorderErr == nil {
			reorderErr = s.appendEvent(txCtx, principal, workspaceID, collectionID, input.RequestID, "collection.reordered", updated.Revision)
		}
		if reorderErr != nil {
			return domain.StoredFeatureResponse{}, reorderErr
		}
		return storedCollection(updated)
	})
	if err != nil {
		return domain.Collection{}, err
	}
	return decodeCollection(stored)
}

var _ domain.CollectionUsecase = (*CollectionService)(nil)
