package domain

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type WorkspaceRole string

const (
	WorkspaceRoleOwner  WorkspaceRole = "owner"
	WorkspaceRoleMember WorkspaceRole = "member" // Legacy migration input only.
	WorkspaceRoleEditor WorkspaceRole = "editor"
	WorkspaceRoleViewer WorkspaceRole = "viewer"
)

type Workspace struct {
	ID         uuid.UUID      `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	PublicID   *string        `json:"publicId,omitempty"`
	ACLVersion int64          `json:"aclVersion" gorm:"default:1"`
	Name       string         `json:"name" gorm:"not null"`
	OwnerID    uuid.UUID      `json:"ownerId" gorm:"type:uuid;not null;index"`
	CreatedAt  time.Time      `json:"createdAt"`
	UpdatedAt  time.Time      `json:"updatedAt"`
	DeletedAt  gorm.DeletedAt `json:"-" gorm:"index"`
}

type WorkspaceMember struct {
	WorkspaceID uuid.UUID     `json:"workspaceId" gorm:"primaryKey"`
	UserID      uuid.UUID     `json:"userId" gorm:"primaryKey"`
	Role        WorkspaceRole `json:"role"`
	CreatedAt   time.Time     `json:"createdAt"`
}

type WorkspaceRepository interface {
	FindByID(ctx Context, id uuid.UUID) (*Workspace, error)
	ListForUser(ctx Context, userID uuid.UUID) ([]Workspace, error)
	Save(ctx Context, workspace *Workspace) error
}

type WorkspaceUsecase interface {
	Create(ctx Context, userID uuid.UUID, name string) (*Workspace, error)
	ListByUser(ctx Context, userID uuid.UUID) ([]Workspace, error)
}
