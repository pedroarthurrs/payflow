-- =============================================================================
-- Seed de desenvolvimento: massa de dados para testes locais.
-- Executa apenas no compose de dev e é idempotente (ON CONFLICT).
--
-- Senha padrão para todos os usuários: "Dev@123456"
-- Hash bcrypt (cost 10): $2a$10$U2ShqBynvPjAUiCnPTypnO6Hjs4IwsLU7dJAmcU.qQkUXfeWBXU1e
-- =============================================================================

-- ===================== ORGANIZATIONS =====================

INSERT INTO organizations (id, name, status) VALUES
    ('11111111-1111-1111-1111-111111111111', 'I9Pay Dev Organization', 'active'),
    ('22222222-2222-2222-2222-222222222222', 'Acme Corp', 'active'),
    ('33333333-3333-3333-3333-333333333333', 'Suspended Org', 'suspended')
ON CONFLICT (id) DO UPDATE SET
    name = EXCLUDED.name,
    status = EXCLUDED.status,
    updated_at = NOW(),
    deleted_at = NULL;

-- ===================== ORGANIZATION CONFIGS =====================

INSERT INTO organization_configs (id, organization_id, support_email, logo_url, primary_color) VALUES
    ('cc000001-0000-0000-0000-000000000001', '11111111-1111-1111-1111-111111111111', 'suporte@i9pay.local', 'https://placehold.co/200x200?text=I9', '#6C63FF'),
    ('cc000002-0000-0000-0000-000000000002', '22222222-2222-2222-2222-222222222222', 'help@acme.local', 'https://placehold.co/200x200?text=AC', '#FF6347')
ON CONFLICT (organization_id) DO UPDATE SET
    support_email = EXCLUDED.support_email,
    logo_url = EXCLUDED.logo_url,
    primary_color = EXCLUDED.primary_color,
    updated_at = NOW();

-- ===================== USERS =====================
-- Senha: password123

INSERT INTO users (id, organization_id, email, name, password, role) VALUES
    -- I9Pay Dev Organization
    ('aa000001-0000-0000-0000-000000000001', '11111111-1111-1111-1111-111111111111', 'dev@i9pay.local',       'Dev User',          '$2a$10$U2ShqBynvPjAUiCnPTypnO6Hjs4IwsLU7dJAmcU.qQkUXfeWBXU1e', 'owner'),
    ('aa000002-0000-0000-0000-000000000002', '11111111-1111-1111-1111-111111111111', 'admin@i9pay.local',     'Admin User',        '$2a$10$U2ShqBynvPjAUiCnPTypnO6Hjs4IwsLU7dJAmcU.qQkUXfeWBXU1e', 'admin'),
    ('aa000003-0000-0000-0000-000000000003', '11111111-1111-1111-1111-111111111111', 'dev2@i9pay.local',      'Developer User',    '$2a$10$U2ShqBynvPjAUiCnPTypnO6Hjs4IwsLU7dJAmcU.qQkUXfeWBXU1e', 'developer'),
    ('aa000008-0000-0000-0000-000000000008', '11111111-1111-1111-1111-111111111111', 'support@i9pay.local',   'Support User',      '$2a$10$U2ShqBynvPjAUiCnPTypnO6Hjs4IwsLU7dJAmcU.qQkUXfeWBXU1e', 'support'),
    -- Acme Corp
    ('aa000004-0000-0000-0000-000000000004', '22222222-2222-2222-2222-222222222222', 'owner@acme.local',      'Acme Owner',        '$2a$10$U2ShqBynvPjAUiCnPTypnO6Hjs4IwsLU7dJAmcU.qQkUXfeWBXU1e', 'owner'),
    ('aa000005-0000-0000-0000-000000000005', '22222222-2222-2222-2222-222222222222', 'admin@acme.local',      'Acme Admin',        '$2a$10$U2ShqBynvPjAUiCnPTypnO6Hjs4IwsLU7dJAmcU.qQkUXfeWBXU1e', 'admin'),
    ('aa000006-0000-0000-0000-000000000006', '22222222-2222-2222-2222-222222222222', 'developer@acme.local',  'Acme Developer',    '$2a$10$U2ShqBynvPjAUiCnPTypnO6Hjs4IwsLU7dJAmcU.qQkUXfeWBXU1e', 'developer'),
    ('aa000007-0000-0000-0000-000000000007', '22222222-2222-2222-2222-222222222222', 'support@acme.local',    'Acme Support',      '$2a$10$U2ShqBynvPjAUiCnPTypnO6Hjs4IwsLU7dJAmcU.qQkUXfeWBXU1e', 'support'),
    ('aa000009-0000-0000-0000-000000000009', '22222222-2222-2222-2222-222222222222', 'support2@acme.local',   'Acme Support II',   '$2a$10$U2ShqBynvPjAUiCnPTypnO6Hjs4IwsLU7dJAmcU.qQkUXfeWBXU1e', 'support')
