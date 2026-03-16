package router

import (
	"i9pay/internal/api/http/address"
	"i9pay/internal/api/http/auth"
	"i9pay/internal/api/http/customer"
	"i9pay/internal/api/http/health"
	"i9pay/internal/api/http/me"
	"i9pay/internal/api/http/root"
	"i9pay/internal/api/middleware"
	"i9pay/internal/config"
	"i9pay/internal/database"
	"i9pay/internal/service"
	"i9pay/internal/token"
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"
)

const accessTokenTTL = 15 * time.Minute
const refreshTokenTTL = 7 * 24 * time.Hour

func New(cfg *config.Config, logger *slog.Logger, db *database.DB) *gin.Engine {
	r := gin.New()
	r.Use(middleware.Recovery())
	r.Use(middleware.RequestID())
	r.Use(middleware.StructuredLogger(logger))
	r.Use(middleware.Errors())

	// Rate limiter stores
	ipStore := middleware.NewRateLimiterStore()
	loginStore := middleware.NewRateLimiterStore()
	tenantStore := middleware.NewRateLimiterStore()

	// Rate limiter windows
	ipWindow := time.Duration(cfg.RateLimitWindowSeconds) * time.Second
	loginWindow := time.Duration(cfg.RateLimitLoginWindowSeconds) * time.Second
	tenantWindow := time.Duration(cfg.RateLimitTenantWindowSeconds) * time.Second

	// Services
	tokenSvc := token.NewService(cfg.JwtSecret, accessTokenTTL, refreshTokenTTL)
	authSvc := service.NewAuthService(tokenSvc)
	addressSvc := service.NewAddressService(db)
	customerSvc := service.NewCustomerService(db, addressSvc)

	// Handlers
	rootHandler := root.NewHandler()
	authHandler := auth.NewHandler(authSvc)
	healthHandler := health.NewHandler(db)
	meHandler := me.NewHandler()
	customerHandler := customer.NewHandler(customerSvc)
	addressHandler := address.NewHandler(addressSvc)

	r.GET("/", rootHandler.Get)

	v1 := r.Group("/v1")
	v1.Use(middleware.RateLimitByIP(ipStore, cfg.RateLimitMaxRequests, ipWindow, "too many requests, try again later"))
	{
		v1.GET("/health", healthHandler.Get)

		// Auth routes — public, no token required.
		authGroup := v1.Group("/auth")
		{
			authGroup.POST("/login", middleware.RateLimitByIP(loginStore, cfg.RateLimitLoginMaxRequests, loginWindow, "too many login attempts, try again later"), authHandler.Login)
			authGroup.POST("/refresh", authHandler.Refresh)
		}

		// Protected routes — require a valid access token.
		protected := v1.Group("/")
		protected.Use(middleware.Auth(tokenSvc))
		protected.Use(middleware.RateLimitByTenant(tenantStore, cfg.RateLimitTenantMaxRequests, tenantWindow))
		{
			protected.GET("/me", meHandler.Get)

			protected.POST("/customers", customerHandler.Create)
			protected.GET("/customers", customerHandler.List)
			protected.GET("/customers/:customer_id", customerHandler.GetByID)
			protected.PUT("/customers/:customer_id", customerHandler.Update)
			protected.DELETE("/customers/:customer_id", customerHandler.Delete)

			protected.POST("/customers/:customer_id/addresses", addressHandler.Create)
			protected.GET("/customers/:customer_id/addresses", addressHandler.List)
			protected.GET("/customers/:customer_id/addresses/:id", addressHandler.GetByID)
			protected.PUT("/customers/:customer_id/addresses/:id", addressHandler.Update)
			protected.DELETE("/customers/:customer_id/addresses/:id", addressHandler.Delete)
		}
	}

	return r
}
