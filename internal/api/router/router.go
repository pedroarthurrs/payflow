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
	"i9pay/internal/domain"
	"i9pay/internal/monitor"
	"i9pay/internal/service"

	"github.com/gin-gonic/gin"
)

const accessTokenTTL = 15 * time.Minute
const refreshTokenTTL = 7 * 24 * time.Hour

// Adapters holds the infrastructure adapters that are wired externally (in main)
// and injected into the router. Keeping them in a struct avoids a long parameter
// list and makes it easy to add new adapters without changing the function signature.
type Adapters struct {
	DB          *database.DB
	HealthCheck *database.HealthChecker
	Queue       domain.Queue
	Gateway     domain.Gateway
	Flags       config.FlagProvider
}

// NewRouter constructs and returns a configured Gin engine with all routes and middleware registered.
func NewRouter(cfg *config.Config, logger *slog.Logger, adapters Adapters) *gin.Engine {
	// Feature flags — MaintenanceProvider wraps the FlagProvider so the middleware
	// always reads the current value; Reload() on the provider propagates automatically.
	maintenanceProvider := middleware.NewMaintenanceProvider(adapters.Flags.IsEnabled(config.FlagMaintenance))

	// Stores
	ipStore := middleware.NewRateLimiterStore()
	loginStore := middleware.NewRateLimiterStore()
	organizationStore := middleware.NewRateLimiterStore()

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
	customerRepo := repository.NewCustomerRepo(adapters.DB.CustomerQ)
	addressRepo := repository.NewAddressRepo(adapters.DB.AddressQ)
	productRepo := repository.NewProductRepo(adapters.DB.ProductQ)
	userRepo := repository.NewUserRepo(adapters.DB.UserQ)
	authRepo := repository.NewAuthRepo(adapters.DB.AuthQ)
	credentialRepo := repository.NewCredentialRepo(adapters.DB.CredentialQ)
	credentialLogRepo := repository.NewCredentialLogRepo(adapters.DB.CredentialLogQ)
	webhookRepo := repository.NewWebhookRepo(adapters.DB.WebhookQ)
	webhookLogRepo := repository.NewWebhookLogRepo(adapters.DB.WebhookLogQ)
	orgSecretReader := repository.NewOrganizationSecretReader(adapters.DB.OrganizationQ)
	idempotencyRepo := repository.NewIdempotencyRepo(adapters.DB.IdempotencyQ)
	paymentRepo := repository.NewPaymentRepo(adapters.DB.PaymentQ)
	paymentFSMRepo := repository.NewPaymentFSMRepo(adapters.DB.PaymentQ)
	txManager := repository.NewTxManager(adapters.DB)
	planRepo := repository.NewPlanRepo(adapters.DB.PlanQ)
	subscriptionRepo := repository.NewSubscriptionRepo(adapters.DB.SubscriptionQ)
	subscriptionItemRepo := repository.NewSubscriptionItemRepo(adapters.DB.SubscriptionQ)
	subscriptionSchRepo := repository.NewSubscriptionScheduleRepo(adapters.DB.SubscriptionQ)
	subscriptionHistRepo := repository.NewSubscriptionHistoryRepo(adapters.DB.SubscriptionQ)

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
	dispatcher := service.NewWebhookDispatcher(webhookRepo, orgSecretReader, adapters.Queue)
	_ = dispatcher // TODO: inject into handler that triggers webhook events
	paymentSvc := service.NewPaymentService(paymentRepo)
	paymentFSMSvc := service.NewPaymentFSMService(txManager, paymentFSMRepo)
	gatewaySvc := service.NewGatewayService(paymentRepo, paymentFSMSvc, adapters.Gateway)
	planSvc := service.NewPlanService(planRepo)
	subscriptionSvc := service.NewSubscriptionService(
		subscriptionRepo,
		subscriptionItemRepo,
		subscriptionSchRepo,
		subscriptionHistRepo,
		planRepo,
		customerRepo,
		txManager,
	)

	// Middleware de login rate limiting (injetado no AuthHandler)
	loginRL := middleware.RateLimitByIP(loginStore, cfg.RateLimitLoginMaxRequests, loginWindow, "too many login attempts, try again later")

	// Handlers
	rootHandler := handler.NewRootHandler()
	authHandler := handler.NewAuthHandler(authSvc, loginRL)
	healthHandler := handler.NewHealthHandler(adapters.HealthCheck, maintenanceProvider)
	meHandler := handler.NewMeHandler()
	customerHandler := handler.NewCustomerHandler(customerSvc)
	addressHandler := handler.NewAddressHandler(addressSvc)
	productHandler := handler.NewProductHandler(productSvc)
	userHandler := handler.NewUserHandler(userSvc)
	credentialHandler := handler.NewCredentialHandler(credentialSvc, credentialLogSvc)
	webhookHandler := handler.NewWebhookHandler(webhookSvc)
	paymentHandler := handler.NewPaymentHandler(paymentSvc, paymentFSMSvc, gatewaySvc)
	planHandler := handler.NewPlanHandler(planSvc)
	subscriptionHandler := handler.NewSubscriptionHandler(subscriptionSvc)

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
	planHandler.RegisterRoutes(protected)
	subscriptionHandler.RegisterRoutes(protected)

	// SPA: serve o frontend embutido para todas as rotas não mapeadas pela API.
	r.NoRoute(SPAHandler())

	return r
}
