-- Contratos ativos de assinatura.
CREATE TABLE IF NOT EXISTS subscriptions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id),
    customer_id UUID NOT NULL,
    status subscription_status_type NOT NULL, -- Situação cíclica atual da assinatura.
    current_period_start TIMESTAMP WITH TIME ZONE NOT NULL,
    current_period_end TIMESTAMP WITH TIME ZONE NOT NULL,
    cancel_at_period_end BOOLEAN DEFAULT false,
    payment_method VARCHAR(50) NOT NULL DEFAULT '',
    billing_locked_until TIMESTAMPTZ NULL,
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

-- Histórico de mudanças de estado das assinaturas.
CREATE TABLE IF NOT EXISTS subscription_status_history (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id),
    subscription_id UUID NOT NULL,

    old_status subscription_status_type,          -- Estado anterior da assinatura.
    new_status subscription_status_type NOT NULL, -- Novo estado da assinatura.
    reason VARCHAR(100),                          -- Motivo operacional que originou a mudança.
    metadata JSONB,
    event_source_id UUID,                         -- Entidade associada que disparou a mudança (payment ou invoice).
    event_source VARCHAR(50),                     -- Origem da transição: "api", "system", "webhook".
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),

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
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),

    CONSTRAINT fk_sub_schedules_subscription_organization
        FOREIGN KEY (organization_id, subscription_id)
        REFERENCES subscriptions (organization_id, id) ON DELETE CASCADE
);

-- Indexes
CREATE INDEX IF NOT EXISTS idx_subscriptions_customer_id
    ON subscriptions (customer_id);

CREATE INDEX IF NOT EXISTS idx_subscriptions_status
    ON subscriptions (status) WHERE status IN ('active', 'past_due');

CREATE INDEX IF NOT EXISTS idx_subscriptions_period_end
    ON subscriptions (current_period_end);

CREATE INDEX IF NOT EXISTS idx_subscriptions_billing
    ON subscriptions(current_period_end)
    WHERE status IN ('active', 'past_due', 'trialing') AND billing_locked_until IS NULL;

CREATE INDEX IF NOT EXISTS idx_subscription_items_subscription_id
    ON subscription_items (subscription_id);

CREATE INDEX IF NOT EXISTS idx_subscription_items_plan_id
    ON subscription_items (plan_id);

CREATE INDEX IF NOT EXISTS idx_sub_history_subscription
    ON subscription_status_history (subscription_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_subs_schedules_pending
    ON subscription_schedules (scheduled_at) WHERE applied_at IS NULL;

-- Triggers
CREATE TRIGGER tr_subscriptions_updated_at
    BEFORE UPDATE ON subscriptions
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER tr_subscription_items_updated_at
    BEFORE UPDATE ON subscription_items
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER tr_subscription_schedules_updated_at
    BEFORE UPDATE ON subscription_schedules
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER tr_subscription_status_history_updated_at
    BEFORE UPDATE ON subscription_status_history
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- NOTE: subscription status history is now recorded by the Go backend (SubscriptionService.Transition),
-- not by a database trigger. This provides rich context (event_source, event_source_id, reason)
-- within the same transaction that updates the status.
-- The trigger fn_log_subscription_status_change and tr_subscriptions_status_history have been removed.
