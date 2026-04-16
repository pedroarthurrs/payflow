# Ledger

**Status:** Not Started
**Created:** 2026-04-15
**Last updated:** 2026-04-15

---

## 1. Overview

The Ledger feature introduces a financial bookkeeping layer that tracks every monetary movement inside the platform. Each Organization has a Wallet that accumulates balance from successful payments and is debited on Payouts. Customers can also have Wallets to receive commissions. Every balance change produces an immutable `ledger_entry` row that provides a complete audit trail.

The migration (`000011_create_wallets.up.sql`) is already applied and contains the `wallets`, `ledger_entries`, `payouts`, and `commission_settings` tables. The entire Go stack — domain, repository, service, and HTTP handler — is missing and must be built.

The feature integrates with the existing payment FSM: when a payment transitions to `succeeded`, the system credits the org's wallet and, if a `commission_setting` exists for the customer, credits the customer's wallet proportionally. Payouts debit the org wallet and create a `payout` record. Manual adjustments are owner-only and always generate a ledger entry.

**Breaking changes:** none. All new endpoints are additive.

---

## 2. Current State

| Layer | File | Status |
|-------|------|--------|
| Migration | `deploy/migrations/000011_create_wallets.up.sql` | ✅ Exists |
| DB types | `wallets`, `ledger_entries`, `payouts`, `commission_settings` | ✅ Exists |
| Domain types | `internal/domain/wallet.go` | ❌ Missing |
| Domain errors | `internal/domain/errors.go` | ⚠️ Partial (ledger errors missing) |
| Domain permissions | `internal/domain/permission.go` | ⚠️ Partial (ledger perms missing) |
| Domain tx handle | `internal/domain/tx.go` | ⚠️ Partial (wallet repo missing) |
| SQLC queries | `internal/database/queries/wallet.sql` | ❌ Missing |
| SQLC generated | `internal/database/dbgen/wallet/` | ❌ Missing |
| Repository | `internal/database/repository/wallet.go` | ❌ Missing |
| DB connection | `internal/database/connection.go` | ⚠️ Partial (WalletQ missing) |
| Service | `internal/service/wallet.go` | ❌ Missing |
| HTTP Handler | `internal/api/handler/wallet.go` | ❌ Missing |
| Router wiring | `internal/api/router/router.go` | ❌ Missing |
| Payment hook | `internal/service/payment.go` (Transition) | ⚠️ Partial (no ledger write on success) |

**Contract changes:**
- `domain.TxHandle` gains `WalletRepo() WalletRepository`
- `service.PaymentService` gains `LedgerSvc *LedgerService` dependency
- `database.DB` gains `WalletQ walletdb.Querier`

---

## 3. Data Model

Migration already exists as `000011_create_wallets.up.sql`. No new migration is required. Reference schema:

```sql
-- wallets: one per org (organization_owner_id) or one per customer (customer_owner_id)
-- Exactly one of organization_owner_id / customer_owner_id must be set (check constraint).

CREATE TABLE IF NOT EXISTS wallets (
    id                   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id      UUID NOT NULL REFERENCES organizations(id),
    organization_owner_id UUID REFERENCES organizations(id),
    customer_owner_id    UUID REFERENCES customers(id),
    balance              INTEGER NOT NULL DEFAULT 0,  -- stored in cents
    currency             VARCHAR(3) DEFAULT 'BRL',
    status               VARCHAR(20) DEFAULT 'active' CHECK (status IN ('active', 'suspended')),
    created_at           TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at           TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    CONSTRAINT check_only_one_owner CHECK (
        (organization_owner_id IS NOT NULL AND customer_owner_id IS NULL) OR
        (organization_owner_id IS NULL  AND customer_owner_id IS NOT NULL)
    )
);

-- ledger_entries: immutable record of each monetary movement
CREATE TABLE IF NOT EXISTS ledger_entries (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id),
    wallet_id       UUID NOT NULL REFERENCES wallets(id),
    type            VARCHAR(10) NOT NULL CHECK (type IN ('credit', 'debit')),
    amount_cents    INTEGER NOT NULL CHECK (amount_cents > 0),
    category        VARCHAR(30) NOT NULL,  -- payment_split | payout | refund | commission | manual_adjustment
    balance_after   INTEGER NOT NULL,       -- snapshot of wallet.balance after this entry
    subscription_id UUID REFERENCES subscriptions(id),
    invoice_id      UUID REFERENCES invoices(id),
    payment_id      UUID REFERENCES payments(id),
    metadata        JSONB,
    created_at      TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at      TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- payouts: withdrawal requests from a wallet
CREATE TABLE IF NOT EXISTS payouts (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id),
    wallet_id       UUID NOT NULL REFERENCES wallets(id),
    amount_cents    INTEGER NOT NULL CHECK (amount_cents > 0),
    fee_cents       INTEGER DEFAULT 0 CHECK (fee_cents >= 0),
    status          VARCHAR(20) DEFAULT 'pending' CHECK (status IN ('pending', 'processing', 'paid', 'failed')),
    bank_info       JSONB,
    created_at      TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at      TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- commission_settings: rules for crediting customer wallets on payment success
CREATE TABLE IF NOT EXISTS commission_settings (
    id                 UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id    UUID NOT NULL REFERENCES organizations(id),
    target_customer_id UUID NOT NULL REFERENCES customers(id),
    type               VARCHAR(20) NOT NULL CHECK (type IN ('percentage', 'fixed')),
    value              NUMERIC(15,2) NOT NULL,  -- percentage (0–100) or fixed cents
    conditions         JSONB,
    is_active          BOOLEAN DEFAULT TRUE,
    created_at         TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at         TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    deleted_at         TIMESTAMP WITH TIME ZONE
);
```

