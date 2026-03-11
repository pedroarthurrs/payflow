CREATE UNIQUE INDEX IF NOT EXISTS unique_customer_email_per_tenant
    ON customers (tenant_id, email)
    WHERE deleted_at IS NULL;

CREATE UNIQUE INDEX IF NOT EXISTS unique_customer_doc_per_tenant
    ON customers (tenant_id, document)
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_customers_tenant_id
    ON customers (tenant_id);

CREATE INDEX IF NOT EXISTS idx_addresses_customer_id
    ON addresses (customer_id);

CREATE INDEX IF NOT EXISTS idx_products_tenant_id
    ON products (tenant_id);

CREATE INDEX IF NOT EXISTS idx_plans_product_id
    ON plans (product_id);

CREATE INDEX IF NOT EXISTS idx_subscriptions_customer_id
    ON subscriptions (customer_id);

CREATE INDEX IF NOT EXISTS idx_subscription_items_subscription_id
    ON subscription_items (subscription_id);

CREATE INDEX IF NOT EXISTS idx_subscription_items_plan_id
    ON subscription_items (plan_id);

CREATE INDEX IF NOT EXISTS idx_invoices_tenant_created_at
    ON invoices (tenant_id, created_at);

CREATE INDEX IF NOT EXISTS idx_invoices_customer_id
    ON invoices (customer_id);

CREATE INDEX IF NOT EXISTS idx_invoices_subscription_id
    ON invoices (subscription_id);

CREATE INDEX IF NOT EXISTS idx_invoice_items_invoice_id
    ON invoice_items (invoice_id);

CREATE INDEX IF NOT EXISTS idx_invoice_settings_tenant_id
    ON invoice_settings (tenant_id);

CREATE INDEX IF NOT EXISTS idx_transactions_invoice_id
    ON transactions (invoice_id);

CREATE INDEX IF NOT EXISTS idx_transactions_tenant_created_at
    ON transactions (tenant_id, created_at);

CREATE INDEX IF NOT EXISTS idx_idempotency_expires_at
    ON idempotency_keys (expires_at);

CREATE INDEX IF NOT EXISTS idx_balance_customer_tenant
    ON customer_balance_transactions (tenant_id, customer_id);

CREATE INDEX IF NOT EXISTS idx_sub_history_subscription
    ON subscription_status_history (subscription_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_subscriptions_status
    ON subscriptions (status) WHERE status IN ('active', 'past_due');

CREATE INDEX IF NOT EXISTS idx_invoices_status
    ON invoices (status) WHERE status IN ('PENDING', 'ERROR');

CREATE INDEX IF NOT EXISTS idx_subscriptions_period_end
    ON subscriptions (current_period_end);

CREATE INDEX IF NOT EXISTS idx_subs_schedules_pending
    ON subscription_schedules (scheduled_at) WHERE applied_at IS NULL;