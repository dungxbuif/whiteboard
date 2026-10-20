package main

import (
	"context"
	"devtool/internal/domain"
	"devtool/internal/usecase"
	"io"
	"log/slog"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
)

type drainRepo struct {
	mu        sync.Mutex
	job       *domain.FeatureJob
	completed chan struct{}
	claims    int
}

type notificationDelivererStub struct {
	eventID int64
}

func (s *notificationDelivererStub) DeliverEvent(_ context.Context, eventID int64) error {
	s.eventID = eventID
	return nil
}

func TestNotificationJobHandlerRequiresAndDeliversEventID(t *testing.T) {
	service := &notificationDelivererStub{}
	handle := notificationJobHandler(service)
	if err := handle(context.Background(), domain.FeatureJob{}); err == nil {
		t.Fatal("notification job without event ID was accepted")
	}
	eventID := int64(42)
	if err := handle(context.Background(), domain.FeatureJob{EventID: &eventID}); err != nil {
		t.Fatal(err)
	}
	if service.eventID != eventID {
		t.Fatalf("delivered event ID=%d want=%d", service.eventID, eventID)
	}
}

func (r *drainRepo) Claim(_ context.Context, _ time.Time, _ int, workerID string, _ time.Duration) ([]domain.FeatureJob, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.claims++
	if r.job == nil {
		return nil, nil
	}
	job := *r.job
	job.LeaseOwner = workerID
	r.job = nil
	return []domain.FeatureJob{job}, nil
}
func (r *drainRepo) Complete(context.Context, uuid.UUID, string) error {
	close(r.completed)
	return nil
}
func (r *drainRepo) Retry(context.Context, uuid.UUID, string, time.Time, string) error { return nil }
func (r *drainRepo) DeadLetter(context.Context, uuid.UUID, string, string) error       { return nil }

func TestLoadConfigRequiresWorkerSettings(t *testing.T) {
	for _, key := range []string{"DATABASE_URL", "FEATURE_WORKER_ID", "FEATURE_WORKER_ADDR", "CASBIN_MODEL_PATH"} {
		t.Setenv(key, "")
	}
	if _, err := loadConfig(); err == nil {
		t.Fatal("missing worker configuration accepted")
	}
}

func TestLoadConfigAcceptsExplicitSettings(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://worker@example.invalid/whiteboard")
	t.Setenv("FEATURE_WORKER_ID", "worker-a")
	t.Setenv("FEATURE_WORKER_ADDR", "127.0.0.1:8090")
	t.Setenv("CASBIN_MODEL_PATH", "configs/casbin_model.conf")
	cfg, err := loadConfig()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.DatabaseURL != os.Getenv("DATABASE_URL") || cfg.WorkerID != "worker-a" || cfg.Addr != "127.0.0.1:8090" || cfg.CasbinModelPath != "configs/casbin_model.conf" {
		t.Fatalf("unexpected config: %#v", cfg)
	}
}

func TestShutdownDrainsActiveWorkAndPersistsOutcome(t *testing.T) {
	started := make(chan struct{})
	release := make(chan struct{})
	contextCanceled := make(chan bool, 1)
	repo := &drainRepo{job: &domain.FeatureJob{ID: uuid.New(), Kind: "blocking", Attempts: 1}, completed: make(chan struct{})}
	jobs, err := usecase.NewFeatureJobs(repo, "worker-a", map[string]usecase.FeatureJobHandler{
		"blocking": func(ctx context.Context, _ domain.FeatureJob) error {
			close(started)
			select {
			case <-ctx.Done():
				contextCanceled <- true
			case <-release:
				contextCanceled <- false
			}
			return nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	gate := newWorkerStopGate()
	done := make(chan struct{})
	var ready atomic.Bool
	go func() {
		defer close(done)
		runWorker(gate, jobs, &ready, slog.New(slog.NewTextHandler(io.Discard, nil)))
	}()
	<-started
	gate.Stop()
	select {
	case canceled := <-contextCanceled:
		if canceled {
			t.Fatal("shutdown canceled active work instead of draining it")
		}
	case <-time.After(50 * time.Millisecond):
	}
	close(release)
	select {
	case <-repo.completed:
	case <-time.After(time.Second):
		t.Fatal("active outcome was not persisted")
	}
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("worker did not stop after active work drained")
	}
}

func TestStopBetweenLoopCheckAndClaimPreventsNewLease(t *testing.T) {
	repo := &drainRepo{job: &domain.FeatureJob{ID: uuid.New(), Kind: "work", Attempts: 1}, completed: make(chan struct{})}
	jobs, err := usecase.NewFeatureJobs(repo, "worker-a", map[string]usecase.FeatureJobHandler{
		"work": func(context.Context, domain.FeatureJob) error { return nil },
	})
	if err != nil {
		t.Fatal(err)
	}
	beforeClaim := make(chan struct{})
	releaseClaim := make(chan struct{})
	gate := newWorkerStopGate()
	gate.beforeClaim = func() {
		close(beforeClaim)
		<-releaseClaim
	}
	done := make(chan struct{})
	var ready atomic.Bool
	go func() {
		defer close(done)
		runWorker(gate, jobs, &ready, slog.New(slog.NewTextHandler(io.Discard, nil)))
	}()
	<-beforeClaim
	gate.Stop()
	close(releaseClaim)
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("stopped worker did not leave claim gate")
	}
	repo.mu.Lock()
	claims := repo.claims
	repo.mu.Unlock()
	if claims != 0 {
		t.Fatalf("worker began %d claims after stop", claims)
	}
}

func TestStopWaitsForAdmittedClaimDecision(t *testing.T) {
	gate := newWorkerStopGate()
	claimStarted := make(chan struct{})
	releaseClaim := make(chan struct{})
	claimDone := make(chan struct{})
	go func() {
		defer close(claimDone)
		allowed, err := gate.Claim(func() error {
			close(claimStarted)
			<-releaseClaim
			return nil
		})
		if err != nil || !allowed {
			t.Errorf("admitted claim result allowed=%v err=%v", allowed, err)
		}
	}()

	<-claimStarted
	stopDone := make(chan struct{})
	go func() {
		gate.Stop()
		close(stopDone)
	}()
	select {
	case <-stopDone:
		t.Fatal("stop returned before the admitted claim decision completed")
	case <-time.After(25 * time.Millisecond):
	}

	close(releaseClaim)
	select {
	case <-claimDone:
	case <-time.After(time.Second):
		t.Fatal("admitted claim did not finish")
	}
	select {
	case <-stopDone:
	case <-time.After(time.Second):
		t.Fatal("stop did not close after the claim decision")
	}
}
