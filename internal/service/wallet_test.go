package service

import (
	"context"
	"testing"

	"i9pay/internal/domain"

	"github.com/google/uuid"
)

// ── txManager stub for wallet tests ──────────────────────────────────────────

type walletTxManager struct {
	repo domain.WalletRepository
}

func (m *walletTxManager) WithTx(ctx context.Context, fn func(domain.TxHandle) error) error {
	return fn(&txHandleWithWallet{walletRepo: m.repo})
}

// ── RequestPayout ─────────────────────────────────────────────────────────────

func TestWalletRequestPayoutHappyPath(t *testing.T) {
	walletID := uuid.New()
	orgID := uuid.New()
	w := &domain.Wallet{
		ID:             walletID,
		OrganizationID: orgID,
		Balance:        10000,
		LockedBalance:  0,
		Status:         domain.WalletStatusActive,
	}

	repo := &controlledWalletRepo{orgWallet: w}
	// GetWalletByID needs to work too
	repo.orgWallet = w

	// GetWalletByID is called for pre-check — override to return our wallet
	fullRepo := &walletServiceRepo{controlledWalletRepo: repo, wallet: w}
	svc2 := NewWalletService(fullRepo, &walletTxManager{repo: fullRepo})

	input := domain.CreatePayoutInput{
		WalletID:    walletID,
		AmountCents: 5000,
		FeeCents:    100,
	}

	payout, err := svc2.RequestPayout(context.Background(), orgID, input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if payout == nil {
		t.Fatal("expected payout, got nil")
	}
	// locked_balance should be incremented
	if len(fullRepo.lockedBalanceUpdates) == 0 {
		t.Error("expected locked_balance to be updated")
	}
	if fullRepo.lockedBalanceUpdates[0] != 5100 {
		t.Errorf("expected locked 5100, got %d", fullRepo.lockedBalanceUpdates[0])
	}
	// balance should NOT be debited
	if len(fullRepo.balanceUpdates) != 0 {
		t.Error("balance should not be debited on payout creation")
	}
}

func TestWalletRequestPayoutInsufficientBalance(t *testing.T) {
	walletID := uuid.New()
	orgID := uuid.New()
	w := &domain.Wallet{
		ID:            walletID,
		Balance:       1000,
		LockedBalance: 900,
		Status:        domain.WalletStatusActive,
	}
	repo := &walletServiceRepo{controlledWalletRepo: &controlledWalletRepo{orgWallet: w}, wallet: w}
	svc := NewWalletService(repo, &walletTxManager{repo: repo})

	_, err := svc.RequestPayout(context.Background(), orgID, domain.CreatePayoutInput{
		WalletID:    walletID,
		AmountCents: 200,
		FeeCents:    0,
	})
	if err != domain.ErrInsufficientBalance {
		t.Errorf("expected ErrInsufficientBalance, got %v", err)
	}
}

func TestWalletRequestPayoutSuspended(t *testing.T) {
	walletID := uuid.New()
	orgID := uuid.New()
	w := &domain.Wallet{
		ID:     walletID,
		Status: domain.WalletStatusSuspended,
	}
	repo := &walletServiceRepo{controlledWalletRepo: &controlledWalletRepo{orgWallet: w}, wallet: w}
	svc := NewWalletService(repo, &walletTxManager{repo: repo})

	_, err := svc.RequestPayout(context.Background(), orgID, domain.CreatePayoutInput{
		WalletID:    walletID,
		AmountCents: 100,
	})
	if err != domain.ErrWalletSuspended {
		t.Errorf("expected ErrWalletSuspended, got %v", err)
	}
}

// ── Adjust ────────────────────────────────────────────────────────────────────

func TestWalletAdjustCreditHappyPath(t *testing.T) {
	walletID := uuid.New()
	orgID := uuid.New()
	w := &domain.Wallet{ID: walletID, Balance: 500, Status: domain.WalletStatusActive}
	repo := &walletServiceRepo{controlledWalletRepo: &controlledWalletRepo{orgWallet: w}, wallet: w}
	svc := NewWalletService(repo, &walletTxManager{repo: repo})

	result, err := svc.Adjust(context.Background(), orgID, walletID, domain.AdjustWalletInput{
		Direction:   domain.LedgerCredit,
		AmountCents: 200,
		Reason:      "test credit",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected wallet result")
	}
	if repo.balanceUpdates[0] != 700 {
		t.Errorf("expected balance 700, got %d", repo.balanceUpdates[0])
	}
}

func TestWalletAdjustDebitHappyPath(t *testing.T) {
	walletID := uuid.New()
	orgID := uuid.New()
	w := &domain.Wallet{ID: walletID, Balance: 1000, Status: domain.WalletStatusActive}
	repo := &walletServiceRepo{controlledWalletRepo: &controlledWalletRepo{orgWallet: w}, wallet: w}
	svc := NewWalletService(repo, &walletTxManager{repo: repo})

	_, err := svc.Adjust(context.Background(), orgID, walletID, domain.AdjustWalletInput{
		Direction:   domain.LedgerDebit,
		AmountCents: 300,
		Reason:      "test debit",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.balanceUpdates[0] != 700 {
		t.Errorf("expected balance 700, got %d", repo.balanceUpdates[0])
	}
}

func TestWalletAdjustDebitInsufficientBalance(t *testing.T) {
	walletID := uuid.New()
	orgID := uuid.New()
	w := &domain.Wallet{ID: walletID, Balance: 100, Status: domain.WalletStatusActive}
	repo := &walletServiceRepo{controlledWalletRepo: &controlledWalletRepo{orgWallet: w}, wallet: w}
	svc := NewWalletService(repo, &walletTxManager{repo: repo})

	_, err := svc.Adjust(context.Background(), orgID, walletID, domain.AdjustWalletInput{
		Direction:   domain.LedgerDebit,
		AmountCents: 500,
		Reason:      "too much",
	})
	if err != domain.ErrInsufficientBalance {
		t.Errorf("expected ErrInsufficientBalance, got %v", err)
	}
}

func TestWalletAdjustInvalidInput(t *testing.T) {
	svc := NewWalletService(&mockWalletRepo{}, &walletTxManager{repo: &mockWalletRepo{}})

	_, err := svc.Adjust(context.Background(), uuid.New(), uuid.New(), domain.AdjustWalletInput{
		Direction:   domain.LedgerCredit,
		AmountCents: 0, // invalid
		Reason:      "test",
	})
	if err != domain.ErrInvalidAdjustment {
		t.Errorf("expected ErrInvalidAdjustment, got %v", err)
	}

	_, err = svc.Adjust(context.Background(), uuid.New(), uuid.New(), domain.AdjustWalletInput{
		Direction:   domain.LedgerCredit,
		AmountCents: 100,
		Reason:      "", // invalid
	})
	if err != domain.ErrInvalidAdjustment {
		t.Errorf("expected ErrInvalidAdjustment for empty reason, got %v", err)
	}
}

// ── walletServiceRepo: extends controlledWalletRepo with GetWalletByID ────────

type walletServiceRepo struct {
	*controlledWalletRepo
	wallet *domain.Wallet
}

func (r *walletServiceRepo) GetWalletByID(_ context.Context, _, _ uuid.UUID) (*domain.Wallet, error) {
	if r.wallet == nil {
		return nil, domain.ErrWalletNotFound
	}
	return r.wallet, nil
}

func (r *walletServiceRepo) GetWalletForUpdate(_ context.Context, _, _ uuid.UUID) (*domain.Wallet, error) {
	if r.wallet == nil {
		return nil, domain.ErrWalletNotFound
	}
	return r.wallet, nil
}

func (r *walletServiceRepo) CreatePayout(_ context.Context, _ uuid.UUID, input domain.CreatePayoutInput) (*domain.Payout, error) {
	return &domain.Payout{
		ID:          uuid.New(),
		WalletID:    input.WalletID,
		AmountCents: input.AmountCents,
		FeeCents:    input.FeeCents,
		Status:      domain.PayoutPending,
	}, nil
}

func (r *walletServiceRepo) UpdateWalletBalance(_ context.Context, _, _ uuid.UUID, newBalance int32) (*domain.Wallet, error) {
	r.balanceUpdates = append(r.balanceUpdates, newBalance)
	w := *r.wallet
	w.Balance = newBalance
	return &w, nil
}

func (r *walletServiceRepo) UpdateWalletLockedBalance(_ context.Context, _, _ uuid.UUID, newLocked int32) (*domain.Wallet, error) {
	r.lockedBalanceUpdates = append(r.lockedBalanceUpdates, newLocked)
	w := *r.wallet
	w.LockedBalance = newLocked
	return &w, nil
}

func (r *walletServiceRepo) InsertLedgerEntry(_ context.Context, _ uuid.UUID, input domain.LedgerEntryInput) (*domain.LedgerEntry, error) {
	r.ledgerEntries = append(r.ledgerEntries, input)
	return &domain.LedgerEntry{ID: uuid.New()}, nil
}
