-- Customers
CREATE UNIQUE INDEX IF NOT EXISTS unique_customer_email_per_organization
    ON customers (organization_id, email)
    WHERE deleted_at IS NULL;

CREATE UNIQUE INDEX IF NOT EXISTS unique_customer_doc_per_organization
    ON customers (organization_id, document)
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_customers_organization_id
    ON customers (organization_id);

-- Addresses
CREATE INDEX IF NOT EXISTS idx_addresses_customer_id
    ON addresses (customer_id);

-- Products
CREATE INDEX IF NOT EXISTS idx_products_organization_id
    ON products (organization_id);

-- Plans
CREATE INDEX IF NOT EXISTS idx_plans_product_id
    ON plans (product_id);

-- Subscriptions
CREATE INDEX IF NOT EXISTS idx_subscriptions_customer_id
    ON subscriptions (customer_id);

CREATE INDEX IF NOT EXISTS idx_subscriptions_status
    ON subscriptions (status) WHERE status IN ('active', 'past_due');

CREATE INDEX IF NOT EXISTS idx_subscriptions_period_end
    ON subscriptions (current_period_end);

-- Subscription Items
CREATE INDEX IF NOT EXISTS idx_subscription_items_subscription_id
    ON subscription_items (subscription_id);

CREATE INDEX IF NOT EXISTS idx_subscription_items_plan_id
    ON subscription_items (plan_id);

-- Subscription Status History
CREATE INDEX IF NOT EXISTS idx_sub_history_subscription
    ON subscription_status_history (subscription_id, created_at DESC);

-- Subscription Schedules
CREATE INDEX IF NOT EXISTS idx_subs_schedules_pending
    ON subscription_schedules (scheduled_at) WHERE applied_at IS NULL;

-- Invoices
CREATE INDEX IF NOT EXISTS idx_invoices_organization_created_at
    ON invoices (organization_id, created_at);

CREATE INDEX IF NOT EXISTS idx_invoices_customer_id
    ON invoices (customer_id);

CREATE INDEX IF NOT EXISTS idx_invoices_subscription_id
    ON invoices (subscription_id);

CREATE INDEX IF NOT EXISTS idx_invoices_status
    ON invoices (status) WHERE status IN ('pending', 'error');

-- Invoice Items
CREATE INDEX IF NOT EXISTS idx_invoice_items_invoice_id
    ON invoice_items (invoice_id);

-- Invoice Logs
CREATE INDEX IF NOT EXISTS idx_invoice_logs_invoice_id
    ON invoice_logs (invoice_id);

-- Invoice Settings
CREATE INDEX IF NOT EXISTS idx_invoice_settings_organization_id
    ON invoice_settings (organization_id);

-- Payments
CREATE INDEX IF NOT EXISTS idx_payments_invoice_id
    ON payments (invoice_id);

CREATE INDEX IF NOT EXISTS idx_payments_organization_created_at
    ON payments (organization_id, created_at);

CREATE INDEX IF NOT EXISTS idx_payments_status
    ON payments (status);

CREATE INDEX IF NOT EXISTS idx_payments_customer_id
    ON payments (organization_id, customer_id);

-- Payment Status History
CREATE INDEX IF NOT EXISTS idx_payment_status_history_payment_id
    ON payment_status_history (payment_id, created_at DESC);

-- Credentials
CREATE INDEX IF NOT EXISTS idx_credentials_organization_id
    ON credentials (organization_id);

-- Credential Logs
CREATE INDEX IF NOT EXISTS idx_credential_logs_organization_id
    ON credential_logs (organization_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_credential_logs_credential_id
    ON credential_logs (credential_id);

-- Wallets
CREATE INDEX IF NOT EXISTS idx_wallets_organization_id
    ON wallets (organization_id);

CREATE INDEX IF NOT EXISTS idx_wallets_customer_owner_id
    ON wallets (customer_owner_id) WHERE customer_owner_id IS NOT NULL;

-- Ledger Entries
CREATE INDEX IF NOT EXISTS idx_ledger_entries_wallet_id
    ON ledger_entries (wallet_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_ledger_entries_organization_id
    ON ledger_entries (organization_id);

-- Payouts
CREATE INDEX IF NOT EXISTS idx_payouts_wallet_id
    ON payouts (wallet_id);

CREATE INDEX IF NOT EXISTS idx_payouts_organization_id
    ON payouts (organization_id);

-- Commission Settings
CREATE INDEX IF NOT EXISTS idx_commission_settings_organization_id
    ON commission_settings (organization_id);

CREATE INDEX IF NOT EXISTS idx_commission_settings_customer_id
    ON commission_settings (target_customer_id);

-- Idempotency Keys
CREATE INDEX IF NOT EXISTS idx_idempotency_expires_at
    ON idempotency_keys (expires_at);

-- Recurrence billing
CREATE INDEX IF NOT EXISTS idx_subscriptions_billing
    ON subscriptions(current_period_end)
    WHERE status IN ('active', 'past_due', 'trialing') AND billing_locked_until IS NULL;

-- User
CREATE INDEX IF NOT EXISTS idx_user_email
    ON "user" (email);

-- Organization Members
CREATE INDEX IF NOT EXISTS idx_org_members_org
    ON organization_members (organization_id) WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_org_members_user
    ON organization_members (user_id) WHERE deleted_at IS NULL;

-- Member Permissions
CREATE INDEX IF NOT EXISTS idx_member_permissions_member
    ON member_permissions (member_id);

CREATE INDEX IF NOT EXISTS idx_member_permissions_org
    ON member_permissions (organization_id);

-- Member Logs
CREATE INDEX IF NOT EXISTS idx_member_logs_member
    ON member_logs (member_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_member_logs_org
    ON member_logs (organization_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_member_logs_created_at
    ON member_logs (created_at DESC);