package main

import (
	"context"
	postgresrepo "devtool/internal/repository/postgres"
	"devtool/internal/usecase"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

const (
	workerPollInterval = time.Second
	workerDrainTimeout = 30 * time.Second
)

type config struct {
	DatabaseURL string
	WorkerID    string
	Addr        string
}

func loadConfig() (config, error) {
	values := map[string]string{
		"DATABASE_URL":        os.Getenv("DATABASE_URL"),
		"FEATURE_WORKER_ID":   os.Getenv("FEATURE_WORKER_ID"),
		"FEATURE_WORKER_ADDR": os.Getenv("FEATURE_WORKER_ADDR"),
	}
	for key, value := range values {
		if value == "" {
			return config{}, fmt.Errorf("missing required environment variable: %s", key)
		}
	}
	return config{DatabaseURL: values["DATABASE_URL"], WorkerID: values["FEATURE_WORKER_ID"], Addr: values["FEATURE_WORKER_ADDR"]}, nil
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	cfg, err := loadConfig()
	if err != nil {
		logger.Error("invalid feature worker configuration", "error", err)
		os.Exit(1)
	}
	db, err := gorm.Open(postgres.Open(cfg.DatabaseURL), &gorm.Config{})
	if err != nil {
		logger.Error("feature worker database connection failed", "error", err)
		os.Exit(1)
	}
	jobs, err := usecase.NewFeatureJobs(postgresrepo.NewJobRepository(db), cfg.WorkerID, map[string]usecase.FeatureJobHandler{})
	if err != nil {
		logger.Error("feature worker setup failed", "error", err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	var ready atomic.Bool
	server := newHealthServer(cfg.Addr, &ready)
	serverErrors := make(chan error, 1)
	go func() {
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErrors <- err
		}
	}()

	workerDone := make(chan struct{})
	go func() {
		defer close(workerDone)
		runWorker(ctx, jobs, &ready, logger)
	}()

	select {
	case <-ctx.Done():
	case err := <-serverErrors:
		logger.Error("feature worker health server stopped", "error", err)
		stop()
	}
	ready.Store(false)
	drainCtx, cancel := context.WithTimeout(context.Background(), workerDrainTimeout)
	defer cancel()
	_ = server.Shutdown(drainCtx)
	select {
	case <-workerDone:
	case <-drainCtx.Done():
		logger.Error("feature worker drain deadline exceeded")
		os.Exit(1)
	}
}

func runWorker(ctx context.Context, jobs *usecase.FeatureJobs, ready *atomic.Bool, logger *slog.Logger) {
	ticker := time.NewTicker(workerPollInterval)
	defer ticker.Stop()
	for {
		if ctx.Err() != nil {
			return
		}
		// A stop signal prevents the next claim. The active bounded operation keeps a
		// live context so its handler and completion/retry write can drain safely.
		result, err := jobs.RunOnce(context.WithoutCancel(ctx), time.Now().UTC(), 1)
		if err != nil {
			ready.Store(false)
			logger.Error("feature job batch failed", "error", err)
		} else {
			ready.Store(true)
			if result.Claimed > 0 {
				logger.Info("feature job batch processed", "claimed", result.Claimed, "completed", result.Completed, "retried", result.Retried, "deadLettered", result.DeadLettered)
				continue
			}
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func newHealthServer(addr string, ready *atomic.Bool) *http.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	mux.HandleFunc("GET /readyz", func(w http.ResponseWriter, _ *http.Request) {
		if !ready.Load() {
			http.Error(w, "not ready", http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ready"))
	})
	return &http.Server{Addr: addr, Handler: mux, ReadHeaderTimeout: 5 * time.Second}
}
