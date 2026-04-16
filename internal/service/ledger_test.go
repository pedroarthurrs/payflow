package service

import (
	"context"
	"testing"

	"i9pay/internal/domain"

	"github.com/google/uuid"
)

// ── helpers ──────────────────────────────────────────────────────────────────

func newOrgWallet(balance, locked int32) *domain.Wallet {
	return &domain.Wallet{
		ID:             uuid.New(),
		OrganizationID: uuid.New(),
		Balance:        balance,
		LockedBalance:  locked,
		Status:         domain.WalletStatusActive,
		Currency:       "BRL",
	}
}

// controlledWalletRepo lets tests inject specific responses per method.
type controlledWalletRepo struct {
	mockWalletRepo // embed no-op base

	orgWallet         *domain.Wallet
	customerWallet    *domain.Wallet
	commissionSetting *domain.CommissionSetting
	existingLock      *domain.WalletLock

	// capture calls
	balanceUpdates       []int32
	lockedBalanceUpdates []int32
	ledgerEntries        []domain.LedgerEntryInput
	locksCreated         []domain.CreateWalletLockInput
	locksReleased        []uuid.UUID
}

func (r *controlledWalletRepo) GetWalletByOrganizationOwner(_ context.Context, _ uuid.UUID) (*domain.Wallet, error) {
	if r.orgWallet == nil {
		return nil, domain.ErrWalletNotFound
	}
	return r.orgWallet, nil
}

func (r *controlledWalletRepo) GetWalletByCustomerOwner(_ context.Context, _, _ uuid.UUID) (*domain.Wallet, error) {
	if r.customerWallet == nil {
		return nil, domain.ErrWalletNotFound
	}
	return r.customerWallet, nil
}

func (r *controlledWalletRepo) GetWalletForUpdate(_ context.Context, _, walletID uuid.UUID) (*domain.Wallet, error) {
	if r.orgWallet != nil && r.orgWallet.ID == walletID {
		return r.orgWallet, nil
	}
	if r.customerWallet != nil && r.customerWallet.ID == walletID {
		return r.customerWallet, nil
	}
	return nil, domain.ErrWalletNotFound
}

func (r *controlledWalletRepo) CreateWallet(_ context.Context, _ uuid.UUID, input domain.CreateWalletInput) (*domain.Wallet, error) {
	w := &domain.Wallet{ID: uuid.New(), Balance: 0, Status: domain.WalletStatusActive, Currency: "BRL"}
	if input.OrganizationOwnerID != nil {
		r.orgWallet = w
	} else {
		r.customerWallet = w
	}
	return w, nil
}

func (r *controlledWalletRepo) UpdateWalletBalance(_ context.Context, _, _ uuid.UUID, newBalance int32) (*domain.Wallet, error) {
	r.balanceUpdates = append(r.balanceUpdates, newBalance)
	w := *r.orgWallet
	w.Balance = newBalance
	return &w, nil
}

func (r *controlledWalletRepo) UpdateWalletLockedBalance(_ context.Context, _, _ uuid.UUID, newLocked int32) (*domain.Wallet, error) {
	r.lockedBalanceUpdates = append(r.lockedBalanceUpdates, newLocked)
	w := *r.orgWallet
	w.LockedBalance = newLocked
	return &w, nil
}

func (r *controlledWalletRepo) InsertLedgerEntry(_ context.Context, _ uuid.UUID, input domain.LedgerEntryInput) (*domain.LedgerEntry, error) {
	r.ledgerEntries = append(r.ledgerEntries, input)
	return &domain.LedgerEntry{ID: uuid.New()}, nil
}

func (r *controlledWalletRepo) GetCommissionSettingByCustomer(_ context.Context, _, _ uuid.UUID) (*domain.CommissionSetting, error) {
	if r.commissionSetting == nil {
		return nil, domain.ErrCommissionNotFound
	}
	return r.commissionSetting, nil
}

func (r *controlledWalletRepo) CreateWalletLock(_ context.Context, _ uuid.UUID, input domain.CreateWalletLockInput) (*domain.WalletLock, error) {
	r.locksCreated = append(r.locksCreated, input)
	lock := &domain.WalletLock{ID: uuid.New(), AmountCents: input.AmountCents, WalletID: input.WalletID}
	return lock, nil
}

func (r *controlledWalletRepo) ReleaseWalletLock(_ context.Context, _, lockID uuid.UUID) error {
	r.locksReleased = append(r.locksReleased, lockID)
	return nil
}

