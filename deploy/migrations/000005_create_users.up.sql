-- Identidade global de usuário (desacoplada de org).
-- Um mesmo login pode acessar múltiplas organizations via organization_members.
CREATE TABLE IF NOT EXISTS "user" (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email      VARCHAR(255) NOT NULL UNIQUE,
    name       VARCHAR(255) NOT NULL,
    password   VARCHAR(255) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    deleted_at TIMESTAMP WITH TIME ZONE
);

-- Vínculo entre um UserAccount e uma Organization, com role e audit.
CREATE TABLE IF NOT EXISTS organization_members (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    user_id         UUID NOT NULL REFERENCES "user"(id) ON DELETE CASCADE,
    role            VARCHAR(20) NOT NULL DEFAULT 'support'
                    CHECK (role IN ('owner', 'admin', 'developer', 'support')),
    invited_by      UUID REFERENCES "user"(id),
    joined_at       TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    created_at      TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at      TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    deleted_at      TIMESTAMP WITH TIME ZONE,

    CONSTRAINT unique_member_per_org UNIQUE (organization_id, user_id)
);

-- Grants e revocations individuais sobre os defaults do role.
-- granted=true  → concede permissão não prevista no role.
-- granted=false → revoga permissão prevista no role.
-- organization_id é desnormalizado para queries filtradas por org sem JOIN adicional.
-- Integridade garantida pelo CASCADE em member_id.
CREATE TABLE IF NOT EXISTS member_permissions (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL,  -- desnormalizado de organization_members
    member_id       UUID NOT NULL REFERENCES organization_members(id) ON DELETE CASCADE,
    permission      VARCHAR(50) NOT NULL,
    granted         BOOLEAN NOT NULL DEFAULT true,
    created_at      TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at      TIMESTAMP WITH TIME ZONE DEFAULT NOW(),

    CONSTRAINT unique_member_permission UNIQUE (member_id, permission)
);

-- Auditoria de ações de membros autenticados dentro de uma organização.
-- Logs sem contexto de org (ex: autenticação pré-seleção) não são registrados aqui.
-- organization_id é desnormalizado para queries filtradas por org sem JOIN adicional.
CREATE TABLE IF NOT EXISTS member_logs (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL,  -- desnormalizado de organization_members
    member_id       UUID NOT NULL REFERENCES organization_members(id) ON DELETE CASCADE,
    action          VARCHAR(100) NOT NULL,
    method          VARCHAR(10),
    path            TEXT,
    status_code     INTEGER,
    latency_ms      INTEGER,
    ip_address      VARCHAR(45),
    user_agent      TEXT,
    metadata        JSONB,
    created_at      TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at      TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Indexes
CREATE INDEX IF NOT EXISTS idx_user_email
    ON "user" (email);

CREATE INDEX IF NOT EXISTS idx_org_members_org
    ON organization_members (organization_id) WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_org_members_user
    ON organization_members (user_id) WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_member_permissions_member
    ON member_permissions (member_id);

CREATE INDEX IF NOT EXISTS idx_member_permissions_org
    ON member_permissions (organization_id);

CREATE INDEX IF NOT EXISTS idx_member_logs_member
    ON member_logs (member_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_member_logs_org
    ON member_logs (organization_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_member_logs_created_at
    ON member_logs (created_at DESC);

-- Triggers
CREATE TRIGGER tr_user_updated_at
    BEFORE UPDATE ON "user"
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER tr_organization_members_updated_at
    BEFORE UPDATE ON organization_members
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER tr_member_permissions_updated_at
    BEFORE UPDATE ON member_permissions
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER tr_member_logs_updated_at
    BEFORE UPDATE ON member_logs
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
