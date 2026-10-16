package domain

import (
	"errors"
	"github.com/golang-jwt/jwt/v5"
	"time"
)

const (
	MutationAudience              = "whiteboard-mutation"
	MutationPurpose               = "mutation-decision"
	MutationLifetime              = 5 * time.Second
	AssetCapabilityLifetime       = time.Minute
	MaxAssetBytes           int64 = 10 * 1024 * 1024
	MaxAssetBatch                 = 100
	ActionSceneUpdate             = "scene:update"
	ActionSceneCommand            = "scene:command"
	ActionSceneRead               = "scene:read"
)

var ErrInvalidCapability = errors.New("invalid capability request")

type SceneMutation struct {
	WorkspaceID      string `json:"workspaceId" validate:"required,uuid"`
	SceneID          string `json:"sceneId" validate:"required,uuid"`
	Action           string `json:"action" validate:"required,oneof=scene:update scene:command scene:read"`
	RequestID        string `json:"requestId" validate:"required,min=1,max=128,ascii"`
	PayloadHash      string `json:"payloadHash" validate:"required,len=64,hexadecimal,lowercase"`
	ExpectedSequence string `json:"expectedSequence,omitempty"`
}
type MutationClaims struct {
	jwt.RegisteredClaims
	Purpose string `json:"purpose"`
	SceneMutation
	PrincipalType PrincipalKind `json:"principalType"`
	PrincipalID   string        `json:"principalId"`
	GrantID       string        `json:"grantId"`
	GrantVersion  string        `json:"grantVersion,omitempty"`
	Access        string        `json:"access"`
	ACLVersion    string        `json:"aclVersion"`
}
type SignedGrant struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expiresAt"`
}
type AssetGrant struct {
	URLs      map[string]AssetUploadResponse `json:"urls"`
	ExpiresAt time.Time                      `json:"expiresAt"`
}
type CapabilityService interface {
	AuthorizeScene(Context, Principal, SceneRef, time.Duration) (time.Time, error)
	AuthorizeCommand(Context, Principal, SceneMutation) (SignedGrant, error)
	CreateAssetCapability(Context, Principal, SceneRef, []AssetUploadRequest) (AssetGrant, error)
}
type ScopedUploadRepository interface {
	PresignUploadConstrained(Context, string, string, string, int64, time.Duration) (string, map[string]string, error)
}
type PrincipalAssetUploader interface {
	PresignPrincipalUpload(Context, Principal, SceneRef, []AssetUploadRequest, time.Duration) (map[string]AssetUploadResponse, error)
}

type ScopedAssetViewer interface {
	PresignScopedView(Context, SceneRef, []string, time.Duration) (map[string]AssetViewResponse, error)
}

const ContextKeyCredentialExpiry = "credential_expiry"
