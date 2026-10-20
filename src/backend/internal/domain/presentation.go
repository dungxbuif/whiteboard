package domain

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

const (
	PresentationAudience        = "whiteboard-presentation"
	PresentationHostPurpose     = "presentation-host"
	PresentationPendingPurpose  = "presentation-pending"
	PresentationViewerPurpose   = "presentation-audience"
	PresentationLinkLifetime    = 24 * time.Hour
	PresentationPendingLifetime = 5 * time.Minute
	PresentationWaitingLimit    = 100
)

var (
	ErrInvalidPresentation = errors.New("invalid presentation request")
	ErrPresentationEnded   = errors.New("presentation session ended")
	ErrPresentationFull    = errors.New("presentation waiting room is full")
)

type PresentationLink struct {
	ID          uuid.UUID  `json:"id" gorm:"type:uuid;primaryKey"`
	WorkspaceID uuid.UUID  `json:"workspaceId" gorm:"type:uuid;not null"`
	SceneID     uuid.UUID  `json:"sceneId" gorm:"type:uuid;not null"`
	SessionID   uuid.UUID  `json:"sessionId" gorm:"type:uuid;not null"`
	TokenHash   string     `json:"-" gorm:"not null"`
	CreatedBy   uuid.UUID  `json:"createdBy" gorm:"type:uuid;not null"`
	Revision    int64      `json:"revision" gorm:"not null;default:1"`
	Active      bool       `json:"active" gorm:"not null;default:true"`
	ExpiresAt   time.Time  `json:"expiresAt" gorm:"not null"`
	RevokedAt   *time.Time `json:"revokedAt,omitempty"`
	CreatedAt   time.Time  `json:"createdAt"`
}

type SessionGrant struct {
	SessionID uuid.UUID `json:"sessionId"`
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expiresAt"`
}

type OneTimeShownLink struct {
	PresentationLink
	Token string `json:"token"`
}

type PendingPresentationGrant struct {
	SessionID   uuid.UUID `json:"sessionId"`
	WorkspaceID uuid.UUID `json:"workspaceId"`
	SceneID     uuid.UUID `json:"sceneId"`
	PrincipalID string    `json:"principalId"`
	DisplayName string    `json:"displayName"`
	Token       string    `json:"token"`
	ExpiresAt   time.Time `json:"expiresAt"`
}

type PresentationRepository interface {
	CreateLink(Context, *PresentationLink) error
	FindLinkByHash(Context, string) (*PresentationLink, error)
	FindLink(Context, SceneRef, uuid.UUID, uuid.UUID) (*PresentationLink, error)
	RevokeLink(Context, SceneRef, uuid.UUID, uuid.UUID) error
}

type PresentationUsecase interface {
	Create(Context, Principal, SceneRef, uuid.UUID) (SessionGrant, error)
	CreateLink(Context, Principal, SceneRef, uuid.UUID, uuid.UUID) (OneTimeShownLink, error)
	Join(Context, string, string) (PendingPresentationGrant, error)
	Admit(Context, Principal, SceneRef, uuid.UUID, string, int64) (SessionGrant, error)
	ResolveAudience(Context, Principal, SceneRef, uuid.UUID) error
	RevokeLink(Context, Principal, SceneRef, uuid.UUID, uuid.UUID, uuid.UUID) error
}

type PresentationRuntimeDecision struct {
	SessionID       uuid.UUID
	WorkspaceID     uuid.UUID
	SceneID         uuid.UUID
	PrincipalID     string
	HostPrincipalID string
	DisplayName     string
	Revision        int64
}

type PresentationRuntimeVerifier interface {
	ResolveHost(Context, SceneRef, uuid.UUID, string) (PresentationRuntimeDecision, error)
	ResolvePending(Context, SceneRef, uuid.UUID, string, string, int64) (PresentationRuntimeDecision, error)
	ResolveAudience(Context, SceneRef, uuid.UUID, string) (PresentationRuntimeDecision, error)
}

type PresentationAudienceAuthority interface {
	ParseAudience(string) (SceneSessionIdentity, uuid.UUID, error)
	ResolveAudience(Context, Principal, SceneRef, uuid.UUID) error
}
