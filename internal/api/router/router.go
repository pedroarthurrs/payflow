package router

import (
	"log/slog"
	"time"

	"i9pay/internal/api/handler"
	"i9pay/internal/api/middleware"
	"i9pay/internal/auth/token"
	"i9pay/internal/config"
	"i9pay/internal/database"
	"i9pay/internal/database/repository"
	"i9pay/internal/monitor"
	"i9pay/internal/queue"
	"i9pay/internal/service"

	"github.com/gin-gonic/gin"
)

const accessTokenTTL = 15 * time.Minute
const refreshTokenTTL = 7 * 24 * time.Hour

// NewRouter constructs and returns a configured Gin engine with all routes and middleware registered.
func NewRouter(cfg *config.Config, logger *slog.Logger, db *database.DB, healthCheck *database.HealthChecker, q queue.Queue) *gin.Engine {
	// Feature flags — backed by env vars by default; swap for AWS AppConfig or
	// any other store by passing a different config.FlagProvider implementation.
	flags := config.NewEnvFlagProvider(
		config.FlagMaintenance,
	)

	// Stores
	ipStore := middleware.NewRateLimiterStore()
	loginStore := middleware.NewRateLimiterStore()
	organizationStore := middleware.NewRateLimiterStore()
	maintenanceProvider := middleware.NewMaintenanceProvider(flags.IsEnabled(config.FlagMaintenance))

	r := gin.New()
	r.RedirectTrailingSlash = false
	r.RedirectFixedPath = false
	r.Use(middleware.Recovery())
	r.Use(monitor.Middleware())
	r.Use(middleware.RequestID())
	r.Use(middleware.StructuredLogger(logger))
	r.Use(middleware.Maintenance(maintenanceProvider))
	r.Use(middleware.Errors())

	// Rate limiter windows
	ipWindow := time.Duration(cfg.RateLimitWindowSeconds) * time.Second
	loginWindow := time.Duration(cfg.RateLimitLoginWindowSeconds) * time.Second
	organizationWindow := time.Duration(cfg.RateLimitOrganizationWindowSeconds) * time.Second

	// Repositories
	customerRepo := repository.NewCustomerRepo(db.CustomerQ)
	addressRepo := repository.NewAddressRepo(db.AddressQ)
	productRepo := repository.NewProductRepo(db.ProductQ)
	userRepo := repository.NewUserRepo(db.UserQ)
	authRepo := repository.NewAuthRepo(db.AuthQ)
	credentialRepo := repository.NewCredentialRepo(db.CredentialQ)
	credentialLogRepo := repository.NewCredentialLogRepo(db.CredentialLogQ)
	webhookRepo := repository.NewWebhookRepo(db.WebhookQ)
	webhookLogRepo := repository.NewWebhookLogRepo(db.WebhookLogQ)
	orgSecretReader := repository.NewOrganizationSecretReader(db.OrganizationQ)
	idempotencyRepo := repository.NewIdempotencyRepo(db.IdempotencyQ)
	paymentRepo := repository.NewPaymentRepo(db.PaymentQ)
	paymentFSMRepo := repository.NewPaymentFSMRepo(db.PaymentQ)
	txManager := repository.NewTxManager(db)

	// Services
	tokenSvc := token.NewService(cfg.JwtSecret, accessTokenTTL, refreshTokenTTL)
	authSvc := service.NewAuthService(tokenSvc, authRepo)
	addressSvc := service.NewAddressService(customerRepo, addressRepo)
	customerSvc := service.NewCustomerService(customerRepo, addressRepo, txManager)
	productSvc := service.NewProductService(productRepo)
	userSvc := service.NewUserService(userRepo)
	credentialSvc := service.NewCredentialService(credentialRepo, txManager, cfg.APIKeyPepper)
	credentialLogSvc := service.NewCredentialLogService(credentialLogRepo)
	webhookSvc := service.NewWebhookService(webhookRepo, webhookLogRepo)
	dispatcher := service.NewWebhookDispatcher(webhookRepo, orgSecretReader, q)
	_ = dispatcher // TODO: inject into handler that triggers webhook events
	paymentSvc := service.NewPaymentService(paymentRepo)
	paymentFSMSvc := service.NewPaymentFSMService(txManager, paymentFSMRepo)

	// Middleware de login rate limiting (injetado no AuthHandler)
	loginRL := middleware.RateLimitByIP(loginStore, cfg.RateLimitLoginMaxRequests, loginWindow, "too many login attempts, try again later")

	// Handlers
	rootHandler := handler.NewRootHandler()
	authHandler := handler.NewAuthHandler(authSvc, loginRL)
	healthHandler := handler.NewHealthHandler(healthCheck, maintenanceProvider)
	meHandler := handler.NewMeHandler()
	customerHandler := handler.NewCustomerHandler(customerSvc)
	addressHandler := handler.NewAddressHandler(addressSvc)
	productHandler := handler.NewProductHandler(productSvc)
	userHandler := handler.NewUserHandler(userSvc)
	credentialHandler := handler.NewCredentialHandler(credentialSvc, credentialLogSvc)
	webhookHandler := handler.NewWebhookHandler(webhookSvc)
	paymentHandler := handler.NewPaymentHandler(paymentSvc, paymentFSMSvc)

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
	protected.Use(middleware.Idempotency(idempotencyRepo))
	protected.Use(middleware.CredentialLog(credentialLogRepo))
	protected.Use(middleware.RateLimitByOrganization(organizationStore, cfg.RateLimitOrganizationMaxRequests, organizationWindow))

	meHandler.RegisterRoutes(protected)
	customerHandler.RegisterRoutes(protected)
	addressHandler.RegisterRoutes(protected)
	productHandler.RegisterRoutes(protected)
	userHandler.RegisterRoutes(protected)
	credentialHandler.RegisterRoutes(protected)
	webhookHandler.RegisterRoutes(protected)
	paymentHandler.RegisterRoutes(protected)

	// SPA: serve o frontend embutido para todas as rotas não mapeadas pela API.
	r.NoRoute(SPAHandler())

	return r
}