**Required SQLC queries** (`internal/database/queries/wallet.sql`):

```sql
-- name: CreateWallet :one
-- name: GetWalletByID :one
-- name: GetWalletByOrganizationOwner :one
-- name: GetWalletByCustomerOwner :one
-- name: GetWalletForUpdate :one   (SELECT ... FOR UPDATE)
-- name: UpdateWalletBalance :one
-- name: InsertLedgerEntry :one
-- name: ListLedgerEntries :many
-- name: CountLedgerEntries :one
-- name: CreatePayout :one
-- name: GetPayoutByID :one
-- name: UpdatePayoutStatus :one
-- name: ListPayouts :many
-- name: UpsertCommissionSetting :one
-- name: GetCommissionSettingByCustomer :one
-- name: ListCommissionSettings :many
-- name: SoftDeleteCommissionSetting :exec
```

---

## 4. Models / Hierarchies

### Wallet ownership

```
Organization
  └── Wallet (organization_owner_id)   ← receives payment splits, funds payouts

Customer
  └── Wallet (customer_owner_id)       ← receives commissions
```

### Ledger entry categories

| Category | Direction | Triggered by |
|---|---|---|
| `payment_split` | credit | Payment → `succeeded` |
| `refund` | debit | Payment → `refunded` / `partially_refunded` |
| `commission` | credit | Payment → `succeeded` + active commission_setting |
| `payout` | debit | POST /wallets/:id/payouts |
| `manual_adjustment` | credit or debit | POST /wallets/:id/adjust (owner only) |

### Payout status flow

```
pending → processing → paid
                    → failed
```

---

## 5. Resource / Permission Catalog

| Permission | Constant | Owner | Admin | Developer | Member |
|---|---|---|---|---|---|
| `wallets:read` | `PermWalletsRead` | ✅ | ✅ | ✅ | ❌ |
| `wallets:payout` | `PermWalletsPayout` | ✅ | ✅ | ❌ | ❌ |
| `wallets:adjust` | `PermWalletsAdjust` | ✅ | ❌ | ❌ | ❌ |
| `commissions:read` | `PermCommissionsRead` | ✅ | ✅ | ✅ | ❌ |
| `commissions:manage` | `PermCommissionsManage` | ✅ | ✅ | ❌ | ❌ |

---

## 6. Main Flows

### Flow A — Payment succeeds → credit org wallet

1. `PaymentService.Transition` is called with `toStatus = succeeded`.
2. Inside the same transaction, after `UpdateStatus`, `LedgerService.CreditOnPaymentSuccess` is called.
3. `CreditOnPaymentSuccess`:
   a. Loads the org wallet via `GetWalletForUpdate` (SELECT FOR UPDATE).
   b. Computes `newBalance = wallet.balance + payment.amount_cents`.
   c. Calls `UpdateWalletBalance`.
   d. Inserts a `ledger_entries` row: `type=credit`, `category=payment_split`, `balance_after=newBalance`, `payment_id=payment.id`.
   e. Checks for an active `commission_setting` for `payment.customer_id`.
   f. If found, computes commission amount (percentage × payment or fixed cents).
   g. Gets/creates the customer wallet (`GetOrCreateCustomerWallet`).
   h. Credits customer wallet similarly: `type=credit`, `category=commission`.