ON CONFLICT (organization_id, email) DO UPDATE SET
    name = EXCLUDED.name,
    password = EXCLUDED.password,
    role = EXCLUDED.role,
    updated_at = NOW(),
    deleted_at = NULL;

-- ===================== CREDENTIALS (API Keys) =====================
-- key_hash é SHA-256 fictício para dev. Prefixo indica ambiente.

INSERT INTO credentials (id, organization_id, name, key_hash, key_prefix, environment, expires_at) VALUES
    ('dd000001-0000-0000-0000-000000000001', '11111111-1111-1111-1111-111111111111', 'Dev Test Key',  'sha256_dev_test_key_hash_0001',  'test_',  'test',       NOW() + INTERVAL '1 year'),
    ('dd000002-0000-0000-0000-000000000002', '11111111-1111-1111-1111-111111111111', 'Dev Prod Key',  'sha256_dev_prod_key_hash_0002',  'prod_',  'production', NOW() + INTERVAL '1 year'),
    ('dd000003-0000-0000-0000-000000000003', '22222222-2222-2222-2222-222222222222', 'Acme Test Key', 'sha256_acme_test_key_hash_0003', 'test_',  'test',       NOW() + INTERVAL '6 months'),
    ('dd000004-0000-0000-0000-000000000004', '22222222-2222-2222-2222-222222222222', 'Acme Prod Key', 'sha256_acme_prod_key_hash_0004', 'prod_',  'production', NOW() + INTERVAL '1 year'),
    ('dd000005-0000-0000-0000-000000000005', '11111111-1111-1111-1111-111111111111', 'Expired Key',   'sha256_expired_key_hash_0005',   'test_',  'test',       NOW() - INTERVAL '30 days')
ON CONFLICT (organization_id, id) DO NOTHING;

-- ===================== CUSTOMERS =====================

INSERT INTO customers (id, organization_id, name, email, phone, document, document_type, external_reference, status, metadata) VALUES
    -- I9Pay customers
    ('bb000001-0000-0000-0000-000000000001', '11111111-1111-1111-1111-111111111111', 'João Silva',        'joao@email.local',      '+5511999990001', '12345678901',     'cpf',  'CRM-001', 'active',   '{"tier": "premium", "source": "website"}'),
    ('bb000002-0000-0000-0000-000000000002', '11111111-1111-1111-1111-111111111111', 'Maria Oliveira',    'maria@email.local',     '+5511999990002', '23456789012',     'cpf',  'CRM-002', 'active',   '{"tier": "basic", "source": "referral"}'),
    ('bb000003-0000-0000-0000-000000000003', '11111111-1111-1111-1111-111111111111', 'Tech Solutions ME', 'contato@techsol.local', '+5511999990003', '12345678000190',  'cnpj', 'CRM-003', 'active',   '{"tier": "enterprise"}'),
    ('bb000004-0000-0000-0000-000000000004', '11111111-1111-1111-1111-111111111111', 'Carlos Souza',      'carlos@email.local',    '+5511999990004', '34567890123',     'cpf',  NULL,      'inactive', NULL),
    ('bb000005-0000-0000-0000-000000000005', '11111111-1111-1111-1111-111111111111', 'Ana Costa',         'ana@email.local',       '+5511999990005', '45678901234',     'cpf',  'CRM-005', 'active',   '{"tier": "basic"}'),
    -- Acme customers
    ('bb000006-0000-0000-0000-000000000006', '22222222-2222-2222-2222-222222222222', 'Pedro Santos',      'pedro@email.local',     '+5521999990006', '56789012345',     'cpf',  'ERP-100', 'active',   '{"plan": "starter"}'),
    ('bb000007-0000-0000-0000-000000000007', '22222222-2222-2222-2222-222222222222', 'Fernanda Lima',     'fernanda@email.local',  '+5521999990007', '67890123456',     'cpf',  'ERP-101', 'active',   NULL),
    ('bb000008-0000-0000-0000-000000000008', '22222222-2222-2222-2222-222222222222', 'Global Trade LTDA', 'global@trade.local',    '+5521999990008', '98765432000110',  'cnpj', 'ERP-102', 'active',   '{"tier": "enterprise", "segment": "import"}'),
    ('bb000009-0000-0000-0000-000000000009', '22222222-2222-2222-2222-222222222222', 'Lucas Mendes',      'lucas@email.local',     '+5521999990009', '78901234567',     'cpf',  NULL,      'inactive', NULL),
    ('bb000010-0000-0000-0000-000000000010', '22222222-2222-2222-2222-222222222222', 'Camila Rocha',      'camila@email.local',    '+5521999990010', '89012345678',     'cpf',  'ERP-104', 'active',   '{"tier": "premium"}')
