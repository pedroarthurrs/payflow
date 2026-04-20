package config

import (
	"os"
	"strings"
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

	// API Keys
	APIKeyPepper string

	// Gateway provider
	GatewayProviderAPIKey string
	GatewayProviderName   string // GATEWAY_PROVIDER_NAME (e.g. "stripe")

	// CORS — comma-separated list of allowed frontend origins.
	// Example: "https://app.i9pay.io,https://dashboard.i9pay.io"
	// In development, defaults to http://localhost:3000.
	AllowedOrigins []string

	// IsProduction is true when GO_ENV == "production".
	// It controls security flags such as Secure on cookies.
	IsProduction bool
}

// Load reads environment variables and returns a populated Config.
func Load() *Config {
	goEnv := os.Getenv("GO_ENV")
	isProd := goEnv == "production"

	allowedOrigins := parseOrigins(os.Getenv("ALLOWED_ORIGINS"))
	if len(allowedOrigins) == 0 {
		// Default: allow local Vite dev server in non-production environments.
		if !isProd {
			allowedOrigins = []string{"http://localhost:3000"}
		}
	}

	cfg := &Config{
		GoEnv:      goEnv,
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

		APIKeyPepper: os.Getenv("API_KEY_PEPPER"),

		GatewayProviderAPIKey: os.Getenv("GATEWAY_PROVIDER_API_KEY"),
		GatewayProviderName:   os.Getenv("GATEWAY_PROVIDER_NAME"),

		AllowedOrigins: allowedOrigins,
		IsProduction:   isProd,
	}

	return cfg
}

// parseOrigins splits a comma-separated string of origins and trims whitespace.
func parseOrigins(raw string) []string {
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if t := strings.TrimSpace(p); t != "" {
			out = append(out, t)
		}
	}
	return out
}
