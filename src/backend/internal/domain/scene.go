package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type Scene struct {
	ID             uuid.UUID
	WorkspaceID    uuid.UUID
	FolderID       *uuid.UUID
	Title          string
	Content        json.RawMessage
	ContentVersion int64
	IsStarred      bool
	DeletedAt      *time.Time
	CreatedBy      *uuid.UUID
	UpdatedBy      *uuid.UUID
	LastOpenedAt   *time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
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

type SceneRepository interface {
	FindByID(ctx Context, workspaceID uuid.UUID, id uuid.UUID) (*Scene, error)
	ListByWorkspace(ctx Context, workspaceID uuid.UUID) ([]Scene, error)
	Save(ctx Context, scene *Scene) error
}
