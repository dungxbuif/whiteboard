package domain

import (
	"encoding/json"
	"errors"
	"github.com/google/uuid"
	"time"
)

var ErrAgentCredential = errors.New("invalid agent credential")
var ErrAgentInput = errors.New("invalid agent input")
var ErrAgentProjection = errors.New("scene projection exceeds limit")

const (
	ScopeWorkspaceRead = "workspace:read"
	ScopeSceneRead     = "scene:read"
	ScopeSceneCreate   = "scene:create"
	ScopeSceneWrite    = "scene:write"
	ScopeSceneMove     = "scene:move"
	ScopeSceneTrash    = "scene:trash"
	ScopeSceneExport   = "scene:export"
)

type AgentTargets struct {
	WorkspaceID uuid.UUID   `json:"workspaceId"`
	SceneIDs    []uuid.UUID `json:"sceneIds"`
}
type AgentKey struct {
	ID          uuid.UUID   `json:"id"`
	WorkspaceID uuid.UUID   `json:"workspaceId"`
	OwnerID     uuid.UUID   `json:"ownerId"`
	Scopes      []string    `json:"scopes" gorm:"serializer:json"`
	SceneIDs    []uuid.UUID `json:"sceneIds" gorm:"serializer:json"`
	TokenHash   string      `json:"-"`
	CreatedAt   time.Time   `json:"createdAt"`
	RevokedAt   *time.Time  `json:"revokedAt"`
}
type KeyOnce struct {
	AgentKey
	Key string `json:"key"`
}
type AgentPage[T any] struct {
	Items      []T    `json:"items"`
	NextCursor string `json:"nextCursor,omitempty"`
}
type AgentWorkspace struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`
}
type AgentScene struct {
	ID             uuid.UUID       `json:"id"`
	WorkspaceID    uuid.UUID       `json:"workspaceId"`
	Title          string          `json:"title"`
	Content        json.RawMessage `json:"content,omitempty"`
	ContentVersion int64           `json:"contentVersion"`
}
type AgentAccessRepository interface {
	AgentSceneState(Context, Principal, SceneRef) (AccessDecision, error)
}
type AgentRepository interface {
	Create(Context, *AgentKey) error
	FindHash(Context, string) (AgentKey, error)
	FindActive(Context, uuid.UUID) (AgentKey, error)
	List(Context, uuid.UUID) ([]AgentKey, error)
	Revoke(Context, uuid.UUID, uuid.UUID) error
	Workspace(Context, uuid.UUID) (AgentWorkspace, error)
	Scenes(Context, AgentKey, uuid.UUID, int) ([]AgentScene, error)
	Scene(Context, SceneRef) (AgentScene, error)
	CreateScene(Context, *Scene) error
}

type AgentScopeRepository interface {
	RequireAgentScope(Context, Principal, SceneRef, string) error
}
