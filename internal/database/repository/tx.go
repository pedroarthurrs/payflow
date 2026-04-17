package repository

import (
	"context"
	"log/slog"

	"i9pay/internal/database"
	addressdb "i9pay/internal/database/dbgen/address"
	credentialdb "i9pay/internal/database/dbgen/credential"
	customerdb "i9pay/internal/database/dbgen/customer"
	invoicedb "i9pay/internal/database/dbgen/invoice"
	paymentdb "i9pay/internal/database/dbgen/payment"
	subscriptiondb "i9pay/internal/database/dbgen/subscription"
	walletdb "i9pay/internal/database/dbgen/wallet"
	"i9pay/internal/domain"
)

// TxManager implements domain.TxManager using the underlying *sql.DB.
type TxManager struct {
	db *database.DB
}

// NewTxManager creates a TxManager.
func NewTxManager(db *database.DB) *TxManager {
	return &TxManager{db: db}
}

// WithTx executes fn inside a database transaction.
func (m *TxManager) WithTx(ctx context.Context, fn func(tx domain.TxHandle) error) error {
	sqlTx, err := m.db.BeginTx(ctx, nil)
	if err != nil {
		slog.Error("failed to begin transaction", "error", err)
		return domain.ErrInternal
	}
	defer func() { _ = sqlTx.Rollback() }()

	handle := &txHandle{
		customerQ:     customerdb.New(sqlTx),
		addressQ:      addressdb.New(sqlTx),
		credentialQ:   credentialdb.New(sqlTx),
		paymentQ:      paymentdb.New(sqlTx),
		subscriptionQ: subscriptiondb.New(sqlTx),
		walletQ:       walletdb.New(sqlTx),
		invoiceQ:      invoicedb.New(sqlTx),
	}

	if err := fn(handle); err != nil {
		return err
	}

	if err := sqlTx.Commit(); err != nil {
		slog.Error("failed to commit transaction", "error", err)
		return domain.ErrInternal
	}
	return nil
}

// txHandle provides scoped repositories within a transaction.
type txHandle struct {
	customerQ     *customerdb.Queries
	addressQ      *addressdb.Queries
	credentialQ   *credentialdb.Queries
	paymentQ      *paymentdb.Queries
	subscriptionQ *subscriptiondb.Queries
	walletQ       *walletdb.Queries
	invoiceQ      *invoicedb.Queries
}

func (h *txHandle) CustomerRepo() domain.CustomerRepository {
	return NewCustomerRepo(h.customerQ)
}

func (h *txHandle) AddressRepo() domain.AddressRepository {
	return NewAddressRepo(h.addressQ)
}

func (h *txHandle) CredentialRepo() domain.CredentialRepository {
	return NewCredentialRepo(h.credentialQ)
}

func (h *txHandle) PaymentFSMRepo() domain.PaymentFSMRepository {
	return NewPaymentFSMRepo(h.paymentQ)
}

func (h *txHandle) SubscriptionRepo() domain.SubscriptionRepository {
	return NewSubscriptionRepo(h.subscriptionQ)
}

func (h *txHandle) SubscriptionItemRepo() domain.SubscriptionItemRepository {
	return NewSubscriptionItemRepo(h.subscriptionQ)
}

func (h *txHandle) SubscriptionScheduleRepo() domain.SubscriptionScheduleRepository {
	return NewSubscriptionScheduleRepo(h.subscriptionQ)
}

func (h *txHandle) WalletRepo() domain.WalletRepository {
	return NewWalletRepo(h.walletQ)
}

func (h *txHandle) InvoiceRepo() domain.InvoiceRepository {
	return NewInvoiceRepo(h.invoiceQ)
}

func (h *txHandle) InvoiceItemRepo() domain.InvoiceItemRepository {
	return NewInvoiceItemRepo(h.invoiceQ)
}

var _ domain.TxManager = (*TxManager)(nil)
var _ domain.TxHandle = (*txHandle)(nil)
