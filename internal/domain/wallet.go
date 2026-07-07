package domain

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// WalletStatus represents the status of a wallet.
type WalletStatus string

// LedgerEntryType represents the direction of a ledger entry.
type LedgerEntryType string

// LedgerCategory represents the category of a ledger entry.
type LedgerCategory string

// PayoutStatus represents the status of a payout.
type PayoutStatus string

// CommissionType represents the type of a commission setting.
type CommissionType string

// WalletLockType represents the type of a wallet lock.
type WalletLockType string

// WalletLockStatus represents the status of a wallet lock.
type WalletLockStatus string

// Wallet status constants.
const (
	WalletStatusActive    WalletStatus = "active"
	WalletStatusSuspended WalletStatus = "suspended"

	LedgerCredit LedgerEntryType = "credit"
	LedgerDebit  LedgerEntryType = "debit"

	CategoryPaymentSplit     LedgerCategory = "payment_split"
	CategoryPayout           LedgerCategory = "payout"
	CategoryRefund           LedgerCategory = "refund"
	CategoryCommission       LedgerCategory = "commission"
	CategoryManualAdjustment LedgerCategory = "manual_adjustment"

	PayoutPending    PayoutStatus = "pending"
	PayoutProcessing PayoutStatus = "processing"
	PayoutPaid       PayoutStatus = "paid"
	PayoutFailed     PayoutStatus = "failed"

	CommissionPercentage CommissionType = "percentage"
	CommissionFixed      CommissionType = "fixed"

	LockTypeDispute WalletLockType = "dispute"
	LockTypePayout  WalletLockType = "payout"

	LockStatusActive   WalletLockStatus = "active"
	LockStatusReleased WalletLockStatus = "released"
)

