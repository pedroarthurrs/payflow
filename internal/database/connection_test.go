package database

import (
	"testing"

	"payflow/internal/config"
)

func TestBuildDSN_WithDefaultDriver(t *testing.T) {
	cfg := &config.Config{
		DbUser:     "user",
		DbPassword: "pass",
		DbHost:     "localhost",
		DbPort:     "5432",
		DbName:     "db",
	}

	dsn, err := buildDSN(cfg, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := "postgres://user:pass@localhost:5432/db?sslmode=disable"
	if dsn != expected {
		t.Fatalf("expected %q, got %q", expected, dsn)
	}
}

func TestBuildDSN_WithDSNOverride(t *testing.T) {
	cfg := &config.Config{
		DbDSN:    "mysql://user:pass@tcp(localhost:3306)/db",
		DbDriver: "mysql",
	}

	dsn, err := buildDSN(cfg, "mysql")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if dsn != cfg.DbDSN {
		t.Fatalf("expected DSN override %q, got %q", cfg.DbDSN, dsn)
	}
}

func TestBuildDSN_UnsupportedDriver(t *testing.T) {
	cfg := &config.Config{}

	_, err := buildDSN(cfg, "sqlserver")
	if err == nil {
		t.Fatal("expected error for unsupported driver")
	}
}
