-- Gerenciamento de acesso por ambiente.
CREATE TABLE IF NOT EXISTS credentials (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    name VARCHAR(50) NOT NULL,
    key_hash TEXT NOT NULL UNIQUE,   -- Hash SHA-256 da chave para validação.
    key_prefix VARCHAR(10) NOT NULL, -- Prefixo indicativo de ambiente da chave.
    environment environment_type NOT NULL,
    last_used_at TIMESTAMP WITH TIME ZONE,
    expires_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    deleted_at TIMESTAMP WITH TIME ZONE,

    CONSTRAINT unique_credential_organization_id UNIQUE (organization_id, id)
);

-- Monitoramento e debug do sistema.
CREATE TABLE IF NOT EXISTS credential_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id),
    credential_id UUID,    -- Credencial utilizada na requisição.
    method VARCHAR(10),    -- Método HTTP da requisição.
    path TEXT,             -- Path da rota acessada.
    request_body JSONB,
    response_body JSONB,
    status_code INTEGER,
    latency_ms INTEGER,
    ip_address VARCHAR(45),
    user_agent TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),

    CONSTRAINT fk_credential_logs_credential_organization
        FOREIGN KEY (organization_id, credential_id)
        REFERENCES credentials (organization_id, id) ON DELETE SET NULL
);

-- Indexes
CREATE INDEX IF NOT EXISTS idx_credentials_organization_id
    ON credentials (organization_id);

CREATE INDEX IF NOT EXISTS idx_credential_logs_organization_id
    ON credential_logs (organization_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_credential_logs_credential_id
    ON credential_logs (credential_id);

-- Triggers
CREATE TRIGGER tr_credentials_updated_at
    BEFORE UPDATE ON credentials
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER tr_credential_logs_updated_at
    BEFORE UPDATE ON credential_logs
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