ON CONFLICT (organization_id, id) DO NOTHING;

-- ===================== ADDRESSES =====================

INSERT INTO addresses (id, organization_id, customer_id, type, postal_code, street, number, complement, neighborhood, city, state, country, is_main) VALUES
    -- João Silva (2 endereços)
    ('ee000001-0000-0000-0000-000000000001', '11111111-1111-1111-1111-111111111111', 'bb000001-0000-0000-0000-000000000001', 'billing',  '01310100', 'Av. Paulista',          '1000', 'Sala 501',  'Bela Vista',    'São Paulo',       'SP', 'BR', true),
    ('ee000002-0000-0000-0000-000000000002', '11111111-1111-1111-1111-111111111111', 'bb000001-0000-0000-0000-000000000001', 'shipping', '04543011', 'Rua Funchal',           '418',  NULL,        'Vila Olímpia',  'São Paulo',       'SP', 'BR', false),
    -- Maria Oliveira
    ('ee000003-0000-0000-0000-000000000003', '11111111-1111-1111-1111-111111111111', 'bb000002-0000-0000-0000-000000000002', 'billing',  '20040020', 'Av. Rio Branco',        '156',  'Andar 12',  'Centro',        'Rio de Janeiro',  'RJ', 'BR', true),
    -- Tech Solutions ME
    ('ee000004-0000-0000-0000-000000000004', '11111111-1111-1111-1111-111111111111', 'bb000003-0000-0000-0000-000000000003', 'billing',  '30130000', 'Av. Afonso Pena',       '4000', 'Bloco B',   'Funcionários',  'Belo Horizonte',  'MG', 'BR', true),
    -- Ana Costa
    ('ee000005-0000-0000-0000-000000000005', '11111111-1111-1111-1111-111111111111', 'bb000005-0000-0000-0000-000000000005', 'home',     '80060150', 'Rua XV de Novembro',    '700',  'Apto 302',  'Centro',        'Curitiba',        'PR', 'BR', true),
    -- Pedro Santos (Acme)
    ('ee000006-0000-0000-0000-000000000006', '22222222-2222-2222-2222-222222222222', 'bb000006-0000-0000-0000-000000000006', 'billing',  '40020010', 'Av. Sete de Setembro',  '200',  NULL,        'Centro',        'Salvador',        'BA', 'BR', true),
    -- Fernanda Lima (Acme)
    ('ee000007-0000-0000-0000-000000000007', '22222222-2222-2222-2222-222222222222', 'bb000007-0000-0000-0000-000000000007', 'billing',  '50030230', 'Rua da Aurora',         '325',  'Sala 10',   'Boa Vista',     'Recife',          'PE', 'BR', true),
    -- Global Trade LTDA (Acme, 2 endereços)
    ('ee000008-0000-0000-0000-000000000008', '22222222-2222-2222-2222-222222222222', 'bb000008-0000-0000-0000-000000000008', 'billing',  '90010280', 'Rua dos Andradas',      '1001', 'Conj 801',  'Centro',        'Porto Alegre',    'RS', 'BR', true),
    ('ee000009-0000-0000-0000-000000000009', '22222222-2222-2222-2222-222222222222', 'bb000008-0000-0000-0000-000000000008', 'shipping', '93010000', 'Av. Brasil',            '500',  'Galpão 3',  'Centro',        'São Leopoldo',    'RS', 'BR', false),
    -- Camila Rocha (Acme)
    ('ee000010-0000-0000-0000-000000000010', '22222222-2222-2222-2222-222222222222', 'bb000010-0000-0000-0000-000000000010', 'billing',  '60060440', 'Rua Barão do Rio Branco','100', NULL,        'Centro',        'Fortaleza',       'CE', 'BR', true)
ON CONFLICT (id) DO NOTHING;

-- ===================== PRODUCTS =====================

INSERT INTO products (id, organization_id, name, active) VALUES
    -- I9Pay products
    ('ff000001-0000-0000-0000-000000000001', '11111111-1111-1111-1111-111111111111', 'Plano Básico',       true),
    ('ff000002-0000-0000-0000-000000000002', '11111111-1111-1111-1111-111111111111', 'Plano Profissional', true),
    ('ff000003-0000-0000-0000-000000000003', '11111111-1111-1111-1111-111111111111', 'Plano Enterprise',   true),
    ('ff000004-0000-0000-0000-000000000004', '11111111-1111-1111-1111-111111111111', 'Add-on: SMS',        true),
    ('ff000005-0000-0000-0000-000000000005', '11111111-1111-1111-1111-111111111111', 'Produto Descontinuado', false),
    -- Acme products
    ('ff000006-0000-0000-0000-000000000006', '22222222-2222-2222-2222-222222222222', 'Starter',            true),
    ('ff000007-0000-0000-0000-000000000007', '22222222-2222-2222-2222-222222222222', 'Business',           true),
    ('ff000008-0000-0000-0000-000000000008', '22222222-2222-2222-2222-222222222222', 'Enterprise',         true)
