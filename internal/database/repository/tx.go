package repository

import (
	"context"
	"log/slog"

	"i9pay/internal/database"
	"i9pay/internal/database/dbgen"
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

	qtx := m.db.Queries.(*dbgen.Queries).WithTx(sqlTx)
	handle := &txHandle{qtx: qtx}

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
	qtx *dbgen.Queries
}

func (h *txHandle) CustomerRepo() domain.CustomerRepository {
	return NewCustomerRepo(h.qtx)
}

func (h *txHandle) AddressRepo() domain.AddressRepository {
	return NewAddressRepo(h.qtx)
}

var _ domain.TxManager = (*TxManager)(nil)
