package domain

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
)

var ErrInvalidLibrary = errors.New("invalid library input")

type LibraryAssetPin struct {
	SourceSceneID uuid.UUID `json:"sourceSceneId"`
	AssetID       uuid.UUID `json:"assetId"`
	FileID        string    `json:"fileId"`
}

type LibraryItem struct {
	ID              uuid.UUID `json:"id"`
	WorkspaceID     uuid.UUID `json:"-"`
	OwnerID         uuid.UUID `json:"-"`
	Name            string    `json:"name"`
	CurrentRevision int64     `json:"currentRevision,string"`
	CreatedAt       time.Time `json:"createdAt"`
	UpdatedAt       time.Time `json:"updatedAt"`
}

type LibraryRevision struct {
	LibraryItemID uuid.UUID       `json:"libraryItemId"`
	WorkspaceID   uuid.UUID       `json:"-"`
	Revision      int64           `json:"revision,string"`
	Content       json.RawMessage `json:"content"`
	ContentHash   string          `json:"contentHash"`
	ByteSize      int64           `json:"byteSize"`
	CreatedAt     time.Time       `json:"createdAt"`
}

type LibraryPage struct {
	Items []LibraryItem `json:"items"`
}

type LibraryImportInput struct {
	RequestID uuid.UUID
	Name      string
	Content   json.RawMessage
	Assets    []LibraryAssetPin
}

type LibraryPublishInput struct {
	RequestID        uuid.UUID
	ExpectedRevision int64
	Name             string
	Content          json.RawMessage
	Assets           []LibraryAssetPin
}

type LibraryInsertInput struct {
	LibraryItemID    uuid.UUID `json:"libraryItemId"`
	NativeItemID     string    `json:"nativeItemId,omitempty"`
	Revision         int64     `json:"revision"`
	ExpectedSequence string    `json:"expectedSequence"`
	RequestID        uuid.UUID `json:"requestId"`
}

type LibraryInsertResult struct {
	RequestID   uuid.UUID `json:"requestId"`
	Sequence    string    `json:"sequence"`
	StateVector string    `json:"stateVector"`
	ElementIDs  []string  `json:"elementIds"`
}

type LibraryRepository interface {
	CreateRevision(Context, LibraryItem, LibraryRevision, []LibraryAssetPin) (LibraryItem, LibraryRevision, error)
	Item(Context, uuid.UUID, uuid.UUID) (LibraryItem, error)
	Revision(Context, uuid.UUID, uuid.UUID, int64) (LibraryRevision, error)
	List(Context, uuid.UUID, string, int) ([]LibraryItem, error)
	RevisionAssets(Context, uuid.UUID, uuid.UUID, int64) ([]LibraryAssetPin, error)
}

type LibraryAssetStager interface {
	Stage(Context, Principal, SceneRef, uuid.UUID, []LibraryAssetPin) (map[string]uuid.UUID, error)
}

type LibraryObjectCopier interface {
	CopyObject(Context, string, string, string, string) error
	Delete(Context, string, string) error
}

type LibraryUsecase interface {
	Import(Context, Principal, uuid.UUID, LibraryImportInput) (LibraryItem, LibraryRevision, error)
	Publish(Context, Principal, uuid.UUID, uuid.UUID, LibraryPublishInput) (LibraryItem, LibraryRevision, error)
	List(Context, Principal, uuid.UUID, string, int) (LibraryPage, error)
	GetRevision(Context, Principal, uuid.UUID, uuid.UUID, int64) (LibraryRevision, error)
	Export(Context, Principal, uuid.UUID, uuid.UUID, int64) (json.RawMessage, error)
	Insert(Context, Principal, SceneRef, LibraryInsertInput) (LibraryInsertResult, error)
}
