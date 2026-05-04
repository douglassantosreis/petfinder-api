package server

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	httpSwagger "github.com/swaggo/http-swagger/v2"
	"github.com/yourname/go-backend/internal/infra/jwt"
	inframoderation "github.com/yourname/go-backend/internal/infra/moderation"
	mongoinfra "github.com/yourname/go-backend/internal/infra/mongo"
	oauthinfra "github.com/yourname/go-backend/internal/infra/oauth"
	"github.com/yourname/go-backend/internal/infra/storage"
	"github.com/yourname/go-backend/internal/interface/http/handler"
	"github.com/yourname/go-backend/internal/interface/http/middleware"
	"github.com/yourname/go-backend/internal/platform/config"
	aduc "github.com/yourname/go-backend/internal/usecase/ad"
	authuc "github.com/yourname/go-backend/internal/usecase/auth"
	messageuc "github.com/yourname/go-backend/internal/usecase/message"
	moderationuc "github.com/yourname/go-backend/internal/usecase/moderation"
	uploaduc "github.com/yourname/go-backend/internal/usecase/upload"
	useruc "github.com/yourname/go-backend/internal/usecase/user"
)

type Server struct {
	handler http.Handler
}

func New(ctx context.Context, cfg config.Config) (*Server, error) {
	dbClient, err := mongoinfra.New(ctx, cfg.MongoURI, cfg.MongoDatabase)
	if err != nil {
		return nil, err
	}

	userRepo := mongoinfra.NewUserRepository(dbClient.DB())
	adRepo := mongoinfra.NewAdRepository(dbClient.DB())
	conversationRepo := mongoinfra.NewConversationRepository(dbClient.DB())
	messageRepo := mongoinfra.NewMessageRepository(dbClient.DB())
	revocationRepo := mongoinfra.NewTokenRevocationRepository(dbClient.DB())
	uploadRepo := mongoinfra.NewUploadRepository(dbClient.DB())

	ensureIndexes(ctx,
		userRepo.EnsureIndexes,
		adRepo.EnsureIndexes,
		conversationRepo.EnsureIndexes,
		messageRepo.EnsureIndexes,
		revocationRepo.EnsureIndexes,
		uploadRepo.EnsureIndexes,
	)

	fileStorage, err := buildStorage(ctx, cfg)
	if err != nil {
		return nil, err
	}

	moderator, err := buildModerator(ctx, cfg)
	if err != nil {
		return nil, err
	}

	tokenProvider := jwt.NewProvider(cfg.JWTSecret, revocationRepo)
	googleProvider := oauthinfra.NewGoogleProvider(cfg.GoogleClientID, cfg.GoogleClientSecret, cfg.GoogleRedirectURL)
	stateManager := oauthinfra.NewStateManager(cfg.OAuthStateSigningSecret)

	moderationService := moderationuc.NewService(moderator, uploadRepo, adRepo, userRepo)
	authService := authuc.NewService(userRepo, googleProvider, stateManager, tokenProvider, time.Duration(cfg.AccessTokenTTLMinutes)*time.Minute, time.Duration(cfg.RefreshTokenTTLHours)*time.Hour)
	userService := useruc.NewService(userRepo)
	adService := aduc.NewService(adRepo, uploadRepo)
	messageService := messageuc.NewService(conversationRepo, messageRepo, adRepo)
	uploadService := uploaduc.NewService(fileStorage, uploadRepo, moderationService, cfg.MaxUploadMB)

	authHandler := handler.NewAuthHandler(authService)
	userHandler := handler.NewUserHandler(userService)
	reportHandler := handler.NewReportHandler(adService)
	messageHandler := handler.NewMessageHandler(messageService)
	uploadHandler := handler.NewUploadHandler(uploadService)

	rateLimiter := middleware.NewRateLimiter(30, time.Minute)

	// auth wraps every authenticated route with token validation + ban check.
	auth := func(next http.Handler) http.Handler {
		return middleware.AuthRequired(tokenProvider, userRepo, next)
	}

	go runOrphanCleanup(uploadService)

	mux := http.NewServeMux()
	mux.Handle("GET /health", middleware.RequestLogger(http.HandlerFunc(handler.HealthCheck)))
	mux.HandleFunc("GET /swagger/", httpSwagger.WrapHandler)
	mux.Handle("GET /uploads/", http.StripPrefix("/uploads/", http.FileServer(http.Dir(cfg.UploadDir))))

	mux.Handle("POST /v1/auth/register", rateLimiter.Middleware(http.HandlerFunc(authHandler.Register)))
	mux.Handle("POST /v1/auth/login", rateLimiter.Middleware(http.HandlerFunc(authHandler.Login)))
	mux.Handle("GET /v1/auth/oauth/google/start", rateLimiter.Middleware(http.HandlerFunc(authHandler.StartGoogleOAuth)))
	mux.Handle("GET /v1/auth/oauth/google/callback", rateLimiter.Middleware(http.HandlerFunc(authHandler.GoogleCallback)))
	mux.Handle("POST /v1/auth/refresh", rateLimiter.Middleware(http.HandlerFunc(authHandler.Refresh)))
	mux.Handle("POST /v1/auth/logout", auth(rateLimiter.Middleware(http.HandlerFunc(authHandler.Logout))))

	mux.Handle("GET /v1/users/me", auth(http.HandlerFunc(userHandler.Me)))
	mux.Handle("PATCH /v1/users/me", auth(http.HandlerFunc(userHandler.UpdateMe)))
	mux.Handle("DELETE /v1/users/me", auth(http.HandlerFunc(userHandler.DeleteMe)))

	mux.Handle("POST /v1/uploads", auth(http.HandlerFunc(uploadHandler.Upload)))

	mux.Handle("POST /v1/reports", auth(http.HandlerFunc(reportHandler.Create)))
	mux.Handle("GET /v1/reports/{id}", auth(http.HandlerFunc(reportHandler.GetByID)))
	mux.Handle("GET /v1/reports", auth(http.HandlerFunc(reportHandler.List)))
	mux.Handle("PATCH /v1/reports/{id}", auth(http.HandlerFunc(reportHandler.Patch)))
	mux.Handle("POST /v1/reports/{id}/resolve", auth(http.HandlerFunc(reportHandler.Resolve)))
	mux.Handle("POST /v1/reports/{id}/archive", auth(http.HandlerFunc(reportHandler.Archive)))

	mux.Handle("POST /v1/reports/{id}/conversations", auth(http.HandlerFunc(messageHandler.StartConversation)))
	mux.Handle("GET /v1/conversations", auth(http.HandlerFunc(messageHandler.ListConversations)))
	mux.Handle("GET /v1/conversations/{id}/messages", auth(http.HandlerFunc(messageHandler.ListMessages)))
	mux.Handle("POST /v1/conversations/{id}/messages", auth(http.HandlerFunc(messageHandler.SendMessage)))

	return &Server{handler: middleware.Recovery(middleware.RequestLogger(mux))}, nil
}

