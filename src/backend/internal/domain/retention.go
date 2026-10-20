package domain

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
)

const (
	DefaultActivityRetentionDays     = 90
	DefaultNotificationRetentionDays = 30
	DefaultAutomaticVersionLimit     = 100
	MinimumRetentionActivationDelay  = 7 * 24 * time.Hour
)

var (
	ErrInvalidRetention = errors.New("invalid retention request")
	ErrPreviewExpired   = errors.New("retention preview expired")
)

type RetentionPolicyInput struct {
	ActivityDays          int  `json:"activityDays"`
	NotificationDays      int  `json:"notificationDays"`
	AutomaticVersionLimit int  `json:"automaticVersionLimit"`
	CommentsDays          *int `json:"commentsDays,omitempty"`
	TrashHardPurge        bool `json:"trashHardPurge"`
}

func DefaultRetentionPolicyInput() RetentionPolicyInput {
	return RetentionPolicyInput{ActivityDays: DefaultActivityRetentionDays, NotificationDays: DefaultNotificationRetentionDays, AutomaticVersionLimit: DefaultAutomaticVersionLimit}
}

type RetentionCounts struct {
	Activity      int64 `json:"activity"`
	Notifications int64 `json:"notifications"`
	Versions      int64 `json:"versions"`
	Comments      int64 `json:"comments"`
}

type RetentionPreview struct {
	ID                    uuid.UUID            `json:"id"`
	WorkspaceID           uuid.UUID            `json:"workspaceId"`
	PolicyRevision        int64                `json:"policyRevision,string"`
	Policy                RetentionPolicyInput `json:"policy"`
	PolicyHash            string               `json:"policyHash"`
	Counts                RetentionCounts      `json:"counts"`
	EarliestDestructiveAt time.Time            `json:"earliestDestructiveAt"`
	ExpiresAt             time.Time            `json:"expiresAt"`
	CreatedAt             time.Time            `json:"createdAt"`
}

type RetentionPolicy struct {
	WorkspaceID uuid.UUID            `json:"workspaceId"`
	Revision    int64                `json:"revision,string"`
	Policy      RetentionPolicyInput `json:"policy"`
	PolicyHash  string               `json:"policyHash"`
	EffectiveAt time.Time            `json:"effectiveAt"`
	ScheduledAt time.Time            `json:"scheduledAt"`
}

type RetentionReference struct {
	WorkspaceID uuid.UUID `json:"workspaceId"`
	Kind        string    `json:"kind"`
	ID          string    `json:"id"`
}

type RetentionHold struct {
	ID        uuid.UUID          `json:"id"`
	Reference RetentionReference `json:"reference"`
	VersionID *uuid.UUID         `json:"versionId,omitempty"`
	Held      bool               `json:"held"`
	UpdatedAt time.Time          `json:"updatedAt"`
}

type RetentionHoldInput struct {
	Reference RetentionReference `json:"reference"`
	VersionID *uuid.UUID         `json:"versionId,omitempty"`
	Held      bool               `json:"held"`
	RequestID uuid.UUID          `json:"requestId"`
}

type RetentionScheduleInput struct {
	PreviewID        uuid.UUID `json:"previewId"`
	ExpectedRevision int64     `json:"expectedRevision,string"`
	RequestID        uuid.UUID `json:"requestId"`
}

type RetentionPurgeItem struct {
	Kind string `json:"kind"`
	ID   string `json:"id"`
}

type PurgeResult struct {
	Scanned int                  `json:"scanned"`
	Purged  RetentionCounts      `json:"purged"`
	Items   []RetentionPurgeItem `json:"items"`
}

type RetentionRepository interface {
	CurrentPolicy(Context, uuid.UUID) (RetentionPolicy, error)
	PreviewCounts(Context, uuid.UUID, RetentionPolicyInput, time.Time) (RetentionCounts, error)
	SavePreview(Context, RetentionPreview) error
	Preview(Context, uuid.UUID, uuid.UUID) (RetentionPreview, error)
	WithMutation(Context, Principal, uuid.UUID, uuid.UUID, string, []byte, func(Context) (StoredFeatureResponse, error)) (StoredFeatureResponse, error)
	Schedule(Context, uuid.UUID, RetentionPreview, int64, time.Time) (RetentionPolicy, error)
	SetHold(Context, RetentionReference, *uuid.UUID, bool, time.Time) (RetentionHold, error)
	PurgeBatch(Context, time.Time, int) (PurgeResult, error)
}

type RetentionUsecase interface {
	Policy(Context, Principal, uuid.UUID) (RetentionPolicy, error)
	PreviewPolicy(Context, Principal, uuid.UUID, RetentionPolicyInput) (RetentionPreview, error)
	Schedule(Context, Principal, uuid.UUID, uuid.UUID, int64, uuid.UUID) (RetentionPolicy, error)
	SetHold(Context, Principal, RetentionReference, *uuid.UUID, bool, uuid.UUID) (RetentionHold, error)
	PurgeBatch(Context, time.Time, int) (PurgeResult, error)
}

type RetentionReceiptResponse struct {
	Status int             `json:"status"`
	Body   json.RawMessage `json:"body"`
}
