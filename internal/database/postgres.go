package database

import (
	"database/sql"
	"fmt"
	"i9pay/internal/database/dbgen"

	_ "github.com/lib/pq"
)

type DBConfig struct {
	User     string
	Password string
	Host     string
	Port     string
	Name     string
}

type DB struct {
	*sql.DB
	Queries *dbgen.Queries
}

func NewConnection(cfg DBConfig) (*DB, error) {
	connStr := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
		cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.Name,
	)

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		return nil, err
	}

	return &DB{
		DB:      db,
		Queries: dbgen.New(db),
	}, nil
}
