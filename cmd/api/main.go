package main

import (
	"i9pay/internal/api/router"
	"i9pay/internal/config"
	"i9pay/internal/database"
	"log/slog"
	"os"

	"github.com/gin-gonic/gin"
)

// SetupLogger creates an slog.Logger based on the environment.
// Production uses JSON output; all other environments use text output.
func SetupLogger(env string) *slog.Logger {
	if env == "production" {
		return slog.New(slog.NewJSONHandler(os.Stdout, nil))
	}
	return slog.New(slog.NewTextHandler(os.Stdout, nil))
}

func main() {
	cfg := config.Load()
	logger := SetupLogger(cfg.GoEnv)

	if cfg.GoEnv == "production" {
		gin.SetMode(gin.ReleaseMode)
	} else {
		gin.SetMode(gin.DebugMode)
	}

	db, err := database.NewConnection(database.DBConfig{
		User:     cfg.DbUser,
		Password: cfg.DbPassword,
		Host:     cfg.DbHost,
		Port:     cfg.DbPort,
		Name:     cfg.DbName,
	})
	if err != nil {
		logger.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	r := router.New(cfg, logger, db)

	r.Run(":" + cfg.ServerPort)
}