func (r *controlledWalletRepo) GetWalletLockByReference(_ context.Context, _, _ uuid.UUID, _ domain.WalletLockType) (*domain.WalletLock, error) {
	if r.existingLock == nil {
		return nil, domain.ErrWalletLockNotFound
	}
	return r.existingLock, nil
}

// txHandleWithWallet wraps mockTxHandle but returns a specific wallet repo.
type txHandleWithWallet struct {
	mockTxHandle
	walletRepo domain.WalletRepository
}

func (h *txHandleWithWallet) WalletRepo() domain.WalletRepository { return h.walletRepo }

// ── tests ─────────────────────────────────────────────────────────────────────

func TestLedgerCreditOnPaymentSuccess(t *testing.T) {
	orgID := uuid.New()
	repo := &controlledWalletRepo{
		orgWallet: newOrgWallet(1000, 0),
	}
	svc := NewLedgerService(repo)
	tx := &txHandleWithWallet{walletRepo: repo}

	p := &domain.PaymentForUpdate{
		ID:          uuid.New(),
		CustomerID:  uuid.New(),
		AmountCents: 500,
	}

	if err := svc.CreditOnPaymentSuccess(context.Background(), tx, orgID, p); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(repo.balanceUpdates) != 1 || repo.balanceUpdates[0] != 1500 {
		t.Errorf("expected balance 1500, got %v", repo.balanceUpdates)
	}
	if len(repo.ledgerEntries) != 1 {
		t.Errorf("expected 1 ledger entry, got %d", len(repo.ledgerEntries))
	}
	if repo.ledgerEntries[0].Category != domain.CategoryPaymentSplit {
		t.Errorf("expected payment_split category")
	}
}

func TestLedgerCreditWithCommissionPercentage(t *testing.T) {
	orgID := uuid.New()
	customerID := uuid.New()
	repo := &controlledWalletRepo{
		orgWallet: newOrgWallet(1000, 0),
		commissionSetting: &domain.CommissionSetting{
			Type:     domain.CommissionPercentage,
			Value:    10,
			IsActive: true,
		},
	}
	svc := NewLedgerService(repo)
	tx := &txHandleWithWallet{walletRepo: repo}

	p := &domain.PaymentForUpdate{ID: uuid.New(), CustomerID: customerID, AmountCents: 1000}

	if err := svc.CreditOnPaymentSuccess(context.Background(), tx, orgID, p); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// org credit + customer commission = 2 entries
	if len(repo.ledgerEntries) != 2 {
		t.Fatalf("expected 2 ledger entries, got %d", len(repo.ledgerEntries))
	}
	if repo.ledgerEntries[1].Category != domain.CategoryCommission {
		t.Errorf("expected commission category for second entry")
	}
	if repo.ledgerEntries[1].AmountCents != 100 { // 10% of 1000
		t.Errorf("expected commission 100, got %d", repo.ledgerEntries[1].AmountCents)
	}
}

func TestLedgerCreditWithCommissionFixed(t *testing.T) {
	orgID := uuid.New()
	repo := &controlledWalletRepo{
		orgWallet: newOrgWallet(0, 0),
		commissionSetting: &domain.CommissionSetting{
			Type:     domain.CommissionFixed,
			Value:    50,
			IsActive: true,
		},
	}
	svc := NewLedgerService(repo)
	tx := &txHandleWithWallet{walletRepo: repo}

	p := &domain.PaymentForUpdate{ID: uuid.New(), CustomerID: uuid.New(), AmountCents: 200}

	if err := svc.CreditOnPaymentSuccess(context.Background(), tx, orgID, p); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if repo.ledgerEntries[1].AmountCents != 50 {
		t.Errorf("expected fixed commission 50, got %d", repo.ledgerEntries[1].AmountCents)
	}
}

func TestLedgerCreditWalletSuspended(t *testing.T) {
	orgID := uuid.New()
	w := newOrgWallet(1000, 0)
	w.Status = domain.WalletStatusSuspended
	repo := &controlledWalletRepo{orgWallet: w}
	svc := NewLedgerService(repo)
	tx := &txHandleWithWallet{walletRepo: repo}

	p := &domain.PaymentForUpdate{ID: uuid.New(), CustomerID: uuid.New(), AmountCents: 500}

	if err := svc.CreditOnPaymentSuccess(context.Background(), tx, orgID, p); err != nil {
		t.Fatalf("suspended wallet should not return error, got: %v", err)
	}
	if len(repo.balanceUpdates) != 0 {
		t.Error("suspended wallet should not update balance")
	}
}

