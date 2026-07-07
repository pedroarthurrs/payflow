package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"payflow/internal/config"
	addressdb "payflow/internal/database/dbgen/address"
	authdb "payflow/internal/database/dbgen/auth"
	credentialdb "payflow/internal/database/dbgen/credential"
	credentiallogdb "payflow/internal/database/dbgen/credentiallog"
	customerdb "payflow/internal/database/dbgen/customer"
	idempotencydb "payflow/internal/database/dbgen/idempotency"
	invoicedb "payflow/internal/database/dbgen/invoice"
	memberlogdb "payflow/internal/database/dbgen/memberlog"
	organizationdb "payflow/internal/database/dbgen/organization"
	paymentdb "payflow/internal/database/dbgen/payment"
	plandb "payflow/internal/database/dbgen/plan"
	productdb "payflow/internal/database/dbgen/product"
	recurrencedb "payflow/internal/database/dbgen/recurrence"
	subscriptiondb "payflow/internal/database/dbgen/subscription"
	walletdb "payflow/internal/database/dbgen/wallet"
	webhookdb "payflow/internal/database/dbgen/webhook"
	webhooklogdb "payflow/internal/database/dbgen/webhooklog"

	_ "github.com/lib/pq"
)

// DB wraps *sql.DB and exposes type-safe SQLC queries separated by domain.
type DB struct {
	*sql.DB
	CustomerQ      customerdb.Querier
	AddressQ       addressdb.Querier
	ProductQ       productdb.Querier
	AuthQ          authdb.Querier
	OrganizationQ  organizationdb.Querier
	CredentialQ    credentialdb.Querier
	CredentialLogQ credentiallogdb.Querier
	MemberLogQ     memberlogdb.Querier
	WebhookQ       webhookdb.Querier
	WebhookLogQ    webhooklogdb.Querier
	IdempotencyQ   idempotencydb.Querier
	PaymentQ       paymentdb.Querier
	PlanQ          plandb.Querier
	SubscriptionQ  subscriptiondb.Querier
	RecurrenceQ    recurrencedb.Querier
	WalletQ        walletdb.Querier
	InvoiceQ       invoicedb.Querier
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
		DB:             db,
		CustomerQ:      customerdb.New(db),
		AddressQ:       addressdb.New(db),
		ProductQ:       productdb.New(db),
		AuthQ:          authdb.New(db),
		OrganizationQ:  organizationdb.New(db),
		CredentialQ:    credentialdb.New(db),
		CredentialLogQ: credentiallogdb.New(db),
		MemberLogQ:     memberlogdb.New(db),
		WebhookQ:       webhookdb.New(db),
		WebhookLogQ:    webhooklogdb.New(db),
		IdempotencyQ:   idempotencydb.New(db),
		PaymentQ:       paymentdb.New(db),
		PlanQ:          plandb.New(db),
		SubscriptionQ:  subscriptiondb.New(db),
		RecurrenceQ:    recurrencedb.New(db),
		WalletQ:        walletdb.New(db),
		InvoiceQ:       invoicedb.New(db),
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