4. Transaction commits; both wallets updated atomically.

**Decision:** if commission > `payment.amount_cents`, skip commission (log warning).

**Error cases:**
- Wallet not found → `ErrWalletNotFound` → HTTP 404 (internal; logged, not surfaced to API caller)
- Wallet suspended → `ErrWalletSuspended` → ledger entry is skipped, warning logged

### Flow B — Payment refunded → debit org wallet

1. `PaymentService.Transition` with `toStatus = refunded` or `partially_refunded`.
2. `LedgerService.DebitOnPaymentRefund` called in same transaction.
3. Loads wallet for update, checks `balance >= refund_amount`; if not, continues anyway but logs overdraft.
4. Inserts `type=debit`, `category=refund`, `balance_after=newBalance`.

### Flow C — Payout request

**Request:** `POST /v1/wallets/:wallet_id/payouts`

1. Auth check: `PermWalletsPayout`.
2. Load wallet; verify `status=active`.
3. Verify `wallet.balance >= amount_cents + fee_cents`; if not → `ErrInsufficientBalance` → HTTP 422.
4. Within transaction:
   a. `GetWalletForUpdate`.
   b. Insert `payouts` row with `status=pending`.
   c. Debit wallet: `UpdateWalletBalance(balance - amount - fee)`.
   d. Insert ledger entry: `type=debit`, `category=payout`, `metadata={payout_id}`.
5. Return payout object HTTP 201.

**Error cases:**

| Condition | Error | HTTP |
|---|---|---|
| Wallet not found | `ErrWalletNotFound` | 404 |
| Wallet suspended | `ErrWalletSuspended` | 422 |
| Insufficient balance | `ErrInsufficientBalance` | 422 |

### Flow D — Manual adjustment (owner only)

**Request:** `POST /v1/wallets/:wallet_id/adjust`

1. Auth check: `PermWalletsAdjust`.
2. Validate: `direction ∈ {credit, debit}`, `amount_cents > 0`, `reason` non-empty.
3. Within transaction:
   a. For debit: verify `balance >= amount_cents`.
   b. Insert ledger entry, update balance.
4. Return updated wallet HTTP 200.

### Flow E — List ledger entries

**Request:** `GET /v1/wallets/:wallet_id/entries`

Filters: `category`, `type`, `created_from`, `created_to`, `page`, `per_page`.

Returns paginated list, most recent first.

---

## 7. Technical Requirements

### `internal/domain/wallet.go` (new file)

```go
package domain

import (
    "context"
    "encoding/json"
    "time"
    "github.com/google/uuid"
)

type WalletStatus string

const (
    WalletStatusActive    WalletStatus = "active"
    WalletStatusSuspended WalletStatus = "suspended"
)

type LedgerEntryType string

const (
    LedgerCredit LedgerEntryType = "credit"
    LedgerDebit  LedgerEntryType = "debit"
)

type LedgerCategory string

const (
    CategoryPaymentSplit     LedgerCategory = "payment_split"
    CategoryPayout           LedgerCategory = "payout"
    CategoryRefund           LedgerCategory = "refund"
    CategoryCommission       LedgerCategory = "commission"
    CategoryManualAdjustment LedgerCategory = "manual_adjustment"
)

type PayoutStatus string

const (
    PayoutPending    PayoutStatus = "pending"
    PayoutProcessing PayoutStatus = "processing"
    PayoutPaid       PayoutStatus = "paid"
    PayoutFailed     PayoutStatus = "failed"
)

type CommissionType string

const (
    CommissionPercentage CommissionType = "percentage"
    CommissionFixed      CommissionType = "fixed"
)

// Wallet is the balance container for an org or customer.
type Wallet struct {
    ID                   uuid.UUID
    OrganizationID       uuid.UUID
    OrganizationOwnerID  *uuid.UUID
    CustomerOwnerID      *uuid.UUID
    Balance              int32
    Currency             string
    Status               WalletStatus
    CreatedAt            time.Time
    UpdatedAt            time.Time
}

// LedgerEntry is an immutable monetary movement record.
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

// Payout represents a withdrawal request.
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

// CreateWalletInput is the input for creating a wallet.
type CreateWalletInput struct {
    OrganizationOwnerID *uuid.UUID
    CustomerOwnerID     *uuid.UUID
    Currency            string  // defaults to "BRL"
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

// AdjustWalletInput is the input for a manual balance adjustment.
type AdjustWalletInput struct {
    Direction   LedgerEntryType  // credit or debit
    AmountCents int32
    Reason      string
}

// LedgerListParams are filters for listing ledger entries.
type LedgerListParams struct {
    ListParams
    Category    *LedgerCategory
    Type        *LedgerEntryType
    PaymentID   *uuid.UUID
    CreatedFrom *time.Time
    CreatedTo   *time.Time
}

// WalletRepository defines persistence operations for wallets and ledger entries.
type WalletRepository interface {
    CreateWallet(ctx context.Context, organizationID uuid.UUID, input CreateWalletInput) (*Wallet, error)
    GetWalletByID(ctx context.Context, organizationID, walletID uuid.UUID) (*Wallet, error)
    GetWalletByOrganizationOwner(ctx context.Context, organizationID uuid.UUID) (*Wallet, error)
    GetWalletByCustomerOwner(ctx context.Context, organizationID, customerID uuid.UUID) (*Wallet, error)
    GetWalletForUpdate(ctx context.Context, organizationID, walletID uuid.UUID) (*Wallet, error)
    UpdateWalletBalance(ctx context.Context, organizationID, walletID uuid.UUID, newBalance int32) (*Wallet, error)
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
}
```

