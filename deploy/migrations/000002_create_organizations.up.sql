-- Entidade principal de gestão (Organizations/Empresas).
CREATE TABLE IF NOT EXISTS organizations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    webhook_secret VARCHAR(255) DEFAULT gen_random_uuid(),
    status VARCHAR(20) DEFAULT 'active' CHECK (status IN ('active', 'suspended')),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    deleted_at TIMESTAMP WITH TIME ZONE
);

-- Configurações globais atreladas a um Organization.
CREATE TABLE IF NOT EXISTS organization_configs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    support_email VARCHAR(255),
    logo_url TEXT,
    primary_color VARCHAR(7) DEFAULT '#000000',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),

    CONSTRAINT unique_organization_config_organization_id UNIQUE (organization_id)
);

-- ----------------------------------------------------
-- FUNCTION: update_updated_at_column
-- Shared trigger function used across all domains.
-- ----------------------------------------------------
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER tr_organizations_updated_at
    BEFORE UPDATE ON organizations
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER tr_organization_configs_updated_at
    BEFORE UPDATE ON organization_configs
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
