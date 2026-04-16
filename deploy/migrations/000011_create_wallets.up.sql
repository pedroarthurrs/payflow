CREATE TABLE IF NOT EXISTS wallets (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id),

    organization_owner_id UUID REFERENCES organizations(id),
    customer_owner_id UUID REFERENCES customers(id),

    balance INTEGER NOT NULL DEFAULT 0,
    locked_balance INTEGER NOT NULL DEFAULT 0,
    currency VARCHAR(3) DEFAULT 'BRL',
    status VARCHAR(20) DEFAULT 'active' CHECK (status IN ('active', 'suspended')),

    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),

    -- Restrição para garantir que a carteira tenha exatamente UM dono
    CONSTRAINT check_only_one_owner CHECK (
        (organization_owner_id IS NOT NULL AND customer_owner_id IS NULL) OR
        (organization_owner_id IS NULL AND customer_owner_id IS NOT NULL)
    )
);

CREATE TABLE IF NOT EXISTS ledger_entries (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id),
    wallet_id UUID NOT NULL REFERENCES wallets(id),

    type VARCHAR(10) NOT NULL CHECK (type IN ('credit', 'debit')), -- Direção monetária da movimentação.
    amount_cents INTEGER NOT NULL CHECK (amount_cents > 0),
    category VARCHAR(30) NOT NULL, -- payment_split, payout, refund, manual_adjustment
    balance_after INTEGER NOT NULL,

    subscription_id UUID REFERENCES subscriptions(id),
    invoice_id UUID REFERENCES invoices(id),
    payment_id UUID REFERENCES payments(id),

    metadata JSONB,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),

    CONSTRAINT fk_ledger_wallet_organization
        FOREIGN KEY (wallet_id) REFERENCES wallets (id)
);

CREATE TABLE IF NOT EXISTS payouts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id),

    wallet_id UUID NOT NULL REFERENCES wallets(id),
    amount_cents INTEGER NOT NULL CHECK (amount_cents > 0),
    fee_cents INTEGER DEFAULT 0 CHECK (fee_cents >= 0),
    status VARCHAR(20) DEFAULT 'pending' CHECK (status IN ('pending', 'processing', 'paid', 'failed')),
    bank_info JSONB,

    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS commission_settings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id),

    target_customer_id UUID NOT NULL REFERENCES customers(id),

    type VARCHAR(20) NOT NULL CHECK (type IN ('percentage', 'fixed')),
    value NUMERIC(15,2) NOT NULL, -- Valor ou Porcentagem
    conditions JSONB,

    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    deleted_at TIMESTAMP WITH TIME ZONE
);

-- Indexes
CREATE INDEX IF NOT EXISTS idx_wallets_organization_id
    ON wallets (organization_id);

CREATE INDEX IF NOT EXISTS idx_wallets_customer_owner_id
    ON wallets (customer_owner_id) WHERE customer_owner_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_ledger_entries_wallet_id
    ON ledger_entries (wallet_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_ledger_entries_organization_id
    ON ledger_entries (organization_id);

CREATE INDEX IF NOT EXISTS idx_payouts_wallet_id
    ON payouts (wallet_id);

CREATE INDEX IF NOT EXISTS idx_payouts_organization_id
    ON payouts (organization_id);

CREATE INDEX IF NOT EXISTS idx_commission_settings_organization_id
    ON commission_settings (organization_id);

CREATE INDEX IF NOT EXISTS idx_commission_settings_customer_id
    ON commission_settings (target_customer_id);

-- Triggers
CREATE TRIGGER tr_wallets_updated_at
    BEFORE UPDATE ON wallets
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER tr_ledger_entries_updated_at
    BEFORE UPDATE ON ledger_entries
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER tr_payouts_updated_at
    BEFORE UPDATE ON payouts
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER tr_commission_settings_updated_at
    BEFORE UPDATE ON commission_settings
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Tabela de locks de saldo (disputas e payouts pendentes)
CREATE TABLE IF NOT EXISTS wallet_locks (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id),
    wallet_id       UUID NOT NULL REFERENCES wallets(id),
    type            VARCHAR(20) NOT NULL CHECK (type IN ('dispute', 'payout')),
    reference_id    UUID NOT NULL,
    amount_cents    INTEGER NOT NULL CHECK (amount_cents > 0),
    status          VARCHAR(20) NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'released')),
    created_at      TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    released_at     TIMESTAMP WITH TIME ZONE
);

CREATE INDEX IF NOT EXISTS idx_wallet_locks_wallet_id
    ON wallet_locks (wallet_id) WHERE status = 'active';

CREATE INDEX IF NOT EXISTS idx_wallet_locks_reference_id
    ON wallet_locks (organization_id, reference_id, type) WHERE status = 'active';

-- Unique constraint para upsert de commission_settings
CREATE UNIQUE INDEX IF NOT EXISTS idx_commission_settings_org_customer
    ON commission_settings (organization_id, target_customer_id)
    WHERE deleted_at IS NULL;
