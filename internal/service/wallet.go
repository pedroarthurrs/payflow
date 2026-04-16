package service

import (
	"context"
	"errors"

	"i9pay/internal/domain"

	"github.com/google/uuid"
)

// WalletService handles wallet operations exposed via HTTP.
type WalletService struct {
	wallets domain.WalletRepository
	txMgr   domain.TxManager
}

// NewWalletService creates a WalletService.
func NewWalletService(wallets domain.WalletRepository, txMgr domain.TxManager) *WalletService {
	return &WalletService{wallets: wallets, txMgr: txMgr}
}

// GetOrgWallet returns the org wallet, creating it if it doesn't exist.
func (s *WalletService) GetOrgWallet(ctx context.Context, organizationID uuid.UUID) (*domain.Wallet, error) {
	wallet, err := s.wallets.GetWalletByOrganizationOwner(ctx, organizationID)
	if err == nil {
		return wallet, nil
	}
	if !errors.Is(err, domain.ErrWalletNotFound) {
		return nil, err
	}
	return s.wallets.CreateWallet(ctx, organizationID, domain.CreateWalletInput{
		OrganizationOwnerID: &organizationID,
		Currency:            "BRL",
	})
}

// GetCustomerWallet returns the customer wallet, creating it if it doesn't exist.
func (s *WalletService) GetCustomerWallet(ctx context.Context, organizationID, customerID uuid.UUID) (*domain.Wallet, error) {
	wallet, err := s.wallets.GetWalletByCustomerOwner(ctx, organizationID, customerID)
	if err == nil {
		return wallet, nil
	}
	if !errors.Is(err, domain.ErrWalletNotFound) {
		return nil, err
	}
	return s.wallets.CreateWallet(ctx, organizationID, domain.CreateWalletInput{
		CustomerOwnerID: &customerID,
		Currency:        "BRL",
	})
}

// GetByID returns a wallet by ID, scoped to the organization.
func (s *WalletService) GetByID(ctx context.Context, organizationID, walletID uuid.UUID) (*domain.Wallet, error) {
	return s.wallets.GetWalletByID(ctx, organizationID, walletID)
}

// RequestPayout creates a payout and locks the amount. Does NOT debit balance.
func (s *WalletService) RequestPayout(ctx context.Context, organizationID uuid.UUID, input domain.CreatePayoutInput) (*domain.Payout, error) {
	// Pre-check outside tx (optimistic guard)
	wallet, err := s.wallets.GetWalletByID(ctx, organizationID, input.WalletID)
	if err != nil {
		return nil, err
	}
	if wallet.Status == domain.WalletStatusSuspended {
		return nil, domain.ErrWalletSuspended
	}
	total := input.AmountCents + input.FeeCents
	if wallet.Balance-wallet.LockedBalance < total {
		return nil, domain.ErrInsufficientBalance
	}

	var payout *domain.Payout
	err = s.txMgr.WithTx(ctx, func(tx domain.TxHandle) error {
		repo := tx.WalletRepo()

		w, err := repo.GetWalletForUpdate(ctx, organizationID, input.WalletID)
		if err != nil {
			return err
		}
		if w.Status == domain.WalletStatusSuspended {
			return domain.ErrWalletSuspended
		}
		if w.Balance-w.LockedBalance < total {
			return domain.ErrInsufficientBalance
		}

		p, err := repo.CreatePayout(ctx, organizationID, input)
		if err != nil {
			return err
		}

		if _, err := repo.CreateWalletLock(ctx, organizationID, domain.CreateWalletLockInput{
			WalletID:    w.ID,
			Type:        domain.LockTypePayout,
			ReferenceID: p.ID,
			AmountCents: total,
		}); err != nil {
			return err
		}

		if _, err := repo.UpdateWalletLockedBalance(ctx, organizationID, w.ID, w.LockedBalance+total); err != nil {
			return err
		}

		payout = p
		return nil
	})
	if err != nil {
		return nil, err
	}
	return payout, nil
}

