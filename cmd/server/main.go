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
	"i9pay/internal/monitor"
	"i9pay/internal/queue"
	"i9pay/internal/worker"

	"github.com/gin-gonic/gin"
)

// SetupLogger creates an slog.Logger based on the environment, wrapped with
// Sentry integration so error-level records are captured as Sentry events.
func SetupLogger(env string) *slog.Logger {
	var handler slog.Handler
	if env == "production" {
		handler = slog.NewJSONHandler(os.Stdout, nil)
	} else {
		handler = slog.NewTextHandler(os.Stdout, nil)
	}
	return slog.New(monitor.NewLogHandler(handler))
}

func run() error {
	cfg := config.Load()
	logger := SetupLogger(cfg.GoEnv)
	monitor.Init(cfg)
	defer monitor.Flush()

	if cfg.GoEnv == "production" {
		gin.SetMode(gin.ReleaseMode)
	} else {
		gin.SetMode(gin.DebugMode)
	}

	db, err := database.New(cfg)
	if err != nil {
		logger.Error("failed to connect to database", "error", err)
		return err
	}
	defer func() {
		if err := db.Close(); err != nil {
			logger.Warn("failed to close database connection", "error", err)
		}
	}()

	healthCheck := database.NewHealthChecker(db)
	healthCheck.Start()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	q := queue.NewInMemoryQueue(cfg.WebhookQueueBufferSize)

	webhookLogRepo := repository.NewWebhookLogRepo(db.WebhookLogQ)
	wkr := worker.NewWebhookWorker(q, webhookLogRepo, &http.Client{Timeout: 30 * time.Second}, cfg.WebhookWorkerCount)
	go wkr.Start(ctx)

	r := router.NewRouter(cfg, logger, db, healthCheck, q)

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
