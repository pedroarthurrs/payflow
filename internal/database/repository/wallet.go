package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"log/slog"

	walletdb "i9pay/internal/database/dbgen/wallet"
	"i9pay/internal/domain"

	"github.com/google/uuid"
	"github.com/sqlc-dev/pqtype"
)

// WalletRepo implements domain.WalletRepository backed by PostgreSQL via sqlc.
type WalletRepo struct {
	q walletdb.Querier
}

// NewWalletRepo creates a WalletRepo.
func NewWalletRepo(q walletdb.Querier) *WalletRepo {
	return &WalletRepo{q: q}
}

// CreateWallet inserts a new wallet for the given organization.
func (r *WalletRepo) CreateWallet(ctx context.Context, organizationID uuid.UUID, input domain.CreateWalletInput) (*domain.Wallet, error) {
	currency := input.Currency
	if currency == "" {
		currency = "BRL"
	}
	row, err := r.q.CreateWallet(ctx, walletdb.CreateWalletParams{
		OrganizationID:      organizationID,
		OrganizationOwnerID: toNullUUID(input.OrganizationOwnerID),
		CustomerOwnerID:     toNullUUID(input.CustomerOwnerID),
		Currency:            sql.NullString{String: currency, Valid: true},
	})
	if err != nil {
		slog.Error("failed to create wallet", "organization_id", organizationID, "error", err)
		return nil, domain.ErrInternal
	}
	w := toDomainWallet(row)
	return &w, nil
}

// GetWalletByID retrieves a wallet by ID scoped to an organization.
func (r *WalletRepo) GetWalletByID(ctx context.Context, organizationID, walletID uuid.UUID) (*domain.Wallet, error) {
	row, err := r.q.GetWalletByID(ctx, walletdb.GetWalletByIDParams{
		OrganizationID: organizationID,
		ID:             walletID,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrWalletNotFound
		}
		slog.Error("failed to get wallet by id", "wallet_id", walletID, "error", err)
		return nil, domain.ErrInternal
	}
	w := toDomainWallet(row)
	return &w, nil
}

// GetWalletByOrganizationOwner retrieves the org-owned wallet for an organization.
func (r *WalletRepo) GetWalletByOrganizationOwner(ctx context.Context, organizationID uuid.UUID) (*domain.Wallet, error) {
	row, err := r.q.GetWalletByOrganizationOwner(ctx, organizationID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrWalletNotFound
		}
		slog.Error("failed to get wallet by org owner", "organization_id", organizationID, "error", err)
		return nil, domain.ErrInternal
	}
	w := toDomainWallet(row)
	return &w, nil
}

// GetWalletByCustomerOwner retrieves the customer-owned wallet.
func (r *WalletRepo) GetWalletByCustomerOwner(ctx context.Context, organizationID, customerID uuid.UUID) (*domain.Wallet, error) {
	row, err := r.q.GetWalletByCustomerOwner(ctx, walletdb.GetWalletByCustomerOwnerParams{
		OrganizationID:  organizationID,
		CustomerOwnerID: uuid.NullUUID{UUID: customerID, Valid: true},
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrWalletNotFound
		}
		slog.Error("failed to get wallet by customer owner", "customer_id", customerID, "error", err)
		return nil, domain.ErrInternal
	}
	w := toDomainWallet(row)
	return &w, nil
}

// GetWalletForUpdate retrieves a wallet with SELECT FOR UPDATE for use inside a transaction.
func (r *WalletRepo) GetWalletForUpdate(ctx context.Context, organizationID, walletID uuid.UUID) (*domain.Wallet, error) {
	row, err := r.q.GetWalletForUpdate(ctx, walletdb.GetWalletForUpdateParams{
		OrganizationID: organizationID,
		ID:             walletID,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrWalletNotFound
		}
		slog.Error("failed to get wallet for update", "wallet_id", walletID, "error", err)
		return nil, domain.ErrInternal
	}
	w := toDomainWallet(row)
	return &w, nil
}

