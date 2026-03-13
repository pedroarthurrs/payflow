package main

import (
	"i9pay/internal/api/router"
	"i9pay/internal/config"
	"i9pay/internal/database"
	"log"

	"github.com/gin-gonic/gin"
)

func main() {
	cfg := config.Load()

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
		log.Fatalf("FATAL: %v", err)
	}
	defer db.Close()

	r := router.New(cfg.JwtSecret, db)

	r.Run(":8080")
}
