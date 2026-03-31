package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"i9pay/internal/config"
	"i9pay/internal/database/dbgen"

	_ "github.com/lib/pq"
)

// DB wraps *sql.DB and exposes type-safe SQLC queries via the Queries field.
type DB struct {
	*sql.DB
	Queries dbgen.Querier
}

// New opens and verifies a database connection using the provided application config.
// It supports a driver from configuration and an optional DSN override.
func New(cfg *config.Config) (*DB, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config required")
	}

	driver := cfg.DbDriver

	connStr, err := buildDSN(cfg, driver)
	if err != nil {
		return nil, err
	}

	db, err := sql.Open(driver, connStr)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		return nil, err
	}

	return &DB{
		DB:      db,
		Queries: dbgen.New(db),
	}, nil
}

func buildDSN(cfg *config.Config, driver string) (string, error) {
	if driver == "" {
		driver = "postgres"
	}

	if cfg.DbDSN != "" {
		return cfg.DbDSN, nil
	}

	switch driver {
	case "postgres", "pgx":
		sslmode := cfg.DbSSLMode

		if sslmode == "" {
			sslmode = "disable"
		}

		return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s",
			cfg.DbUser, cfg.DbPassword, cfg.DbHost, cfg.DbPort, cfg.DbName, sslmode,
		), nil
	default:
		return "", fmt.Errorf("unsupported db driver: %q", driver)
	}
}
