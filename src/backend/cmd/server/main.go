package main

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"github.com/casbin/casbin/v3"
	gormadapter "github.com/casbin/gorm-adapter/v3"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"devtool/internal/domain"
	httpdelivery "devtool/internal/delivery/http"
	"devtool/internal/delivery/http/handler"
	middledelivery "devtool/internal/delivery/http/middleware"
	postgresrepo "devtool/internal/repository/postgres"
	"devtool/internal/usecase"
)

type config struct {
	Addr               string
	DatabaseURL        string
	GoogleClientID     string
	GoogleClientSecret string
	GoogleRedirectURL  string
	JWTSecret          string
	CasbinModelPath    string
}

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

	// OAuth Config
	oauthConfig := &oauth2.Config{
		ClientID:     cfg.GoogleClientID,
		ClientSecret: cfg.GoogleClientSecret,
		RedirectURL:  cfg.GoogleRedirectURL,
		Scopes: []string{
			"https://www.googleapis.com/auth/userinfo.email",
			"https://www.googleapis.com/auth/userinfo.profile",
			"openid",
		},
		Endpoint: google.Endpoint,
	}

	// Usecases
	authUsecase := usecase.NewAuthUsecase(oauthConfig, userRepo, []byte(cfg.JWTSecret))
	workspaceUsecase := usecase.NewWorkspaceUsecase(workspaceRepo)

	// Handlers
	authHandler := handler.NewAuthHandler(authUsecase)
	workspaceHandler := handler.NewWorkspaceHandler(workspaceUsecase)

	// Middlewares
	authMiddleware := middledelivery.AuthMiddleware([]byte(cfg.JWTSecret))

	server := &http.Server{
		Addr:              cfg.Addr,
		Handler:           httpdelivery.NewRouter(authHandler, workspaceHandler, authMiddleware),
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
		"GOOGLE_CLIENT_ID",
		"GOOGLE_CLIENT_SECRET",
		"DATABASE_URL",
		"JWT_SECRET",
	}

	for _, key := range required {
		if os.Getenv(key) == "" {
			return config{}, fmt.Errorf("missing required environment variable: %s", key)
		}
	}

	return config{
		Addr:               getEnv("HTTP_ADDR", ":8080"),
		DatabaseURL:        os.Getenv("DATABASE_URL"),
		GoogleClientID:     os.Getenv("GOOGLE_CLIENT_ID"),
		GoogleClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
		GoogleRedirectURL:  getEnv("GOOGLE_REDIRECT_URL", "http://localhost:8080/auth/google/callback"),
		JWTSecret:          os.Getenv("JWT_SECRET"),
		CasbinModelPath:    getEnv("CASBIN_MODEL_PATH", "configs/casbin_model.conf"),
	}, nil
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