// UpdateWalletBalance sets the wallet balance to newBalance.
func (r *WalletRepo) UpdateWalletBalance(ctx context.Context, organizationID, walletID uuid.UUID, newBalance int32) (*domain.Wallet, error) {
	row, err := r.q.UpdateWalletBalance(ctx, walletdb.UpdateWalletBalanceParams{
		OrganizationID: organizationID,
		ID:             walletID,
		Balance:        newBalance,
	})
	if err != nil {
		slog.Error("failed to update wallet balance", "wallet_id", walletID, "error", err)
		return nil, domain.ErrInternal
	}
	w := toDomainWallet(row)
	return &w, nil
}

// UpdateWalletLockedBalance sets the wallet locked_balance to newLockedBalance.
func (r *WalletRepo) UpdateWalletLockedBalance(ctx context.Context, organizationID, walletID uuid.UUID, newLockedBalance int32) (*domain.Wallet, error) {
	row, err := r.q.UpdateWalletLockedBalance(ctx, walletdb.UpdateWalletLockedBalanceParams{
		OrganizationID: organizationID,
		ID:             walletID,
		LockedBalance:  newLockedBalance,
	})
	if err != nil {
		slog.Error("failed to update wallet locked balance", "wallet_id", walletID, "error", err)
		return nil, domain.ErrInternal
	}
	w := toDomainWallet(row)
	return &w, nil
}

// InsertLedgerEntry persists an immutable ledger entry.
func (r *WalletRepo) InsertLedgerEntry(ctx context.Context, organizationID uuid.UUID, input domain.LedgerEntryInput) (*domain.LedgerEntry, error) {
	row, err := r.q.InsertLedgerEntry(ctx, walletdb.InsertLedgerEntryParams{
		OrganizationID: organizationID,
		WalletID:       input.WalletID,
		Type:           string(input.Type),
		AmountCents:    input.AmountCents,
		Category:       string(input.Category),
		BalanceAfter:   input.BalanceAfter,
		SubscriptionID: toNullUUID(input.SubscriptionID),
		InvoiceID:      toNullUUID(input.InvoiceID),
		PaymentID:      toNullUUID(input.PaymentID),
		Metadata:       pqtype.NullRawMessage{RawMessage: input.Metadata, Valid: input.Metadata != nil},
	})
	if err != nil {
		slog.Error("failed to insert ledger entry", "wallet_id", input.WalletID, "error", err)
		return nil, domain.ErrInternal
	}
	e := toDomainLedgerEntry(row)
	return &e, nil
}

// ListLedgerEntries returns paginated ledger entries with optional filters.
func (r *WalletRepo) ListLedgerEntries(ctx context.Context, organizationID, walletID uuid.UUID, params domain.LedgerListParams) ([]domain.LedgerEntry, error) {
	rows, err := r.q.ListLedgerEntries(ctx, walletdb.ListLedgerEntriesParams{
		OrganizationID: organizationID,
		WalletID:       walletID,
		Category:       toNullString((*string)(params.Category)),
		Type:           toNullString((*string)(params.Type)),
		CreatedFrom:    toNullTime(params.CreatedFrom),
		CreatedTo:      toNullTime(params.CreatedTo),
		Limit:          params.Limit,
		Offset:         params.Offset,
	})
	if err != nil {
		slog.Error("failed to list ledger entries", "wallet_id", walletID, "error", err)
		return nil, domain.ErrInternal
	}
	entries := make([]domain.LedgerEntry, len(rows))
	for i := range rows {
		entries[i] = toDomainLedgerEntry(rows[i])
	}
	return entries, nil
}

// CountLedgerEntries returns the total count of ledger entries matching the filters.
func (r *WalletRepo) CountLedgerEntries(ctx context.Context, organizationID, walletID uuid.UUID, params domain.LedgerListParams) (int64, error) {
	count, err := r.q.CountLedgerEntries(ctx, walletdb.CountLedgerEntriesParams{
		OrganizationID: organizationID,
		WalletID:       walletID,
		Category:       toNullString((*string)(params.Category)),
		Type:           toNullString((*string)(params.Type)),
		CreatedFrom:    toNullTime(params.CreatedFrom),
		CreatedTo:      toNullTime(params.CreatedTo),
	})
	if err != nil {
		slog.Error("failed to count ledger entries", "wallet_id", walletID, "error", err)
		return 0, domain.ErrInternal
	}
	return count, nil
}

