package usecase

import (
	"context"
	"devtool/internal/domain"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
)

type featureJobRepoStub struct {
	mu           sync.Mutex
	jobs         []domain.FeatureJob
	completed    []uuid.UUID
	retried      []uuid.UUID
	dead         []uuid.UUID
	retryReasons []string
	deadReasons  []string
}

func (r *featureJobRepoStub) Claim(_ context.Context, _ time.Time, limit int, workerID string, lease time.Duration) ([]domain.FeatureJob, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if limit > len(r.jobs) {
		limit = len(r.jobs)
	}
	claimed := append([]domain.FeatureJob(nil), r.jobs[:limit]...)
	r.jobs = r.jobs[limit:]
	return claimed, nil
}

func (r *featureJobRepoStub) Complete(_ context.Context, id uuid.UUID, _ string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.completed = append(r.completed, id)
	return nil
}

func (r *featureJobRepoStub) Retry(_ context.Context, id uuid.UUID, _ string, _ time.Time, reason string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.retried = append(r.retried, id)
	r.retryReasons = append(r.retryReasons, reason)
	return nil
}

func (r *featureJobRepoStub) DeadLetter(_ context.Context, id uuid.UUID, _ string, reason string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.dead = append(r.dead, id)
	r.deadReasons = append(r.deadReasons, reason)
	return nil
}

func TestCrashAfterEffectRetriesSafely(t *testing.T) {
	now := time.Date(2026, 10, 20, 12, 0, 0, 0, time.UTC)
	id := uuid.New()
	repo := &featureJobRepoStub{jobs: []domain.FeatureJob{{ID: id, Kind: "notify", Attempts: 1}}}
	effects := map[uuid.UUID]bool{}
	jobs, err := NewFeatureJobs(repo, "worker-a", map[string]FeatureJobHandler{
		"notify": func(_ context.Context, job domain.FeatureJob) error {
			effects[job.ID] = true
			return errors.New("acknowledgement lost secret=private")
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	result, err := jobs.RunOnce(context.Background(), now, 100)
	if err != nil {
		t.Fatal(err)
	}
	if result.Retried != 1 || len(repo.retried) != 1 || !effects[id] {
		t.Fatalf("expected idempotent effect to be retried, got %#v", result)
	}
	if len(repo.retryReasons) != 1 || repo.retryReasons[0] != "handler failed" {
		t.Fatalf("unsafe handler detail persisted: %#v", repo.retryReasons)
	}
}

func TestFeatureJobBackoffAndDeadLetter(t *testing.T) {
	now := time.Date(2026, 10, 20, 12, 0, 0, 0, time.UTC)
	repo := &featureJobRepoStub{jobs: []domain.FeatureJob{
		{ID: uuid.New(), Kind: "fail", Attempts: 2},
		{ID: uuid.New(), Kind: "fail", Attempts: 10},
	}}
	jobs, err := NewFeatureJobs(repo, "worker-a", map[string]FeatureJobHandler{
		"fail": func(context.Context, domain.FeatureJob) error { return errors.New("failed") },
	})
	if err != nil {
		t.Fatal(err)
	}
	result, err := jobs.RunOnce(context.Background(), now, 100)
	if err != nil {
		t.Fatal(err)
	}
	if result.Retried != 1 || result.DeadLettered != 1 {
		t.Fatalf("unexpected result: %#v", result)
	}
}

func TestShutdownDrainsWithinDeadline(t *testing.T) {
	repo := &featureJobRepoStub{jobs: []domain.FeatureJob{{ID: uuid.New(), Kind: "blocking", Attempts: 1}}}
	started := make(chan struct{})
	jobs, err := NewFeatureJobs(repo, "worker-a", map[string]FeatureJobHandler{
		"blocking": func(ctx context.Context, _ domain.FeatureJob) error {
			close(started)
			<-ctx.Done()
			return ctx.Err()
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		_, _ = jobs.RunOnce(ctx, time.Now(), 1)
	}()
	<-started
	cancel()
	select {
	case <-done:
	case <-time.After(250 * time.Millisecond):
		t.Fatal("worker did not drain after cancellation")
	}
}
