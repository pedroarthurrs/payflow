package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	DbUser     string
	DbPassword string
	DbHost     string
	DbPort     string
	DbName     string
	ServerPort string
	GoEnv      string
	JwtSecret  string
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

	return &Config{
		GoEnv:      goEnv,
		DbUser:     mustEnv("DB_USER"),
		DbPassword: mustEnv("DB_PASSWORD"),
		DbHost:     mustEnv("DB_HOST"),
		DbPort:     env("DB_PORT", "5432"),
		DbName:     mustEnv("DB_NAME"),
		ServerPort: env("SERVER_PORT", "8080"),
		JwtSecret:  mustEnv("JWT_SECRET"),
	}
}
