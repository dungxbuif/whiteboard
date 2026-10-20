package domain

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

var ErrInvalidNotification = errors.New("invalid notification input")

type Notification struct {
	ID          uuid.UUID  `json:"id"`
	UserID      uuid.UUID  `json:"-"`
	WorkspaceID uuid.UUID  `json:"workspaceId"`
	SceneID     uuid.UUID  `json:"sceneId"`
	EventID     int64      `json:"-"`
	Kind        string     `json:"kind"`
	ActorName   string     `json:"actorName"`
	ThreadID    uuid.UUID  `json:"threadId"`
	DeepLink    string     `json:"deepLink"`
	ReadAt      *time.Time `json:"readAt,omitempty"`
	CreatedAt   time.Time  `json:"createdAt"`
}

type NotificationPage struct {
	Items      []Notification `json:"items"`
	NextCursor string         `json:"nextCursor,omitempty"`
}

type NotificationRepository interface {
	Event(Context, int64) (WorkspaceEvent, error)
	Subscribers(Context, SceneRef, uuid.UUID) ([]uuid.UUID, error)
	ActorName(Context, WorkspaceEvent) (string, error)
	Create(Context, Notification) error
	List(Context, uuid.UUID, *uuid.UUID, int) ([]Notification, error)
	Get(Context, uuid.UUID, uuid.UUID) (Notification, error)
	MarkRead(Context, uuid.UUID, uuid.UUID, uuid.UUID) error
}

type NotificationUsecase interface {
	DeliverEvent(Context, int64) error
	List(Context, uuid.UUID, string, int) (NotificationPage, error)
	MarkRead(Context, uuid.UUID, uuid.UUID, uuid.UUID) error
}
