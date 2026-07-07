-- Configuração dos webhooks.
CREATE TABLE IF NOT EXISTS webhook_endpoints (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    url TEXT NOT NULL,
    description TEXT,
    enabled_events TEXT[], -- Identificadores dos eventos habilitados no endpoint.
    environment environment_type DEFAULT 'test',
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    deleted_at TIMESTAMP WITH TIME ZONE,

    CONSTRAINT unique_webhook_endpoint_organization_id UNIQUE (organization_id, id)
);

-- Monitoramento e debug dos webhooks.
CREATE TABLE IF NOT EXISTS webhook_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id),
    endpoint_id UUID NOT NULL,
    event_type VARCHAR(100) NOT NULL, -- Identificador do evento disparado.
    payload JSONB NOT NULL,
    response_body TEXT,
    response_status INTEGER,          -- Código HTTP retornado pelo endpoint.
    duration_ms INTEGER,
    attempt_number INTEGER DEFAULT 1,
    success BOOLEAN DEFAULT false,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),

    CONSTRAINT fk_webhook_logs_endpoint_organization
        FOREIGN KEY (organization_id, endpoint_id)
        REFERENCES webhook_endpoints (organization_id, id) ON DELETE CASCADE
);

-- Triggers
CREATE TRIGGER tr_webhook_endpoints_updated_at
    BEFORE UPDATE ON webhook_endpoints
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER tr_webhook_logs_updated_at
    BEFORE UPDATE ON webhook_logs
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
