package domain

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Folder struct {
	ID          uuid.UUID      `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	WorkspaceID uuid.UUID      `json:"workspaceId" gorm:"type:uuid;not null;index"`
	ParentID    *uuid.UUID     `json:"parentId" gorm:"type:uuid;index"`
	Name        string         `json:"name" gorm:"not null"`
	Path        string         `json:"path" gorm:"not null;index"`
	Depth       int            `json:"depth" gorm:"not null;default:0"`
	IsStarred   bool           `json:"isStarred" gorm:"not null;default:false"`
	CreatedBy   *uuid.UUID     `json:"createdBy,omitempty" gorm:"type:uuid"`
	CreatedAt   time.Time      `json:"createdAt"`
	UpdatedAt   time.Time      `json:"updatedAt"`
	DeletedAt   gorm.DeletedAt `json:"-" gorm:"index"`
}

type FolderRepository interface {
	FindByID(ctx Context, workspaceID uuid.UUID, id uuid.UUID) (*Folder, error)
	ListByWorkspace(ctx Context, workspaceID uuid.UUID) ([]Folder, error)
	ListByParent(ctx Context, workspaceID uuid.UUID, parentID *uuid.UUID) ([]Folder, error)
	Save(ctx Context, folder *Folder) error
	MaxDepthByPath(ctx Context, workspaceID uuid.UUID, pathPrefix string) (int, error)
	DeleteRecursive(ctx Context, workspaceID uuid.UUID, folderID uuid.UUID) error
	RestoreRecursive(ctx Context, workspaceID uuid.UUID, folderID uuid.UUID) error
	UpdatePathRecursive(ctx Context, workspaceID uuid.UUID, oldPathPrefix, newPathPrefix string, depthDelta int) error
}

type FolderUsecase interface {
	Create(ctx Context, workspaceID uuid.UUID, userID uuid.UUID, name string, parentID *uuid.UUID) (*Folder, error)
	Move(ctx Context, workspaceID uuid.UUID, folderID uuid.UUID, newParentID *uuid.UUID) error
	Delete(ctx Context, workspaceID uuid.UUID, folderID uuid.UUID) error
	Restore(ctx Context, workspaceID uuid.UUID, folderID uuid.UUID) error
	ListAll(ctx Context, workspaceID uuid.UUID) ([]Folder, error)
	List(ctx Context, workspaceID uuid.UUID, parentID *uuid.UUID) ([]Folder, error)
	ToggleStar(ctx Context, workspaceID uuid.UUID, folderID uuid.UUID) error
}
