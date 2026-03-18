package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"i9pay/internal/database/dbgen"

	_ "github.com/lib/pq"
)

// DBConfig holds the parameters required to open a PostgreSQL connection.
type DBConfig struct {
	User     string
	Password string
	Host     string
	Port     string
	Name     string
}

// DB wraps *sql.DB and exposes type-safe SQLC queries via the Queries field.
type DB struct {
	*sql.DB
	Queries dbgen.Querier
}

// NewConnection opens and verifies a PostgreSQL connection using the provided config.
func NewConnection(cfg DBConfig) (*DB, error) {
	connStr := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
		cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.Name,
	)

	db, err := sql.Open("postgres", connStr)
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
