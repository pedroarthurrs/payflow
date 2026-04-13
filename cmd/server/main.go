package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"i9pay/internal/api/router"
	"i9pay/internal/config"
	"i9pay/internal/database"
	"i9pay/internal/database/repository"
	"i9pay/internal/domain"
	"i9pay/internal/gateway"
	"i9pay/internal/monitor"
	"i9pay/internal/queue"
	"i9pay/internal/worker"

	"github.com/gin-gonic/gin"
)

func setupLogger(cfg *config.Config) *slog.Logger {
	var logHandler slog.Handler
	if cfg.GoEnv == "production" {
		logHandler = slog.NewJSONHandler(os.Stdout, nil)
	} else {
		logHandler = slog.NewTextHandler(os.Stdout, nil)
	}
	return slog.New(monitor.NewLogHandler(logHandler))
}

func setupDatabase(cfg *config.Config, logger *slog.Logger) (*database.DB, *database.HealthChecker, error) {
	db, err := database.New(cfg)
	if err != nil {
		logger.Error("failed to connect to database", "error", err)
		monitor.Flush()
		return nil, nil, err
	}

	healthCheck := database.NewHealthChecker(db)
	healthCheck.Start()

	return db, healthCheck, nil
}

func setupInfrastructure(cfg *config.Config, logger *slog.Logger) (domain.Queue, domain.Gateway, config.FlagProvider) {
	jobQueue := queue.New("memory", cfg.WebhookQueueBufferSize)
	if jobQueue == nil {
		logger.Error("failed to initialize job queue")
		os.Exit(1)
	}

	paymentGateway := gateway.New(cfg.GatewayProviderName, cfg.GatewayProviderAPIKey)
	if paymentGateway == nil {
		logger.Warn("no payment gateway configured; gateway operations will be unavailable",
			"provider", cfg.GatewayProviderName)
	}

	featureFlags := config.NewEnvFlagProvider(
		config.FlagMaintenance,
	)

	return jobQueue, paymentGateway, featureFlags
}

func startWorkers(ctx context.Context, db *database.DB, jobQueue domain.Queue, cfg *config.Config) {
	webhookLogRepo := repository.NewWebhookLogRepo(db.WebhookLogQ)
	webhookWorker := worker.NewWebhookWorker(jobQueue, webhookLogRepo, &http.Client{Timeout: 30 * time.Second}, cfg.WebhookWorkerCount)
	go webhookWorker.Start(ctx)

	cleanupRepo := repository.NewCleanupRepo(db.CredentialLogQ, db.WebhookLogQ, db.IdempotencyQ)
	cleanupWorker := worker.NewCleanupWorker(cleanupRepo)
	go cleanupWorker.Start(ctx)
}

func run() error {
	cfg := config.Load()

	logger := setupLogger(cfg)

	monitor.Init(cfg)
	defer monitor.Flush()

	if cfg.GoEnv == "production" {
		gin.SetMode(gin.ReleaseMode)
	} else {
		gin.SetMode(gin.DebugMode)
	}

	db, healthCheck, err := setupDatabase(cfg, logger)
	if err != nil {
		return err
	}
	defer func() {
		if err := db.Close(); err != nil {
			logger.Warn("failed to close database connection", "error", err)
		}
	}()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	jobQueue, paymentGateway, featureFlags := setupInfrastructure(cfg, logger)

	startWorkers(ctx, db, jobQueue, cfg)

	r := router.NewRouter(cfg, logger, router.Adapters{
		DB:          db,
		HealthCheck: healthCheck,
		Queue:       jobQueue,
		Gateway:     paymentGateway,
		Flags:       featureFlags,
	})

	if err := r.Run(":" + cfg.ServerPort); err != nil {
		logger.Error("server stopped", "error", err)
		return err
	}

	return nil
}

func main() {
	if err := run(); err != nil {
		os.Exit(1)
	}
}
