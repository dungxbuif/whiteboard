package domain

import (
	"time"

	"github.com/google/uuid"
)

type WorkspaceRole string

const (
	WorkspaceRoleOwner  WorkspaceRole = "owner"
	WorkspaceRoleMember WorkspaceRole = "member"
)

type Workspace struct {
	ID        uuid.UUID
	Name      string
	OwnerID   uuid.UUID
	CreatedAt time.Time
	UpdatedAt time.Time
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