### `internal/domain/errors.go` — additions

```go
ErrWalletNotFound        = errors.New("wallet not found")
ErrWalletSuspended       = errors.New("wallet is suspended")
ErrInsufficientBalance   = errors.New("insufficient wallet balance")
ErrCommissionNotFound    = errors.New("commission setting not found")
ErrPayoutNotFound        = errors.New("payout not found")
ErrInvalidAdjustment     = errors.New("invalid manual adjustment")
```

### `internal/domain/permission.go` — additions

```go
PermWalletsRead      Permission = "wallets:read"
PermWalletsPayout    Permission = "wallets:payout"
PermWalletsAdjust    Permission = "wallets:adjust"
PermCommissionsRead  Permission = "commissions:read"
PermCommissionsManage Permission = "commissions:manage"
```

Add to `roleDefaultPermissions`:
- `RoleOwner` / `RoleAdmin`: `PermWalletsRead`, `PermWalletsPayout`, `PermCommissionsRead`, `PermCommissionsManage`
- `RoleOwner` only: `PermWalletsAdjust`
- `RoleDeveloper`: `PermWalletsRead`, `PermCommissionsRead`

### `internal/domain/tx.go` — addition

```go
// TxHandle gains:
WalletRepo() WalletRepository
```

### `internal/database/queries/wallet.sql` (new file)

Full SQLC annotated queries for all methods listed in `WalletRepository`. Key patterns:
- `GetWalletForUpdate`: `SELECT ... FROM wallets WHERE ... FOR UPDATE`
- `UpdateWalletBalance`: `UPDATE wallets SET balance=$3, updated_at=NOW() WHERE organization_id=$1 AND id=$2 RETURNING *`
- `InsertLedgerEntry`: `INSERT INTO ledger_entries (...) VALUES (...) RETURNING *`

### `internal/database/dbgen/wallet/` (generated)

Run `make sqlc` after adding `wallet.sql` to `sqlc.yml` under a new `wallet` package.

Update `sqlc.yml`:
```yaml
- queries: "internal/database/queries/wallet.sql"
  schema: "deploy/migrations"
  out: "internal/database/dbgen/wallet"
  package: "walletdb"
```

### `internal/database/repository/wallet.go` (new file)

```go
// WalletRepo implements domain.WalletRepository backed by PostgreSQL via sqlc.
type WalletRepo struct {
    q walletdb.Querier
}

func NewWalletRepo(q walletdb.Querier) *WalletRepo

// All methods delegate to r.q.* and map rows to domain types.
// GetWalletByID / GetWalletForUpdate return ErrWalletNotFound on sql.ErrNoRows.
```

### `internal/database/connection.go` — addition

```go
WalletQ walletdb.Querier
```

Wired in `New()` alongside existing queriers.

### `internal/database/repository/tx.go` — addition

