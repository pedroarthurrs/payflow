package service

import (
	"context"
	"errors"
	"log/slog"
	"math"

	"i9pay/internal/domain"

	"github.com/google/uuid"
)

// LedgerService handles ledger operations triggered by payment FSM transitions.
// It is called inside existing transactions — never opens its own.
type LedgerService struct {
	wallets domain.WalletRepository
}

// NewLedgerService creates a LedgerService.
func NewLedgerService(wallets domain.WalletRepository) *LedgerService {
	return &LedgerService{wallets: wallets}
}

// CreditOnPaymentSuccess credits the org wallet and, if a commission setting exists,
// credits the customer wallet. Called inside the payment FSM transaction.
func (s *LedgerService) CreditOnPaymentSuccess(
	ctx context.Context,
	tx domain.TxHandle,
	organizationID uuid.UUID,
	p *domain.PaymentForUpdate,
) error {
	repo := tx.WalletRepo()

	orgWallet, err := s.getOrCreateOrgWallet(ctx, repo, organizationID)
	if err != nil {
		return err
	}

	if orgWallet.Status == domain.WalletStatusSuspended {
		slog.Warn("org wallet suspended, skipping credit", "wallet_id", orgWallet.ID, "payment_id", p.ID)
		return nil
	}

	newBalance := orgWallet.Balance + p.AmountCents
	if _, err := repo.UpdateWalletBalance(ctx, organizationID, orgWallet.ID, newBalance); err != nil {
		return err
	}

	paymentID := p.ID
	if _, err := repo.InsertLedgerEntry(ctx, organizationID, domain.LedgerEntryInput{
		WalletID:     orgWallet.ID,
		Type:         domain.LedgerCredit,
		AmountCents:  p.AmountCents,
		Category:     domain.CategoryPaymentSplit,
		BalanceAfter: newBalance,
		PaymentID:    &paymentID,
	}); err != nil {
		return err
	}

	// Commission
	setting, err := repo.GetCommissionSettingByCustomer(ctx, organizationID, p.CustomerID)
	if err != nil {
		if errors.Is(err, domain.ErrCommissionNotFound) {
			return nil
		}
		return err
	}

	if !setting.IsActive {
		return nil
	}

	commission := s.computeCommission(setting, p.AmountCents)
	if commission <= 0 {
		return nil
	}
	if commission > p.AmountCents {
		slog.Warn("commission exceeds payment amount, skipping", "commission", commission, "amount", p.AmountCents, "payment_id", p.ID)
		return nil
	}

	customerWallet, err := s.getOrCreateCustomerWallet(ctx, repo, organizationID, p.CustomerID)
	if err != nil {
		return err
	}

	if customerWallet.Status == domain.WalletStatusSuspended {
		slog.Warn("customer wallet suspended, skipping commission", "wallet_id", customerWallet.ID)
		return nil
	}

	newCustomerBalance := customerWallet.Balance + commission
	if _, err := repo.UpdateWalletBalance(ctx, organizationID, customerWallet.ID, newCustomerBalance); err != nil {
		return err
	}

	if _, err := repo.InsertLedgerEntry(ctx, organizationID, domain.LedgerEntryInput{
		WalletID:     customerWallet.ID,
		Type:         domain.LedgerCredit,
		AmountCents:  commission,
		Category:     domain.CategoryCommission,
		BalanceAfter: newCustomerBalance,
		PaymentID:    &paymentID,
	}); err != nil {
		return err
	}

	return nil
}

// LockOnPaymentDisputed creates a dispute lock and increments locked_balance.
func (s *LedgerService) LockOnPaymentDisputed(
	ctx context.Context,
	tx domain.TxHandle,
	organizationID uuid.UUID,
	p *domain.PaymentForUpdate,
) error {
	repo := tx.WalletRepo()

	orgWallet, err := repo.GetWalletByOrganizationOwner(ctx, organizationID)
	if err != nil {
		if errors.Is(err, domain.ErrWalletNotFound) {
			return nil // no wallet yet, nothing to lock
		}
		return err
	}

	wallet, err := repo.GetWalletForUpdate(ctx, organizationID, orgWallet.ID)
	if err != nil {
		return err
	}

	if _, err := repo.CreateWalletLock(ctx, organizationID, domain.CreateWalletLockInput{
		WalletID:    wallet.ID,
		Type:        domain.LockTypeDispute,
		ReferenceID: p.ID,
		AmountCents: p.AmountCents,
	}); err != nil {
		return err
	}

	newLocked := wallet.LockedBalance + p.AmountCents
	if _, err := repo.UpdateWalletLockedBalance(ctx, organizationID, wallet.ID, newLocked); err != nil {
		return err
	}

	return nil
}

