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
	claimLimits  []int
	claimTimes   []time.Time
}

func (r *featureJobRepoStub) Claim(_ context.Context, now time.Time, limit int, workerID string, lease time.Duration) ([]domain.FeatureJob, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.claimLimits = append(r.claimLimits, limit)
	r.claimTimes = append(r.claimTimes, now)
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
	effectCount := 0
	handlerCalls := 0
	jobs, err := NewFeatureJobs(repo, "worker-a", map[string]FeatureJobHandler{
		"notify": func(_ context.Context, job domain.FeatureJob) error {
			handlerCalls++
			if !effects[job.ID] {
				effects[job.ID] = true
				effectCount++
			}
			if handlerCalls == 1 {
				return errors.New("acknowledgement lost secret=private")
			}
			return nil
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
	repo.jobs = append(repo.jobs, domain.FeatureJob{ID: id, Kind: "notify", Attempts: 2})
	result, err = jobs.RunOnce(context.Background(), now.Add(time.Minute), 100)
	if err != nil {
		t.Fatal(err)
	}
	if result.Completed != 1 || effectCount != 1 || len(repo.completed) != 1 {
		t.Fatalf("redelivery duplicated effect: result=%#v effects=%d", result, effectCount)
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

func TestRunOnceDoesNotLeaseQueuedJobs(t *testing.T) {
	repo := &featureJobRepoStub{jobs: []domain.FeatureJob{
		{ID: uuid.New(), Kind: "work", Attempts: 1},
		{ID: uuid.New(), Kind: "work", Attempts: 1},
		{ID: uuid.New(), Kind: "work", Attempts: 1},
	}}
	jobs, err := NewFeatureJobs(repo, "worker-a", map[string]FeatureJobHandler{
		"work": func(context.Context, domain.FeatureJob) error { return nil },
	})
	if err != nil {
		t.Fatal(err)
	}
	result, err := jobs.RunOnce(context.Background(), time.Now(), 3)
	if err != nil {
		t.Fatal(err)
	}
	if result.Completed != 3 {
		t.Fatalf("unexpected result: %#v", result)
	}
	for _, limit := range repo.claimLimits {
		if limit != 1 {
			t.Fatalf("queued jobs were leased as a batch: limits=%v", repo.claimLimits)
		}
	}
}

func TestRunOnceRefreshesClockBeforeEachLease(t *testing.T) {
	start := time.Date(2026, 10, 20, 12, 0, 0, 0, time.UTC)
	later := start.Add(59 * time.Second)
	repo := &featureJobRepoStub{jobs: []domain.FeatureJob{
		{ID: uuid.New(), Kind: "work", Attempts: 1},
		{ID: uuid.New(), Kind: "work", Attempts: 1},
	}}
	jobs, err := NewFeatureJobs(repo, "worker-a", map[string]FeatureJobHandler{
		"work": func(context.Context, domain.FeatureJob) error { return nil },
	})
	if err != nil {
		t.Fatal(err)
	}
	jobs.now = func() time.Time { return later }
	if _, err := jobs.RunOnce(context.Background(), start, 2); err != nil {
		t.Fatal(err)
	}
	if len(repo.claimTimes) < 2 || !repo.claimTimes[1].Equal(later) {
		t.Fatalf("second job received an aging lease base: %v", repo.claimTimes)
	}
}
