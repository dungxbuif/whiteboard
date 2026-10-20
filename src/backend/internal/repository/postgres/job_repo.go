package postgres

import (
	"context"
	"devtool/internal/domain"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type JobRepository struct{ db *gorm.DB }

func NewJobRepository(db *gorm.DB) *JobRepository { return &JobRepository{db: db} }

func (r *JobRepository) Claim(ctx context.Context, now time.Time, limit int, workerID string, lease time.Duration) ([]domain.FeatureJob, error) {
	if limit <= 0 || limit > domain.FeatureJobBatchLimit || workerID == "" || lease <= 0 {
		return nil, errors.New("invalid feature job claim")
	}
	var jobs []domain.FeatureJob
	err := r.db.WithContext(ctx).Raw(`
		WITH candidates AS (
			SELECT id FROM feature_jobs
			WHERE completed_at IS NULL
			  AND dead_lettered_at IS NULL
			  AND available_at <= ?
			  AND (lease_expires_at IS NULL OR lease_expires_at <= ?)
			ORDER BY available_at, id
			FOR UPDATE SKIP LOCKED
			LIMIT ?
		)
		UPDATE feature_jobs AS jobs
		SET lease_owner = ?, lease_expires_at = ?, attempts = attempts + 1, updated_at = ?
		FROM candidates
		WHERE jobs.id = candidates.id
		RETURNING jobs.id, jobs.workspace_id, jobs.event_id, jobs.kind, jobs.payload,
		          jobs.attempts, jobs.available_at, jobs.lease_owner, jobs.lease_expires_at`,
		now, now, limit, workerID, now.Add(lease), now).Scan(&jobs).Error
	return jobs, err
}

func (r *JobRepository) Complete(ctx context.Context, id uuid.UUID, workerID string) error {
	result := r.db.WithContext(ctx).Exec(`UPDATE feature_jobs
		SET completed_at=now(), lease_owner=NULL, lease_expires_at=NULL, updated_at=now()
		WHERE id=? AND lease_owner=? AND completed_at IS NULL AND dead_lettered_at IS NULL`, id, workerID)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return domain.ErrInaccessible
	}
	return nil
}

func (r *JobRepository) Retry(ctx context.Context, id uuid.UUID, workerID string, availableAt time.Time, message string) error {
	result := r.db.WithContext(ctx).Exec(`UPDATE feature_jobs
		SET available_at=?, last_error=?, lease_owner=NULL, lease_expires_at=NULL, updated_at=now()
		WHERE id=? AND lease_owner=? AND completed_at IS NULL AND dead_lettered_at IS NULL`, availableAt, message, id, workerID)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return domain.ErrInaccessible
	}
	return nil
}

func (r *JobRepository) DeadLetter(ctx context.Context, id uuid.UUID, workerID, message string) error {
	result := r.db.WithContext(ctx).Exec(`UPDATE feature_jobs
		SET dead_lettered_at=now(), last_error=?, lease_owner=NULL, lease_expires_at=NULL, updated_at=now()
		WHERE id=? AND lease_owner=? AND completed_at IS NULL AND dead_lettered_at IS NULL`, message, id, workerID)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return domain.ErrInaccessible
	}
	return nil
}
