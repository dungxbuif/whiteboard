package domain

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

const (
	FeatureJobBatchLimit       = 100
	FeatureJobLeaseDuration    = 60 * time.Second
	FeatureJobOperationTimeout = 30 * time.Second
	FeatureJobMaxAttempts      = 10
)

type FeatureJob struct {
	ID             uuid.UUID       `json:"id" gorm:"type:uuid;primaryKey"`
	WorkspaceID    *uuid.UUID      `json:"workspaceId,omitempty" gorm:"type:uuid"`
	EventID        *int64          `json:"eventId,omitempty"`
	Kind           string          `json:"kind"`
	Payload        json.RawMessage `json:"payload" gorm:"type:jsonb"`
	Attempts       int             `json:"attempts"`
	AvailableAt    time.Time       `json:"availableAt"`
	LeaseOwner     string          `json:"leaseOwner,omitempty"`
	LeaseExpiresAt *time.Time      `json:"leaseExpiresAt,omitempty"`
}

type JobBatchResult struct {
	Claimed      int `json:"claimed"`
	Completed    int `json:"completed"`
	Retried      int `json:"retried"`
	DeadLettered int `json:"deadLettered"`
}

type FeatureJobRepository interface {
	Claim(context.Context, time.Time, int, string, time.Duration) ([]FeatureJob, error)
	Complete(context.Context, uuid.UUID, string) error
	Retry(context.Context, uuid.UUID, string, time.Time, string) error
	DeadLetter(context.Context, uuid.UUID, string, string) error
}
