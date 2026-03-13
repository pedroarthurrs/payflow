package router

import (
	"i9pay/internal/api/handler"
	"i9pay/internal/api/middleware"
	"i9pay/internal/database"
	i9token "i9pay/internal/token"
	"time"

	"github.com/gin-gonic/gin"
)

// accessTokenTTL is the lifetime of an access token.
// Short so that a compromised token has limited exposure.
const accessTokenTTL = 15 * time.Minute

// refreshTokenTTL is the lifetime of a refresh token.
// Longer-lived so clients can stay authenticated without re-entering credentials.
const refreshTokenTTL = 7 * 24 * time.Hour

func New(jwtSecret string, db *database.DB) *gin.Engine {
	r := gin.New()
	r.Use(middleware.Recovery())
	r.Use(gin.Logger())
	r.Use(middleware.Errors())

	tokenSvc := i9token.NewService(jwtSecret, accessTokenTTL, refreshTokenTTL)

	rootHandler := handler.NewRootHandler()
	loginHandler := handler.NewLoginHandler(tokenSvc)
	refreshHandler := handler.NewRefreshHandler(tokenSvc)
	healthHandler := handler.NewHealthHandler(db)
	meHandler := handler.NewMeHandler()

	r.GET("/", rootHandler.Get)

	v1 := r.Group("/v1")
	{
		v1.GET("/health", healthHandler.Get)

		// Auth routes — public, no token required.
		auth := v1.Group("/auth")
		{
			auth.POST("/login", loginHandler.Post)
			auth.POST("/refresh", refreshHandler.Post)
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
