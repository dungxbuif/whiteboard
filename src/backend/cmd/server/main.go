package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/casbin/casbin/v3"
	gormadapter "github.com/casbin/gorm-adapter/v3"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	httpdelivery "devtool/internal/delivery/http"
	"devtool/internal/delivery/http/handler"
	middledelivery "devtool/internal/delivery/http/middleware"
	"devtool/internal/domain"
	"devtool/internal/realtime"
	geminirepo "devtool/internal/repository/gemini"
	postgresrepo "devtool/internal/repository/postgres"
	s3repo "devtool/internal/repository/s3"
	"devtool/internal/usecase"
)

type config struct {
	Addr                 string
	DatabaseURL          string
	JWTSecret            string
	CasbinModelPath      string
	S3Endpoint           string
	S3AccessKey          string
	S3SecretKey          string
	S3Bucket             string
	S3Region             string
	GoogleClientID       string
	GoogleClientSecret   string
	GoogleRedirectURL    string
	FrontendURL          string
	GeminiAPIKey         string
	GeminiModelFast      string
	GeminiModelReasoning string
	TURNURL              string
	TURNUsername         string
	TURNPassword         string
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

	// S3 Configuration
	awsCfg, err := awsconfig.LoadDefaultConfig(context.Background(),
		awsconfig.WithRegion(cfg.S3Region),
		awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(cfg.S3AccessKey, cfg.S3SecretKey, "")),
	)
	if err != nil {
		logger.Error("failed to load aws config", "error", err)
		os.Exit(1)
	}
	s3Client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(cfg.S3Endpoint)
		o.UsePathStyle = true
	})

	// Repositories
	userRepo := postgresrepo.NewUserRepository(db)
	workspaceRepo := postgresrepo.NewWorkspaceRepository(db)
	folderRepo := postgresrepo.NewFolderRepository(db)
	sceneRepo := postgresrepo.NewSceneRepository(db)
	searchRepo := postgresrepo.NewSearchRepository(db)
	scenePermissionRepo := postgresrepo.NewScenePermissionRepository(db)
	sceneAssetRepo := postgresrepo.NewSceneAssetRepository(db)
	s3Repo := s3repo.NewS3Repository(s3Client)

	// Usecases
	authUsecase := usecase.NewAuthUsecase(userRepo, []byte(cfg.JWTSecret), usecase.OAuthConfig{
		ClientID: cfg.GoogleClientID, ClientSecret: cfg.GoogleClientSecret,
		RedirectURL: cfg.GoogleRedirectURL, FrontendURL: cfg.FrontendURL,
	})
	workspaceUsecase := usecase.NewWorkspaceUsecase(workspaceRepo)
	folderUsecase := usecase.NewFolderUsecase(folderRepo)
	sceneUsecase := usecase.NewSceneUsecase(sceneRepo, folderRepo)
	searchUsecase := usecase.NewSearchUsecase(searchRepo)
	scenePermissionUsecase := usecase.NewScenePermissionUsecase(scenePermissionRepo, sceneRepo)
	assetUsecase := usecase.NewAssetUsecase(sceneRepo, sceneAssetRepo, s3Repo, cfg.S3Bucket, 1*time.Hour)
	aiRepo := geminirepo.New(cfg.GeminiAPIKey, cfg.GeminiModelFast)
	aiUsecase := usecase.NewAIUsecase(aiRepo, cfg.GeminiModelFast, cfg.GeminiModelReasoning)

	// Handlers
	authHandler := handler.NewAuthHandler(authUsecase, cfg.FrontendURL)
	workspaceHandler := handler.NewWorkspaceHandler(workspaceUsecase)
	folderHandler := handler.NewFolderHandler(folderUsecase)
	sceneHandler := handler.NewSceneHandler(sceneUsecase)
	searchHandler := handler.NewSearchHandler(searchUsecase)
	sharingHandler := handler.NewSharingHandler(scenePermissionUsecase)
	publicHandler := handler.NewPublicHandler(scenePermissionUsecase)
	assetHandler := handler.NewAssetHandler(assetUsecase)
	aiHandler := handler.NewAIHandler(aiUsecase)
	realtimeServer, err := realtime.NewServer([]byte(cfg.JWTSecret))
	if err != nil {
		logger.Error("failed to create realtime server", "error", err)
		os.Exit(1)
	}
	realtimeServer.SetTURNConfig(cfg.TURNURL, cfg.TURNUsername, cfg.TURNPassword)

	// Middlewares
	authMiddleware := middledelivery.AuthMiddleware([]byte(cfg.JWTSecret))

	server := &http.Server{
		Addr:              cfg.Addr,
		Handler:           httpdelivery.NewRouterWithSharing(authHandler, workspaceHandler, folderHandler, sceneHandler, assetHandler, authMiddleware, searchHandler, sharingHandler, publicHandler, realtimeServer.Handler(), aiHandler.Handler(), http.HandlerFunc(realtimeServer.ConfigHandler)),
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
		"S3_ENDPOINT",
		"S3_ACCESS_KEY",
		"S3_SECRET_KEY",
		"S3_BUCKET",
		"GOOGLE_CLIENT_ID",
		"GOOGLE_CLIENT_SECRET",
		"GOOGLE_REDIRECT_URL",
		"FRONTEND_URL",
	}

	for _, key := range required {
		if os.Getenv(key) == "" {
			return config{}, fmt.Errorf("missing required environment variable: %s", key)
		}
	}

	return config{
		Addr:                 getEnv("HTTP_ADDR", defaultHTTPAddr),
		DatabaseURL:          os.Getenv("DATABASE_URL"),
		JWTSecret:            os.Getenv("JWT_SECRET"),
		CasbinModelPath:      os.Getenv("CASBIN_MODEL_PATH"),
		S3Endpoint:           os.Getenv("S3_ENDPOINT"),
		S3AccessKey:          os.Getenv("S3_ACCESS_KEY"),
		S3SecretKey:          os.Getenv("S3_SECRET_KEY"),
		S3Bucket:             os.Getenv("S3_BUCKET"),
		S3Region:             getEnv("S3_REGION", "auto"),
		GoogleClientID:       os.Getenv("GOOGLE_CLIENT_ID"),
		GoogleClientSecret:   os.Getenv("GOOGLE_CLIENT_SECRET"),
		GoogleRedirectURL:    os.Getenv("GOOGLE_REDIRECT_URL"),
		FrontendURL:          os.Getenv("FRONTEND_URL"),
		GeminiAPIKey:         os.Getenv("GEMINI_API_KEY"),
		GeminiModelFast:      getEnv("GEMINI_MODEL_FAST", "gemini-1.5-flash"),
		GeminiModelReasoning: getEnv("GEMINI_MODEL_REASONING", "gemini-1.5-pro"),
		TURNURL:              os.Getenv("TURN_URL"),
		TURNUsername:         os.Getenv("TURN_USERNAME"),
		TURNPassword:         os.Getenv("TURN_PASSWORD"),
	}, nil
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
