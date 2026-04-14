package config

import (
	"os"
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

		APIKeyPepper: os.Getenv("API_KEY_PEPPER"),

		GatewayProviderAPIKey: os.Getenv("GATEWAY_PROVIDER_API_KEY"),
		GatewayProviderName:   os.Getenv("GATEWAY_PROVIDER_NAME"),
	}

	return cfg
}
