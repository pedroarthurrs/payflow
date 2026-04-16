-- Registros financeiros brutos vinculados a faturas.
CREATE TABLE IF NOT EXISTS payments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id),
    customer_id UUID NOT NULL REFERENCES customers(id), -- Associação direta ao cliente.
    invoice_id UUID,
    subscription_id UUID NULL,

    gateway_name VARCHAR(50),       -- Gateway que processou o pagamento.
    gateway_reference VARCHAR(255), -- Identificador externo atribuído pelo gateway.
    payment_method VARCHAR(50),     -- Meio de pagamento utilizado.
    amount_cents INTEGER NOT NULL CHECK (amount_cents >= 0),
    status payment_status_type NOT NULL, -- Estado financeiro do pagamento.
    currency VARCHAR(3) DEFAULT 'BRL',

    idempotency_key VARCHAR(255) UNIQUE, -- Token para blindagem contra retentativas duplicadas.
    error_log JSONB,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),

    CONSTRAINT fk_payments_invoice_organization
        FOREIGN KEY (organization_id, invoice_id)
        REFERENCES invoices (organization_id, id) ON DELETE SET NULL,

    CONSTRAINT fk_payments_subscription_organization
        FOREIGN KEY (organization_id, subscription_id)
        REFERENCES subscriptions (organization_id, id) ON DELETE SET NULL
);

-- Histórico de mudanças de estado das transações.
CREATE TABLE IF NOT EXISTS payment_status_history (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id  UUID NOT NULL REFERENCES organizations(id),
    payment_id   UUID NOT NULL REFERENCES payments(id),
    old_status       payment_status_type,
    new_status       payment_status_type NOT NULL,
    event_source     VARCHAR(50),
    event_source_id  UUID,
    reason           TEXT,
    created_at       TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at       TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Controle de idempotência das requisições.
CREATE TABLE IF NOT EXISTS idempotency_keys (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,

    idempotency_key TEXT NOT NULL,  -- Hash identificador da requisição do consumidor.
    request_path TEXT NOT NULL,     -- Endpoint relacionado ao token gerado.
    response_code INTEGER,          -- Código HTTP cacheado para replay.
    response_body JSONB,            -- Resposta serializada para replay em chamadas repetidas.
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),

    CONSTRAINT unique_idempotency_key_per_organization
        UNIQUE (organization_id, idempotency_key)
);

-- Indexes
CREATE INDEX IF NOT EXISTS idx_payments_invoice_id
    ON payments (invoice_id);

CREATE INDEX IF NOT EXISTS idx_payments_organization_created_at
    ON payments (organization_id, created_at);

CREATE INDEX IF NOT EXISTS idx_payments_status
    ON payments (status);

CREATE INDEX IF NOT EXISTS idx_payments_customer_id
    ON payments (organization_id, customer_id);

CREATE INDEX IF NOT EXISTS idx_payment_status_history_payment_id
    ON payment_status_history (payment_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_idempotency_expires_at
    ON idempotency_keys (expires_at);

-- Triggers
CREATE TRIGGER tr_payments_updated_at
    BEFORE UPDATE ON payments
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER tr_payment_status_history_updated_at
    BEFORE UPDATE ON payment_status_history
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER tr_idempotency_keys_updated_at
    BEFORE UPDATE ON idempotency_keys
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