func TestLedgerCreditCommissionExceedsAmount(t *testing.T) {
	orgID := uuid.New()
	repo := &controlledWalletRepo{
		orgWallet: newOrgWallet(0, 0),
		commissionSetting: &domain.CommissionSetting{
			Type:     domain.CommissionFixed,
			Value:    999,
			IsActive: true,
		},
	}
	svc := NewLedgerService(repo)
	tx := &txHandleWithWallet{walletRepo: repo}

	p := &domain.PaymentForUpdate{ID: uuid.New(), CustomerID: uuid.New(), AmountCents: 100}

	if err := svc.CreditOnPaymentSuccess(context.Background(), tx, orgID, p); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Only org entry, commission skipped
	if len(repo.ledgerEntries) != 1 {
		t.Errorf("expected 1 entry (commission skipped), got %d", len(repo.ledgerEntries))
	}
}

func TestLedgerDebitOnRefund(t *testing.T) {
	orgID := uuid.New()
	repo := &controlledWalletRepo{orgWallet: newOrgWallet(1000, 0)}
	svc := NewLedgerService(repo)
	tx := &txHandleWithWallet{walletRepo: repo}

	p := &domain.PaymentForUpdate{ID: uuid.New(), CustomerID: uuid.New(), AmountCents: 300}

	if err := svc.DebitOnPaymentRefund(context.Background(), tx, orgID, p); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(repo.balanceUpdates) != 1 || repo.balanceUpdates[0] != 700 {
		t.Errorf("expected balance 700, got %v", repo.balanceUpdates)
	}
	if repo.ledgerEntries[0].Category != domain.CategoryRefund {
		t.Errorf("expected refund category")
	}
}

func TestLedgerLockOnDisputed(t *testing.T) {
	orgID := uuid.New()
	repo := &controlledWalletRepo{orgWallet: newOrgWallet(1000, 0)}
	svc := NewLedgerService(repo)
	tx := &txHandleWithWallet{walletRepo: repo}

	p := &domain.PaymentForUpdate{ID: uuid.New(), CustomerID: uuid.New(), AmountCents: 400}

	if err := svc.LockOnPaymentDisputed(context.Background(), tx, orgID, p); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(repo.locksCreated) != 1 {
		t.Fatalf("expected 1 lock created, got %d", len(repo.locksCreated))
	}
	if repo.locksCreated[0].Type != domain.LockTypeDispute {
		t.Errorf("expected dispute lock type")
	}
	if len(repo.lockedBalanceUpdates) != 1 || repo.lockedBalanceUpdates[0] != 400 {
		t.Errorf("expected locked_balance 400, got %v", repo.lockedBalanceUpdates)
	}
}

func TestLedgerReleaseLockIfDisputed(t *testing.T) {
	orgID := uuid.New()
	lockID := uuid.New()
	walletID := uuid.New()
	w := &domain.Wallet{ID: walletID, Balance: 1000, LockedBalance: 400, Status: domain.WalletStatusActive}
	repo := &controlledWalletRepo{
		orgWallet: w,
		existingLock: &domain.WalletLock{
			ID:          lockID,
			WalletID:    walletID,
			AmountCents: 400,
		},
	}
	svc := NewLedgerService(repo)
	tx := &txHandleWithWallet{walletRepo: repo}

	p := &domain.PaymentForUpdate{ID: uuid.New(), CustomerID: uuid.New(), AmountCents: 400}

	if err := svc.ReleaseLockIfDisputed(context.Background(), tx, orgID, p); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(repo.locksReleased) != 1 {
		t.Errorf("expected 1 lock released")
	}
	if len(repo.lockedBalanceUpdates) != 1 || repo.lockedBalanceUpdates[0] != 0 {
		t.Errorf("expected locked_balance 0, got %v", repo.lockedBalanceUpdates)
	}
}

func TestLedgerReleaseLockIdempotent(t *testing.T) {
	orgID := uuid.New()
	repo := &controlledWalletRepo{orgWallet: newOrgWallet(1000, 0)} // no existingLock
	svc := NewLedgerService(repo)
	tx := &txHandleWithWallet{walletRepo: repo}

	p := &domain.PaymentForUpdate{ID: uuid.New(), CustomerID: uuid.New(), AmountCents: 400}

	// Should not error when lock not found
	if err := svc.ReleaseLockIfDisputed(context.Background(), tx, orgID, p); err != nil {
		t.Fatalf("expected no error for missing lock, got: %v", err)
	}
}