// CreatePayout inserts a new payout record with status=pending.
func (r *WalletRepo) CreatePayout(ctx context.Context, organizationID uuid.UUID, input domain.CreatePayoutInput) (*domain.Payout, error) {
	row, err := r.q.CreatePayout(ctx, walletdb.CreatePayoutParams{
		OrganizationID: organizationID,
		WalletID:       input.WalletID,
		AmountCents:    input.AmountCents,
		FeeCents:       sql.NullInt32{Int32: input.FeeCents, Valid: true},
		BankInfo:       pqtype.NullRawMessage{RawMessage: input.BankInfo, Valid: input.BankInfo != nil},
	})
	if err != nil {
		slog.Error("failed to create payout", "wallet_id", input.WalletID, "error", err)
		return nil, domain.ErrInternal
	}
	p := toDomainPayout(row)
	return &p, nil
}

// GetPayoutByID retrieves a payout by ID scoped to an organization.
func (r *WalletRepo) GetPayoutByID(ctx context.Context, organizationID, payoutID uuid.UUID) (*domain.Payout, error) {
	row, err := r.q.GetPayoutByID(ctx, walletdb.GetPayoutByIDParams{
		OrganizationID: organizationID,
		ID:             payoutID,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrPayoutNotFound
		}
		slog.Error("failed to get payout by id", "payout_id", payoutID, "error", err)
		return nil, domain.ErrInternal
	}
	p := toDomainPayout(row)
	return &p, nil
}

// UpdatePayoutStatus transitions a payout to the given status.
func (r *WalletRepo) UpdatePayoutStatus(ctx context.Context, organizationID, payoutID uuid.UUID, status domain.PayoutStatus) (*domain.Payout, error) {
	row, err := r.q.UpdatePayoutStatus(ctx, walletdb.UpdatePayoutStatusParams{
		OrganizationID: organizationID,
		ID:             payoutID,
		Status:         sql.NullString{String: string(status), Valid: true},
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrPayoutNotFound
		}
		slog.Error("failed to update payout status", "payout_id", payoutID, "error", err)
		return nil, domain.ErrInternal
	}
	p := toDomainPayout(row)
	return &p, nil
}

// ListPayouts returns paginated payouts for a wallet.
func (r *WalletRepo) ListPayouts(ctx context.Context, organizationID, walletID uuid.UUID, params domain.ListParams) ([]domain.Payout, error) {
	rows, err := r.q.ListPayouts(ctx, walletdb.ListPayoutsParams{
		OrganizationID: organizationID,
		WalletID:       walletID,
		Limit:          params.Limit,
		Offset:         params.Offset,
	})
	if err != nil {
		slog.Error("failed to list payouts", "wallet_id", walletID, "error", err)
		return nil, domain.ErrInternal
	}
	payouts := make([]domain.Payout, len(rows))
	for i := range rows {
		payouts[i] = toDomainPayout(rows[i])
	}
	return payouts, nil
}

// UpsertCommissionSetting creates or updates a commission setting for a customer.
func (r *WalletRepo) UpsertCommissionSetting(ctx context.Context, organizationID uuid.UUID, input domain.CommissionSetting) (*domain.CommissionSetting, error) {
	row, err := r.q.UpsertCommissionSetting(ctx, walletdb.UpsertCommissionSettingParams{
		OrganizationID:   organizationID,
		TargetCustomerID: input.TargetCustomerID,
		Type:             string(input.Type),
		Value:            formatDecimal(input.Value),
		Conditions:       pqtype.NullRawMessage{RawMessage: input.Conditions, Valid: input.Conditions != nil},
		IsActive:         sql.NullBool{Bool: input.IsActive, Valid: true},
	})
	if err != nil {
		slog.Error("failed to upsert commission setting", "customer_id", input.TargetCustomerID, "error", err)
		return nil, domain.ErrInternal
	}
	cs := toDomainCommissionSetting(row)
	return &cs, nil
}

