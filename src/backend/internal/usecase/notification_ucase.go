package usecase

import (
	"devtool/internal/domain"
	"errors"
	"fmt"

	"github.com/google/uuid"
)

type NotificationService struct {
	repo     domain.NotificationRepository
	features domain.FeatureService
	access   domain.AccessService
}

func NewNotificationService(repo domain.NotificationRepository, features domain.FeatureService, access domain.AccessService) *NotificationService {
	return &NotificationService{repo: repo, features: features, access: access}
}

func notificationThreadID(event domain.WorkspaceEvent) (uuid.UUID, error) {
	value, ok := event.SafeMetadata["threadId"].(string)
	if !ok {
		return uuid.Nil, domain.ErrInvalidNotification
	}
	return uuid.Parse(value)
}

func (s *NotificationService) DeliverEvent(ctx domain.Context, eventID int64) error {
	if eventID < 1 || s.repo == nil || s.features == nil || s.access == nil {
		return domain.ErrInvalidNotification
	}
	event, err := s.repo.Event(ctx, eventID)
	if err != nil {
		return err
	}
	if event.SceneID == nil || len(event.Kind) < len("comment.") || event.Kind[:len("comment.")] != "comment." {
		return nil
	}
	threadID, err := notificationThreadID(event)
	if err != nil {
		return err
	}
	ref := domain.SceneRef{WorkspaceID: event.WorkspaceID, SceneID: *event.SceneID}
	return s.access.WithWorkspaceLock(ctx, event.WorkspaceID, uuid.Nil, func(txCtx domain.Context) error {
		current, loadErr := s.repo.Event(txCtx, eventID)
		if loadErr != nil {
			return loadErr
		}
		recipients, loadErr := s.repo.Subscribers(txCtx, ref, threadID)
		if loadErr != nil {
			return loadErr
		}
		actorName, loadErr := s.repo.ActorName(txCtx, current)
		if loadErr != nil {
			return loadErr
		}
		for _, userID := range recipients {
			if current.ActorKind == domain.EventActorUser && current.ActorID == userID.String() {
				continue
			}
			if requireErr := s.features.Require(txCtx, domain.UserPrincipal(userID), ref, domain.FeatureCommentRead); requireErr != nil {
				if errors.Is(requireErr, domain.ErrAccessDenied) || errors.Is(requireErr, domain.ErrInaccessible) || errors.Is(requireErr, domain.ErrFeatureDisabled) {
					continue
				}
				return requireErr
			}
			value := domain.Notification{ID: uuid.New(), UserID: userID, WorkspaceID: ref.WorkspaceID, SceneID: ref.SceneID, EventID: eventID, Kind: current.Kind, ActorName: actorName, ThreadID: threadID}
			if createErr := s.repo.Create(txCtx, value); createErr != nil {
				return createErr
			}
		}
		return nil
	})
}

func (s *NotificationService) authorized(ctx domain.Context, userID uuid.UUID, value domain.Notification) (bool, error) {
	ref := domain.SceneRef{WorkspaceID: value.WorkspaceID, SceneID: value.SceneID}
	err := s.access.WithWorkspaceLock(ctx, value.WorkspaceID, userID, func(txCtx domain.Context) error {
		return s.features.Require(txCtx, domain.UserPrincipal(userID), ref, domain.FeatureCommentRead)
	})
	if err == nil {
		return true, nil
	}
	if errors.Is(err, domain.ErrAccessDenied) || errors.Is(err, domain.ErrInaccessible) || errors.Is(err, domain.ErrFeatureDisabled) {
		return false, nil
	}
	return false, err
}

func (s *NotificationService) List(ctx domain.Context, userID uuid.UUID, cursor string, limit int) (domain.NotificationPage, error) {
	if userID == uuid.Nil || limit < 1 || limit > 50 || s.repo == nil || s.features == nil || s.access == nil {
		return domain.NotificationPage{}, domain.ErrInvalidNotification
	}
	var cursorID *uuid.UUID
	if cursor != "" {
		id, err := uuid.Parse(cursor)
		if err != nil {
			return domain.NotificationPage{}, domain.ErrInvalidNotification
		}
		cursorID = &id
	}
	items := make([]domain.Notification, 0, limit)
	next := ""
	for len(items) < limit {
		rows, err := s.repo.List(ctx, userID, cursorID, 51)
		if err != nil {
			return domain.NotificationPage{}, err
		}
		if len(rows) == 0 {
			break
		}
		scan := rows
		if len(scan) > 50 {
			scan = scan[:50]
		}
		for index, value := range scan {
			id := value.ID
			cursorID = &id
			allowed, authErr := s.authorized(ctx, userID, value)
			if authErr != nil {
				return domain.NotificationPage{}, authErr
			}
			if allowed {
				value.DeepLink = fmt.Sprintf("/w/%s/scenes/%s?thread=%s", value.WorkspaceID, value.SceneID, value.ThreadID)
				items = append(items, value)
			}
			if len(items) == limit {
				if index < len(scan)-1 || len(rows) > 50 {
					next = value.ID.String()
				}
				return domain.NotificationPage{Items: items, NextCursor: next}, nil
			}
		}
		if len(rows) <= 50 {
			break
		}
	}
	return domain.NotificationPage{Items: items, NextCursor: next}, nil
}

func (s *NotificationService) MarkRead(ctx domain.Context, userID, id, requestID uuid.UUID) error {
	if userID == uuid.Nil || id == uuid.Nil || requestID == uuid.Nil || s.repo == nil || s.features == nil || s.access == nil {
		return domain.ErrInvalidNotification
	}
	value, err := s.repo.Get(ctx, userID, id)
	if err != nil {
		return err
	}
	return s.access.WithWorkspaceLock(ctx, value.WorkspaceID, userID, func(txCtx domain.Context) error {
		current, loadErr := s.repo.Get(txCtx, userID, id)
		if loadErr != nil {
			return loadErr
		}
		ref := domain.SceneRef{WorkspaceID: current.WorkspaceID, SceneID: current.SceneID}
		if loadErr = s.features.Require(txCtx, domain.UserPrincipal(userID), ref, domain.FeatureCommentRead); loadErr != nil {
			return loadErr
		}
		return s.repo.MarkRead(txCtx, userID, id, requestID)
	})
}

var _ domain.NotificationUsecase = (*NotificationService)(nil)
