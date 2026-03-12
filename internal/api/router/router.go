package router

import (
	"i9pay/internal/api/handler"
	"i9pay/internal/api/middleware"

	"github.com/gin-gonic/gin"
)

func New(jwtSecret string) *gin.Engine {
	r := gin.Default()

	rootHandler := handler.NewRootHandler()
	loginHandler := handler.NewLoginHandler(jwtSecret)
	meHandler := handler.NewMeHandler()

	r.GET("/", rootHandler.Get)
	r.POST("/login", loginHandler.Post)

	// Protected routes — require a valid JWT
	protected := r.Group("/")
	protected.Use(middleware.Auth(jwtSecret))
	{
		protected.GET("/me", meHandler.Get)
	}

	return r
}