ON CONFLICT (organization_id, id) DO NOTHING;

-- ===================== PLANS =====================

INSERT INTO plans (id, organization_id, product_id, name, slug, amount_cents, currency, interval, interval_count, active) VALUES
    -- I9Pay plans
    ('11aaa001-0000-0000-0000-000000000001', '11111111-1111-1111-1111-111111111111', 'ff000001-0000-0000-0000-000000000001', 'Básico Mensal',        'basico-mensal',        2990,   'BRL', 'month', 1, true),
    ('11aaa002-0000-0000-0000-000000000002', '11111111-1111-1111-1111-111111111111', 'ff000001-0000-0000-0000-000000000001', 'Básico Anual',         'basico-anual',         29900,  'BRL', 'year',  1, true),
    ('11aaa003-0000-0000-0000-000000000003', '11111111-1111-1111-1111-111111111111', 'ff000002-0000-0000-0000-000000000002', 'Profissional Mensal',  'profissional-mensal',  7990,   'BRL', 'month', 1, true),
    ('11aaa004-0000-0000-0000-000000000004', '11111111-1111-1111-1111-111111111111', 'ff000002-0000-0000-0000-000000000002', 'Profissional Anual',   'profissional-anual',   79900,  'BRL', 'year',  1, true),
    ('11aaa005-0000-0000-0000-000000000005', '11111111-1111-1111-1111-111111111111', 'ff000003-0000-0000-0000-000000000003', 'Enterprise Mensal',    'enterprise-mensal',    24990,  'BRL', 'month', 1, true),
    ('11aaa006-0000-0000-0000-000000000006', '11111111-1111-1111-1111-111111111111', 'ff000004-0000-0000-0000-000000000004', 'SMS 1000 créditos',    'sms-1000',             4990,   'BRL', 'month', 1, true),
    -- Acme plans
    ('11aaa007-0000-0000-0000-000000000007', '22222222-2222-2222-2222-222222222222', 'ff000006-0000-0000-0000-000000000006', 'Starter Monthly',      'starter-monthly',      1990,   'BRL', 'month', 1, true),
    ('11aaa008-0000-0000-0000-000000000008', '22222222-2222-2222-2222-222222222222', 'ff000007-0000-0000-0000-000000000007', 'Business Monthly',     'business-monthly',     4990,   'BRL', 'month', 1, true),
    ('11aaa009-0000-0000-0000-000000000009', '22222222-2222-2222-2222-222222222222', 'ff000007-0000-0000-0000-000000000007', 'Business Quarterly',   'business-quarterly',   12990,  'BRL', 'month', 3, true),
    ('11aaa010-0000-0000-0000-000000000010', '22222222-2222-2222-2222-222222222222', 'ff000008-0000-0000-0000-000000000008', 'Enterprise Monthly',   'enterprise-monthly',   19990,  'BRL', 'month', 1, true)
ON CONFLICT (organization_id, id) DO NOTHING;

-- ===================== SUBSCRIPTIONS =====================

