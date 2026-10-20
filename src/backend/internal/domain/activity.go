package domain

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

var ErrInvalidActivity = errors.New("invalid activity request")

type ActivityCursor struct {
	OccurredAt time.Time
	ID         string
}

type ActivityItem struct {
	ID           string         `json:"id"`
	WorkspaceID  uuid.UUID      `json:"workspaceId"`
	SceneID      *uuid.UUID     `json:"sceneId,omitempty"`
	ActorKind    EventActorKind `json:"actorKind"`
	ActorName    string         `json:"actorName"`
	Kind         string         `json:"kind"`
	SubjectID    string         `json:"subjectId,omitempty"`
	OccurredAt   time.Time      `json:"occurredAt"`
	SafeMetadata map[string]any `json:"safeMetadata"`
}

type ActivityPage struct {
	Items         []ActivityItem `json:"items"`
	NextCursor    string         `json:"nextCursor,omitempty"`
	RetainedSince time.Time      `json:"retainedSince"`
}

type ActivityRepository interface {
	ListActivity(Context, uuid.UUID, uuid.UUID, *uuid.UUID, bool, *ActivityCursor, int) ([]ActivityItem, error)
	ActivityRetainedSince(Context, uuid.UUID, uuid.UUID, *uuid.UUID, bool) (time.Time, error)
}

type ActivityUsecase interface {
	List(Context, Principal, uuid.UUID, *uuid.UUID, string, int) (ActivityPage, error)
}
