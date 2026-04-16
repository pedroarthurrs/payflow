# Implementation Tracker - Ledger

**Implementation progress tracking document.**

**Last updated:** 2026-04-15
**Owner:** —

---

## Overall Status

**Phase 1 (Data & Domain):** 0% (0/5)
**Phase 2 (Repository):** 0% (0/3)
**Phase 3 (Services):** 0% (0/3)
**Phase 4 (HTTP Layer):** 0% (0/3)
**Phase 5 (Tests):** 0% (0/3)

**Total Progress:** 0/17 (0%)

---

## Phase 1: Data & Domain (5 steps)

### 1.1 Add SQLC queries — wallet.sql
- **Status:** ⬜ Not Started
- **Expected file:** `internal/database/queries/wallet.sql`
- **Verification:**
  ```bash
  cat internal/database/queries/wallet.sql | grep "-- name:" | wc -l
  # Expected: 16
  ```
- **Notes:** See FEATURE_SPEC §3 for the full list of 16 query names.
- **Blocker:** None
- **Associated PR:** #???

### 1.2 Update sqlc.yml and regenerate wallet package
- **Status:** ⬜ Not Started
- **Expected file:** `sqlc.yml`, `internal/database/dbgen/wallet/`
- **Verification:**
  ```bash
  make sqlc
  ls internal/database/dbgen/wallet/
  ```
- **Notes:** Add a new `wallet` entry with `package: "walletdb"`.
- **Blocker:** 1.1
- **Associated PR:** #???

### 1.3 Implement domain types — wallet.go
- **Status:** ⬜ Not Started
- **Expected file:** `internal/domain/wallet.go`
- **Verification:**
  ```bash
  go build ./internal/domain/...
  grep "WalletRepository" internal/domain/wallet.go
  ```
- **Notes:** Full type list in FEATURE_SPEC §7.
- **Blocker:** None
- **Associated PR:** #???

### 1.4 Add new error sentinels
- **Status:** ⬜ Not Started
- **Expected file:** `internal/domain/errors.go`
- **Verification:**
  ```bash
  grep "ErrWalletNotFound\|ErrInsufficientBalance\|ErrWalletSuspended\|ErrPayoutNotFound\|ErrCommissionNotFound\|ErrInvalidAdjustment" internal/domain/errors.go | wc -l
  # Expected: 6
  ```
- **Notes:**
- **Blocker:** None
- **Associated PR:** #???

### 1.5 Add ledger permissions and role assignments
- **Status:** ⬜ Not Started
- **Expected file:** `internal/domain/permission.go`
- **Verification:**
  ```bash
  grep "PermWallets\|PermCommissions" internal/domain/permission.go | wc -l
  # Expected: 5
  ```
- **Notes:** `PermWalletsAdjust` must be Owner-only.
- **Blocker:** None
- **Associated PR:** #???

---

## Phase 2: Repository (3 steps)

### 2.1 Wire WalletQ into database.DB
- **Status:** ⬜ Not Started
- **Expected file:** `internal/database/connection.go`
- **Verification:**
  ```bash
  go build ./internal/database/...
  grep "WalletQ" internal/database/connection.go
  ```
- **Notes:**
- **Blocker:** 1.2
- **Associated PR:** #???

### 2.2 Implement repository/wallet.go
- **Status:** ⬜ Not Started
- **Expected file:** `internal/database/repository/wallet.go`
- **Verification:**
  ```bash
  go build ./internal/database/repository/...
  grep "WalletRepo\b" internal/database/repository/wallet.go
  ```
- **Notes:** `GetWalletByID` and `GetWalletForUpdate` must map `sql.ErrNoRows` → `domain.ErrWalletNotFound`.
- **Blocker:** 2.1
- **Associated PR:** #???

### 2.3 Add WalletRepo() to TxHandle interface and implementation
- **Status:** ⬜ Not Started
- **Expected files:** `internal/domain/tx.go`, `internal/database/repository/tx.go`
- **Verification:**
  ```bash
  go build ./...
  grep "WalletRepo()" internal/domain/tx.go
  ```
- **Notes:**
- **Blocker:** 2.2
- **Associated PR:** #???

---

## Phase 3: Services (3 steps)

### 3.1 Implement service/ledger.go
- **Status:** ⬜ Not Started
- **Expected file:** `internal/service/ledger.go`
- **Verification:**
  ```bash
  go build ./internal/service/...
  grep "CreditOnPaymentSuccess\|DebitOnPaymentRefund" internal/service/ledger.go
  ```
- **Notes:** Ledger errors must be non-fatal (logged, not returned) to avoid blocking FSM transitions.
- **Blocker:** 2.3
- **Associated PR:** #???

### 3.2 Inject LedgerService into PaymentService.Transition
- **Status:** ⬜ Not Started
- **Expected file:** `internal/service/payment.go`
- **Verification:**
  ```bash
  go test ./internal/service/... -run TestPaymentTransition -v
  ```
