package domain

import (
	"errors"
	"github.com/google/uuid"
	"time"
)

type PrincipalKind string

const (
	PrincipalUser   PrincipalKind = "user"
	PrincipalPublic PrincipalKind = "public"
	PrincipalAgent  PrincipalKind = "agent"
	AccessView                    = "view"
	AccessEdit                    = "edit"
	AccessManage                  = "manage"
)

var (
	ErrAccessDenied   = errors.New("access denied")
	ErrInaccessible   = errors.New("resource not found")
	ErrOwnerProtected = errors.New("owner cannot be removed or demoted")
)

type Principal struct {
	Kind                PrincipalKind
	ID                  string
	UserID              *uuid.UUID
	GrantID             string
	GrantVersion        string
	DisplayName         string
	CredentialExpiresAt time.Time
}

func UserPrincipal(id uuid.UUID) Principal {
	return Principal{Kind: PrincipalUser, ID: id.String(), UserID: &id}
}

type SceneRef struct {
	WorkspaceID uuid.UUID
	SceneID     uuid.UUID
}
type AccessDecision struct {
	Access     string `json:"access"`
	Owner      bool   `json:"owner"`
	ACLVersion int64  `json:"aclVersion"`
}
type AccessState struct {
	Role       WorkspaceRole
	Override   WorkspaceRole
	ACLVersion int64
}
type SceneMemberOverride struct {
	WorkspaceID uuid.UUID     `json:"workspaceId"`
	SceneID     uuid.UUID     `json:"sceneId"`
	UserID      uuid.UUID     `json:"userId"`
	Role        WorkspaceRole `json:"role"`
}
type AccessRepository interface {
	WorkspaceState(Context, uuid.UUID, uuid.UUID) (AccessState, error)
	SceneState(Context, uuid.UUID, SceneRef) (AccessState, error)
	ResolveWorkspaceID(Context, string) (uuid.UUID, error)
	ResolveSceneID(Context, uuid.UUID, string) (uuid.UUID, error)
	WithWorkspaceLock(Context, uuid.UUID, uuid.UUID, func(Context) error) error
	ListMembers(Context, uuid.UUID) ([]WorkspaceMember, error)
	ChangeMember(Context, uuid.UUID, uuid.UUID, *WorkspaceRole) error
	ChangeOverride(Context, SceneRef, uuid.UUID, *WorkspaceRole) error
	DescendantScenes(Context, uuid.UUID, uuid.UUID) ([]uuid.UUID, error)
}
type AccessService interface {
	Resolve(Context, Principal, SceneRef) (AccessDecision, error)
	ResolveWorkspace(Context, Principal, uuid.UUID) (AccessDecision, error)
	RequireOwner(Context, uuid.UUID, uuid.UUID) error
	RequireFolderEdit(Context, Principal, uuid.UUID, uuid.UUID) error
	ResolveWorkspaceID(Context, string) (uuid.UUID, error)
	ResolveSceneID(Context, uuid.UUID, string) (uuid.UUID, error)
	WithWorkspaceLock(Context, uuid.UUID, uuid.UUID, func(Context) error) error
	ListMembers(Context, uuid.UUID, uuid.UUID) ([]WorkspaceMember, error)
	SetMember(Context, uuid.UUID, uuid.UUID, uuid.UUID, *WorkspaceRole) error
	SetOverride(Context, uuid.UUID, SceneRef, uuid.UUID, *WorkspaceRole) error
}
