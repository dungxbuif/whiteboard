package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	httpdelivery "devtool/internal/delivery/http"
)

type config struct {
	Addr        string
	DatabaseURL string
}

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	cfg := loadConfig()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var db *pgxpool.Pool
	if cfg.DatabaseURL != "" {
		pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
		if err != nil {
			logger.Warn("database pool not initialized", "error", err)
		} else if err := pool.Ping(ctx); err != nil {
			logger.Warn("database ping failed", "error", err)
			pool.Close()
		} else {
			db = pool
			defer db.Close()
			logger.Info("database connection initialized")
		}
	}

	server := &http.Server{
		Addr:              cfg.Addr,
		Handler:           httpdelivery.NewRouter(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	logger.Info("starting server", "addr", cfg.Addr)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logger.Error("server stopped", "error", err)
		os.Exit(1)
	}
}

func loadConfig() config {
	return config{
		Addr:        getEnv("HTTP_ADDR", ":8080"),
		DatabaseURL: getEnv("DATABASE_URL", defaultDatabaseURL()),
	}
}

func defaultDatabaseURL() string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=disable",
		getEnv("POSTGRES_USER", "dev"),
		getEnv("POSTGRES_PASSWORD", "password"),
		getEnv("POSTGRES_HOST", "localhost"),
		getEnv("POSTGRES_PORT", "5432"),
		getEnv("POSTGRES_DB", "devtool"),
	)
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
