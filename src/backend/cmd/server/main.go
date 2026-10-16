package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	httpdelivery "devtool/internal/delivery/http"
	"devtool/internal/delivery/http/handler"
	middledelivery "devtool/internal/delivery/http/middleware"
	"devtool/internal/domain"
	"devtool/internal/realtime"
	openairepo "devtool/internal/repository/openai"
	postgresrepo "devtool/internal/repository/postgres"
	s3repo "devtool/internal/repository/s3"
	"devtool/internal/usecase"
)

type config struct {
	Addr             string
	DatabaseURL      string
	JWTSecret        string
	CollabJWTSecret  string
	CasbinModelPath  string
	S3Endpoint       string
	S3AccessKey      string
	S3SecretKey      string
	S3Bucket         string
	S3Region         string
	FrontendURL      string
	AIAPIKey         string
	AIBaseURL        string
	AIModelFast      string
	AIModelReasoning string
	TURNURL          string
	TURNUsername     string
	TURNPassword     string
	TURNAuthSecret   string
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

	accessService, err := usecase.NewAccessService(postgresrepo.NewAccessRepository(db), cfg.CasbinModelPath)
	if err != nil {
		logger.Error("failed to create access policy", "error", err)
		os.Exit(1)
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
	shareLinkRepo := postgresrepo.NewShareLinkRepository(db)
	shareLinks := usecase.NewShareLinkService(shareLinkRepo, accessService)
	rooms, err := usecase.NewRoomService(shareLinkRepo, accessService, []byte(cfg.CollabJWTSecret))
	if err != nil {
		logger.Error("failed to configure room credentials", "error", err)
		os.Exit(1)
	}
	invitationUsecase := usecase.NewInvitationUsecase(postgresrepo.NewInvitationRepository(db), accessService, cfg.FrontendURL)
	authUsecase := usecase.NewAuthUsecase(userRepo, []byte(cfg.JWTSecret))
	workspaceUsecase := usecase.NewWorkspaceUsecase(workspaceRepo)
	folderUsecase := usecase.NewFolderUsecase(folderRepo)
	sceneUsecase := usecase.NewSceneUsecase(sceneRepo, folderRepo)
	searchUsecase := usecase.NewSearchUsecase(searchRepo)
	scenePermissionUsecase := usecase.NewScenePermissionUsecase(scenePermissionRepo, sceneRepo)
	assetUsecase := usecase.NewAssetUsecase(sceneRepo, sceneAssetRepo, s3Repo, cfg.S3Bucket, 1*time.Hour)
	aiRepo := openairepo.New(cfg.AIAPIKey, cfg.AIBaseURL, cfg.AIModelFast)
	aiUsecase := usecase.NewAIUsecase(aiRepo, cfg.AIModelFast, cfg.AIModelReasoning)

	// Handlers
	invitationHandler := handler.NewInvitationHandler(invitationUsecase)
	authHandler := handler.NewAuthHandler(authUsecase, cfg.FrontendURL)
	workspaceHandler := handler.NewWorkspaceHandler(workspaceUsecase)
	folderHandler := handler.NewFolderHandler(folderUsecase)
	sceneHandler := handler.NewSceneHandler(sceneUsecase)
	searchHandler := handler.NewSearchHandler(searchUsecase)
	sharingHandler := handler.NewSharingHandler(scenePermissionUsecase, shareLinks)
	publicHandler := handler.NewPublicHandler(scenePermissionUsecase, handler.NewRoomHandler(rooms, accessService))
	assetHandler := handler.NewAssetHandler(assetUsecase)
	aiHandler := handler.NewAIHandler(aiUsecase)
	realtimeServer, err := realtime.NewServer([]byte(cfg.JWTSecret))
	if err != nil {
		logger.Error("failed to create realtime server", "error", err)
		os.Exit(1)
	}
	realtimeServer.SetTURNConfig(cfg.TURNURL, cfg.TURNUsername, cfg.TURNPassword)
	if cfg.TURNAuthSecret != "" {
		realtimeServer.SetTURNAuthSecret(cfg.TURNAuthSecret, cfg.TURNUsername)
	}
	realtimeServer.SetWorkspaceAuthorizer(func(ctx context.Context, workspaceID uuid.UUID, userID string) bool {
		userUUID, err := uuid.Parse(userID)
		if err != nil {
			return false
		}
		workspace, err := workspaceRepo.FindByID(ctx, workspaceID)
		return err == nil && workspace.OwnerID == userUUID
	})

	// Middlewares
	authMiddleware := middledelivery.AuthMiddleware([]byte(cfg.JWTSecret))

	server := &http.Server{
		Addr:              cfg.Addr,
		Handler:           httpdelivery.NewRouterWithAccess(authHandler, workspaceHandler, folderHandler, sceneHandler, assetHandler, authMiddleware, accessService, searchHandler, sharingHandler, publicHandler, realtimeServer.Handler(), aiHandler.Handler(), http.HandlerFunc(realtimeServer.ConfigHandler), invitationHandler),
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
		"COLLAB_JWT_SECRET",
		"CASBIN_MODEL_PATH",
		"S3_ENDPOINT",
		"S3_ACCESS_KEY",
		"S3_SECRET_KEY",
		"S3_BUCKET",
		"FRONTEND_URL",
	}

	for _, key := range required {
		if os.Getenv(key) == "" {
			return config{}, fmt.Errorf("missing required environment variable: %s", key)
		}
	}

	if len(os.Getenv("COLLAB_JWT_SECRET")) < 32 || os.Getenv("COLLAB_JWT_SECRET") == os.Getenv("JWT_SECRET") {
		return config{}, fmt.Errorf("COLLAB_JWT_SECRET must contain at least 32 bytes and differ from JWT_SECRET")
	}

	frontend, err := url.Parse(os.Getenv("FRONTEND_URL"))
	if err != nil || frontend.Hostname() == "" || (frontend.Scheme != "https" && frontend.Scheme != "http") || frontend.User != nil || frontend.RawQuery != "" || frontend.ForceQuery || frontend.Fragment != "" || (frontend.Path != "" && frontend.Path != "/") {
		return config{}, fmt.Errorf("FRONTEND_URL must be an HTTP(S) origin without credentials, path, query, or fragment")
	}
	return config{
		Addr:             getEnv("HTTP_ADDR", defaultHTTPAddr),
		DatabaseURL:      os.Getenv("DATABASE_URL"),
		JWTSecret:        os.Getenv("JWT_SECRET"),
		CollabJWTSecret:  os.Getenv("COLLAB_JWT_SECRET"),
		CasbinModelPath:  os.Getenv("CASBIN_MODEL_PATH"),
		S3Endpoint:       os.Getenv("S3_ENDPOINT"),
		S3AccessKey:      os.Getenv("S3_ACCESS_KEY"),
		S3SecretKey:      os.Getenv("S3_SECRET_KEY"),
		S3Bucket:         os.Getenv("S3_BUCKET"),
		S3Region:         getEnv("S3_REGION", "auto"),
		FrontendURL:      os.Getenv("FRONTEND_URL"),
		AIAPIKey:         os.Getenv("AI_API_KEY"),
		AIBaseURL:        getEnv("AI_BASE_URL", "https://api.deepseek.com/v1"),
		AIModelFast:      getEnv("AI_MODEL_FAST", "deepseek-chat"),
		AIModelReasoning: getEnv("AI_MODEL_REASONING", "deepseek-reasoner"),
		TURNURL:          os.Getenv("TURN_URL"),
		TURNUsername:     os.Getenv("TURN_USERNAME"),
		TURNPassword:     os.Getenv("TURN_PASSWORD"),
		TURNAuthSecret:   os.Getenv("TURN_AUTH_SECRET"),
	}, nil
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