// GetCommissionSettingByCustomer retrieves the active commission setting for a customer.
func (r *WalletRepo) GetCommissionSettingByCustomer(ctx context.Context, organizationID, customerID uuid.UUID) (*domain.CommissionSetting, error) {
	row, err := r.q.GetCommissionSettingByCustomer(ctx, walletdb.GetCommissionSettingByCustomerParams{
		OrganizationID:   organizationID,
		TargetCustomerID: customerID,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrCommissionNotFound
		}
		slog.Error("failed to get commission setting", "customer_id", customerID, "error", err)
		return nil, domain.ErrInternal
	}
	cs := toDomainCommissionSetting(row)
	return &cs, nil
}

// ListCommissionSettings returns paginated commission settings for an organization.
func (r *WalletRepo) ListCommissionSettings(ctx context.Context, organizationID uuid.UUID, params domain.ListParams) ([]domain.CommissionSetting, error) {
	rows, err := r.q.ListCommissionSettings(ctx, walletdb.ListCommissionSettingsParams{
		OrganizationID: organizationID,
		Limit:          params.Limit,
		Offset:         params.Offset,
	})
	if err != nil {
		slog.Error("failed to list commission settings", "organization_id", organizationID, "error", err)
		return nil, domain.ErrInternal
	}
	settings := make([]domain.CommissionSetting, len(rows))
	for i := range rows {
		settings[i] = toDomainCommissionSetting(rows[i])
	}
	return settings, nil
}

// SoftDeleteCommissionSetting marks a commission setting as deleted.
func (r *WalletRepo) SoftDeleteCommissionSetting(ctx context.Context, organizationID, settingID uuid.UUID) error {
	err := r.q.SoftDeleteCommissionSetting(ctx, walletdb.SoftDeleteCommissionSettingParams{
		OrganizationID:   organizationID,
		TargetCustomerID: settingID,
	})
	if err != nil {
		slog.Error("failed to soft delete commission setting", "setting_id", settingID, "error", err)
		return domain.ErrInternal
	}
	return nil
}

// CreateWalletLock creates an active balance lock (dispute or payout).
func (r *WalletRepo) CreateWalletLock(ctx context.Context, organizationID uuid.UUID, input domain.CreateWalletLockInput) (*domain.WalletLock, error) {
	row, err := r.q.CreateWalletLock(ctx, walletdb.CreateWalletLockParams{
		OrganizationID: organizationID,
		WalletID:       input.WalletID,
		Type:           string(input.Type),
		ReferenceID:    input.ReferenceID,
		AmountCents:    input.AmountCents,
	})
	if err != nil {
		slog.Error("failed to create wallet lock", "wallet_id", input.WalletID, "error", err)
		return nil, domain.ErrInternal
	}
	l := toDomainWalletLock(row)
	return &l, nil
}

// ReleaseWalletLock marks a wallet lock as released.
func (r *WalletRepo) ReleaseWalletLock(ctx context.Context, organizationID, lockID uuid.UUID) error {
	err := r.q.ReleaseWalletLock(ctx, walletdb.ReleaseWalletLockParams{
		OrganizationID: organizationID,
		ID:             lockID,
	})
	if err != nil {
		slog.Error("failed to release wallet lock", "lock_id", lockID, "error", err)
		return domain.ErrInternal
	}
	return nil
}

// GetWalletLockByReference retrieves an active lock by reference ID and type.
func (r *WalletRepo) GetWalletLockByReference(ctx context.Context, organizationID, referenceID uuid.UUID, lockType domain.WalletLockType) (*domain.WalletLock, error) {
	row, err := r.q.GetWalletLockByReference(ctx, walletdb.GetWalletLockByReferenceParams{
		OrganizationID: organizationID,
		ReferenceID:    referenceID,
		Type:           string(lockType),
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrWalletLockNotFound
		}
		slog.Error("failed to get wallet lock by reference", "reference_id", referenceID, "error", err)
		return nil, domain.ErrInternal
	}
	l := toDomainWalletLock(row)
	return &l, nil
}

// --- mappers ---

