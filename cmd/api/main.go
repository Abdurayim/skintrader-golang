package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"

	"skintrader-go/internal/config"
	"skintrader-go/internal/handler"
	"skintrader-go/internal/middleware"
	"skintrader-go/internal/repository/cache"
	"skintrader-go/internal/repository/postgres"
	"skintrader-go/internal/router"
	"skintrader-go/internal/service"
	"skintrader-go/internal/worker"

	jwtpkg "skintrader-go/internal/pkg/jwt"
)

func main() {
	// Logger
	logger := setupLogger()

	// Config
	cfg, err := config.Load()
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to load config")
	}

	// Set Gin mode
	if cfg.Server.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	// PostgreSQL
	pool, err := setupDatabase(cfg, logger)
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to connect to database")
	}
	defer pool.Close()

	// DragonflyDB (Redis-compatible)
	rdb := setupRedis(cfg, logger)
	defer rdb.Close()

	// JWT Manager
	jwtManager := jwtpkg.NewManager(cfg.JWT.Secret, cfg.JWT.AccessTokenExpiry, cfg.JWT.RefreshTokenExpiry)

	// Repositories
	repos := setupRepositories(pool)

	// Cache
	cacheRepo := cache.NewRedisCache(rdb)

	// Services
	services := setupServices(cfg, repos, cacheRepo, jwtManager, logger)

	// Rate limiter
	rateLimiter := middleware.NewRateLimiter(rdb)

	// Auth middleware
	authMiddleware := middleware.NewAuthMiddleware(jwtManager, services.User, services.Admin)

	// Handlers
	handlers := setupHandlers(cfg, services, jwtManager, authMiddleware, logger)

	// Router
	r := gin.New()
	router.Setup(r, cfg, logger, handlers, &router.Dependencies{
		AuthMiddleware: authMiddleware,
		RateLimiter:    rateLimiter,
	})

	// Subscription cleanup worker
	cleanupWorker := worker.NewSubscriptionCleanup(services.Subscription, logger)
	cleanupWorker.Start()
	defer cleanupWorker.Stop()

	// Start server
	srv := &http.Server{
		Addr:         ":" + cfg.Server.Port,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Graceful shutdown
	go func() {
		logger.Info().Str("port", cfg.Server.Port).Msg("starting server")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal().Err(err).Msg("server failed")
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info().Msg("shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Fatal().Err(err).Msg("forced shutdown")
	}

	logger.Info().Msg("server stopped")
}

func setupLogger() zerolog.Logger {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	return zerolog.New(os.Stdout).With().Timestamp().Caller().Logger()
}

func setupDatabase(cfg *config.Config, logger zerolog.Logger) (*pgxpool.Pool, error) {
	ctx := context.Background()

	poolCfg, err := pgxpool.ParseConfig(cfg.Database.URL)
	if err != nil {
		return nil, fmt.Errorf("parse database config: %w", err)
	}

	poolCfg.MaxConns = cfg.Database.MaxConns
	poolCfg.MinConns = cfg.Database.MinConns
	poolCfg.MaxConnLifetime = cfg.Database.MaxConnLife
	poolCfg.MaxConnIdleTime = cfg.Database.MaxConnIdle
	poolCfg.HealthCheckPeriod = cfg.Database.HealthCheck

	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return nil, fmt.Errorf("create pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("ping database: %w", err)
	}

	logger.Info().Msg("connected to PostgreSQL")
	return pool, nil
}

func setupRedis(cfg *config.Config, logger zerolog.Logger) *redis.Client {
	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.Addr(),
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})

	ctx := context.Background()
	if err := rdb.Ping(ctx).Err(); err != nil {
		logger.Warn().Err(err).Msg("failed to connect to DragonflyDB/Redis - cache disabled")
	} else {
		logger.Info().Msg("connected to DragonflyDB")
	}

	return rdb
}

func setupRepositories(pool *pgxpool.Pool) *postgres.Repositories {
	return postgres.NewRepositories(pool)
}

func setupServices(cfg *config.Config, repos *postgres.Repositories, cacheRepo *cache.RedisCache, jwtManager *jwtpkg.Manager, logger zerolog.Logger) *service.Services {
	return service.NewServices(cfg, repos, cacheRepo, jwtManager, logger)
}

func setupHandlers(cfg *config.Config, services *service.Services, jwtManager *jwtpkg.Manager, authMiddleware *middleware.AuthMiddleware, logger zerolog.Logger) *router.Handlers {
	adminHandler := handler.NewAdminHandler(
		services.Admin,
		services.User,
		services.Post,
		services.Game,
		services.Subscription,
		services.Report,
		services.KYC,
		services.AdminLog,
		authMiddleware,
		logger,
	)
	adminHandler.SetJWTManager(jwtManager)
	adminHandler.SetTopupRepo(services.BalanceTopup)

	return &router.Handlers{
		Auth:         handler.NewAuthHandler(services.Auth, services.KYC, authMiddleware, logger),
		User:         handler.NewUserHandler(services.User, services.Post, services.Image, authMiddleware, logger),
		Post:         handler.NewPostHandler(services.Post, services.User, services.Game, services.Image, authMiddleware, logger),
		Game:         handler.NewGameHandler(services.Game, logger),
		Message:      handler.NewMessageHandler(services.Message, services.Conversation, services.User, authMiddleware, logger),
		Subscription: handler.NewSubscriptionHandler(services.Subscription, services.Payment, authMiddleware, logger),
		Balance:      handler.NewBalanceHandler(services.User, services.BalanceTopup, cfg.Payment.TopupCardNumber, logger),
		Payment:      handler.NewPaymentHandler(services.Payment, authMiddleware, logger),
		Report:       handler.NewReportHandler(services.Report, authMiddleware, logger),
		Admin:        adminHandler,
	}
}