INSERT INTO subscriptions (id, organization_id, customer_id, status, current_period_start, current_period_end, cancel_at_period_end, metadata) VALUES
    -- I9Pay subscriptions
    ('55aaa001-0000-0000-0000-000000000001', '11111111-1111-1111-1111-111111111111', 'bb000001-0000-0000-0000-000000000001', 'active',   NOW() - INTERVAL '15 days', NOW() + INTERVAL '15 days', false, '{"source": "website"}'),
    ('55aaa002-0000-0000-0000-000000000002', '11111111-1111-1111-1111-111111111111', 'bb000002-0000-0000-0000-000000000002', 'active',   NOW() - INTERVAL '10 days', NOW() + INTERVAL '20 days', false, NULL),
    ('55aaa003-0000-0000-0000-000000000003', '11111111-1111-1111-1111-111111111111', 'bb000003-0000-0000-0000-000000000003', 'active',   NOW() - INTERVAL '5 days',  NOW() + INTERVAL '25 days', false, '{"contract": "annual"}'),
    ('55aaa004-0000-0000-0000-000000000004', '11111111-1111-1111-1111-111111111111', 'bb000001-0000-0000-0000-000000000001', 'active',   NOW() - INTERVAL '15 days', NOW() + INTERVAL '15 days', false, '{"addon": true}'),
    ('55aaa005-0000-0000-0000-000000000005', '11111111-1111-1111-1111-111111111111', 'bb000004-0000-0000-0000-000000000004', 'canceled', NOW() - INTERVAL '60 days', NOW() - INTERVAL '30 days', false, '{"cancel_reason": "too_expensive"}'),
    ('55aaa006-0000-0000-0000-000000000006', '11111111-1111-1111-1111-111111111111', 'bb000005-0000-0000-0000-000000000005', 'past_due', NOW() - INTERVAL '35 days', NOW() - INTERVAL '5 days',  false, NULL),
    ('55aaa007-0000-0000-0000-000000000007', '11111111-1111-1111-1111-111111111111', 'bb000002-0000-0000-0000-000000000002', 'trialing', NOW(),                       NOW() + INTERVAL '14 days', false, '{"trial_source": "promo"}'),
    -- Acme subscriptions
    ('55aaa008-0000-0000-0000-000000000008', '22222222-2222-2222-2222-222222222222', 'bb000006-0000-0000-0000-000000000006', 'active',   NOW() - INTERVAL '20 days', NOW() + INTERVAL '10 days', false, NULL),
    ('55aaa009-0000-0000-0000-000000000009', '22222222-2222-2222-2222-222222222222', 'bb000008-0000-0000-0000-000000000008', 'active',   NOW() - INTERVAL '8 days',  NOW() + INTERVAL '22 days', false, '{"contract": "enterprise"}'),
    ('55aaa010-0000-0000-0000-000000000010', '22222222-2222-2222-2222-222222222222', 'bb000010-0000-0000-0000-000000000010', 'paused',   NOW() - INTERVAL '45 days', NOW() - INTERVAL '15 days', false, '{"pause_reason": "vacation"}'),
    ('55aaa011-0000-0000-0000-000000000011', '22222222-2222-2222-2222-222222222222', 'bb000007-0000-0000-0000-000000000007', 'active',   NOW() - INTERVAL '3 days',  NOW() + INTERVAL '27 days', true,  '{"cancel_scheduled": true}')
ON CONFLICT (organization_id, id) DO NOTHING;

-- ===================== SUBSCRIPTION ITEMS =====================

INSERT INTO subscription_items (id, organization_id, subscription_id, plan_id, quantity) VALUES
    -- João: Profissional Mensal + SMS addon
    ('66aaa001-0000-0000-0000-000000000001', '11111111-1111-1111-1111-111111111111', '55aaa001-0000-0000-0000-000000000001', '11aaa003-0000-0000-0000-000000000003', 1),
    ('66aaa002-0000-0000-0000-000000000002', '11111111-1111-1111-1111-111111111111', '55aaa004-0000-0000-0000-000000000004', '11aaa006-0000-0000-0000-000000000006', 2),
    -- Maria: Básico Mensal
    ('66aaa003-0000-0000-0000-000000000003', '11111111-1111-1111-1111-111111111111', '55aaa002-0000-0000-0000-000000000002', '11aaa001-0000-0000-0000-000000000001', 1),
    -- Tech Solutions: Enterprise Mensal
    ('66aaa004-0000-0000-0000-000000000004', '11111111-1111-1111-1111-111111111111', '55aaa003-0000-0000-0000-000000000003', '11aaa005-0000-0000-0000-000000000005', 1),
    -- Carlos (canceled): Básico Mensal
    ('66aaa005-0000-0000-0000-000000000005', '11111111-1111-1111-1111-111111111111', '55aaa005-0000-0000-0000-000000000005', '11aaa001-0000-0000-0000-000000000001', 1),
    -- Ana (past_due): Profissional Anual
    ('66aaa006-0000-0000-0000-000000000006', '11111111-1111-1111-1111-111111111111', '55aaa006-0000-0000-0000-000000000006', '11aaa004-0000-0000-0000-000000000004', 1),
    -- Maria (trialing): Enterprise Mensal
    ('66aaa007-0000-0000-0000-000000000007', '11111111-1111-1111-1111-111111111111', '55aaa007-0000-0000-0000-000000000007', '11aaa005-0000-0000-0000-000000000005', 1),
    -- Acme: Pedro Starter
    ('66aaa008-0000-0000-0000-000000000008', '22222222-2222-2222-2222-222222222222', '55aaa008-0000-0000-0000-000000000008', '11aaa007-0000-0000-0000-000000000007', 1),
    -- Acme: Global Trade Enterprise
    ('66aaa009-0000-0000-0000-000000000009', '22222222-2222-2222-2222-222222222222', '55aaa009-0000-0000-0000-000000000009', '11aaa010-0000-0000-0000-000000000010', 3),
    -- Acme: Camila Business (paused)
    ('66aaa010-0000-0000-0000-000000000010', '22222222-2222-2222-2222-222222222222', '55aaa010-0000-0000-0000-000000000010', '11aaa008-0000-0000-0000-000000000008', 1),
    -- Acme: Fernanda Business (cancel_at_period_end)
    ('66aaa011-0000-0000-0000-000000000011', '22222222-2222-2222-2222-222222222222', '55aaa011-0000-0000-0000-000000000011', '11aaa008-0000-0000-0000-000000000008', 1)
