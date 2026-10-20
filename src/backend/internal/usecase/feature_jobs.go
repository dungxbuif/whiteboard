package usecase

import (
	"context"
	"devtool/internal/domain"
	"errors"
	"fmt"
	"time"
)

type FeatureJobHandler func(context.Context, domain.FeatureJob) error

type FeatureJobs struct {
	repo     domain.FeatureJobRepository
	workerID string
	handlers map[string]FeatureJobHandler
	now      func() time.Time
}

var featureJobBackoff = [...]time.Duration{
	time.Minute,
	2 * time.Minute,
	4 * time.Minute,
	8 * time.Minute,
	16 * time.Minute,
	32 * time.Minute,
	60 * time.Minute,
}

func NewFeatureJobs(repo domain.FeatureJobRepository, workerID string, handlers map[string]FeatureJobHandler) (*FeatureJobs, error) {
	if repo == nil || workerID == "" {
		return nil, errors.New("feature job repository and worker ID are required")
	}
	copyHandlers := make(map[string]FeatureJobHandler, len(handlers))
	for kind, handler := range handlers {
		if kind == "" || handler == nil {
			return nil, errors.New("feature job handlers require a kind and function")
		}
		copyHandlers[kind] = handler
	}
	return &FeatureJobs{repo: repo, workerID: workerID, handlers: copyHandlers, now: func() time.Time { return time.Now().UTC() }}, nil
}

func (s *FeatureJobs) RunOnce(ctx context.Context, now time.Time, limit int) (domain.JobBatchResult, error) {
	if limit <= 0 || limit > domain.FeatureJobBatchLimit {
		limit = domain.FeatureJobBatchLimit
	}
	result := domain.JobBatchResult{}
	for processed := 0; processed < limit; processed++ {
		if err := ctx.Err(); err != nil {
			return result, err
		}
		claimNow := now
		if processed > 0 {
			claimNow = s.now()
		}
		claimed, err := s.repo.Claim(ctx, claimNow, 1, s.workerID, domain.FeatureJobLeaseDuration)
		if err != nil {
			return result, err
		}
		if len(claimed) == 0 {
			break
		}
		if len(claimed) != 1 {
			return result, errors.New("feature job repository returned more than one single-job lease")
		}
		job := claimed[0]
		result.Claimed++
		handler, ok := s.handlers[job.Kind]
		if !ok {
			if err := s.repo.DeadLetter(ctx, job.ID, s.workerID, "no handler registered"); err != nil {
				return result, err
			}
			result.DeadLettered++
			continue
		}
		opCtx, cancel := context.WithTimeout(ctx, domain.FeatureJobOperationTimeout)
		handleErr := handler(opCtx, job)
		cancel()
		if handleErr == nil {
			if err := s.repo.Complete(ctx, job.ID, s.workerID); err != nil {
				return result, err
			}
			result.Completed++
			continue
		}
		message := featureJobFailureReason(handleErr)
		if job.Attempts >= domain.FeatureJobMaxAttempts {
			if err := s.repo.DeadLetter(ctx, job.ID, s.workerID, message); err != nil {
				return result, err
			}
			result.DeadLettered++
			continue
		}
		delay := featureJobBackoff[min(max(job.Attempts, 1)-1, len(featureJobBackoff)-1)]
		if err := s.repo.Retry(ctx, job.ID, s.workerID, claimNow.Add(delay), message); err != nil {
			return result, fmt.Errorf("schedule feature job retry: %w", err)
		}
		result.Retried++
	}
	return result, nil
}

func featureJobFailureReason(err error) string {
	if errors.Is(err, context.DeadlineExceeded) {
		return "operation deadline exceeded"
	}
	if errors.Is(err, context.Canceled) {
		return "operation canceled"
	}
	return "handler failed"
}
