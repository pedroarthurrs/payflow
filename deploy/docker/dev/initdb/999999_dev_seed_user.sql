-- Seed de desenvolvimento: organization + usuário para login local.
-- Executa apenas no compose de dev e é idempotente.

INSERT INTO organizations (id, name, status)
VALUES (
    '11111111-1111-1111-1111-111111111111',
    'I9Pay Dev Organization',
    'active'
)
ON CONFLICT (id) DO UPDATE
SET
    name = EXCLUDED.name,
    status = EXCLUDED.status,
    updated_at = NOW(),
    deleted_at = NULL;

INSERT INTO users (id, organization_id, email, name, password, role)
VALUES (
    '22222222-2222-2222-2222-222222222222',
    '11111111-1111-1111-1111-111111111111',
    'dev@i9pay.local',
    'Dev User',
    '$2a$10$U2ShqBynvPjAUiCnPTypnO6Hjs4IwsLU7dJAmcU.qQkUXfeWBXU1e',
    'owner'
)
ON CONFLICT (organization_id, email) DO UPDATE
SET
    name = EXCLUDED.name,
    password = EXCLUDED.password,
    role = EXCLUDED.role,
    updated_at = NOW(),
    deleted_at = NULL;