ON CONFLICT (id) DO NOTHING;

-- ===================== INVOICES =====================

INSERT INTO invoices (id, organization_id, customer_id, subscription_id, invoice_type, operation_type, amount_cents, currency, status) VALUES
    -- I9Pay invoices
    ('77aaa001-0000-0000-0000-000000000001', '11111111-1111-1111-1111-111111111111', 'bb000001-0000-0000-0000-000000000001', '55aaa001-0000-0000-0000-000000000001', 'national', 3, 7990,  'BRL', 'paid'),
    ('77aaa002-0000-0000-0000-000000000002', '11111111-1111-1111-1111-111111111111', 'bb000001-0000-0000-0000-000000000001', '55aaa004-0000-0000-0000-000000000004', 'national', 3, 9980,  'BRL', 'paid'),
    ('77aaa003-0000-0000-0000-000000000003', '11111111-1111-1111-1111-111111111111', 'bb000002-0000-0000-0000-000000000002', '55aaa002-0000-0000-0000-000000000002', 'national', 3, 2990,  'BRL', 'paid'),
    ('77aaa004-0000-0000-0000-000000000004', '11111111-1111-1111-1111-111111111111', 'bb000003-0000-0000-0000-000000000003', '55aaa003-0000-0000-0000-000000000003', 'national', 3, 24990, 'BRL', 'paid'),
    ('77aaa005-0000-0000-0000-000000000005', '11111111-1111-1111-1111-111111111111', 'bb000005-0000-0000-0000-000000000005', '55aaa006-0000-0000-0000-000000000006', 'national', 3, 79900, 'BRL', 'pending'),
    ('77aaa006-0000-0000-0000-000000000006', '11111111-1111-1111-1111-111111111111', 'bb000004-0000-0000-0000-000000000004', '55aaa005-0000-0000-0000-000000000005', 'national', 3, 2990,  'BRL', 'cancelled'),
    -- Acme invoices
    ('77aaa007-0000-0000-0000-000000000007', '22222222-2222-2222-2222-222222222222', 'bb000006-0000-0000-0000-000000000006', '55aaa008-0000-0000-0000-000000000008', 'national', 3, 1990,  'BRL', 'paid'),
    ('77aaa008-0000-0000-0000-000000000008', '22222222-2222-2222-2222-222222222222', 'bb000008-0000-0000-0000-000000000008', '55aaa009-0000-0000-0000-000000000009', 'international', 3, 59970, 'BRL', 'paid'),
    ('77aaa009-0000-0000-0000-000000000009', '22222222-2222-2222-2222-222222222222', 'bb000010-0000-0000-0000-000000000010', '55aaa010-0000-0000-0000-000000000010', 'national', 3, 4990,  'BRL', 'pending'),
    ('77aaa010-0000-0000-0000-000000000010', '22222222-2222-2222-2222-222222222222', 'bb000007-0000-0000-0000-000000000007', '55aaa011-0000-0000-0000-000000000011', 'national', 3, 4990,  'BRL', 'paid')
ON CONFLICT (organization_id, id) DO NOTHING;

-- ===================== INVOICE ITEMS =====================

INSERT INTO invoice_items (id, organization_id, invoice_id, description, quantity, unit_price_cents, total_price_cents) VALUES
    ('88aaa001-0000-0000-0000-000000000001', '11111111-1111-1111-1111-111111111111', '77aaa001-0000-0000-0000-000000000001', 'Plano Profissional Mensal',  1.0, 7990,  7990),
    ('88aaa002-0000-0000-0000-000000000002', '11111111-1111-1111-1111-111111111111', '77aaa002-0000-0000-0000-000000000002', 'SMS 1000 créditos x2',       2.0, 4990,  9980),
    ('88aaa003-0000-0000-0000-000000000003', '11111111-1111-1111-1111-111111111111', '77aaa003-0000-0000-0000-000000000003', 'Plano Básico Mensal',        1.0, 2990,  2990),
    ('88aaa004-0000-0000-0000-000000000004', '11111111-1111-1111-1111-111111111111', '77aaa004-0000-0000-0000-000000000004', 'Plano Enterprise Mensal',    1.0, 24990, 24990),
    ('88aaa005-0000-0000-0000-000000000005', '11111111-1111-1111-1111-111111111111', '77aaa005-0000-0000-0000-000000000005', 'Plano Profissional Anual',   1.0, 79900, 79900),
    ('88aaa006-0000-0000-0000-000000000006', '11111111-1111-1111-1111-111111111111', '77aaa006-0000-0000-0000-000000000006', 'Plano Básico Mensal',        1.0, 2990,  2990),
    ('88aaa007-0000-0000-0000-000000000007', '22222222-2222-2222-2222-222222222222', '77aaa007-0000-0000-0000-000000000007', 'Starter Monthly',            1.0, 1990,  1990),
    ('88aaa008-0000-0000-0000-000000000008', '22222222-2222-2222-2222-222222222222', '77aaa008-0000-0000-0000-000000000008', 'Enterprise Monthly x3',      3.0, 19990, 59970),
    ('88aaa009-0000-0000-0000-000000000009', '22222222-2222-2222-2222-222222222222', '77aaa009-0000-0000-0000-000000000009', 'Business Monthly',           1.0, 4990,  4990),
    ('88aaa010-0000-0000-0000-000000000010', '22222222-2222-2222-2222-222222222222', '77aaa010-0000-0000-0000-000000000010', 'Business Monthly',           1.0, 4990,  4990)
