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
	ID        uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Name      string         `gorm:"not null"`
	OwnerID   uuid.UUID      `gorm:"type:uuid;not null;index"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`
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
