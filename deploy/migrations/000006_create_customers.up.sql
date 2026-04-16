-- Clientes finais de cada SaaS.
CREATE TABLE IF NOT EXISTS customers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id),
    name VARCHAR(255) NOT NULL,
    email VARCHAR(255) NOT NULL,
    phone VARCHAR(20),
    document VARCHAR(20) NOT NULL,
    document_type VARCHAR(20) NOT NULL,   -- Tipificação padronizada do documento.
    external_reference VARCHAR(255),       -- Referência ao cliente em sistema externo (CRM, ERP, etc.).
    status VARCHAR(20) DEFAULT 'active' CHECK (status IN ('active', 'inactive')),
    metadata JSONB,                     -- Metadados customizados vinculados à entidade.
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    deleted_at TIMESTAMP WITH TIME ZONE,

    CONSTRAINT unique_customer_organization_id UNIQUE (organization_id, id)
);

-- Endereços vinculados a clientes.
CREATE TABLE IF NOT EXISTS addresses (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id),
    customer_id UUID NOT NULL,
    type address_type DEFAULT 'billing',
    postal_code VARCHAR(10) NOT NULL,
    street TEXT NOT NULL,
    number VARCHAR(10),
    complement VARCHAR(100),
    neighborhood VARCHAR(100),
    city VARCHAR(100) NOT NULL,
    state CHAR(2) NOT NULL,
    country CHAR(2) DEFAULT 'BR',
    is_main BOOLEAN DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    deleted_at TIMESTAMP WITH TIME ZONE,

    CONSTRAINT fk_addresses_customer_organization
        FOREIGN KEY (organization_id, customer_id)
        REFERENCES customers (organization_id, id) ON DELETE CASCADE
);

-- Indexes
CREATE UNIQUE INDEX IF NOT EXISTS unique_customer_email_per_organization
    ON customers (organization_id, email)
    WHERE deleted_at IS NULL;

CREATE UNIQUE INDEX IF NOT EXISTS unique_customer_doc_per_organization
    ON customers (organization_id, document)
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_customers_organization_id
    ON customers (organization_id);

CREATE INDEX IF NOT EXISTS idx_addresses_customer_id
    ON addresses (customer_id);

-- Triggers
CREATE TRIGGER tr_customers_updated_at
    BEFORE UPDATE ON customers
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER tr_addresses_updated_at
    BEFORE UPDATE ON addresses
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
