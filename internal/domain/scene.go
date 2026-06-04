package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type Scene struct {
	ID          uuid.UUID
	WorkspaceID uuid.UUID
	FolderID    *uuid.UUID
	Title       string
	Content     json.RawMessage
	IsStarred   bool
	IsTrashed   bool
	CreatedBy   uuid.UUID
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type SceneRepository interface {
	FindByID(ctx Context, workspaceID uuid.UUID, id uuid.UUID) (*Scene, error)
	ListByWorkspace(ctx Context, workspaceID uuid.UUID) ([]Scene, error)
	Save(ctx Context, scene *Scene) error
}
