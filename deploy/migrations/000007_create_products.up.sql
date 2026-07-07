-- Produtos comercializados pelos Organizations.
CREATE TABLE IF NOT EXISTS products (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id),
    name VARCHAR(255) NOT NULL,
    active BOOLEAN DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    deleted_at TIMESTAMP WITH TIME ZONE,

    CONSTRAINT unique_product_organization_id UNIQUE (organization_id, id)
);

-- Regras de preço e recorrência dos produtos.
CREATE TABLE IF NOT EXISTS plans (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id),
    product_id UUID NOT NULL,
    name VARCHAR(255),                -- Nome de exibição público do plano.
    slug TEXT,
    amount_cents INTEGER NOT NULL CHECK (amount_cents >= 0),
    currency VARCHAR(3) DEFAULT 'BRL',
    interval VARCHAR(20) CHECK (interval IN ('day', 'week', 'month', 'year')),
    interval_count INTEGER DEFAULT 1, -- Multiplicador de distância entre ciclos (ex: a cada 3 meses).
    active BOOLEAN DEFAULT true,
    version INTEGER DEFAULT 1,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    deleted_at TIMESTAMP WITH TIME ZONE,

    CONSTRAINT fk_plans_product_organization
        FOREIGN KEY (organization_id, product_id)
        REFERENCES products (organization_id, id) ON DELETE CASCADE,

    CONSTRAINT unique_plan_organization_id UNIQUE (organization_id, id)
);

-- Indexes
CREATE INDEX IF NOT EXISTS idx_products_organization_id
    ON products (organization_id);

CREATE INDEX IF NOT EXISTS idx_plans_product_id
    ON plans (product_id);

-- Triggers
CREATE TRIGGER tr_products_updated_at
    BEFORE UPDATE ON products
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER tr_plans_updated_at
    BEFORE UPDATE ON plans
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
