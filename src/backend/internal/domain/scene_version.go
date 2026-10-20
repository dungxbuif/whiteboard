package domain

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
)

var ErrInvalidVersionInput = errors.New("invalid version input")

type VersionCreator struct {
	Kind PrincipalKind `json:"kind"`
	ID   string        `json:"id"`
}

type SceneVersion struct {
	ID               uuid.UUID                    `json:"id"`
	WorkspaceID      uuid.UUID                    `json:"-"`
	SceneID          uuid.UUID                    `json:"sceneId"`
	DocumentID       uuid.UUID                    `json:"documentId"`
	Sequence         string                       `json:"sequence"`
	Hash             string                       `json:"hash"`
	Content          json.RawMessage              `json:"content,omitempty"`
	Assets           map[string]AssetViewResponse `json:"assets,omitempty"`
	Label            *string                      `json:"label,omitempty"`
	Pinned           bool                         `json:"pinned"`
	Kind             string                       `json:"kind"`
	Creator          VersionCreator               `json:"creator"`
	MetadataRevision int64                        `json:"metadataRevision,string"`
	CreatedAt        time.Time                    `json:"createdAt"`
	UpdatedAt        time.Time                    `json:"updatedAt"`
}

type CreateVersionInput struct {
	RequestID        uuid.UUID `json:"requestId"`
	ExpectedSequence string    `json:"expectedSequence"`
	Label            *string   `json:"label,omitempty"`
	Pinned           bool      `json:"pinned"`
}

type VersionMetadataInput struct {
	RequestID        uuid.UUID
	ExpectedRevision int64
	Label            *string
	Pinned           *bool
}

type VersionPage struct {
	Items      []SceneVersion `json:"items"`
	NextCursor string         `json:"nextCursor,omitempty"`
}

type SceneDiff struct {
	AddedIDs              []string `json:"addedIds"`
	RemovedIDs            []string `json:"removedIds"`
	ChangedIDs            []string `json:"changedIds"`
	ChangedAppStateFields []string `json:"changedAppStateFields"`
}

type RestoreVersionInput struct {
	VersionID        uuid.UUID `json:"versionId"`
	RequestID        uuid.UUID `json:"requestId"`
	ExpectedSequence string    `json:"expectedSequence"`
}

type RestoreVersionResult struct {
	RequestID           uuid.UUID `json:"requestId"`
	Sequence            string    `json:"sequence"`
	StateVector         string    `json:"stateVector"`
	PreRestoreVersionID uuid.UUID `json:"preRestoreVersionId"`
}

type SceneVersionRepository interface {
	ListVersions(Context, SceneRef, *uuid.UUID, int) ([]SceneVersion, error)
	Version(Context, SceneRef, uuid.UUID) (SceneVersion, error)
	SetVersionMetadata(Context, SceneRef, uuid.UUID, int64, *string, *bool) (SceneVersion, error)
}

type SceneVersionUsecase interface {
	CurrentProjection(Context, Principal, SceneRef) (json.RawMessage, error)
	Duplicate(Context, Principal, SceneRef, string, *uuid.UUID) (*Scene, error)
	Create(Context, Principal, SceneRef, CreateVersionInput) (SceneVersion, error)
	List(Context, Principal, SceneRef, string, int) (VersionPage, error)
	Get(Context, Principal, SceneRef, uuid.UUID) (SceneVersion, error)
	SetMetadata(Context, Principal, SceneRef, uuid.UUID, VersionMetadataInput) (SceneVersion, error)
	Diff(Context, Principal, SceneRef, uuid.UUID, string) (SceneDiff, error)
	Restore(Context, Principal, SceneRef, RestoreVersionInput) (RestoreVersionResult, error)
}
