-- Criação de tipos personalizados
CREATE TYPE environment_type AS ENUM ('test', 'production');
CREATE TYPE address_type AS ENUM ('billing', 'shipping', 'home');
CREATE TYPE subscription_status_type AS ENUM (
    'trialing',     -- Período de teste
    'active',       -- Ativo: Assinatura válida
    'past_due',     -- Vencido: Pagamento não efetuado mas acesso não bloqueado
    'unpaid',       -- Não pago: Pagamento não efetuado e acesso bloqueado
    'canceled',     -- Cancelado: Assinatura cancelada, não pode ser retomada
    'paused',       -- Pausa: Acesso bloqueado mas pode ser retomado
    'incomplete'    -- Incompleto: Pagamento em processamento
);

CREATE TYPE transaction_status_type AS ENUM (
    'pending',      -- Aguardando ação (ex: Boleto gerado, PIX aguardando scan)
    'authorized',   -- Cartão: Valor reservado, mas não capturado (comum em fluxos de 2 etapas)
    'succeeded',    -- Sucesso total: Dinheiro confirmado
    'failed',       -- Falha: Cartão recusado, erro no gateway
    'canceled',     -- Cancelada antes de ser paga
    'disputed',     -- Chargeback: O cliente contestou a compra no banco
    'refunded',     -- Estornada: Você devolveu o dinheiro ao cliente
    'partially_refunded' -- Estorno parcial
);

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

-- Usuários com acesso ao painel de cada Organization.
CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    email VARCHAR(255) NOT NULL,
    name VARCHAR(255) NOT NULL,
    password VARCHAR(255) NOT NULL,
    role VARCHAR(20) NOT NULL DEFAULT 'member' CHECK (role IN ('owner', 'admin', 'member')),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    deleted_at TIMESTAMP WITH TIME ZONE,

    CONSTRAINT unique_user_email_per_organization UNIQUE (organization_id, email)
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

    CONSTRAINT fk_credential_logs_credential_organization
        FOREIGN KEY (organization_id, credential_id)
        REFERENCES credentials (organization_id, id) ON DELETE SET NULL
);

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

    CONSTRAINT fk_webhook_logs_endpoint_organization
        FOREIGN KEY (organization_id, endpoint_id)
        REFERENCES webhook_endpoints (organization_id, id) ON DELETE CASCADE
);

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

-- Contratos ativos de assinatura.
CREATE TABLE IF NOT EXISTS subscriptions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id),
    customer_id UUID NOT NULL,
    status subscription_status_type NOT NULL, -- Situação cíclica atual da assinatura.
    current_period_start TIMESTAMP WITH TIME ZONE NOT NULL,
    current_period_end TIMESTAMP WITH TIME ZONE NOT NULL,
    cancel_at_period_end BOOLEAN DEFAULT false,
    metadata JSONB,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),

    CONSTRAINT fk_subscriptions_customer_organization
        FOREIGN KEY (organization_id, customer_id)
        REFERENCES customers (organization_id, id),

    CONSTRAINT unique_subscription_organization_id UNIQUE (organization_id, id)
);

-- Itens vinculados às assinaturas, detalhando os planos contratados.
CREATE TABLE IF NOT EXISTS subscription_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id),
    subscription_id UUID NOT NULL,
    plan_id UUID NOT NULL,
    quantity INTEGER DEFAULT 1 CHECK (quantity > 0),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),

    CONSTRAINT fk_sub_items_subscription_organization
        FOREIGN KEY (organization_id, subscription_id)
        REFERENCES subscriptions (organization_id, id) ON DELETE CASCADE,

    CONSTRAINT fk_sub_items_plan_organization
        FOREIGN KEY (organization_id, plan_id)
        REFERENCES plans (organization_id, id)
);

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

-- Registros financeiros brutos vinculados a faturas.
CREATE TABLE IF NOT EXISTS transactions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id),
    invoice_id UUID,

    gateway_name VARCHAR(50),       -- Gateway que processou o pagamento.
    gateway_reference VARCHAR(255), -- Identificador externo atribuído pelo gateway.
    payment_method VARCHAR(50),     -- Meio de pagamento utilizado.
    amount_cents INTEGER NOT NULL CHECK (amount_cents >= 0),
    status transaction_status_type NOT NULL, -- Estado financeiro do pagamento.

    idempotency_key VARCHAR(255) UNIQUE, -- Token para blindagem contra retentativas duplicadas.
    error_log JSONB,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),

    CONSTRAINT fk_transactions_invoice_organization
        FOREIGN KEY (organization_id, invoice_id)
        REFERENCES invoices (organization_id, id) ON DELETE SET NULL
);

-- Controle de idempotência das requisições.
CREATE TABLE IF NOT EXISTS idempotency_keys (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,

    idempotency_key TEXT NOT NULL,  -- Hash identificador da requisição do consumidor.
    request_path TEXT NOT NULL,     -- Endpoint relacionado ao token gerado.
    response_code INTEGER,          -- Código HTTP cacheado para replay.
    response_body JSONB,            -- Resposta serializada para replay em chamadas repetidas.
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),

    CONSTRAINT unique_idempotency_key_per_organization
        UNIQUE (organization_id, idempotency_key)
);

-- Movimentações de saldo do cliente.
CREATE TABLE IF NOT EXISTS customer_balance_transactions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id),
    customer_id UUID NOT NULL,

    type VARCHAR(10) NOT NULL CHECK (type IN ('credit', 'debit')), -- Direção monetária da movimentação.
    amount_cents INTEGER NOT NULL CHECK (amount_cents > 0),
    description TEXT,

    subscription_id UUID REFERENCES subscriptions(id), -- Assinatura que originou a movimentação.
    invoice_id UUID REFERENCES invoices(id),           -- Fatura que originou a movimentação.

    metadata JSONB,                                    -- Dados auxiliares, incluindo cálculos de pro-rata.
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    deleted_at TIMESTAMP WITH TIME ZONE,

    CONSTRAINT fk_balance_customer_organization
        FOREIGN KEY (organization_id, customer_id)
        REFERENCES customers (organization_id, id)
);

-- Histórico de mudanças de estado das assinaturas.
CREATE TABLE IF NOT EXISTS subscription_status_history (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id),
    subscription_id UUID NOT NULL,

    old_status subscription_status_type,          -- Estado anterior da assinatura.
    new_status subscription_status_type NOT NULL, -- Novo estado da assinatura.
    reason VARCHAR(100),                          -- Motivo operacional que originou a mudança.
    metadata JSONB,
    event_source_id UUID,                         -- Entidade associada que disparou a mudança (transaction ou invoice).
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),

    CONSTRAINT fk_sub_history_subscription_organization
        FOREIGN KEY (organization_id, subscription_id)
        REFERENCES subscriptions (organization_id, id) ON DELETE CASCADE
);

-- Agendamento de alterações futuras em assinaturas.
CREATE TABLE IF NOT EXISTS subscription_schedules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id),
    subscription_id UUID NOT NULL,

    scheduled_at TIMESTAMP WITH TIME ZONE NOT NULL,
    target_status subscription_status_type, -- Estado programado para aplicação na data agendada.
    target_plan_id UUID,                    -- Plano a ser aplicado na data agendada.
    applied_at TIMESTAMP WITH TIME ZONE,    -- Preenchido pelo job após execução bem-sucedida.
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),

    CONSTRAINT fk_sub_schedules_subscription_organization
        FOREIGN KEY (organization_id, subscription_id)
        REFERENCES subscriptions (organization_id, id) ON DELETE CASCADE
);