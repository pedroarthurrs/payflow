-- ----------------------------------------------------
-- TRIGGER: UPDATE_AT
-- ----------------------------------------------------
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Aplicando na tabela de Tenants
CREATE TRIGGER tr_tenants_updated_at
    BEFORE UPDATE ON tenants
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Aplicando na tabela de Customers
CREATE TRIGGER tr_customers_updated_at
    BEFORE UPDATE ON customers
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Aplicando na tabela de Invoices
CREATE TRIGGER tr_invoices_updated_at
    BEFORE UPDATE ON invoices
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Aplicando na tabela de Invoice Settings
CREATE TRIGGER tr_invoice_settings_updated_at
    BEFORE UPDATE ON invoice_settings
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
------------------------------- FIM ---------------------------------------

-- ----------------------------------------------------
-- TRIGGER: LOG DO STATUS DE SUBS
-- ----------------------------------------------------

-- Como o trigger é automático, ele não sabe sozinho se o motivo foi um "Cartão Recusado" ou um "Cancelamento Voluntário".
-- Uma técnica avançada para resolver isso é usar uma variável de configuração temporária na sua transação SQL:
--
-- No seu Backend (Node/Go/Python), antes de dar o Update:
-- SET LOCAL app.change_reason = 'customer_request';
--
-- No Trigger, você captura isso:
-- reason := current_setting('app.change_reason', true);

CREATE OR REPLACE FUNCTION fn_log_subscription_status_change()
RETURNS TRIGGER AS $$
BEGIN
    -- Verifica se o status foi alterado
    IF (OLD.status IS DISTINCT FROM NEW.status) THEN
        INSERT INTO subscription_status_history (
            tenant_id,
            subscription_id,
            old_status,
            new_status,
            reason,
            metadata
        ) VALUES (
            NEW.tenant_id,
            NEW.id,
            OLD.status,
            NEW.status,
            'status_changed_by_system', -- Motivo genérico, pode ser refinado via App
            jsonb_build_object(
                'triggered_at', NOW(),
                'previous_period_end', OLD.current_period_end
            )
        );
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER tr_subscriptions_status_history
    AFTER UPDATE ON subscriptions
    FOR EACH ROW
    EXECUTE FUNCTION fn_log_subscription_status_change();


-- ----------------------------------------------------
-- TRIGGER: SNAPSHOT DE CUSTOMER NA INVOICE
-- ----------------------------------------------------
CREATE OR REPLACE FUNCTION fn_build_invoice_customer_snapshot()
RETURNS TRIGGER AS $$
BEGIN
    -- Só cria automático se não tiver sido enviado manualmente na requisição
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
    -- Identifica o ID da invoice modificado
    IF TG_OP = 'DELETE' THEN
        v_invoice_id := OLD.invoice_id;
    ELSE
        v_invoice_id := NEW.invoice_id;
    END IF;

    -- Atualiza a nota fiscal com o array consolidado dos itens atuais
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

    RETURN NULL; -- Retorna NULL no trigger AFTER
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER tr_invoice_items_snapshot
    AFTER INSERT OR UPDATE OR DELETE ON invoice_items
    FOR EACH ROW
    EXECUTE FUNCTION fn_build_invoice_items_snapshot();

------------------------------- FIM ---------------------------------------

