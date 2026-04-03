package router

import (
	"log/slog"
	"time"

	"i9pay/internal/api/handler"
	"i9pay/internal/api/middleware"
	"i9pay/internal/auth/token"
	"i9pay/internal/config"
	"i9pay/internal/database"
	"i9pay/internal/monitor"
	"i9pay/internal/service"

	"github.com/gin-gonic/gin"
)

const accessTokenTTL = 15 * time.Minute
const refreshTokenTTL = 7 * 24 * time.Hour

// NewRouter constructs and returns a configured Gin engine with all routes and middleware registered.
func NewRouter(cfg *config.Config, logger *slog.Logger, db *database.DB, healthCheck *database.HealthChecker) *gin.Engine {
	r := gin.New()
	r.Use(middleware.Recovery())
	r.Use(monitor.Middleware())
	r.Use(middleware.RequestID())
	r.Use(middleware.StructuredLogger(logger))
	r.Use(middleware.Errors())

	// Rate limiter stores
	ipStore := middleware.NewRateLimiterStore()
	loginStore := middleware.NewRateLimiterStore()
	organizationStore := middleware.NewRateLimiterStore()

	// Rate limiter windows
	ipWindow := time.Duration(cfg.RateLimitWindowSeconds) * time.Second
	loginWindow := time.Duration(cfg.RateLimitLoginWindowSeconds) * time.Second
	organizationWindow := time.Duration(cfg.RateLimitOrganizationWindowSeconds) * time.Second

	// Services
	tokenSvc := token.NewService(cfg.JwtSecret, accessTokenTTL, refreshTokenTTL)
	authSvc := service.NewAuthService(tokenSvc, db)
	addressSvc := service.NewAddressService(db)
	customerSvc := service.NewCustomerService(db, addressSvc)
	userSvc := service.NewUserService(db)

	// Middleware de login rate limiting (injetado no AuthHandler)
	loginRL := middleware.RateLimitByIP(loginStore, cfg.RateLimitLoginMaxRequests, loginWindow, "too many login attempts, try again later")

	// Handlers
	rootHandler := handler.NewRootHandler()
	authHandler := handler.NewAuthHandler(authSvc, loginRL)
	healthHandler := handler.NewHealthHandler(healthCheck)
	meHandler := handler.NewMeHandler()
	customerHandler := handler.NewCustomerHandler(customerSvc)
	addressHandler := handler.NewAddressHandler(addressSvc)
	userHandler := handler.NewUserHandler(userSvc)

	// Rota raiz (fora do grupo de versão)
	rootHandler.RegisterRoutes(&r.RouterGroup)

	// Grupo de versão /v1
	v1 := r.Group("/v1")
	v1.Use(middleware.RateLimitByIP(ipStore, cfg.RateLimitMaxRequests, ipWindow, "too many requests, try again later"))

	// Rotas públicas
	healthHandler.RegisterRoutes(v1)
	authHandler.RegisterRoutes(v1)

	// Rotas protegidas
	protected := v1.Group("/")
	protected.Use(middleware.Auth(tokenSvc))
	protected.Use(middleware.RateLimitByOrganization(organizationStore, cfg.RateLimitOrganizationMaxRequests, organizationWindow))

	meHandler.RegisterRoutes(protected)
	customerHandler.RegisterRoutes(protected)
	addressHandler.RegisterRoutes(protected)
	userHandler.RegisterRoutes(protected)

	return r
}
