package router

import (
	"i9pay/internal/api/handler"
	"i9pay/internal/api/middleware"
	"i9pay/internal/database"

	"github.com/gin-gonic/gin"
)

func New(jwtSecret string, db *database.DB) *gin.Engine {
	r := gin.New()
	r.Use(middleware.Recovery())
	r.Use(gin.Logger())
	r.Use(middleware.Errors())

	rootHandler := handler.NewRootHandler()
	loginHandler := handler.NewLoginHandler(jwtSecret)
	healthHandler := handler.NewHealthHandler(db)
	meHandler := handler.NewMeHandler()

	r.GET("/", rootHandler.Get)
	r.GET("/health", healthHandler.Get)
	r.POST("/login", loginHandler.Post)

	// Protected routes — require a valid JWT
	protected := r.Group("/")
	protected.Use(middleware.Auth(jwtSecret))
	{
		protected.GET("/me", meHandler.Get)
	}

	return r
}