// ReleaseLockIfDisputed releases an active dispute lock. Idempotent — no-op if lock not found.
func (s *LedgerService) ReleaseLockIfDisputed(
	ctx context.Context,
	tx domain.TxHandle,
	organizationID uuid.UUID,
	p *domain.PaymentForUpdate,
) error {
	repo := tx.WalletRepo()

	lock, err := repo.GetWalletLockByReference(ctx, organizationID, p.ID, domain.LockTypeDispute)
	if err != nil {
		if errors.Is(err, domain.ErrWalletLockNotFound) {
			return nil
		}
		return err
	}

	if err := repo.ReleaseWalletLock(ctx, organizationID, lock.ID); err != nil {
		return err
	}

	wallet, err := repo.GetWalletForUpdate(ctx, organizationID, lock.WalletID)
	if err != nil {
		return err
	}

	newLocked := wallet.LockedBalance - lock.AmountCents
	if newLocked < 0 {
		newLocked = 0
	}
	if _, err := repo.UpdateWalletLockedBalance(ctx, organizationID, wallet.ID, newLocked); err != nil {
		return err
	}

	return nil
}

// DebitOnPaymentRefund debits the org wallet. Allows overdraft (logs warning).
// Also releases any active dispute lock.
func (s *LedgerService) DebitOnPaymentRefund(
	ctx context.Context,
	tx domain.TxHandle,
	organizationID uuid.UUID,
	p *domain.PaymentForUpdate,
) error {
	repo := tx.WalletRepo()

	orgWallet, err := repo.GetWalletByOrganizationOwner(ctx, organizationID)
	if err != nil {
		if errors.Is(err, domain.ErrWalletNotFound) {
			return nil
		}
		return err
	}

	wallet, err := repo.GetWalletForUpdate(ctx, organizationID, orgWallet.ID)
	if err != nil {
		return err
	}

	if wallet.Status == domain.WalletStatusSuspended {
		slog.Warn("org wallet suspended, skipping debit on refund", "wallet_id", wallet.ID, "payment_id", p.ID)
		return nil
	}

	newBalance := wallet.Balance - p.AmountCents
	if newBalance < 0 {
		slog.Warn("wallet overdraft on refund", "wallet_id", wallet.ID, "balance", wallet.Balance, "debit", p.AmountCents)
	}

	if _, err := repo.UpdateWalletBalance(ctx, organizationID, wallet.ID, newBalance); err != nil {
		return err
	}

	paymentID := p.ID
	if _, err := repo.InsertLedgerEntry(ctx, organizationID, domain.LedgerEntryInput{
		WalletID:     wallet.ID,
		Type:         domain.LedgerDebit,
		AmountCents:  p.AmountCents,
		Category:     domain.CategoryRefund,
		BalanceAfter: newBalance,
		PaymentID:    &paymentID,
	}); err != nil {
		return err
	}

	// Release dispute lock if one exists (idempotent)
	if err := s.ReleaseLockIfDisputed(ctx, tx, organizationID, p); err != nil {
		slog.Warn("failed to release dispute lock on refund", "payment_id", p.ID, "error", err)
	}

	return nil
}

// computeCommission calculates the commission amount in cents.
// percentage: floor(amountCents * value / 100)
// fixed: int32(value) — already in cents
// Returns 0 if setting is nil or inactive.
func (s *LedgerService) computeCommission(setting *domain.CommissionSetting, amountCents int32) int32 {
	if setting == nil || !setting.IsActive {
		return 0
	}
	switch setting.Type {
	case domain.CommissionPercentage:
		return int32(math.Floor(float64(amountCents) * setting.Value / 100))
	case domain.CommissionFixed:
		return int32(setting.Value)
	}
	return 0
}

// getOrCreateOrgWallet returns the org wallet, creating it if it doesn't exist.
func (s *LedgerService) getOrCreateOrgWallet(ctx context.Context, repo domain.WalletRepository, organizationID uuid.UUID) (*domain.Wallet, error) {
	wallet, err := repo.GetWalletByOrganizationOwner(ctx, organizationID)
	if err == nil {
		return wallet, nil
	}
	if !errors.Is(err, domain.ErrWalletNotFound) {
		return nil, err
	}
	return repo.CreateWallet(ctx, organizationID, domain.CreateWalletInput{
		OrganizationOwnerID: &organizationID,
		Currency:            "BRL",
	})
}

// getOrCreateCustomerWallet returns the customer wallet, creating it if it doesn't exist.
func (s *LedgerService) getOrCreateCustomerWallet(ctx context.Context, repo domain.WalletRepository, organizationID, customerID uuid.UUID) (*domain.Wallet, error) {
	wallet, err := repo.GetWalletByCustomerOwner(ctx, organizationID, customerID)
	if err == nil {
		return wallet, nil
	}
	if !errors.Is(err, domain.ErrWalletNotFound) {
		return nil, err
	}
	return repo.CreateWallet(ctx, organizationID, domain.CreateWalletInput{
		CustomerOwnerID: &customerID,
		Currency:        "BRL",
	})
}
