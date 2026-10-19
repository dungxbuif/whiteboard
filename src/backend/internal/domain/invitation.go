package domain

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

const InvitationLifetime = 7 * 24 * time.Hour

var (
	ErrInvitationInvalid    = errors.New("invitation is invalid or unavailable")
	ErrInvitationExpired    = errors.New("invitation expired")
	ErrInvitationEmail      = errors.New("invitation email does not match")
	ErrRegistrationRejected = errors.New("cannot register; log in if you already have an account or ask for a new invitation")
	ErrInvitationInput      = errors.New("invalid invitation input")
)

type Invitation struct {
	ID          uuid.UUID     `json:"id" gorm:"type:uuid;primaryKey"`
	WorkspaceID uuid.UUID     `json:"workspaceId"`
	Email       string        `json:"email"`
	Role        WorkspaceRole `json:"role"`
	InviterID   uuid.UUID     `json:"inviterId"`
	TokenHash   string        `json:"-"`
	CreatedAt   time.Time     `json:"createdAt"`
	ExpiresAt   time.Time     `json:"expiresAt"`
	AcceptedAt  *time.Time    `json:"acceptedAt,omitempty"`
	RevokedAt   *time.Time    `json:"revokedAt,omitempty"`
}

func (Invitation) TableName() string { return "workspace_invitations" }

type InviteResult struct {
	Invitation Invitation `json:"invitation"`
	InviteURL  string     `json:"inviteUrl"`
}
type RegistrationInput struct {
	Email    string `json:"email" validate:"required,email,max=254"`
	Password string `json:"password" validate:"required,min=8,max=72"`
	Name     string `json:"name" validate:"required,max=200"`
}
type UserProvisioningRepository interface {
	FindDefaultWorkspace(Context, uuid.UUID) (*Workspace, error)
	CreateDefaultWorkspace(Context, *Workspace) error
}
type InvitationRepository interface {
	UserProvisioningRepository
	Create(Context, *Invitation) error
	ListPending(Context, uuid.UUID, time.Time) ([]Invitation, error)
	Revoke(Context, uuid.UUID, uuid.UUID, time.Time) error
	FindByHash(Context, string, bool) (*Invitation, error)
	FindUser(Context, uuid.UUID) (*User, error)
	EmailExists(Context, string) (bool, error)
	CreateUser(Context, *User) error
	Join(Context, *Invitation, uuid.UUID, time.Time) (WorkspaceMember, error)
}
type InvitationUsecase interface {
	CreateInvitation(Context, uuid.UUID, uuid.UUID, string, WorkspaceRole) (InviteResult, error)
	ListInvitations(Context, uuid.UUID, uuid.UUID) ([]Invitation, error)
	RevokeInvitation(Context, uuid.UUID, uuid.UUID, uuid.UUID) error
	AcceptInvitation(Context, uuid.UUID, string) (WorkspaceMember, error)
	RegisterWithInvitation(Context, string, RegistrationInput) (User, error)
}