ON CONFLICT (id) DO NOTHING;

-- ===================== PAYMENTS =====================

INSERT INTO payments (id, organization_id, invoice_id, gateway_name, gateway_reference, payment_method, amount_cents, status, idempotency_key) VALUES
    -- I9Pay transactions
    ('99aaa001-0000-0000-0000-000000000001', '11111111-1111-1111-1111-111111111111', '77aaa001-0000-0000-0000-000000000001', 'stripe',  'pi_dev_001', 'credit_card', 7990,  'succeeded',          'idem_dev_001'),
    ('99aaa002-0000-0000-0000-000000000002', '11111111-1111-1111-1111-111111111111', '77aaa002-0000-0000-0000-000000000002', 'stripe',  'pi_dev_002', 'credit_card', 9980,  'succeeded',          'idem_dev_002'),
    ('99aaa003-0000-0000-0000-000000000003', '11111111-1111-1111-1111-111111111111', '77aaa003-0000-0000-0000-000000000003', 'stripe',  'pi_dev_003', 'pix',         2990,  'succeeded',          'idem_dev_003'),
    ('99aaa004-0000-0000-0000-000000000004', '11111111-1111-1111-1111-111111111111', '77aaa004-0000-0000-0000-000000000004', 'stripe',  'pi_dev_004', 'credit_card', 24990, 'succeeded',          'idem_dev_004'),
    ('99aaa005-0000-0000-0000-000000000005', '11111111-1111-1111-1111-111111111111', '77aaa005-0000-0000-0000-000000000005', 'stripe',  'pi_dev_005', 'boleto',      79900, 'pending',            'idem_dev_005'),
    ('99aaa006-0000-0000-0000-000000000006', '11111111-1111-1111-1111-111111111111', '77aaa001-0000-0000-0000-000000000001', 'stripe',  'pi_dev_006', 'credit_card', 7990,  'refunded',           'idem_dev_006'),
    ('99aaa007-0000-0000-0000-000000000007', '11111111-1111-1111-1111-111111111111', '77aaa003-0000-0000-0000-000000000003', 'stripe',  'pi_dev_007', 'credit_card', 2990,  'failed',             'idem_dev_007'),
    -- Acme transactions
    ('99aaa008-0000-0000-0000-000000000008', '22222222-2222-2222-2222-222222222222', '77aaa007-0000-0000-0000-000000000007', 'pagarme', 'tr_acme_001', 'pix',        1990,  'succeeded',          'idem_acme_001'),
    ('99aaa009-0000-0000-0000-000000000009', '22222222-2222-2222-2222-222222222222', '77aaa008-0000-0000-0000-000000000008', 'pagarme', 'tr_acme_002', 'credit_card',59970, 'succeeded',          'idem_acme_002'),
    ('99aaa010-0000-0000-0000-000000000010', '22222222-2222-2222-2222-222222222222', '77aaa009-0000-0000-0000-000000000009', 'pagarme', 'tr_acme_003', 'boleto',     4990,  'pending',            'idem_acme_003'),
    ('99aaa011-0000-0000-0000-000000000011', '22222222-2222-2222-2222-222222222222', '77aaa010-0000-0000-0000-000000000010', 'pagarme', 'tr_acme_004', 'pix',        4990,  'succeeded',          'idem_acme_004'),
    ('99aaa012-0000-0000-0000-000000000012', '22222222-2222-2222-2222-222222222222', '77aaa008-0000-0000-0000-000000000008', 'pagarme', 'tr_acme_005', 'credit_card',59970, 'partially_refunded', 'idem_acme_005')