```go
// TxHandle implementation gains:
func (h *txHandle) WalletRepo() domain.WalletRepository {
    return NewWalletRepo(walletdb.New(h.tx))
}
```

### `internal/service/wallet.go` (new file)

```go
// WalletService handles wallet read/write operations and manual adjustments.
type WalletService struct {
    wallets domain.WalletRepository
    txMgr   domain.TxManager
}

func NewWalletService(wallets domain.WalletRepository, txMgr domain.TxManager) *WalletService

// GetOrgWallet returns the org-owned wallet; creates one if missing.
func (s *WalletService) GetOrgWallet(ctx context.Context, organizationID uuid.UUID) (*domain.Wallet, error)

// GetCustomerWallet returns the customer-owned wallet; creates one if missing (GetOrCreate).
func (s *WalletService) GetCustomerWallet(ctx context.Context, organizationID, customerID uuid.UUID) (*domain.Wallet, error)

// GetByID returns any wallet by ID scoped to org.
func (s *WalletService) GetByID(ctx context.Context, organizationID, walletID uuid.UUID) (*domain.Wallet, error)

// Adjust applies a manual credit or debit within a transaction.
func (s *WalletService) Adjust(ctx context.Context, organizationID, walletID uuid.UUID, input domain.AdjustWalletInput) (*domain.Wallet, error)

// RequestPayout deducts balance and inserts payout record.
func (s *WalletService) RequestPayout(ctx context.Context, organizationID uuid.UUID, input domain.CreatePayoutInput) (*domain.Payout, error)

// ListEntries returns paginated ledger entries for a wallet.
func (s *WalletService) ListEntries(ctx context.Context, organizationID, walletID uuid.UUID, params domain.LedgerListParams) ([]domain.LedgerEntry, int64, error)

// ListPayouts returns paginated payouts for a wallet.
func (s *WalletService) ListPayouts(ctx context.Context, organizationID, walletID uuid.UUID, params domain.ListParams) ([]domain.Payout, error)
```

### `internal/service/ledger.go` (new file)

```go
// LedgerService is the hook called by PaymentService on state transitions.
// It has no HTTP handler — it is injected into PaymentService.
type LedgerService struct {
    wallets domain.WalletRepository
    commissions domain.WalletRepository  // same repo, different method group
}

func NewLedgerService(wallets domain.WalletRepository) *LedgerService

// CreditOnPaymentSuccess credits the org wallet and, optionally, the customer wallet.
// Must be called inside an existing transaction (receives domain.TxHandle).
func (s *LedgerService) CreditOnPaymentSuccess(ctx context.Context, tx domain.TxHandle, organizationID uuid.UUID, p *domain.Payment) error

// DebitOnPaymentRefund debits the org wallet.
// Must be called inside an existing transaction.
func (s *LedgerService) DebitOnPaymentRefund(ctx context.Context, tx domain.TxHandle, organizationID uuid.UUID, p *domain.Payment) error

// computeCommission returns the commission amount in cents.
// Returns 0 if no active setting exists.
func (s *LedgerService) computeCommission(setting *domain.CommissionSetting, amountCents int32) int32
```

### `internal/service/payment.go` — modifications

Add `LedgerSvc *LedgerService` field to `PaymentService`.

In `Transition`, after `InsertHistory`, add:
```go
switch toStatus {
case domain.StatusSucceeded:
    if err := s.ledger.CreditOnPaymentSuccess(ctx, tx, organizationID, payment); err != nil {
        slog.Warn("ledger credit failed, payment still succeeded", "error", err)
        // Non-fatal: payment FSM must not roll back due to ledger issues.
    }
case domain.StatusRefunded, domain.StatusPartiallyRefunded:
    if err := s.ledger.DebitOnPaymentRefund(ctx, tx, organizationID, payment); err != nil {
        slog.Warn("ledger debit failed on refund", "error", err)
    }
}
```

> Note: ledger errors are non-fatal for the FSM transition — they are logged and do not roll back the payment status change. This avoids coupling payment finality to wallet consistency.

### `internal/api/handler/wallet.go` (new file)

```go
// WalletHandler handles wallet, ledger, and payout HTTP endpoints.
type WalletHandler struct {
    walletSvc *service.WalletService
}

func NewWalletHandler(walletSvc *service.WalletService) *WalletHandler

func (h *WalletHandler) RegisterRoutes(rg *gin.RouterGroup)
// Routes:
//   GET    /wallets/me                            → GetOrgWallet
//   GET    /wallets/:wallet_id                    → GetByID
//   GET    /wallets/:wallet_id/entries            → ListEntries
//   POST   /wallets/:wallet_id/payouts            → RequestPayout
//   GET    /wallets/:wallet_id/payouts            → ListPayouts
//   POST   /wallets/:wallet_id/adjust             → Adjust
```

