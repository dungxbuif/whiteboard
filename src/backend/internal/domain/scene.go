package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Scene struct {
	ID             uuid.UUID       `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	WorkspaceID    uuid.UUID       `json:"workspaceId" gorm:"type:uuid;not null;index"`
	FolderID       *uuid.UUID      `json:"folderId" gorm:"type:uuid;index"`
	Title          string          `json:"title" gorm:"not null"`
	Content        json.RawMessage `json:"content" gorm:"type:jsonb;not null"`
	ContentVersion int64           `json:"contentVersion" gorm:"not null;default:1"`
	IsStarred      bool            `json:"isStarred" gorm:"not null;default:false"`
	DeletedAt      gorm.DeletedAt  `json:"deletedAt,omitempty" gorm:"index"`
	CreatedBy      *uuid.UUID      `json:"createdBy,omitempty" gorm:"type:uuid"`
	UpdatedBy      *uuid.UUID      `json:"updatedBy,omitempty" gorm:"type:uuid"`
	LastOpenedAt   *time.Time      `json:"lastOpenedAt,omitempty"`
	CreatedAt      time.Time       `json:"createdAt"`
	UpdatedAt      time.Time       `json:"updatedAt"`
}

type SceneRepository interface {
	Create(ctx Context, scene *Scene) error
	FindByID(ctx Context, workspaceID uuid.UUID, id uuid.UUID) (*Scene, error)
	List(ctx Context, workspaceID uuid.UUID, folderID *uuid.UUID, deletedOnly bool) ([]Scene, error)
	Update(ctx Context, scene *Scene) error
	SoftDelete(ctx Context, workspaceID uuid.UUID, id uuid.UUID) error
	Restore(ctx Context, workspaceID uuid.UUID, id uuid.UUID) error
	SetStar(ctx Context, workspaceID uuid.UUID, id uuid.UUID, isStarred bool) error
}

type SceneUsecase interface {
	Create(ctx Context, workspaceID, userID uuid.UUID, title string, folderID *uuid.UUID, content json.RawMessage) (*Scene, error)
	Get(ctx Context, workspaceID, id uuid.UUID) (*Scene, error)
	List(ctx Context, workspaceID uuid.UUID, folderID *uuid.UUID, deletedOnly bool) ([]Scene, error)
	Update(ctx Context, workspaceID, userID, id uuid.UUID, title *string, folderID **uuid.UUID, content *json.RawMessage) (*Scene, error)
	SetStar(ctx Context, workspaceID, id uuid.UUID, isStarred bool) error
	Delete(ctx Context, workspaceID, id uuid.UUID) error
	Restore(ctx Context, workspaceID, id uuid.UUID) error
}

type SceneAssetKind string

const (
	SceneAssetKindExcalidrawFile SceneAssetKind = "excalidraw_file"
	SceneAssetKindThumbnail      SceneAssetKind = "scene_thumbnail"
)

type SceneAsset struct {
	ID            uuid.UUID
	WorkspaceID   uuid.UUID
	SceneID       *uuid.UUID
	UploadedBy    *uuid.UUID
	AssetKind     SceneAssetKind
	FileID        *string
	StorageBucket string
	StorageKey    string
	MimeType      string
	ByteSize      int64
	SHA256        *string
	Width         *int
	Height        *int
	DeletedAt     *time.Time
	CreatedAt     time.Time
	UpdatedAt     time.Time
}
