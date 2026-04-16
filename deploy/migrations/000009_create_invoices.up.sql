-- Notas fiscais emitidas por assinatura ou cobrança avulsa.
CREATE TABLE IF NOT EXISTS invoices (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    customer_id UUID NOT NULL,
    subscription_id UUID, -- Assinatura originadora, quando aplicável.

    -- Configurações contábeis adotadas.
    invoice_type VARCHAR(20) NOT NULL CHECK (invoice_type IN ('national', 'international')),
    operation_type INTEGER NOT NULL DEFAULT 3,
    amount_cents INTEGER NOT NULL CHECK (amount_cents >= 0),
    currency VARCHAR(3) DEFAULT 'BRL',

    -- Estado transacional frente ao processamento externo.
    status VARCHAR(20) DEFAULT 'pending' CHECK (status IN ('pending', 'sent', 'paid', 'error', 'cancelled')),
    erp_reference VARCHAR(100), -- Identificador gerado pelo ERP após a emissão.

    -- Snapshots imutáveis gerados no ato da emissão (via Triggers).
    customer_snapshot JSONB, -- Dados do cliente no momento da emissão.
    items_snapshot JSONB,    -- Lista de itens no momento da emissão.

    last_sync_log JSONB,     -- Log da última sincronização com o ERP.
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),

    CONSTRAINT fk_invoices_customer_organization
        FOREIGN KEY (organization_id, customer_id)
        REFERENCES customers (organization_id, id),

    CONSTRAINT fk_invoices_subscription_organization
        FOREIGN KEY (organization_id, subscription_id)
        REFERENCES subscriptions (organization_id, id),

    CONSTRAINT unique_invoice_organization_id UNIQUE (organization_id, id)
);

-- Itens de linha da nota fiscal.
CREATE TABLE IF NOT EXISTS invoice_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id),
    invoice_id UUID NOT NULL,
    description TEXT NOT NULL,
    quantity DECIMAL(12, 4) NOT NULL DEFAULT 1.0 CHECK (quantity > 0),
    unit_price_cents INTEGER NOT NULL CHECK (unit_price_cents >= 0),
    total_price_cents INTEGER NOT NULL CHECK (total_price_cents >= 0),
    metadata JSONB,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),

    CONSTRAINT fk_invoice_items_invoice_organization
        FOREIGN KEY (organization_id, invoice_id)
        REFERENCES invoices (organization_id, id) ON DELETE CASCADE
);

-- Auditoria técnica das operações com o ERP.
CREATE TABLE IF NOT EXISTS invoice_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id),
    invoice_id UUID NOT NULL,

    -- Identificação da operação realizada.
    event_type VARCHAR(50) NOT NULL, -- Identificador da atividade ou trigger que originou a requisição.
    invoice_status VARCHAR(20),
    erp_reference VARCHAR(100),

    -- Resposta e telemetria do provedor.
    erp_error_code VARCHAR(50),
    erp_error_message TEXT,
    http_status INTEGER,
    request_url TEXT,
    request_payload JSONB,
    response_body JSONB,

    attempt_number INTEGER NOT NULL DEFAULT 1,
    success BOOLEAN NOT NULL DEFAULT false,
    message TEXT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),

    CONSTRAINT fk_invoice_logs_invoice_organization
        FOREIGN KEY (organization_id, invoice_id)
        REFERENCES invoices (organization_id, id) ON DELETE CASCADE
);

-- Configurações de emissão fiscal por Organization.
CREATE TABLE IF NOT EXISTS invoice_settings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,

    -- Modalidade de mercado (nacional ou internacional).
    type VARCHAR(20) NOT NULL CHECK (type IN ('national', 'international')),

    -- Credenciais de integração com o ERP.
    erp_api_key TEXT,
    erp_api_secret TEXT,
    erp_account_id VARCHAR(100),

    -- Configurações fiscais para emissão de NFS-e (mercado nacional).
    nfse_service_code VARCHAR(20),      -- Código tributário municipal do serviço.
    nfse_item_list_service VARCHAR(20), -- Código da lista de serviços (Lei Complementar 116/2003).
    tax_rate DECIMAL(5, 2),             -- Alíquota padrão aplicada na base de cálculo.
    invoice_series VARCHAR(10),         -- Série atribuída à numeração da nota fiscal.

    -- Configurações fiscais para emissão internacional.
    iso_currency CHAR(3) DEFAULT 'USD',
    export_regime VARCHAR(50), -- Regime fiscal aplicável a remessas internacionais.

    -- Comportamento de emissão automática.
    issue_on VARCHAR(20) DEFAULT 'payment_confirmed', -- Gatilho que dispara a emissão da nota fiscal.

    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),

    CONSTRAINT unique_organization_invoice_type UNIQUE (organization_id, type)
);

