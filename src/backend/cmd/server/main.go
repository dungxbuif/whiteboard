package main

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	httpdelivery "devtool/internal/delivery/http"
	"devtool/internal/delivery/http/handler"
	middledelivery "devtool/internal/delivery/http/middleware"
	"devtool/internal/domain"
	collabrepo "devtool/internal/repository/collab"
	openairepo "devtool/internal/repository/openai"
	postgresrepo "devtool/internal/repository/postgres"
	s3repo "devtool/internal/repository/s3"
	"devtool/internal/usecase"
)

type config struct {
	Addr                   string
	DatabaseURL            string
	JWTSecret              string
	CollabJWTSecret        string
	AuthorityServiceSecret string
	MutationGrantSecret    string
	AuthorityListenAddr    string
	CollabInternalURL      string
	CollabInternalSecret   string
	CasbinModelPath        string
	S3Endpoint             string
	S3AccessKey            string
	S3SecretKey            string
	S3Bucket               string
	S3Region               string
	FrontendURL            string
	AIAPIKey               string
	AIBaseURL              string
	AIModelFast            string
	AIModelReasoning       string
	TURNURL                string
	TURNAuthSecret         string
}

const defaultHTTPAddr = ":8080"
const defaultAuthorityAddr = "127.0.0.1:8081"

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
	assetUsecase := usecase.NewAssetUsecase(sceneRepo, sceneAssetRepo, s3Repo, cfg.S3Bucket, domain.AssetCapabilityLifetime)
	capabilities, err := usecase.NewCapabilityService(accessService, shareLinkRepo, assetUsecase, []byte(cfg.MutationGrantSecret))
	if err != nil {
		logger.Error("failed to configure mutation authority", "error", err)
		os.Exit(1)
	}
	capabilityHandler, err := handler.NewCapabilityHandler(capabilities, rooms, accessService, assetUsecase, []byte(cfg.AuthorityServiceSecret))
	if err != nil {
		logger.Error("failed to configure private authority", "error", err)
		os.Exit(1)
	}
	if err = capabilityHandler.SetTURNConfig(cfg.TURNURL, cfg.TURNAuthSecret); err != nil {
		logger.Error("failed to configure scoped TURN", "error", err)
		os.Exit(1)
	}
	aiRepo := openairepo.New(cfg.AIAPIKey, cfg.AIBaseURL, cfg.AIModelFast)
	aiUsecase := usecase.NewAIUsecase(aiRepo, cfg.AIModelFast, cfg.AIModelReasoning)

	// Handlers
	credentialLimiter, err := middledelivery.NewDefaultCredentialRateLimiter()
	if err != nil {
		logger.Error("failed to configure credential rate limiting", "error", err)
		os.Exit(1)
	}
	invitationHandler := handler.NewInvitationHandler(invitationUsecase)
	invitationHandler.SetCredentialLimiter(credentialLimiter)
	authHandler := handler.NewAuthHandler(authUsecase, cfg.FrontendURL)
	authHandler.SetCredentialLimiter(credentialLimiter)
	workspaceHandler := handler.NewWorkspaceHandler(workspaceUsecase)
	folderHandler := handler.NewFolderHandler(folderUsecase)
	sceneHandler := handler.NewSceneHandler(sceneUsecase)
	searchHandler := handler.NewSearchHandler(searchUsecase)
	sharingHandler := handler.NewSharingHandler(scenePermissionUsecase, shareLinks)
	roomHandler := handler.NewRoomHandler(rooms, accessService, sceneUsecase)
	roomHandler.SetCredentialLimiter(credentialLimiter)
	publicHandler := handler.NewPublicHandler(scenePermissionUsecase, roomHandler)
	publicHandler.SetCredentialLimiter(credentialLimiter)
	assetHandler := handler.NewAssetHandler(assetUsecase)
	assetHandler.SetCapabilityService(capabilities)
	assetHandler.Capabilities = capabilityHandler
	aiHandler := handler.NewAIHandler(aiUsecase)
	// Middlewares
	authMiddleware := middledelivery.AuthMiddleware([]byte(cfg.JWTSecret), userRepo)

	agents := usecase.NewAgentService(postgresrepo.NewAgentRepository(db), accessService)
	agentHandler := handler.NewAgentHandler(agents, rooms, accessService)
	features := usecase.NewFeatureService(postgresrepo.NewFeatureRepository(db), accessService)
	featureHandler := handler.NewFeatureHandler(features)
	agentHandler.SetExportAssets(usecase.NewExportAssets(agents, sceneAssetRepo, s3Repo.(domain.ExportObjectReader)))
	if cfg.CollabInternalURL != "" {
		remote, e := collabrepo.NewAgentClient(cfg.CollabInternalURL, cfg.CollabInternalSecret)
		if e != nil {
			logger.Error("invalid collaboration bridge configuration")
			os.Exit(1)
		}
		agentHandler.SetBridge(usecase.NewAgentBridge(agents, rooms, capabilities, remote))
	}
	server := &http.Server{
		Addr:              cfg.Addr,
		Handler:           httpdelivery.NewRouterWithAgents(authHandler, workspaceHandler, folderHandler, sceneHandler, assetHandler, authMiddleware, accessService, searchHandler, sharingHandler, publicHandler, nil, aiHandler.Handler(), nil, invitationHandler, agentHandler, featureHandler),
		ReadHeaderTimeout: domain.DefaultReadHeaderTimeout,
	}

	privateServer := &http.Server{Addr: cfg.AuthorityListenAddr, Handler: capabilityHandler.AuthorityHandler(), ReadHeaderTimeout: domain.DefaultReadHeaderTimeout, ReadTimeout: 10 * time.Second, WriteTimeout: 10 * time.Second}
	privateListener, err := net.Listen("tcp", cfg.AuthorityListenAddr)
	if err != nil {
		logger.Error("failed to bind private authority", "error", err)
		os.Exit(1)
	}
	go func() {
		if e := privateServer.Serve(privateListener); e != nil && e != http.ErrServerClosed {
			logger.Error("private authority stopped", "error", e)
			os.Exit(1)
		}
	}()
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
		"AUTHORITY_SERVICE_SECRET",
		"MUTATION_GRANT_SECRET",
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
	if len(os.Getenv("JWT_SECRET")) < 32 {
		return config{}, fmt.Errorf("JWT_SECRET must contain at least 32 bytes")
	}

	if len(os.Getenv("COLLAB_JWT_SECRET")) < 32 || os.Getenv("COLLAB_JWT_SECRET") == os.Getenv("JWT_SECRET") {
		return config{}, fmt.Errorf("COLLAB_JWT_SECRET must contain at least 32 bytes and differ from JWT_SECRET")
	}

	secrets := []string{os.Getenv("JWT_SECRET"), os.Getenv("COLLAB_JWT_SECRET")}
	for _, key := range []string{"AUTHORITY_SERVICE_SECRET", "MUTATION_GRANT_SECRET"} {
		value := os.Getenv(key)
		if len(value) < 32 {
			return config{}, fmt.Errorf("%s must contain at least 32 bytes", key)
		}
		for _, prior := range secrets {
			if prior == value {
				return config{}, fmt.Errorf("%s must use a distinct secret", key)
			}
		}
		secrets = append(secrets, value)
	}
	authorityAddr := getEnv("AUTHORITY_LISTEN_ADDR", defaultAuthorityAddr)
	host, port, e := net.SplitHostPort(authorityAddr)
	ip := net.ParseIP(host)
	n, pe := strconv.Atoi(port)
	if e != nil || pe != nil || n < 1 || n > 65535 || ip == nil || (!ip.IsLoopback() && !ip.IsPrivate()) {
		return config{}, fmt.Errorf("AUTHORITY_LISTEN_ADDR must bind an explicit loopback or private IP and valid port")
	}
	if os.Getenv("TURN_URL") != "" && len(os.Getenv("TURN_AUTH_SECRET")) < 32 {
		return config{}, fmt.Errorf("TURN_URL requires TURN_AUTH_SECRET of at least 32 bytes")
	}
	frontend, err := url.Parse(os.Getenv("FRONTEND_URL"))
	if err != nil || frontend.Hostname() == "" || (frontend.Scheme != "https" && frontend.Scheme != "http") || frontend.User != nil || frontend.RawQuery != "" || frontend.ForceQuery || frontend.Fragment != "" || (frontend.Path != "" && frontend.Path != "/") {
		return config{}, fmt.Errorf("FRONTEND_URL must be an HTTP(S) origin without credentials, path, query, or fragment")
	}
	if (os.Getenv("COLLAB_INTERNAL_URL") == "") != (os.Getenv("COLLAB_INTERNAL_SECRET") == "") {
		return config{}, fmt.Errorf("COLLAB_INTERNAL_URL and COLLAB_INTERNAL_SECRET must be configured together")
	}
	if os.Getenv("COLLAB_INTERNAL_URL") != "" {
		if _, e := collabrepo.NewAgentClient(os.Getenv("COLLAB_INTERNAL_URL"), os.Getenv("COLLAB_INTERNAL_SECRET")); e != nil {
			return config{}, e
		}
		for _, prior := range secrets {
			if prior == os.Getenv("COLLAB_INTERNAL_SECRET") {
				return config{}, fmt.Errorf("COLLAB_INTERNAL_SECRET must be distinct")
			}
		}
	}
	return config{
		Addr:                   getEnv("HTTP_ADDR", defaultHTTPAddr),
		DatabaseURL:            os.Getenv("DATABASE_URL"),
		JWTSecret:              os.Getenv("JWT_SECRET"),
		CollabJWTSecret:        os.Getenv("COLLAB_JWT_SECRET"),
		AuthorityServiceSecret: os.Getenv("AUTHORITY_SERVICE_SECRET"),
		MutationGrantSecret:    os.Getenv("MUTATION_GRANT_SECRET"),
		AuthorityListenAddr:    authorityAddr,
		CollabInternalURL:      os.Getenv("COLLAB_INTERNAL_URL"),
		CollabInternalSecret:   os.Getenv("COLLAB_INTERNAL_SECRET"),
		CasbinModelPath:        os.Getenv("CASBIN_MODEL_PATH"),
		S3Endpoint:             os.Getenv("S3_ENDPOINT"),
		S3AccessKey:            os.Getenv("S3_ACCESS_KEY"),
		S3SecretKey:            os.Getenv("S3_SECRET_KEY"),
		S3Bucket:               os.Getenv("S3_BUCKET"),
		S3Region:               getEnv("S3_REGION", "auto"),
		FrontendURL:            os.Getenv("FRONTEND_URL"),
		AIAPIKey:               os.Getenv("AI_API_KEY"),
		AIBaseURL:              getEnv("AI_BASE_URL", "https://api.deepseek.com/v1"),
		AIModelFast:            getEnv("AI_MODEL_FAST", "deepseek-chat"),
		AIModelReasoning:       getEnv("AI_MODEL_REASONING", "deepseek-reasoner"),
		TURNURL:                os.Getenv("TURN_URL"),
		TURNAuthSecret:         os.Getenv("TURN_AUTH_SECRET"),
	}, nil
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
