package domain

import (
	"errors"
	"github.com/google/uuid"
	"time"
)

const (
	RoomIssuer           = "whiteboard-api"
	RoomAudience         = "whiteboard-collab"
	SceneSessionAudience = "whiteboard-scene-session"
	RoomPurpose          = "room"
	SceneSessionPurpose  = "scene-session"
	RoomTokenLifetime    = 10 * time.Minute
	MaxPublicDisplayName = 100
)

var (
	ErrSharePassword       = errors.New("share password required or invalid")
	ErrInvalidShareInput   = errors.New("invalid public link input")
	ErrInvalidSceneSession = errors.New("invalid or expired scene session")
)

type ShareLink struct {
	ID                 uuid.UUID  `json:"id" gorm:"type:uuid;primaryKey"`
	WorkspaceID        uuid.UUID  `json:"-"`
	SceneID            uuid.UUID  `json:"-"`
	TokenHash          string     `json:"-"`
	PasswordHash       string     `json:"-"`
	Access             string     `json:"access"`
	AllowComments      bool       `json:"allowComments"`
	Active             bool       `json:"active"`
	ExpiresAt          *time.Time `json:"expiresAt"`
	LegacyPermissionID *uuid.UUID `json:"-"`
	CreatedAt          time.Time  `json:"-"`
	UpdatedAt          time.Time  `json:"-"`
	Revision           int64      `json:"-" gorm:"not null;default:1"`
}
type CreateShareLink struct {
	Access        string     `json:"access" validate:"required,oneof=view edit"`
	AllowComments bool       `json:"allowComments"`
	Password      string     `json:"password,omitempty" validate:"omitempty,min=6,max=72"`
	ExpiresAt     *time.Time `json:"expiresAt,omitempty"`
}
type CreatedShareLink struct {
	ShareLink
	Token string `json:"token"`
}
type ShareLinkRepository interface {
	Create(Context, *ShareLink) error
	List(Context, SceneRef) ([]ShareLink, error)
	FindByHash(Context, string) (*ShareLink, error)
	FindByID(Context, uuid.UUID) (*ShareLink, error)
	Revoke(Context, SceneRef, uuid.UUID) error
	UserDisplayName(Context, uuid.UUID) (string, error)
}
type PublicAccessRepository interface {
	PublicSceneState(Context, Principal, SceneRef) (AccessDecision, error)
}
type ShareLinkService interface {
	Create(Context, uuid.UUID, SceneRef, CreateShareLink) (CreatedShareLink, error)
	List(Context, uuid.UUID, SceneRef) ([]ShareLink, error)
	Revoke(Context, uuid.UUID, SceneRef, uuid.UUID) error
}
type SceneSession struct {
	Token       string    `json:"token"`
	ExpiresAt   time.Time `json:"expiresAt"`
	WorkspaceID uuid.UUID `json:"workspaceId"`
	SceneID     uuid.UUID `json:"sceneId"`
	Access      string    `json:"access"`
	PrincipalID string    `json:"principalId"`
	DisplayName string    `json:"displayName"`
}
type RoomToken struct {
	SceneSession *SceneSession `json:"sceneSession,omitempty"`
	Token        string        `json:"token"`
	ExpiresAt    time.Time     `json:"expiresAt"`
}
type SceneSessionIdentity struct {
	ExpiresAt time.Time
	Principal Principal
	SceneRef  SceneRef
}
type RoomService interface {
	ExchangePublicSession(Context, string, string, string) (SceneSession, error)
	MintRoomToken(Context, Principal, SceneRef) (RoomToken, error)
	ParseSceneSession(string) (SceneSessionIdentity, error)
}
