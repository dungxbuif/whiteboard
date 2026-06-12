package domain

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type WorkspaceRole string

const (
	WorkspaceRoleOwner  WorkspaceRole = "owner"
	WorkspaceRoleMember WorkspaceRole = "member"
)

type Workspace struct {
	ID        uuid.UUID      `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Name      string         `json:"name" gorm:"not null"`
	OwnerID   uuid.UUID      `json:"ownerId" gorm:"type:uuid;not null;index"`
	CreatedAt time.Time      `json:"createdAt"`
	UpdatedAt time.Time      `json:"updatedAt"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
}

type WorkspaceMember struct {
	WorkspaceID uuid.UUID
	UserID      uuid.UUID
	Role        WorkspaceRole
	CreatedAt   time.Time
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
