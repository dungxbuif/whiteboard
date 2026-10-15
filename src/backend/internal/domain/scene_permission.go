package domain

import (
	"time"

	"github.com/google/uuid"
)

type ScenePermission struct {
	ID           uuid.UUID `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	WorkspaceID  uuid.UUID `json:"workspaceId" gorm:"type:uuid;not null;index"`
	SceneID      uuid.UUID `json:"sceneId" gorm:"type:uuid;not null;uniqueIndex"`
	Token        string    `json:"token" gorm:"not null;uniqueIndex"`
	Active       bool      `json:"active" gorm:"not null;default:true"`
	PasswordHash string    `json:"-"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

type ScenePermissionRepository interface {
	FindBySceneID(ctx Context, workspaceID, sceneID uuid.UUID) (*ScenePermission, error)
	FindActiveByToken(ctx Context, token string) (*ScenePermission, error)
	Save(ctx Context, permission *ScenePermission) error
}

type ScenePermissionUsecase interface {
	Share(ctx Context, workspaceID, sceneID uuid.UUID, active bool, password string) (*ScenePermission, error)
	GetPublicScene(ctx Context, token, password string) (*Scene, error)
}
