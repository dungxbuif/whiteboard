package domain

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Folder struct {
	ID          uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	WorkspaceID uuid.UUID      `gorm:"type:uuid;not null;index"`
	ParentID    *uuid.UUID     `gorm:"type:uuid;index"`
	Name        string         `gorm:"not null"`
	Path        string         `gorm:"not null;index"`
	Depth       int            `gorm:"not null;default:0"`
	IsStarred   bool           `gorm:"not null;default:false"`
	CreatedBy   *uuid.UUID     `gorm:"type:uuid"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   gorm.DeletedAt `gorm:"index"`
}

type FolderRepository interface {
	FindByID(ctx Context, workspaceID uuid.UUID, id uuid.UUID) (*Folder, error)
	ListByWorkspace(ctx Context, workspaceID uuid.UUID) ([]Folder, error)
	ListByParent(ctx Context, workspaceID uuid.UUID, parentID *uuid.UUID) ([]Folder, error)
	Save(ctx Context, folder *Folder) error
	DeleteRecursive(ctx Context, workspaceID uuid.UUID, folderID uuid.UUID) error
	UpdatePathRecursive(ctx Context, workspaceID uuid.UUID, oldPathPrefix, newPathPrefix string) error
}

type FolderUsecase interface {
	Create(ctx Context, workspaceID uuid.UUID, userID uuid.UUID, name string, parentID *uuid.UUID) (*Folder, error)
	Move(ctx Context, workspaceID uuid.UUID, folderID uuid.UUID, newParentID *uuid.UUID) error
	Delete(ctx Context, workspaceID uuid.UUID, folderID uuid.UUID) error
	List(ctx Context, workspaceID uuid.UUID, parentID *uuid.UUID) ([]Folder, error)
	ToggleStar(ctx Context, workspaceID uuid.UUID, folderID uuid.UUID) error
}
