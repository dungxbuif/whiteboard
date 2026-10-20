package domain

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

type EventActorKind string

const (
	EventActorUser   EventActorKind = "user"
	EventActorPublic EventActorKind = "public"
	EventActorAgent  EventActorKind = "agent"
	EventActorSystem EventActorKind = "system"
)

var (
	ErrEventInput          = errors.New("invalid workspace event")
	ErrEventSourceReused   = errors.New("workspace event source key reused")
	ErrUnsafeEventMetadata = errors.New("unsafe workspace event metadata")
)

type EventInput struct {
	WorkspaceID  uuid.UUID
	SceneID      *uuid.UUID
	ActorKind    EventActorKind
	ActorID      string
	Kind         string
	SubjectID    string
	OccurredAt   time.Time
	SafeMetadata map[string]any
	Source       string
	SourceKey    string
}

type WorkspaceEvent struct {
	ID           string         `json:"id" gorm:"primaryKey;column:id"`
	WorkspaceID  uuid.UUID      `json:"workspaceId"`
	SceneID      *uuid.UUID     `json:"sceneId,omitempty"`
	ActorKind    EventActorKind `json:"actorKind"`
	ActorID      string         `json:"actorId"`
	Kind         string         `json:"kind"`
	SubjectID    string         `json:"subjectId,omitempty"`
	OccurredAt   time.Time      `json:"occurredAt"`
	SafeMetadata map[string]any `json:"safeMetadata" gorm:"serializer:json;type:jsonb"`
	Source       string         `json:"-"`
	SourceKey    string         `json:"-"`
}

func (WorkspaceEvent) TableName() string { return "workspace_events" }

type WorkspaceEventPage struct {
	Items      []WorkspaceEvent `json:"items"`
	NextCursor string           `json:"nextCursor,omitempty"`
}

type EventRepository interface {
	AppendEvent(Context, EventInput) (WorkspaceEvent, error)
	ProjectAuditBatch(Context, int) (int, error)
	ListWorkspaceEvents(Context, uuid.UUID, string, int) ([]WorkspaceEvent, error)
}
