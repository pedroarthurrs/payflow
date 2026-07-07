package router

import (
	"context"
	"log/slog"
	"time"

	"payflow/internal/api/handler"
	"payflow/internal/api/middleware"
	"payflow/internal/auth/token"
	"payflow/internal/config"
	"payflow/internal/database"
	"payflow/internal/database/repository"
	"payflow/internal/domain"
	"payflow/internal/monitor"
	"payflow/internal/service"
	"payflow/internal/worker"

	"github.com/gin-gonic/gin"
)

const accessTokenTTL = 15 * time.Minute
const refreshTokenTTL = 7 * 24 * time.Hour

// Adapters holds the infrastructure adapters that are wired externally (in main)
// and injected into the router. Keeping them in a struct avoids a long parameter
// list and makes it easy to add new adapters without changing the function signature.
type Adapters struct {
	DB              *database.DB
	HealthCheck     *database.HealthChecker
	Queue           domain.Queue
	RecurrenceQueue domain.Queue
	Gateway         domain.Gateway
	Flags           config.FlagProvider
	Context         context.Context
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
	r.Use(middleware.CORS(cfg.AllowedOrigins))
	r.Use(middleware.Recovery())
	r.Use(monitor.Middleware())
	r.Use(middleware.RequestID())
	r.Use(middleware.StructuredLogger(logger))
	r.Use(middleware.Maintenance(maintenanceProvider))
	r.Use(middleware.Errors())

	// Rate limiter windows
	ipWindow := config.RateLimitWindow
	loginWindow := config.RateLimitLoginWindow
	organizationWindow := config.RateLimitOrganizationWindow

	// Repositories
	customerRepo := repository.NewCustomerRepo(adapters.DB.CustomerQ)
	addressRepo := repository.NewAddressRepo(adapters.DB.AddressQ)
	productRepo := repository.NewProductRepo(adapters.DB.ProductQ)
	authRepo := repository.NewAuthRepo(adapters.DB.AuthQ, adapters.DB.DB)
	credentialRepo := repository.NewCredentialRepo(adapters.DB.CredentialQ)
	credentialLogRepo := repository.NewCredentialLogRepo(adapters.DB.CredentialLogQ)
	memberLogRepo := repository.NewMemberLogRepo(adapters.DB.MemberLogQ)
	webhookRepo := repository.NewWebhookRepo(adapters.DB.WebhookQ)
	webhookLogRepo := repository.NewWebhookLogRepo(adapters.DB.WebhookLogQ)
	orgSecretReader := repository.NewOrganizationSecretReader(adapters.DB.OrganizationQ)
	idempotencyRepo := repository.NewIdempotencyRepo(adapters.DB.IdempotencyQ)
	paymentRepo := repository.NewPaymentRepo(adapters.DB.PaymentQ)
	paymentFSMRepo := repository.NewPaymentFSMRepo(adapters.DB.PaymentQ)
	txManager := repository.NewTxManager(adapters.DB)
	walletRepo := repository.NewWalletRepo(adapters.DB.WalletQ)
	planRepo := repository.NewPlanRepo(adapters.DB.PlanQ)
	subscriptionRepo := repository.NewSubscriptionRepo(adapters.DB.SubscriptionQ)
	subscriptionItemRepo := repository.NewSubscriptionItemRepo(adapters.DB.SubscriptionQ)
	subscriptionSchRepo := repository.NewSubscriptionScheduleRepo(adapters.DB.SubscriptionQ)
	subscriptionHistRepo := repository.NewSubscriptionHistoryRepo(adapters.DB.SubscriptionQ)
	memberRepo := repository.NewMemberRepo(adapters.DB.DB)
	userAccountRepo := repository.NewUserAccountRepo(adapters.DB.DB)
	permissionRepo := repository.NewPermissionRepo(adapters.DB.DB)

	// Services
	tokenSvc := token.NewService(cfg.JwtSecret, accessTokenTTL, refreshTokenTTL)
	authSvc := service.NewAuthService(tokenSvc, authRepo)
	addressSvc := service.NewAddressService(customerRepo, addressRepo)
	customerSvc := service.NewCustomerService(customerRepo, addressRepo, txManager)
	productSvc := service.NewProductService(productRepo)
	credentialSvc := service.NewCredentialService(credentialRepo, credentialLogRepo, txManager, cfg.APIKeyPepper)
	webhookSvc := service.NewWebhookService(webhookRepo, webhookLogRepo)
	webhookScheduler := worker.NewWebhookScheduler(webhookRepo, orgSecretReader, adapters.Queue)
	ledgerSvc := service.NewLedgerService(walletRepo)
	walletSvc := service.NewWalletService(walletRepo, txManager)
	paymentSvc := service.NewPaymentService(paymentRepo, txManager, paymentFSMRepo, ledgerSvc)
	gatewaySvc := service.NewGatewayService(paymentSvc, adapters.Gateway)
	planSvc := service.NewPlanService(planRepo)
	subscriptionFSMRepo := repository.NewSubscriptionFSMRepo(adapters.DB.SubscriptionQ)
	subscriptionSvc := service.NewSubscriptionService(
		subscriptionRepo,
		subscriptionItemRepo,
		subscriptionSchRepo,
		subscriptionHistRepo,
		planRepo,
		customerRepo,
		txManager,
		subscriptionFSMRepo,
	)
	memberSvc := service.NewMemberServiceFull(memberRepo, userAccountRepo, memberLogRepo)
	permSvc := service.NewPermissionService(memberRepo, permissionRepo)

	// Invoice
	invoiceRepo := repository.NewInvoiceRepo(adapters.DB.InvoiceQ)
	invoiceItemRepo := repository.NewInvoiceItemRepo(adapters.DB.InvoiceQ)
	invoiceLogRepo := repository.NewInvoiceLogRepo(adapters.DB.InvoiceQ)
	invoiceSettingRepo := repository.NewInvoiceSettingRepo(adapters.DB.InvoiceQ)
	invoiceSvc := service.NewInvoiceService(
		invoiceRepo, invoiceItemRepo, invoiceLogRepo, invoiceSettingRepo,
		customerRepo, subscriptionRepo, subscriptionItemRepo, planRepo,
		txManager, webhookScheduler,
	)

	// Recurrence engine
	recurrenceRepo := repository.NewRecurrenceRepo(adapters.DB.DB, adapters.DB.RecurrenceQ)
	recurrenceSvc := service.NewRecurrenceService(
		subscriptionRepo,
		subscriptionItemRepo,
		subscriptionSchRepo,
		planRepo,
		recurrenceRepo,
		paymentRepo,
		gatewaySvc,
		invoiceSvc,
		txManager,
		config.RecurrenceMaxAttempts,
		subscriptionSvc,
	)
	recurrenceWorker := worker.NewRecurrenceWorker(adapters.RecurrenceQueue, recurrenceSvc, config.RecurrenceWorkerCount, config.RecurrenceMaxAttempts)
	recurrenceScheduler := worker.NewRecurrenceScheduler(recurrenceRepo, adapters.RecurrenceQueue, config.RecurrenceBatchSize, config.RecurrenceScanCron)
	go recurrenceWorker.Start(adapters.Context)
	go recurrenceScheduler.Start(adapters.Context)

	// Middleware de login rate limiting (injetado no AuthHandler)
	loginRL := middleware.RateLimitByIP(loginStore, config.RateLimitLoginMaxRequests, loginWindow, "too many login attempts, try again later")

	// Handlers
	rootHandler := handler.NewRootHandler()
	authHandler := handler.NewAuthHandler(authSvc, loginRL, cfg.IsProduction)
	healthHandler := handler.NewHealthHandler(adapters.HealthCheck, maintenanceProvider)
	meHandler := handler.NewMeHandler(permSvc)
	customerHandler := handler.NewCustomerHandler(customerSvc)
	addressHandler := handler.NewAddressHandler(addressSvc)
	productHandler := handler.NewProductHandler(productSvc)
	credentialHandler := handler.NewCredentialHandler(credentialSvc)
	webhookHandler := handler.NewWebhookHandler(webhookSvc)
	paymentHandler := handler.NewPaymentHandler(paymentSvc, gatewaySvc)
	planHandler := handler.NewPlanHandler(planSvc)
	subscriptionHandler := handler.NewSubscriptionHandler(subscriptionSvc)
	memberHandler := handler.NewMemberHandler(memberSvc, permSvc)
	walletHandler := handler.NewWalletHandler(walletSvc, permSvc)
	commissionHandler := handler.NewCommissionHandler(walletRepo, permSvc)
	invoiceHandler := handler.NewInvoiceHandler(invoiceSvc, permSvc)

	// Rota raiz (fora do grupo de versão)
	rootHandler.RegisterRoutes(&r.RouterGroup)

	// Grupo de versão /v1
	v1 := r.Group("/v1")
	v1.Use(middleware.RateLimitByIP(ipStore, config.RateLimitMaxRequests, ipWindow, "too many requests, try again later"))

	// Rotas públicas
	healthHandler.RegisterRoutes(v1)
	authHandler.RegisterRoutes(v1)

	// Rotas protegidas
	protected := v1.Group("/")
	protected.Use(middleware.Auth(tokenSvc))
	protected.Use(middleware.CSRFProtect(cfg.IsProduction))
	protected.Use(middleware.Idempotency(idempotencyRepo))
	protected.Use(middleware.CredentialLog(credentialLogRepo))
	protected.Use(middleware.MemberLog(memberLogRepo))
	protected.Use(middleware.RateLimitByOrganization(organizationStore, config.RateLimitOrganizationMaxRequests, organizationWindow))

	meHandler.RegisterRoutes(protected)
	customerHandler.RegisterRoutes(protected)
	addressHandler.RegisterRoutes(protected)
	productHandler.RegisterRoutes(protected)
	credentialHandler.RegisterRoutes(protected)
	webhookHandler.RegisterRoutes(protected)
	paymentHandler.RegisterRoutes(protected)
	planHandler.RegisterRoutes(protected)
	subscriptionHandler.RegisterRoutes(protected)
	memberHandler.RegisterRoutes(protected)
	walletHandler.RegisterRoutes(protected)
	commissionHandler.RegisterRoutes(protected)
	invoiceHandler.RegisterRoutes(protected)

	// SPA: serve o frontend embutido para todas as rotas não mapeadas pela API.
	r.NoRoute(SPAHandler())

	return r
}
