package domain

import (
	"time"

	"github.com/google/uuid"
)

type Folder struct {
	ID          uuid.UUID
	WorkspaceID uuid.UUID
	ParentID    *uuid.UUID
	Name        string
	Path        string
	Depth       int
	IsStarred   bool
	DeletedAt   *time.Time
	CreatedBy   *uuid.UUID
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type FolderRepository interface {
	FindByID(ctx Context, workspaceID uuid.UUID, id uuid.UUID) (*Folder, error)
	ListByWorkspace(ctx Context, workspaceID uuid.UUID) ([]Folder, error)
	Save(ctx Context, folder *Folder) error
}
