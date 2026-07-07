package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"payflow/internal/api/router"
	"payflow/internal/config"
	"payflow/internal/database"
	"payflow/internal/database/repository"
	"payflow/internal/domain"
	"payflow/internal/gateway"
	"payflow/internal/monitor"
	"payflow/internal/queue"
	"payflow/internal/worker"

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

func setupInfrastructure(cfg *config.Config, logger *slog.Logger) (jobQueue, recurrenceQueue domain.Queue, paymentGateway domain.Gateway, featureFlags config.FlagProvider) {
	jobQueue = queue.New("memory", config.WebhookQueueBufferSize)
	if jobQueue == nil {
		logger.Error("failed to initialize job queue")
		os.Exit(1)
	}

	recurrenceQueue = queue.New("memory", config.RecurrenceQueueBufferSize)
	if recurrenceQueue == nil {
		logger.Error("failed to initialize recurrence queue")
		os.Exit(1)
	}

	paymentGateway = gateway.New(cfg.GatewayProviderName, cfg.GatewayProviderAPIKey)
	if paymentGateway == nil {
		logger.Warn("no payment gateway configured; gateway operations will be unavailable",
			"provider", cfg.GatewayProviderName)
	}

	featureFlags = config.NewEnvFlagProvider(
		config.FlagMaintenance,
	)

	return
}

func startWorkers(ctx context.Context, db *database.DB, jobQueue domain.Queue) {
	webhookLogRepo := repository.NewWebhookLogRepo(db.WebhookLogQ)
	webhookWorker := worker.NewWebhookWorker(jobQueue, webhookLogRepo, &http.Client{Timeout: 30 * time.Second}, config.WebhookWorkerCount)
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

	jobQueue, recurrenceQueue, paymentGateway, featureFlags := setupInfrastructure(cfg, logger)

	startWorkers(ctx, db, jobQueue)

	r := router.NewRouter(cfg, logger, router.Adapters{
		DB:              db,
		HealthCheck:     healthCheck,
		Queue:           jobQueue,
		RecurrenceQueue: recurrenceQueue,
		Gateway:         paymentGateway,
		Flags:           featureFlags,
		Context:         ctx,
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
