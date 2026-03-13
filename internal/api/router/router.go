package router

import (
	"i9pay/internal/api/http/auth"
	"i9pay/internal/api/http/health"
	"i9pay/internal/api/http/me"
	"i9pay/internal/api/http/root"
	"i9pay/internal/api/middleware"
	"i9pay/internal/database"
	"i9pay/internal/service"
	"i9pay/internal/token"
	"time"

	"github.com/gin-gonic/gin"
)

const accessTokenTTL = 15 * time.Minute
const refreshTokenTTL = 7 * 24 * time.Hour

func New(jwtSecret string, db *database.DB) *gin.Engine {
	r := gin.New()
	r.Use(middleware.Recovery())
	r.Use(gin.Logger())
	r.Use(middleware.Errors())

	// Services
	tokenSvc := token.NewService(jwtSecret, accessTokenTTL, refreshTokenTTL)
	authSvc := service.NewAuthService(tokenSvc)

	// Handlers
	rootHandler := root.NewHandler()
	authHandler := auth.NewHandler(authSvc)
	healthHandler := health.NewHandler(db)
	meHandler := me.NewHandler()

	r.GET("/", rootHandler.Get)

	v1 := r.Group("/v1")
	{
		v1.GET("/health", healthHandler.Get)

		// Auth routes — public, no token required.
		authGroup := v1.Group("/auth")
		{
			authGroup.POST("/login", authHandler.Login)
			authGroup.POST("/refresh", authHandler.Refresh)
		}

		// Protected routes — require a valid access token.
		protected := v1.Group("/")
		protected.Use(middleware.Auth(tokenSvc))
		{
			protected.GET("/me", meHandler.Get)
		}
	}

	return r
}
