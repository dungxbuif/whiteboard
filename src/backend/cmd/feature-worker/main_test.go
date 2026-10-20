package main

import (
	"os"
	"testing"
)

func TestLoadConfigRequiresWorkerSettings(t *testing.T) {
	for _, key := range []string{"DATABASE_URL", "FEATURE_WORKER_ID", "FEATURE_WORKER_ADDR"} {
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
	cfg, err := loadConfig()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.DatabaseURL != os.Getenv("DATABASE_URL") || cfg.WorkerID != "worker-a" || cfg.Addr != "127.0.0.1:8090" {
		t.Fatalf("unexpected config: %#v", cfg)
	}
}
