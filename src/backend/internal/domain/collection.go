package domain

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

type CollectionVisibility string

const (
	CollectionWorkspace CollectionVisibility = "workspace"
	CollectionPrivate   CollectionVisibility = "private"
)

var ErrInvalidCollection = errors.New("invalid collection input")
var ErrCollectionNameConflict = errors.New("collection name already exists")

type CollectionOperation string

const (
	CollectionCreate      CollectionOperation = "create"
	CollectionUpdate      CollectionOperation = "update"
	CollectionDelete      CollectionOperation = "delete"
	CollectionAddScene    CollectionOperation = "add_scene"
	CollectionRemoveScene CollectionOperation = "remove_scene"
	CollectionReorder     CollectionOperation = "reorder"
)

type CollectionScene struct {
	ID        uuid.UUID `json:"id"`
	PublicID  string    `json:"publicId,omitempty"`
	Title     string    `json:"title"`
	Position  int       `json:"position"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type Collection struct {
	ID            uuid.UUID            `json:"id"`
	WorkspaceID   uuid.UUID            `json:"-"`
	CreatorID     uuid.UUID            `json:"-"`
	Name          string               `json:"name"`
	Visibility    CollectionVisibility `json:"visibility"`
	Revision      int64                `json:"revision,string"`
	SceneCount    int                  `json:"sceneCount"`
	Mine          bool                 `json:"mine"`
	CanSetPrivate bool                 `json:"canSetPrivate"`
	Scenes        []CollectionScene    `json:"scenes,omitempty"`
	CreatedAt     time.Time            `json:"createdAt"`
	UpdatedAt     time.Time            `json:"updatedAt"`
}

type CollectionPage struct {
	Items      []Collection `json:"items"`
	NextCursor string       `json:"nextCursor,omitempty"`
}

type CollectionCreateInput struct {
	RequestID  uuid.UUID
	Name       string
	Visibility CollectionVisibility
}

type CollectionUpdateInput struct {
	RequestID        uuid.UUID
	ExpectedRevision int64
	Name             string
	Visibility       CollectionVisibility
}

type CollectionRevisionInput struct {
	RequestID        uuid.UUID
	ExpectedRevision int64
}

type CollectionSceneInput struct {
	RequestID        uuid.UUID
	ExpectedRevision int64
	SceneID          uuid.UUID
}

type CollectionReorderInput struct {
	RequestID        uuid.UUID
	ExpectedRevision int64
	SceneIDs         []uuid.UUID
}

type CollectionRepository interface {
	WithMutation(Context, Principal, uuid.UUID, uuid.UUID, CollectionOperation, []byte, func(Context) (StoredFeatureResponse, error)) (StoredFeatureResponse, error)
	Create(Context, Collection) (Collection, error)
	Update(Context, uuid.UUID, uuid.UUID, uuid.UUID, int64, string, CollectionVisibility) (Collection, error)
	Delete(Context, uuid.UUID, uuid.UUID, uuid.UUID, int64, bool) error
	Get(Context, uuid.UUID, uuid.UUID, uuid.UUID, bool) (Collection, error)
	List(Context, uuid.UUID, uuid.UUID, bool, *uuid.UUID, int) ([]Collection, error)
	AddScene(Context, uuid.UUID, uuid.UUID, uuid.UUID, uuid.UUID, int64) (Collection, error)
	RemoveScene(Context, uuid.UUID, uuid.UUID, uuid.UUID, uuid.UUID, int64) (Collection, error)
	Reorder(Context, uuid.UUID, uuid.UUID, uuid.UUID, int64, []uuid.UUID) (Collection, error)
}

type CollectionUsecase interface {
	Create(Context, Principal, uuid.UUID, CollectionCreateInput) (Collection, error)
	Update(Context, Principal, uuid.UUID, uuid.UUID, CollectionUpdateInput) (Collection, error)
	Delete(Context, Principal, uuid.UUID, uuid.UUID, CollectionRevisionInput) error
	Get(Context, Principal, uuid.UUID, uuid.UUID) (Collection, error)
	List(Context, Principal, uuid.UUID, string, int) (CollectionPage, error)
	AddScene(Context, Principal, uuid.UUID, uuid.UUID, CollectionSceneInput) (Collection, error)
	RemoveScene(Context, Principal, uuid.UUID, uuid.UUID, CollectionSceneInput) (Collection, error)
	Reorder(Context, Principal, uuid.UUID, uuid.UUID, CollectionReorderInput) (Collection, error)
}
