package main

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"github.com/casbin/casbin/v3"
	gormadapter "github.com/casbin/gorm-adapter/v3"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	httpdelivery "devtool/internal/delivery/http"
	"devtool/internal/delivery/http/handler"
	middledelivery "devtool/internal/delivery/http/middleware"
	"devtool/internal/domain"
	postgresrepo "devtool/internal/repository/postgres"
	"devtool/internal/usecase"
)

type config struct {
	Addr            string
	DatabaseURL     string
	JWTSecret       string
	CasbinModelPath string
}

const defaultHTTPAddr = ":8080"

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	cfg, err := loadConfig()
	if err != nil {
		logger.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	db, err := gorm.Open(postgres.Open(cfg.DatabaseURL), &gorm.Config{})
	if err != nil {
		logger.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	logger.Info("database connection initialized with GORM")

	adapter, err := gormadapter.NewAdapterByDB(db)
	if err != nil {
		logger.Error("failed to create casbin adapter", "error", err)
		os.Exit(1)
	}

	enforcer, err := casbin.NewEnforcer(cfg.CasbinModelPath, adapter)
	if err != nil {
		logger.Error("failed to create casbin enforcer", "error", err)
		os.Exit(1)
	}

	if err := enforcer.LoadPolicy(); err != nil {
		logger.Warn("failed to load casbin policy", "error", err)
	}

	// Repositories
	userRepo := postgresrepo.NewUserRepository(db)
	workspaceRepo := postgresrepo.NewWorkspaceRepository(db)
	folderRepo := postgresrepo.NewFolderRepository(db)
	sceneRepo := postgresrepo.NewSceneRepository(db)

	// Usecases
	authUsecase := usecase.NewAuthUsecase(userRepo, []byte(cfg.JWTSecret))
	workspaceUsecase := usecase.NewWorkspaceUsecase(workspaceRepo)
	folderUsecase := usecase.NewFolderUsecase(folderRepo)
	sceneUsecase := usecase.NewSceneUsecase(sceneRepo, folderRepo)

	// Handlers
	authHandler := handler.NewAuthHandler(authUsecase)
	workspaceHandler := handler.NewWorkspaceHandler(workspaceUsecase)
	folderHandler := handler.NewFolderHandler(folderUsecase)
	sceneHandler := handler.NewSceneHandler(sceneUsecase)

	// Middlewares
	authMiddleware := middledelivery.AuthMiddleware([]byte(cfg.JWTSecret))

	server := &http.Server{
		Addr:              cfg.Addr,
		Handler:           httpdelivery.NewRouter(authHandler, workspaceHandler, folderHandler, sceneHandler, authMiddleware),
		ReadHeaderTimeout: domain.DefaultReadHeaderTimeout,
	}

	logger.Info("starting server", "addr", cfg.Addr)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logger.Error("server stopped", "error", err)
		os.Exit(1)
	}
}

func loadConfig() (config, error) {
	required := []string{
		"DATABASE_URL",
		"JWT_SECRET",
		"CASBIN_MODEL_PATH",
	}

	for _, key := range required {
		if os.Getenv(key) == "" {
			return config{}, fmt.Errorf("missing required environment variable: %s", key)
		}
	}

	return config{
		Addr:            getEnv("HTTP_ADDR", defaultHTTPAddr),
		DatabaseURL:     os.Getenv("DATABASE_URL"),
		JWTSecret:       os.Getenv("JWT_SECRET"),
		CasbinModelPath: os.Getenv("CASBIN_MODEL_PATH"),
	}, nil
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