func toDomainWallet(row walletdb.Wallet) domain.Wallet {
	w := domain.Wallet{
		ID:             row.ID,
		OrganizationID: row.OrganizationID,
		Balance:        row.Balance,
		LockedBalance:  row.LockedBalance,
	}
	if row.OrganizationOwnerID.Valid {
		w.OrganizationOwnerID = &row.OrganizationOwnerID.UUID
	}
	if row.CustomerOwnerID.Valid {
		w.CustomerOwnerID = &row.CustomerOwnerID.UUID
	}
	if row.Currency.Valid {
		w.Currency = row.Currency.String
	}
	if row.Status.Valid {
		w.Status = domain.WalletStatus(row.Status.String)
	}
	if row.CreatedAt.Valid {
		w.CreatedAt = row.CreatedAt.Time
	}
	if row.UpdatedAt.Valid {
		w.UpdatedAt = row.UpdatedAt.Time
	}
	return w
}

func toDomainLedgerEntry(row walletdb.LedgerEntry) domain.LedgerEntry {
	e := domain.LedgerEntry{
		ID:             row.ID,
		OrganizationID: row.OrganizationID,
		WalletID:       row.WalletID,
		Type:           domain.LedgerEntryType(row.Type),
		AmountCents:    row.AmountCents,
		Category:       domain.LedgerCategory(row.Category),
		BalanceAfter:   row.BalanceAfter,
	}
	if row.SubscriptionID.Valid {
		e.SubscriptionID = &row.SubscriptionID.UUID
	}
	if row.InvoiceID.Valid {
		e.InvoiceID = &row.InvoiceID.UUID
	}
	if row.PaymentID.Valid {
		e.PaymentID = &row.PaymentID.UUID
	}
	if row.Metadata.Valid {
		e.Metadata = json.RawMessage(row.Metadata.RawMessage)
	}
	if row.CreatedAt.Valid {
		e.CreatedAt = row.CreatedAt.Time
	}
	return e
}

func toDomainPayout(row walletdb.Payout) domain.Payout {
	p := domain.Payout{
		ID:             row.ID,
		OrganizationID: row.OrganizationID,
		WalletID:       row.WalletID,
		AmountCents:    row.AmountCents,
	}
	if row.FeeCents.Valid {
		p.FeeCents = row.FeeCents.Int32
	}
	if row.Status.Valid {
		p.Status = domain.PayoutStatus(row.Status.String)
	}
	if row.BankInfo.Valid {
		p.BankInfo = json.RawMessage(row.BankInfo.RawMessage)
	}
	if row.CreatedAt.Valid {
		p.CreatedAt = row.CreatedAt.Time
	}
	if row.UpdatedAt.Valid {
		p.UpdatedAt = row.UpdatedAt.Time
	}
	return p
}

func toDomainCommissionSetting(row walletdb.CommissionSetting) domain.CommissionSetting {
	cs := domain.CommissionSetting{
		ID:               row.ID,
		OrganizationID:   row.OrganizationID,
		TargetCustomerID: row.TargetCustomerID,
		Type:             domain.CommissionType(row.Type),
	}
	if v, err := parseDecimal(row.Value); err == nil {
		cs.Value = v
	}
	if row.Conditions.Valid {
		cs.Conditions = json.RawMessage(row.Conditions.RawMessage)
	}
	if row.IsActive.Valid {
		cs.IsActive = row.IsActive.Bool
	}
	if row.CreatedAt.Valid {
		cs.CreatedAt = row.CreatedAt.Time
	}
	if row.UpdatedAt.Valid {
		cs.UpdatedAt = row.UpdatedAt.Time
	}
	if row.DeletedAt.Valid {
		cs.DeletedAt = &row.DeletedAt.Time
	}
	return cs
}

func toDomainWalletLock(row walletdb.WalletLock) domain.WalletLock {
	l := domain.WalletLock{
		ID:             row.ID,
		OrganizationID: row.OrganizationID,
		WalletID:       row.WalletID,
		Type:           domain.WalletLockType(row.Type),
		ReferenceID:    row.ReferenceID,
		AmountCents:    row.AmountCents,
		Status:         domain.WalletLockStatus(row.Status),
	}
	if row.CreatedAt.Valid {
		l.CreatedAt = row.CreatedAt.Time
	}
	if row.ReleasedAt.Valid {
		l.ReleasedAt = &row.ReleasedAt.Time
	}
	return l
}

var _ domain.WalletRepository = (*WalletRepo)(nil)
