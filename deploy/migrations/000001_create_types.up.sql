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

CREATE TYPE payment_status_type AS ENUM (
    'pending',      -- Aguardando ação (ex: Boleto gerado, PIX aguardando scan)
    'authorized',   -- Cartão: Valor reservado, mas não capturado (comum em fluxos de 2 etapas)
    'succeeded',    -- Sucesso total: Dinheiro confirmado
    'failed',       -- Falha: Cartão recusado, erro no gateway
    'canceled',     -- Cancelada antes de ser paga
    'disputed',     -- Chargeback: O cliente contestou a compra no banco
    'refunded',     -- Estornada: Você devolveu o dinheiro ao cliente
    'partially_refunded' -- Estorno parcial
);