func (s *Server) Routes() http.Handler {
	return s.handler
}

func ensureIndexes(ctx context.Context, fns ...func(context.Context) error) {
	for _, fn := range fns {
		if err := fn(ctx); err != nil {
			slog.Error("ensure indexes failed", "error", err)
		}
	}
}

// buildModerator selects the image moderation backend.
// local storage → noop (auto-approve)
// s3 → rekognition (requires -tags rekognition + AWS credentials)
func buildModerator(ctx context.Context, cfg config.Config) (moderationuc.Moderator, error) {
	if cfg.StorageProvider == "s3" {
		slog.Info("moderation provider: rekognition", "bucket", cfg.S3Bucket)
		return inframoderation.NewRekognitionModerator(ctx, inframoderation.RekognitionConfig{
			Region:          cfg.S3Region,
			AccessKeyID:     cfg.AWSAccessKeyID,
			SecretAccessKey: cfg.AWSSecretAccessKey,
			S3Bucket:        cfg.S3Bucket,
		})
	}
	slog.Info("moderation provider: noop (auto-approve)")
	return &inframoderation.NoOpModerator{}, nil
}

// buildStorage returns S3Storage when STORAGE_PROVIDER=s3, LocalStorage otherwise.
func buildStorage(ctx context.Context, cfg config.Config) (uploaduc.Storage, error) {
	if cfg.StorageProvider == "s3" {
		slog.Info("storage provider: s3", "bucket", cfg.S3Bucket, "region", cfg.S3Region)
		return storage.NewS3Storage(ctx, storage.S3Config{
			Bucket:          cfg.S3Bucket,
			Region:          cfg.S3Region,
			Endpoint:        cfg.S3Endpoint,
			AccessKeyID:     cfg.AWSAccessKeyID,
			SecretAccessKey: cfg.AWSSecretAccessKey,
			PublicURL:       cfg.UploadBaseURL,
		})
	}
	slog.Info("storage provider: local", "dir", cfg.UploadDir)
	return storage.NewLocalStorage(cfg.UploadDir, cfg.UploadBaseURL)
}

func runOrphanCleanup(svc *uploaduc.Service) {
	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()
	for range ticker.C {
		n, err := svc.CleanupOrphans(context.Background(), 24*time.Hour)
		if err != nil {
			slog.Error("orphan cleanup failed", "error", err)
		} else if n > 0 {
			slog.Info("orphan uploads removed", "count", n)
		}
	}
}