- **Notes:** See FEATURE_SPEC §7 for the non-fatal hook pattern.
- **Blocker:** 3.1
- **Associated PR:** #???

### 3.3 Implement service/wallet.go
- **Status:** ⬜ Not Started
- **Expected file:** `internal/service/wallet.go`
- **Verification:**
  ```bash
  go build ./internal/service/...
  grep "RequestPayout\|GetOrgWallet\|GetCustomerWallet\|Adjust\|ListEntries" internal/service/wallet.go | wc -l
  # Expected: 5
  ```
- **Notes:** `GetOrgWallet` and `GetCustomerWallet` must use GetOrCreate semantics.
- **Blocker:** 2.3
- **Associated PR:** #???

---

## Phase 4: HTTP Layer (3 steps)

### 4.1 Implement handler/wallet.go (6 routes)
- **Status:** ⬜ Not Started
- **Expected file:** `internal/api/handler/wallet.go`
- **Verification:**
  ```bash
  go build ./internal/api/...
  grep "RegisterRoutes\|RequestPayout\|ListEntries\|Adjust" internal/api/handler/wallet.go
  ```
- **Notes:** See FEATURE_SPEC §8 for request/response shapes and error codes.
- **Blocker:** 3.3
- **Associated PR:** #???

### 4.2 Implement handler/commission.go (3 routes)
- **Status:** ⬜ Not Started
- **Expected file:** `internal/api/handler/commission.go`
- **Verification:**
  ```bash
  go build ./internal/api/...
  grep "RegisterRoutes" internal/api/handler/commission.go
  ```
- **Notes:**
- **Blocker:** 3.3
- **Associated PR:** #???

### 4.3 Wire everything in router.go
- **Status:** ⬜ Not Started
- **Expected file:** `internal/api/router/router.go`
- **Verification:**
  ```bash
  go build ./cmd/server/...
  # Start server and verify:
  curl -s -o /dev/null -w "%{http_code}" http://localhost:8080/v1/wallets/me
  # Expected: 401 (not 404)
  ```
- **Notes:** Register `walletHandler` and `commissionHandler` in the protected `/v1` group.
- **Blocker:** 4.1, 4.2
- **Associated PR:** #???

---

## Phase 5: Tests (3 steps)

### 5.1 Unit tests for LedgerService
- **Status:** ⬜ Not Started
- **Expected file:** `internal/service/ledger_test.go`
- **Verification:**
  ```bash
  go test ./internal/service/... -run TestLedger -v
  ```
- **Notes:** Use mock WalletRepository. Cover: credit on succeed, debit on refund, commission calculation, wallet suspended (skip), zero commission (skip).
- **Blocker:** 3.1
- **Associated PR:** #???

### 5.2 Unit tests for WalletService.RequestPayout
- **Status:** ⬜ Not Started
- **Expected file:** `internal/service/wallet_test.go`
- **Verification:**
  ```bash
  go test ./internal/service/... -run TestWalletPayout -v
  ```
- **Notes:** Cover: happy path, insufficient balance (422), suspended wallet (422).
- **Blocker:** 3.3
- **Associated PR:** #???

### 5.3 Handler tests for WalletHandler
- **Status:** ⬜ Not Started
- **Expected file:** `internal/api/handler/wallet_test.go`
- **Verification:**
  ```bash
  go test ./internal/api/handler/... -run TestWallet -v
  ```
- **Notes:** Table-driven, cover 401/403/422/201/200 for each route.
- **Blocker:** 4.1
- **Associated PR:** #???

---

## General Notes

- Phase dependencies: Phase N+1 does not start until Phase N is at 100%
- Reference document: `FEATURE_SPEC.md` in this folder
- PRs should reference this tracker by step number (e.g. "Step 3.2 — LedgerService hooks")
- Ledger errors are **non-fatal** for the payment FSM — never roll back a payment transition due to ledger issues

---

## Known Risks

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| Ledger/wallet out of sync if non-fatal hook fails silently | Medium | Medium | Add monitoring counter for skipped ledger credits; reconciliation job in future |
| Commission overdraft (commission > payment amount) | Low | Low | `computeCommission` caps at `payment.amount_cents`; log warning |
| Concurrent payout + credit race on wallet balance | Low | High | `GetWalletForUpdate` (SELECT FOR UPDATE) inside transaction ensures serialized updates |
| `sqlc` model naming conflicts with existing `payment` package | Low | Low | Use dedicated `walletdb` package in separate `dbgen/wallet/` folder |

---

## Status Legend

| Icon | Meaning |
|------|---------|
| ✅ | COMPLETE - passed all verifications |
| ⚠️ | PARTIAL - passed some verifications |
| ⬜ | NOT STARTED |
| 🔴 | BLOCKED - prerequisite not met |