### `internal/api/handler/commission.go` (new file)

```go
// CommissionHandler handles commission_settings CRUD.
type CommissionHandler struct {
    wallets domain.WalletRepository
}

func NewCommissionHandler(walletRepo domain.WalletRepository) *CommissionHandler

func (h *CommissionHandler) RegisterRoutes(rg *gin.RouterGroup)
// Routes:
//   GET    /commissions              → ListCommissionSettings
//   PUT    /commissions/:customer_id → UpsertCommissionSetting
//   DELETE /commissions/:customer_id → SoftDeleteCommissionSetting
```

---

## 8. Expected API

| Method | Route | Description | Permission |
|--------|-------|-------------|------------|
| GET | `/v1/wallets/me` | Get org wallet | `wallets:read` |
| GET | `/v1/wallets/:wallet_id` | Get any wallet by ID | `wallets:read` |
| GET | `/v1/wallets/:wallet_id/entries` | List ledger entries | `wallets:read` |
| POST | `/v1/wallets/:wallet_id/payouts` | Request payout | `wallets:payout` |
| GET | `/v1/wallets/:wallet_id/payouts` | List payouts | `wallets:read` |
| POST | `/v1/wallets/:wallet_id/adjust` | Manual adjustment | `wallets:adjust` |
| GET | `/v1/commissions` | List commission settings | `commissions:read` |
| PUT | `/v1/commissions/:customer_id` | Upsert commission setting | `commissions:manage` |
| DELETE | `/v1/commissions/:customer_id` | Delete commission setting | `commissions:manage` |

### GET `/v1/wallets/me`

**Response 200:**
```json
{
  "id": "uuid",
  "organization_id": "uuid",
  "balance": 150000,
  "currency": "BRL",
  "status": "active",
  "created_at": "...",
  "updated_at": "..."
}
```

### GET `/v1/wallets/:wallet_id/entries`

**Query params:** `category`, `type`, `created_from`, `created_to`, `page` (default 1), `per_page` (default 20, max 100)

**Response 200:**
```json
{
  "data": [
    {
      "id": "uuid",
      "wallet_id": "uuid",
      "type": "credit",
      "amount_cents": 5000,
      "category": "payment_split",
      "balance_after": 150000,
      "payment_id": "uuid",
      "created_at": "..."
    }
  ],
  "total": 42,
  "page": 1,
  "per_page": 20
}
```

### POST `/v1/wallets/:wallet_id/payouts`

**Request:**
```json
{
  "amount_cents": 50000,
  "fee_cents": 500,
  "bank_info": { "bank": "341", "branch": "0001", "account": "12345-6" }
}
```

**Response 201:**
```json
{
  "id": "uuid",
  "wallet_id": "uuid",
  "amount_cents": 50000,
  "fee_cents": 500,
  "status": "pending",
  "bank_info": { "bank": "341", "branch": "0001", "account": "12345-6" },
  "created_at": "..."
}
```

**Errors:**
- `422 Unprocessable Entity` — `ErrInsufficientBalance`, `ErrWalletSuspended`
- `404 Not Found` — `ErrWalletNotFound`

### POST `/v1/wallets/:wallet_id/adjust`

**Request:**
```json
{
  "direction": "credit",
  "amount_cents": 1000,
  "reason": "Manual correction for invoice #X"
}
```

**Response 200:** updated wallet object (same shape as GET `/wallets/me`).

**Errors:**
- `422` — `ErrInsufficientBalance` (for debit), `ErrInvalidAdjustment`
- `403` — missing `wallets:adjust` permission

### PUT `/v1/commissions/:customer_id`

**Request:**
```json
{
  "type": "percentage",
  "value": 10.0,
  "conditions": {}
}
```

**Response 200:** commission setting object.

---

## 9. Impact Mapping

