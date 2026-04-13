package config

import (
	"log/slog"
	"os"
	"strconv"
)

// Config holds all runtime configuration loaded from environment variables.
type Config struct {
	GoEnv      string
	JwtSecret  string
	ServerPort string
	SentryDSN  string

	// Database connection
	DbDriver   string
	DbDSN      string
	DbUser     string
	DbPassword string
	DbHost     string
	DbPort     string
	DbName     string
	DbSSLMode  string

	// Rate limiting — IP (public routes)
	RateLimitWindowSeconds int
	RateLimitMaxRequests   int

	// Rate limiting — Login
	RateLimitLoginWindowSeconds int
	RateLimitLoginMaxRequests   int

	// Rate limiting — Organization (protected routes)
	RateLimitOrganizationWindowSeconds int
	RateLimitOrganizationMaxRequests   int

	// API Keys
	APIKeyPepper string

	// Webhook worker queue
	WebhookQueueBufferSize int
	WebhookWorkerCount     int
	WebhookQueueDriver     string

	// Gateway provider
	GatewayProviderAPIKey string // GATEWAY_PROVIDER_API_KEY
	GatewayProviderName   string // GATEWAY_PROVIDER_NAME (e.g. "stripe")
}

// envInt reads an environment variable and converts it to int.
// Returns the fallback value if the variable is empty or not a valid integer.
func envInt(key string, fallback int) int {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return n
}

// Load reads environment variables and returns a populated Config.
func Load() *Config {
	cfg := &Config{
		GoEnv:      os.Getenv("GO_ENV"),
		ServerPort: os.Getenv("SERVER_PORT"),
		JwtSecret:  os.Getenv("JWT_SECRET"),
		SentryDSN:  os.Getenv("SENTRY_DSN"),
		DbDriver:   os.Getenv("DB_DRIVER"),
		DbDSN:      os.Getenv("DB_DSN"),
		DbUser:     os.Getenv("DB_USER"),
		DbPassword: os.Getenv("DB_PASSWORD"),
		DbHost:     os.Getenv("DB_HOST"),
		DbPort:     os.Getenv("DB_PORT"),
		DbName:     os.Getenv("DB_NAME"),
		DbSSLMode:  os.Getenv("DB_SSLMODE"),

		RateLimitWindowSeconds:             envInt("RATE_LIMIT_WINDOW_SECONDS", 60),
		RateLimitMaxRequests:               envInt("RATE_LIMIT_MAX_REQUESTS", 100),
		RateLimitLoginWindowSeconds:        envInt("RATE_LIMIT_LOGIN_WINDOW_SECONDS", 300),
		RateLimitLoginMaxRequests:          envInt("RATE_LIMIT_LOGIN_MAX_REQUESTS", 5),
		RateLimitOrganizationWindowSeconds: envInt("RATE_LIMIT_ORGANIZATION_WINDOW_SECONDS", 60),
		RateLimitOrganizationMaxRequests:   envInt("RATE_LIMIT_ORGANIZATION_MAX_REQUESTS", 200),

		APIKeyPepper: os.Getenv("API_KEY_PEPPER"),

		WebhookQueueBufferSize: envInt("WEBHOOK_QUEUE_BUFFER_SIZE", 1000),
		WebhookWorkerCount:     envInt("WEBHOOK_WORKER_COUNT", 5),
		WebhookQueueDriver:     os.Getenv("WEBHOOK_QUEUE_DRIVER"),

		GatewayProviderAPIKey: os.Getenv("GATEWAY_PROVIDER_API_KEY"),
		GatewayProviderName:   os.Getenv("GATEWAY_PROVIDER_NAME"),
	}

	if cfg.GatewayProviderAPIKey == "" {
		slog.Warn("GATEWAY_PROVIDER_API_KEY is empty; gateway operations will fail at runtime")
	}

	return cfg
}
