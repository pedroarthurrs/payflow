package config

import (
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	GoEnv      string
	JwtSecret  string
	ServerPort string

	// Database connection
	DbUser     string
	DbPassword string
	DbHost     string
	DbPort     string
	DbName     string

	// Rate limiting — IP (public routes)
	RateLimitWindowSeconds int
	RateLimitMaxRequests   int

	// Rate limiting — Login
	RateLimitLoginWindowSeconds int
	RateLimitLoginMaxRequests   int

	// Rate limiting — Tenant (protected routes)
	RateLimitTenantWindowSeconds int
	RateLimitTenantMaxRequests   int
}

func Load() *Config {
	err := godotenv.Load()
	goEnv := os.Getenv("GO_ENV")

	if err != nil && goEnv != "production" {
		log.Fatalf("FATAL: .env required for development mode")
	}

	suffix := map[string]string{
		"development": "_DEV",
		"staging":     "_STAGING",
	}[goEnv]

	env := func(key, fallback string) string {
		if v := os.Getenv(key + suffix); v != "" {
			return v
		}
		return fallback
	}

	mustEnv := func(key string) string {
		k := key + suffix
		if v := os.Getenv(k); v != "" {
			return v
		}
		log.Fatalf("FATAL: required environment variable %s is missing", k)
		return ""
	}

	envInt := func(key string, fallback int) int {
		if v := os.Getenv(key); v != "" {
			if n, err := strconv.Atoi(v); err == nil {
				return n
			}
		}
		return fallback
	}

	return &Config{
		GoEnv:      goEnv,
		DbUser:     mustEnv("DB_USER"),
		DbPassword: mustEnv("DB_PASSWORD"),
		DbHost:     mustEnv("DB_HOST"),
		DbPort:     env("DB_PORT", "5432"),
		DbName:     mustEnv("DB_NAME"),
		ServerPort: env("SERVER_PORT", "8080"),
		JwtSecret:  mustEnv("JWT_SECRET"),

		RateLimitWindowSeconds:       envInt("RATE_LIMIT_WINDOW_SECONDS", 60),
		RateLimitMaxRequests:         envInt("RATE_LIMIT_MAX_REQUESTS", 100),
		RateLimitLoginWindowSeconds:  envInt("RATE_LIMIT_LOGIN_WINDOW_SECONDS", 300),
		RateLimitLoginMaxRequests:    envInt("RATE_LIMIT_LOGIN_MAX_REQUESTS", 5),
		RateLimitTenantWindowSeconds: envInt("RATE_LIMIT_TENANT_WINDOW_SECONDS", 60),
		RateLimitTenantMaxRequests:   envInt("RATE_LIMIT_TENANT_MAX_REQUESTS", 200),
	}
}