// Wallet represents a wallet entity in the domain layer.
type Wallet struct {
	ID                  uuid.UUID
	OrganizationID      uuid.UUID
	OrganizationOwnerID *uuid.UUID
	CustomerOwnerID     *uuid.UUID
	Balance             int32
	LockedBalance       int32
	Currency            string
	Status              WalletStatus
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

// LedgerEntry represents an immutable record of a monetary movement.
type LedgerEntry struct {
	ID             uuid.UUID
	OrganizationID uuid.UUID
	WalletID       uuid.UUID
	Type           LedgerEntryType
	AmountCents    int32
	Category       LedgerCategory
	BalanceAfter   int32
	SubscriptionID *uuid.UUID
	InvoiceID      *uuid.UUID
	PaymentID      *uuid.UUID
	Metadata       json.RawMessage
	CreatedAt      time.Time
}

// Payout represents a withdrawal request from a wallet.
type Payout struct {
	ID             uuid.UUID
	OrganizationID uuid.UUID
	WalletID       uuid.UUID
	AmountCents    int32
	FeeCents       int32
	Status         PayoutStatus
	BankInfo       json.RawMessage
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// CommissionSetting defines how a customer earns commission on payments.
type CommissionSetting struct {
	ID               uuid.UUID
	OrganizationID   uuid.UUID
	TargetCustomerID uuid.UUID
	Type             CommissionType
	Value            float64
	Conditions       json.RawMessage
	IsActive         bool
	CreatedAt        time.Time
	UpdatedAt        time.Time
	DeletedAt        *time.Time
}

// WalletLock represents a temporary balance reservation.
type WalletLock struct {
	ID             uuid.UUID
	OrganizationID uuid.UUID
	WalletID       uuid.UUID
	Type           WalletLockType
	ReferenceID    uuid.UUID
	AmountCents    int32
	Status         WalletLockStatus
	CreatedAt      time.Time
	ReleasedAt     *time.Time
}

// CreateWalletInput is the input for creating a wallet.
type CreateWalletInput struct {
	OrganizationOwnerID *uuid.UUID
	CustomerOwnerID     *uuid.UUID
	Currency            string
}

// LedgerEntryInput is the input for inserting a ledger entry.
type LedgerEntryInput struct {
	WalletID       uuid.UUID
	Type           LedgerEntryType
	AmountCents    int32
	Category       LedgerCategory
	BalanceAfter   int32
	SubscriptionID *uuid.UUID
	InvoiceID      *uuid.UUID
	PaymentID      *uuid.UUID
	Metadata       json.RawMessage
}

// CreatePayoutInput is the input for creating a payout.
type CreatePayoutInput struct {
	WalletID    uuid.UUID
	AmountCents int32
	FeeCents    int32
	BankInfo    json.RawMessage
}

// CreateWalletLockInput is the input for creating a wallet lock.
type CreateWalletLockInput struct {
	WalletID    uuid.UUID
	Type        WalletLockType
	ReferenceID uuid.UUID
	AmountCents int32
}

// AdjustWalletInput is the input for a manual wallet adjustment.
type AdjustWalletInput struct {
	Direction   LedgerEntryType
	AmountCents int32
	Reason      string
}

// LedgerListParams carries optional filters for listing ledger entries.
type LedgerListParams struct {
	ListParams
	Category    *LedgerCategory
	Type        *LedgerEntryType
	PaymentID   *uuid.UUID
	CreatedFrom *time.Time
	CreatedTo   *time.Time
}

// WalletRepository defines persistence operations for the wallet domain.
type WalletRepository interface {
	CreateWallet(ctx context.Context, organizationID uuid.UUID, input CreateWalletInput) (*Wallet, error)
	GetWalletByID(ctx context.Context, organizationID, walletID uuid.UUID) (*Wallet, error)
	GetWalletByOrganizationOwner(ctx context.Context, organizationID uuid.UUID) (*Wallet, error)
	GetWalletByCustomerOwner(ctx context.Context, organizationID, customerID uuid.UUID) (*Wallet, error)
	GetWalletForUpdate(ctx context.Context, organizationID, walletID uuid.UUID) (*Wallet, error)
	UpdateWalletBalance(ctx context.Context, organizationID, walletID uuid.UUID, newBalance int32) (*Wallet, error)
	UpdateWalletLockedBalance(ctx context.Context, organizationID, walletID uuid.UUID, newLockedBalance int32) (*Wallet, error)
	InsertLedgerEntry(ctx context.Context, organizationID uuid.UUID, input LedgerEntryInput) (*LedgerEntry, error)
	ListLedgerEntries(ctx context.Context, organizationID, walletID uuid.UUID, params LedgerListParams) ([]LedgerEntry, error)
	CountLedgerEntries(ctx context.Context, organizationID, walletID uuid.UUID, params LedgerListParams) (int64, error)
	CreatePayout(ctx context.Context, organizationID uuid.UUID, input CreatePayoutInput) (*Payout, error)
	GetPayoutByID(ctx context.Context, organizationID, payoutID uuid.UUID) (*Payout, error)
	UpdatePayoutStatus(ctx context.Context, organizationID, payoutID uuid.UUID, status PayoutStatus) (*Payout, error)
	ListPayouts(ctx context.Context, organizationID, walletID uuid.UUID, params ListParams) ([]Payout, error)
	UpsertCommissionSetting(ctx context.Context, organizationID uuid.UUID, input CommissionSetting) (*CommissionSetting, error)
	GetCommissionSettingByCustomer(ctx context.Context, organizationID, customerID uuid.UUID) (*CommissionSetting, error)
	ListCommissionSettings(ctx context.Context, organizationID uuid.UUID, params ListParams) ([]CommissionSetting, error)
	SoftDeleteCommissionSetting(ctx context.Context, organizationID, settingID uuid.UUID) error
	// Locks
	CreateWalletLock(ctx context.Context, organizationID uuid.UUID, input CreateWalletLockInput) (*WalletLock, error)
	ReleaseWalletLock(ctx context.Context, organizationID, lockID uuid.UUID) error
	GetWalletLockByReference(ctx context.Context, organizationID uuid.UUID, referenceID uuid.UUID, lockType WalletLockType) (*WalletLock, error)
}
