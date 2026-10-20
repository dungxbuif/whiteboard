package domain

import (
	"encoding/json"
	"errors"

	"github.com/google/uuid"
)

type FeatureAction string

const (
	FeatureCommentRead       FeatureAction = "comment:read"
	FeatureCommentWrite      FeatureAction = "comment:write"
	FeatureCommentModerate   FeatureAction = "comment:moderate"
	FeatureActivityRead      FeatureAction = "activity:read"
	FeatureVersionRead       FeatureAction = "version:read"
	FeatureVersionCreate     FeatureAction = "version:create"
	FeatureVersionRestore    FeatureAction = "version:restore"
	FeatureLibraryRead       FeatureAction = "library:read"
	FeatureLibraryWrite      FeatureAction = "library:write"
	FeatureCollectionManage  FeatureAction = "collection:manage"
	FeaturePresentationHost  FeatureAction = "presentation:host"
	FeatureScreenPublish     FeatureAction = "screen:publish"
	FeatureEnterpriseManage  FeatureAction = "enterprise:manage"
	FeatureRequestHashLength               = 64
)

var (
	ErrFeatureDisabled     = errors.New("feature disabled")
	ErrInvalidFeatureInput = errors.New("invalid feature input")
	ErrRequestIDReused     = errors.New("request ID reused with different mutation")
	ErrRevisionConflict    = errors.New("revision conflict")
)

type WorkspaceFeaturePolicy struct {
	WorkspaceID               uuid.UUID `json:"workspaceId" gorm:"type:uuid;primaryKey"`
	Revision                  int64     `json:"-" gorm:"not null;default:1"`
	MemberComments            bool      `json:"memberComments" gorm:"not null;default:true"`
	MemberHistory             bool      `json:"memberHistory" gorm:"not null;default:true"`
	CommentsEnabled           bool      `json:"commentsEnabled" gorm:"not null;default:false"`
	ActivityEnabled           bool      `json:"activityEnabled" gorm:"not null;default:false"`
	VersionsEnabled           bool      `json:"versionsEnabled" gorm:"not null;default:false"`
	LibrariesEnabled          bool      `json:"librariesEnabled" gorm:"not null;default:false"`
	CollectionsEnabled        bool      `json:"collectionsEnabled" gorm:"not null;default:false"`
	PresentationEnabled       bool      `json:"presentationEnabled" gorm:"not null;default:false"`
	ScreenShareEnabled        bool      `json:"screenShareEnabled" gorm:"not null;default:false"`
	EnterpriseIdentityEnabled bool      `json:"enterpriseIdentityEnabled" gorm:"not null;default:false"`
	RetentionEnabled          bool      `json:"retentionEnabled" gorm:"not null;default:false"`
}

func DefaultWorkspaceFeaturePolicy(workspaceID uuid.UUID) WorkspaceFeaturePolicy {
	return WorkspaceFeaturePolicy{
		WorkspaceID:    workspaceID,
		Revision:       1,
		MemberComments: true,
		MemberHistory:  true,
	}
}

type WorkspaceFeaturePolicyPatch struct {
	MemberComments            *bool `json:"memberComments,omitempty"`
	MemberHistory             *bool `json:"memberHistory,omitempty"`
	CommentsEnabled           *bool `json:"commentsEnabled,omitempty"`
	ActivityEnabled           *bool `json:"activityEnabled,omitempty"`
	VersionsEnabled           *bool `json:"versionsEnabled,omitempty"`
	LibrariesEnabled          *bool `json:"librariesEnabled,omitempty"`
	CollectionsEnabled        *bool `json:"collectionsEnabled,omitempty"`
	PresentationEnabled       *bool `json:"presentationEnabled,omitempty"`
	ScreenShareEnabled        *bool `json:"screenShareEnabled,omitempty"`
	EnterpriseIdentityEnabled *bool `json:"enterpriseIdentityEnabled,omitempty"`
	RetentionEnabled          *bool `json:"retentionEnabled,omitempty"`
}

func (p WorkspaceFeaturePolicyPatch) Empty() bool {
	return p.MemberComments == nil && p.MemberHistory == nil &&
		p.CommentsEnabled == nil && p.ActivityEnabled == nil &&
		p.VersionsEnabled == nil && p.LibrariesEnabled == nil &&
		p.CollectionsEnabled == nil && p.PresentationEnabled == nil &&
		p.ScreenShareEnabled == nil && p.EnterpriseIdentityEnabled == nil &&
		p.RetentionEnabled == nil
}

type StoredFeatureResponse struct {
	Status int             `json:"status"`
	Body   json.RawMessage `json:"body"`
}

type FeatureRepository interface {
	Policy(Context, uuid.UUID) (WorkspaceFeaturePolicy, error)
	PublicCommentsAllowed(Context, Principal, SceneRef) (bool, error)
	UpdatePolicy(Context, uuid.UUID, int64, WorkspaceFeaturePolicyPatch) (WorkspaceFeaturePolicy, error)
	WithMutation(Context, Principal, SceneRef, FeatureAction, uuid.UUID, string, func(Context) (StoredFeatureResponse, error)) (StoredFeatureResponse, error)
}

type FeatureService interface {
	Require(Context, Principal, SceneRef, FeatureAction) error
	Policy(Context, Principal, uuid.UUID) (WorkspaceFeaturePolicy, error)
	UpdatePolicy(Context, Principal, uuid.UUID, int64, uuid.UUID, WorkspaceFeaturePolicyPatch) (WorkspaceFeaturePolicy, error)
	Mutate(Context, Principal, SceneRef, FeatureAction, uuid.UUID, []byte, func(Context) (StoredFeatureResponse, error)) (StoredFeatureResponse, error)
}