| Existing Endpoint / Component | Change | New Behavior |
|---|---|---|
| `POST /v1/payments/:id/refund` | Indirect (via FSM Transition) | Triggers `DebitOnPaymentRefund` |
| `POST /v1/payments` (gateway charge, succeeds) | Indirect (via FSM Transition on webhook) | Triggers `CreditOnPaymentSuccess` |
| `service.PaymentService.Transition` | New `LedgerSvc` dependency | Calls ledger hooks post-transition |
| `domain.TxHandle` | New method `WalletRepo()` | All transaction impls must expose wallet repo |
| `database.DB` | New `WalletQ` field | Wired in `connection.go` |
| `router.go` | New repos, services, handlers | `walletHandler`, `commissionHandler` registered |
| `domain.permission.go` | New permissions | 5 new constants + role assignments |

---

## 10. Recommended Implementation Order

### Phase 1: Data & Domain (no deps)

**1.1** — Add `wallet.sql` queries to `internal/database/queries/wallet.sql`
- File: `internal/database/queries/wallet.sql`
- Verification: `cat internal/database/queries/wallet.sql | grep "-- name:"` shows all 16 query names

**1.2** — Update `sqlc.yml` with wallet package config and regenerate
- File: `sqlc.yml`, `internal/database/dbgen/wallet/`
- Verification: `make sqlc` exits 0; `ls internal/database/dbgen/wallet/` shows generated files

**1.3** — Add domain types and errors to `internal/domain/wallet.go`
- File: `internal/domain/wallet.go`
- Verification: `go build ./internal/domain/...` exits 0

**1.4** — Add new error sentinels to `internal/domain/errors.go`
- File: `internal/domain/errors.go`
- Verification: `grep "ErrWalletNotFound\|ErrInsufficientBalance" internal/domain/errors.go`

**1.5** — Add ledger permissions + role assignments to `internal/domain/permission.go`
- File: `internal/domain/permission.go`
- Verification: `grep "PermWallets\|PermCommissions" internal/domain/permission.go`

---

### Phase 2: Repository (depends on Phase 1)

**2.1** — Add `WalletQ walletdb.Querier` to `database.DB` and wire in `New()`
- File: `internal/database/connection.go`
- Verification: `go build ./internal/database/...` exits 0

**2.2** — Implement `internal/database/repository/wallet.go`
- File: `internal/database/repository/wallet.go`
- Verification: `go build ./internal/database/repository/...` exits 0

**2.3** — Add `WalletRepo()` to `domain.TxHandle` interface and implement in `repository/tx.go`
- Files: `internal/domain/tx.go`, `internal/database/repository/tx.go`
- Verification: `go build ./...` exits 0

---

### Phase 3: Services (depends on Phase 2)

**3.1** — Implement `internal/service/ledger.go` with `CreditOnPaymentSuccess` and `DebitOnPaymentRefund`
- File: `internal/service/ledger.go`
- Verification: `go build ./internal/service/...` exits 0

**3.2** — Inject `LedgerService` into `PaymentService`; call hooks in `Transition`
- File: `internal/service/payment.go`
- Verification: `go test ./internal/service/... -run TestPaymentTransition` passes

**3.3** — Implement `internal/service/wallet.go`
- File: `internal/service/wallet.go`
- Verification: `go build ./internal/service/...` exits 0

---

### Phase 4: HTTP Layer (depends on Phase 3)

**4.1** — Implement `internal/api/handler/wallet.go` with all 6 routes
- File: `internal/api/handler/wallet.go`
- Verification: `go build ./internal/api/...` exits 0

**4.2** — Implement `internal/api/handler/commission.go` with 3 routes
- File: `internal/api/handler/commission.go`
- Verification: `go build ./internal/api/...` exits 0

**4.3** — Wire repos, services, handlers in `router.go`; register routes under `/v1`
- File: `internal/api/router/router.go`
- Verification: `go build ./cmd/server/...` exits 0; `curl /v1/wallets/me` returns 401 (not 404)

---

### Phase 5: Tests (depends on Phase 4)

**5.1** — Unit tests for `LedgerService` (mock WalletRepository)
- File: `internal/service/ledger_test.go`
- Verification: `go test ./internal/service/... -run TestLedger` passes

**5.2** — Unit tests for `WalletService.RequestPayout` covering insufficient balance
- File: `internal/service/wallet_test.go`
- Verification: `go test ./internal/service/... -run TestWallet` passes

**5.3** — Handler tests for `WalletHandler` (table-driven, 401/403/422/201)
- File: `internal/api/handler/wallet_test.go`
- Verification: `go test ./internal/api/handler/... -run TestWallet` passes