// UpdatePayoutStatus transitions a payout to paid or failed, handling balance and lock accordingly.
func (s *WalletService) UpdatePayoutStatus(ctx context.Context, organizationID, payoutID uuid.UUID, status domain.PayoutStatus) (*domain.Payout, error) {
	var updated *domain.Payout
	err := s.txMgr.WithTx(ctx, func(tx domain.TxHandle) error {
		repo := tx.WalletRepo()

		lock, err := repo.GetWalletLockByReference(ctx, organizationID, payoutID, domain.LockTypePayout)
		if err != nil {
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

		if status == domain.PayoutPaid {
			newBalance := wallet.Balance - lock.AmountCents
			if _, err := repo.UpdateWalletBalance(ctx, organizationID, wallet.ID, newBalance); err != nil {
				return err
			}
			if _, err := repo.InsertLedgerEntry(ctx, organizationID, domain.LedgerEntryInput{
				WalletID:     wallet.ID,
				Type:         domain.LedgerDebit,
				AmountCents:  lock.AmountCents,
				Category:     domain.CategoryPayout,
				BalanceAfter: newBalance,
			}); err != nil {
				return err
			}
		}

		p, err := repo.UpdatePayoutStatus(ctx, organizationID, payoutID, status)
		if err != nil {
			return err
		}
		updated = p
		return nil
	})
	if err != nil {
		return nil, err
	}
	return updated, nil
}

// Adjust applies a manual credit or debit to a wallet.
func (s *WalletService) Adjust(ctx context.Context, organizationID, walletID uuid.UUID, input domain.AdjustWalletInput) (*domain.Wallet, error) {
	if input.AmountCents <= 0 || input.Reason == "" {
		return nil, domain.ErrInvalidAdjustment
	}

	var result *domain.Wallet
	err := s.txMgr.WithTx(ctx, func(tx domain.TxHandle) error {
		repo := tx.WalletRepo()

		wallet, err := repo.GetWalletForUpdate(ctx, organizationID, walletID)
		if err != nil {
			return err
		}

		var newBalance int32
		switch input.Direction {
		case domain.LedgerCredit:
			newBalance = wallet.Balance + input.AmountCents
		case domain.LedgerDebit:
			if wallet.Balance < input.AmountCents {
				return domain.ErrInsufficientBalance
			}
			newBalance = wallet.Balance - input.AmountCents
		default:
			return domain.ErrInvalidAdjustment
		}

		w, err := repo.UpdateWalletBalance(ctx, organizationID, walletID, newBalance)
		if err != nil {
			return err
		}

		if _, err := repo.InsertLedgerEntry(ctx, organizationID, domain.LedgerEntryInput{
			WalletID:     walletID,
			Type:         input.Direction,
			AmountCents:  input.AmountCents,
			Category:     domain.CategoryManualAdjustment,
			BalanceAfter: newBalance,
		}); err != nil {
			return err
		}

		result = w
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

// ListEntries returns ledger entries with pagination and filters.
func (s *WalletService) ListEntries(ctx context.Context, organizationID, walletID uuid.UUID, params domain.LedgerListParams) ([]domain.LedgerEntry, int64, error) {
	entries, err := s.wallets.ListLedgerEntries(ctx, organizationID, walletID, params)
	if err != nil {
		return nil, 0, err
	}
	total, err := s.wallets.CountLedgerEntries(ctx, organizationID, walletID, params)
	if err != nil {
		return nil, 0, err
	}
	return entries, total, nil
}

// ListPayouts returns payouts for a wallet with pagination.
func (s *WalletService) ListPayouts(ctx context.Context, organizationID, walletID uuid.UUID, params domain.ListParams) ([]domain.Payout, error) {
	return s.wallets.ListPayouts(ctx, organizationID, walletID, params)
}
