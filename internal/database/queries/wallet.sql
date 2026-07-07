-- name: CreateWallet :one
INSERT INTO wallets (organization_id, organization_owner_id, customer_owner_id, currency)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: GetWalletByID :one
SELECT * FROM wallets
WHERE organization_id = $1 AND id = $2;

-- name: GetWalletByOrganizationOwner :one
SELECT * FROM wallets
WHERE organization_id = $1 AND organization_owner_id = $1;

-- name: GetWalletByCustomerOwner :one
SELECT * FROM wallets
WHERE organization_id = $1 AND customer_owner_id = $2;

-- name: GetWalletForUpdate :one
SELECT * FROM wallets
WHERE organization_id = $1 AND id = $2
FOR UPDATE;

-- name: UpdateWalletBalance :one
UPDATE wallets
SET balance = $3, updated_at = NOW()
WHERE organization_id = $1 AND id = $2
RETURNING *;

-- name: UpdateWalletLockedBalance :one
UPDATE wallets
SET locked_balance = $3, updated_at = NOW()
WHERE organization_id = $1 AND id = $2
RETURNING *;

-- name: InsertLedgerEntry :one
INSERT INTO ledger_entries (
    organization_id, wallet_id, type, amount_cents, category,
    balance_after, subscription_id, invoice_id, payment_id, metadata
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
RETURNING *;

-- name: ListLedgerEntries :many
SELECT * FROM ledger_entries
WHERE organization_id = $1 AND wallet_id = $2
  AND (sqlc.narg('category')::text IS NULL OR category = sqlc.narg('category'))
  AND (sqlc.narg('type')::text IS NULL OR type = sqlc.narg('type'))
  AND (sqlc.narg('created_from')::timestamptz IS NULL OR created_at >= sqlc.narg('created_from'))
  AND (sqlc.narg('created_to')::timestamptz IS NULL OR created_at <= sqlc.narg('created_to'))
ORDER BY created_at DESC
LIMIT $3 OFFSET $4;

-- name: CountLedgerEntries :one
SELECT COUNT(*) FROM ledger_entries
WHERE organization_id = $1 AND wallet_id = $2
  AND (sqlc.narg('category')::text IS NULL OR category = sqlc.narg('category'))
  AND (sqlc.narg('type')::text IS NULL OR type = sqlc.narg('type'))
  AND (sqlc.narg('created_from')::timestamptz IS NULL OR created_at >= sqlc.narg('created_from'))
  AND (sqlc.narg('created_to')::timestamptz IS NULL OR created_at <= sqlc.narg('created_to'));

-- name: CreatePayout :one
INSERT INTO payouts (organization_id, wallet_id, amount_cents, fee_cents, bank_info)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: GetPayoutByID :one
SELECT * FROM payouts
WHERE organization_id = $1 AND id = $2;

-- name: UpdatePayoutStatus :one
UPDATE payouts
SET status = $3, updated_at = NOW()
WHERE organization_id = $1 AND id = $2
RETURNING *;

-- name: ListPayouts :many
SELECT * FROM payouts
WHERE organization_id = $1 AND wallet_id = $2
ORDER BY created_at DESC
LIMIT $3 OFFSET $4;

-- name: UpsertCommissionSetting :one
INSERT INTO commission_settings (
    organization_id, target_customer_id, type, value, conditions, is_active
) VALUES ($1, $2, $3, $4, $5, $6)
ON CONFLICT (organization_id, target_customer_id)
WHERE deleted_at IS NULL
DO UPDATE SET type = EXCLUDED.type, value = EXCLUDED.value, conditions = EXCLUDED.conditions,
              is_active = EXCLUDED.is_active, updated_at = NOW(), deleted_at = NULL
RETURNING *;

-- name: GetCommissionSettingByCustomer :one
SELECT * FROM commission_settings
WHERE organization_id = $1 AND target_customer_id = $2 AND deleted_at IS NULL;

-- name: ListCommissionSettings :many
SELECT * FROM commission_settings
WHERE organization_id = $1 AND deleted_at IS NULL
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: SoftDeleteCommissionSetting :exec
UPDATE commission_settings
SET deleted_at = NOW(), is_active = false, updated_at = NOW()
WHERE organization_id = $1 AND target_customer_id = $2 AND deleted_at IS NULL;

-- name: CreateWalletLock :one
INSERT INTO wallet_locks (organization_id, wallet_id, type, reference_id, amount_cents)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: ReleaseWalletLock :exec
UPDATE wallet_locks
SET status = 'released', released_at = NOW()
WHERE organization_id = $1 AND id = $2 AND status = 'active';

-- name: GetWalletLockByReference :one
SELECT * FROM wallet_locks
WHERE organization_id = $1 AND reference_id = $2 AND type = $3 AND status = 'active';