-- Indexes
CREATE INDEX IF NOT EXISTS idx_invoices_organization_created_at
    ON invoices (organization_id, created_at);

CREATE INDEX IF NOT EXISTS idx_invoices_customer_id
    ON invoices (customer_id);

CREATE INDEX IF NOT EXISTS idx_invoices_subscription_id
    ON invoices (subscription_id);

CREATE INDEX IF NOT EXISTS idx_invoices_status
    ON invoices (status) WHERE status IN ('pending', 'error');

CREATE INDEX IF NOT EXISTS idx_invoice_items_invoice_id
    ON invoice_items (invoice_id);

CREATE INDEX IF NOT EXISTS idx_invoice_logs_invoice_id
    ON invoice_logs (invoice_id);

CREATE INDEX IF NOT EXISTS idx_invoice_settings_organization_id
    ON invoice_settings (organization_id);

-- Triggers
CREATE TRIGGER tr_invoices_updated_at
    BEFORE UPDATE ON invoices
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER tr_invoice_items_updated_at
    BEFORE UPDATE ON invoice_items
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER tr_invoice_logs_updated_at
    BEFORE UPDATE ON invoice_logs
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER tr_invoice_settings_updated_at
    BEFORE UPDATE ON invoice_settings
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- ----------------------------------------------------
-- TRIGGER: SNAPSHOT DE CUSTOMER NA INVOICE
-- ----------------------------------------------------
CREATE OR REPLACE FUNCTION fn_build_invoice_customer_snapshot()
RETURNS TRIGGER AS $$
BEGIN
    IF NEW.customer_snapshot IS NULL THEN
        NEW.customer_snapshot = (
            SELECT jsonb_build_object(
                'id', c.id,
                'name', c.name,
                'email', c.email,
                'document', c.document,
                'document_type', c.document_type,
                'address', (
                    SELECT row_to_json(a.*)
                    FROM addresses a
                    WHERE a.customer_id = c.id AND a.is_main = true
                    LIMIT 1
                )
            )
            FROM customers c
            WHERE c.id = NEW.customer_id
        );
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER tr_invoices_customer_snapshot
    BEFORE INSERT ON invoices
    FOR EACH ROW
    EXECUTE FUNCTION fn_build_invoice_customer_snapshot();

-- ----------------------------------------------------
-- TRIGGER: SNAPSHOT DOS ITENS DA INVOICE
-- ----------------------------------------------------
CREATE OR REPLACE FUNCTION fn_build_invoice_items_snapshot()
RETURNS TRIGGER AS $$
DECLARE
    v_invoice_id UUID;
BEGIN
    IF TG_OP = 'DELETE' THEN
        v_invoice_id := OLD.invoice_id;
    ELSE
        v_invoice_id := NEW.invoice_id;
    END IF;

    UPDATE invoices
    SET items_snapshot = (
        SELECT COALESCE(jsonb_agg(
            jsonb_build_object(
                'id', id,
                'description', description,
                'quantity', quantity,
                'unit_price_cents', unit_price_cents,
                'total_price_cents', total_price_cents
            )
        ), '[]'::jsonb)
        FROM invoice_items
        WHERE invoice_id = v_invoice_id
    )
    WHERE id = v_invoice_id;

    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER tr_invoice_items_snapshot
    AFTER INSERT OR UPDATE OR DELETE ON invoice_items
    FOR EACH ROW
    EXECUTE FUNCTION fn_build_invoice_items_snapshot();
