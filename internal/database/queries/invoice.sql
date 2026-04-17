-- =============================================================================
-- Invoices
-- =============================================================================

-- name: CreateInvoice :one
INSERT INTO invoices (
    organization_id,
    customer_id,
    subscription_id,
    invoice_type,
    operation_type,
    amount_cents,
    currency,
    status
) VALUES ($1, $2, $3, $4, $5, $6, $7, 'pending')
RETURNING id, organization_id, customer_id, subscription_id,
          invoice_type, operation_type, amount_cents, currency, status,
          erp_reference, customer_snapshot, items_snapshot, last_sync_log,
          created_at, updated_at;

-- name: GetInvoiceByID :one
SELECT id, organization_id, customer_id, subscription_id,
       invoice_type, operation_type, amount_cents, currency, status,
       erp_reference, customer_snapshot, items_snapshot, last_sync_log,
       created_at, updated_at
FROM invoices
WHERE organization_id = $1 AND id = $2
LIMIT 1;

-- name: ListInvoices :many
SELECT id, organization_id, customer_id, subscription_id,
       invoice_type, operation_type, amount_cents, currency, status,
       erp_reference, customer_snapshot, items_snapshot, last_sync_log,
       created_at, updated_at
FROM invoices
WHERE organization_id = $1
  AND (sqlc.narg('customer_id')::uuid IS NULL OR customer_id = sqlc.narg('customer_id'))
  AND (sqlc.narg('subscription_id')::uuid IS NULL OR subscription_id = sqlc.narg('subscription_id'))
  AND (sqlc.narg('status')::varchar IS NULL OR status = sqlc.narg('status'))
  AND (sqlc.narg('invoice_type')::varchar IS NULL OR invoice_type = sqlc.narg('invoice_type'))
  AND (sqlc.narg('created_from')::timestamptz IS NULL OR created_at >= sqlc.narg('created_from'))
  AND (sqlc.narg('created_to')::timestamptz IS NULL OR created_at <= sqlc.narg('created_to'))
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: CountInvoices :one
SELECT count(*) FROM invoices
WHERE organization_id = $1
  AND (sqlc.narg('customer_id')::uuid IS NULL OR customer_id = sqlc.narg('customer_id'))
  AND (sqlc.narg('subscription_id')::uuid IS NULL OR subscription_id = sqlc.narg('subscription_id'))
  AND (sqlc.narg('status')::varchar IS NULL OR status = sqlc.narg('status'))
  AND (sqlc.narg('invoice_type')::varchar IS NULL OR invoice_type = sqlc.narg('invoice_type'))
  AND (sqlc.narg('created_from')::timestamptz IS NULL OR created_at >= sqlc.narg('created_from'))
  AND (sqlc.narg('created_to')::timestamptz IS NULL OR created_at <= sqlc.narg('created_to'));

-- name: UpdateInvoiceStatus :one
UPDATE invoices
SET status = $3, updated_at = NOW()
WHERE organization_id = $1 AND id = $2
RETURNING id, organization_id, customer_id, subscription_id,
          invoice_type, operation_type, amount_cents, currency, status,
          erp_reference, customer_snapshot, items_snapshot, last_sync_log,
          created_at, updated_at;

-- name: UpdateInvoiceERPReference :exec
UPDATE invoices
SET erp_reference = $3, updated_at = NOW()
WHERE organization_id = $1 AND id = $2;

-- name: UpdateInvoiceLastSyncLog :exec
UPDATE invoices
SET last_sync_log = $3, updated_at = NOW()
WHERE organization_id = $1 AND id = $2;

-- =============================================================================
-- Invoice Items
-- =============================================================================

-- name: CreateInvoiceItem :one
INSERT INTO invoice_items (
    organization_id,
    invoice_id,
    description,
    quantity,
    unit_price_cents,
    total_price_cents,
    metadata
) VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: ListInvoiceItemsByInvoice :many
SELECT * FROM invoice_items
WHERE organization_id = $1 AND invoice_id = $2
ORDER BY created_at ASC;

-- name: DeleteInvoiceItemsByInvoice :exec
DELETE FROM invoice_items
WHERE organization_id = $1 AND invoice_id = $2;

-- =============================================================================
-- Invoice Logs
-- =============================================================================

-- name: CreateInvoiceLog :one
INSERT INTO invoice_logs (
    organization_id,
    invoice_id,
    event_type,
    invoice_status,
    erp_reference,
    erp_error_code,
    erp_error_message,
    http_status,
    request_url,
    request_payload,
    response_body,
    attempt_number,
    success,
    message
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
RETURNING *;

-- name: ListInvoiceLogsByInvoice :many
SELECT * FROM invoice_logs
WHERE organization_id = $1 AND invoice_id = $2
ORDER BY created_at DESC
LIMIT $3 OFFSET $4;

-- name: CountInvoiceLogsByInvoice :one
SELECT count(*) FROM invoice_logs
WHERE organization_id = $1 AND invoice_id = $2;

-- =============================================================================
-- Invoice Settings
-- =============================================================================

-- name: UpsertInvoiceSetting :one
INSERT INTO invoice_settings (
    organization_id,
    type,
    erp_api_key,
    erp_api_secret,
    erp_account_id,
    nfse_service_code,
    nfse_item_list_service,
    tax_rate,
    invoice_series,
    iso_currency,
    export_regime,
    issue_on,
    is_active
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
ON CONFLICT (organization_id, type) DO UPDATE SET
    erp_api_key = EXCLUDED.erp_api_key,
    erp_api_secret = EXCLUDED.erp_api_secret,
    erp_account_id = EXCLUDED.erp_account_id,
    nfse_service_code = EXCLUDED.nfse_service_code,
    nfse_item_list_service = EXCLUDED.nfse_item_list_service,
    tax_rate = EXCLUDED.tax_rate,
    invoice_series = EXCLUDED.invoice_series,
    iso_currency = EXCLUDED.iso_currency,
    export_regime = EXCLUDED.export_regime,
    issue_on = EXCLUDED.issue_on,
    is_active = EXCLUDED.is_active,
    updated_at = NOW()
RETURNING *;

-- name: GetInvoiceSettingByOrgAndType :one
SELECT * FROM invoice_settings
WHERE organization_id = $1 AND type = $2
LIMIT 1;

-- name: ListInvoiceSettings :many
SELECT * FROM invoice_settings
WHERE organization_id = $1
ORDER BY created_at DESC;

-- name: DeleteInvoiceSetting :exec
DELETE FROM invoice_settings
WHERE organization_id = $1 AND id = $2;
