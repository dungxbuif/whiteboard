package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Scene struct {
	ID             uuid.UUID       `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	PublicID       *string         `json:"publicId,omitempty"`
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
	PresignedURL  *string
	URLExpiresAt  *time.Time
	DeletedAt     *time.Time
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type SceneAssetRepository interface {
	Create(ctx Context, asset *SceneAsset) error
	GetByFileIDs(ctx Context, workspaceID uuid.UUID, fileIDs []string) ([]SceneAsset, error)
	UpdatePresignedURL(ctx Context, id uuid.UUID, url string, expiresAt time.Time) error
}

type S3Repository interface {
	PresignUpload(ctx Context, bucket, key string, expires time.Duration) (string, error)
	PresignDownload(ctx Context, bucket, key string, expires time.Duration) (string, error)
	Delete(ctx Context, bucket, key string) error
}

type AssetUsecase interface {
	PresignUpload(ctx Context, workspaceID, userID, sceneID uuid.UUID, assets []AssetUploadRequest) (map[string]AssetUploadResponse, error)
	PresignView(ctx Context, workspaceID, sceneID uuid.UUID, fileIDs []string) (map[string]AssetViewResponse, error)
}

type AssetUploadRequest struct {
	FileID   string  `json:"fileId" validate:"required"`
	MimeType string  `json:"mimeType" validate:"required"`
	ByteSize int64   `json:"byteSize" validate:"gte=0"`
	SHA256   *string `json:"sha256"`
	Width    *int    `json:"width"`
	Height   *int    `json:"height"`
}

type AssetUploadResponse struct {
	AssetID    uuid.UUID `json:"assetId"`
	UploadUrl  string    `json:"uploadUrl"`
	StorageKey string    `json:"storageKey"`
	ExpiresAt  time.Time `json:"expiresAt"`
}

type AssetViewResponse struct {
	AssetID   uuid.UUID `json:"assetId"`
	ViewUrl   string    `json:"viewUrl"`
	MimeType  string    `json:"mimeType"`
	ExpiresAt time.Time `json:"expiresAt"`
}