ON CONFLICT (id) DO NOTHING;

-- ===================== USER (novo modelo de identidade global) =====================
-- Senha padrão para todos: "Dev@123456"
-- Hash bcrypt (cost 10): $2a$10$U2ShqBynvPjAUiCnPTypnO6Hjs4IwsLU7dJAmcU.qQkUXfeWBXU1e

INSERT INTO "user" (id, email, name, password) VALUES
    ('ua000001-0000-0000-0000-000000000001', 'dev@i9pay.local',       'Dev User',        '$2a$10$U2ShqBynvPjAUiCnPTypnO6Hjs4IwsLU7dJAmcU.qQkUXfeWBXU1e'),
    ('ua000002-0000-0000-0000-000000000002', 'admin@i9pay.local',     'Admin User',      '$2a$10$U2ShqBynvPjAUiCnPTypnO6Hjs4IwsLU7dJAmcU.qQkUXfeWBXU1e'),
    ('ua000003-0000-0000-0000-000000000003', 'dev2@i9pay.local',      'Developer User',  '$2a$10$U2ShqBynvPjAUiCnPTypnO6Hjs4IwsLU7dJAmcU.qQkUXfeWBXU1e'),
    ('ua000004-0000-0000-0000-000000000004', 'support@i9pay.local',   'Support User',    '$2a$10$U2ShqBynvPjAUiCnPTypnO6Hjs4IwsLU7dJAmcU.qQkUXfeWBXU1e'),
    ('ua000005-0000-0000-0000-000000000005', 'owner@acme.local',      'Acme Owner',      '$2a$10$U2ShqBynvPjAUiCnPTypnO6Hjs4IwsLU7dJAmcU.qQkUXfeWBXU1e'),
    ('ua000006-0000-0000-0000-000000000006', 'admin@acme.local',      'Acme Admin',      '$2a$10$U2ShqBynvPjAUiCnPTypnO6Hjs4IwsLU7dJAmcU.qQkUXfeWBXU1e'),
    ('ua000007-0000-0000-0000-000000000007', 'developer@acme.local',  'Acme Developer',  '$2a$10$U2ShqBynvPjAUiCnPTypnO6Hjs4IwsLU7dJAmcU.qQkUXfeWBXU1e'),
    ('ua000008-0000-0000-0000-000000000008', 'support@acme.local',    'Acme Support',    '$2a$10$U2ShqBynvPjAUiCnPTypnO6Hjs4IwsLU7dJAmcU.qQkUXfeWBXU1e')
ON CONFLICT (email) DO UPDATE SET
    name = EXCLUDED.name,
    password = EXCLUDED.password,
    updated_at = NOW(),
    deleted_at = NULL;

-- ===================== ORGANIZATION MEMBERS =====================
-- dev@i9pay.local também pertence à Acme Corp — útil para testar o fluxo multi-org.

INSERT INTO organization_members (id, organization_id, user_id, role) VALUES
    -- I9Pay Dev Organization
    ('om000001-0000-0000-0000-000000000001', '11111111-1111-1111-1111-111111111111', 'ua000001-0000-0000-0000-000000000001', 'owner'),
    ('om000002-0000-0000-0000-000000000002', '11111111-1111-1111-1111-111111111111', 'ua000002-0000-0000-0000-000000000002', 'admin'),
    ('om000003-0000-0000-0000-000000000003', '11111111-1111-1111-1111-111111111111', 'ua000003-0000-0000-0000-000000000003', 'developer'),
    ('om000004-0000-0000-0000-000000000004', '11111111-1111-1111-1111-111111111111', 'ua000004-0000-0000-0000-000000000004', 'support'),
    -- Acme Corp
    ('om000005-0000-0000-0000-000000000005', '22222222-2222-2222-2222-222222222222', 'ua000005-0000-0000-0000-000000000005', 'owner'),
    ('om000006-0000-0000-0000-000000000006', '22222222-2222-2222-2222-222222222222', 'ua000006-0000-0000-0000-000000000006', 'admin'),
    ('om000007-0000-0000-0000-000000000007', '22222222-2222-2222-2222-222222222222', 'ua000007-0000-0000-0000-000000000007', 'developer'),
    ('om000008-0000-0000-0000-000000000008', '22222222-2222-2222-2222-222222222222', 'ua000008-0000-0000-0000-000000000008', 'support'),
    -- dev@i9pay.local também está na Acme Corp (para testar login multi-org)
    ('om000009-0000-0000-0000-000000000009', '22222222-2222-2222-2222-222222222222', 'ua000001-0000-0000-0000-000000000001', 'admin')
ON CONFLICT (organization_id, user_id) DO UPDATE SET
    role = EXCLUDED.role,
    updated_at = NOW(),
    deleted_at = NULL;
