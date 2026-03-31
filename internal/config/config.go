package config

import (
	"os"
	"strconv"
)

// Config holds all runtime configuration loaded from environment variables.
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

	// Rate limiting — Organization (protected routes)
	RateLimitOrganizationWindowSeconds int
	RateLimitOrganizationMaxRequests   int
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
	return &Config{
		GoEnv:      os.Getenv("GO_ENV"),
		ServerPort: os.Getenv("SERVER_PORT"),
		JwtSecret:  os.Getenv("JWT_SECRET"),
		DbUser:     os.Getenv("DB_USER"),
		DbPassword: os.Getenv("DB_PASSWORD"),
		DbHost:     os.Getenv("DB_HOST"),
		DbPort:     os.Getenv("DB_PORT"),
		DbName:     os.Getenv("DB_NAME"),

		RateLimitWindowSeconds:             envInt("RATE_LIMIT_WINDOW_SECONDS", 60),
		RateLimitMaxRequests:               envInt("RATE_LIMIT_MAX_REQUESTS", 100),
		RateLimitLoginWindowSeconds:        envInt("RATE_LIMIT_LOGIN_WINDOW_SECONDS", 300),
		RateLimitLoginMaxRequests:          envInt("RATE_LIMIT_LOGIN_MAX_REQUESTS", 5),
		RateLimitOrganizationWindowSeconds: envInt("RATE_LIMIT_ORGANIZATION_WINDOW_SECONDS", 60),
		RateLimitOrganizationMaxRequests:   envInt("RATE_LIMIT_ORGANIZATION_MAX_REQUESTS", 200),
	}
}
