# Funcionalidades Financeiras do PayFlow
---

## Sumário

O PayFlow é uma plataforma de gestão financeira projetada para modelos de negócio de **Software como Serviço (SaaS)**

Este documento apresenta uma visão de alto nível das funcionalidades financeiras, focando nas **regras de negócio** e no **valor estratégico** de cada componente.

---

## 1. Catálogo de Produtos e Planos

O catálogo é a base de qualquer estratégia de monetização. Ele oferece flexibilidade para que as organizações modelem suas ofertas comerciais.

### 1.1. Produtos
Um **Produto** é a representação de um serviço ou linha de negócio. Ele agrupa diferentes formas de cobrança.
- **Regra de Negócio:** Produtos podem ser ativados ou desativados para controlar sua disponibilidade no mercado sem impactar clientes existentes.

### 1.2. Planos de Preço
Um **Plano** define como um produto é vendido.
- **Precificação:** Define um valor fixo e a moeda.
- **Recorrência:** Estabelece a frequência de cobrança (ex: mensal, anual, semanal). Isso permite desde assinaturas simples até ciclos de faturamento customizados (ex: a cada 3 meses).
- **Regras de Negócio:**
    - Um plano pode ser desativado para impedir novas assinaturas, mas continua ativo para os clientes que já o contrataram, garantindo a continuidade do serviço.
    - A alteração de um plano não afeta assinantes atuais, a menos que eles sejam migrados explicitamente.

---

## 2. Gestão de Assinaturas e Recorrência

O motor de assinaturas é o coração do PayFlow, gerenciando o relacionamento contínuo com o cliente.

### 2.1. Modelo de Assinatura
A **Assinatura** formaliza o acordo de serviço recorrente com um cliente.
- **Itens da Assinatura:** Uma assinatura pode conter múltiplos planos e quantidades. Isso viabiliza modelos de negócio complexos como:
    - **Preço por Usuário:** A quantidade do item reflete o número de licenças.
    - **Módulos Adicionais (Add-ons):** Inclusão de diferentes planos na mesma assinatura.
    - **Bundles:** Pacotes de serviços combinados.

### 2.2. Ciclo de Vida da Assinatura
O status de uma assinatura reflete a saúde do relacionamento com o cliente. O sistema gerencia automaticamente as transições entre os seguintes estados:
- `trialing`: Período de teste gratuito.
- `active`: Cliente ativo e com pagamentos em dia.
- `past_due`: Pagamento pendente. O sistema inicia um processo de retentativa de cobrança (dunning).
- `unpaid`: Inadimplente por um período prolongado.
- `paused`: Pausa temporária na cobrança e no serviço.
- `canceled`: Assinatura encerrada.

**Regra de Negócio:** Cada mudança de estado é registrada em um histórico imutável, garantindo uma trilha de auditoria completa para cada cliente.

### 2.3. Agendamento de Alterações (Schedules)
Permite programar mudanças futuras em uma assinatura, como:
- **Upgrade ou Downgrade de Plano:** A alteração ocorre automaticamente no início do próximo ciclo de faturamento.
- **Cancelamento Agendado:** O cliente continua com acesso até o fim do período já pago.

### 2.4. Motor de Recorrência
É o componente que automatiza o ciclo de faturamento. Em cada ciclo, o motor é responsável por:
1.  Identificar as assinaturas que precisam ser renovadas.
2.  Gerar a fatura correspondente ao período.
3.  Iniciar o processo de pagamento.
4.  Avançar o ciclo da assinatura para a próxima data de faturamento.

**Regra de Negócio:** O processo é desenhado para ser tolerante a falhas e evitar cobranças duplicadas, mesmo em cenários de alta concorrência.

---

## 3. Processamento de Pagamentos

O PayFlow orquestra o fluxo de cobrança, desde a solicitação até a confirmação.

### 3.1. Modelo de Pagamento
Um **Pagamento** é uma única transação financeira.
- **Rastreabilidade:** Cada pagamento está vinculado a um cliente e, opcionalmente, a uma fatura ou assinatura.
- **Idempotência:** **Regra de Negócio Crítica:** Uma tentativa de pagamento duplicada (ex: por um clique duplo do usuário ou falha de rede) não resultará em uma cobrança duplicada. O sistema identifica e bloqueia a segunda operação, retornando o resultado da primeira.

### 3.2. Ciclo de Vida do Pagamento
O status de um pagamento indica sua fase no fluxo de cobrança:
- `pending`: Aguardando processamento.
- `authorized`: O valor foi reservado no meio de pagamento do cliente.
- `succeeded`: O valor foi capturado com sucesso.
- `failed`: A cobrança falhou.
- `canceled`: A cobrança foi cancelada antes da conclusão.
- `refunded`: O valor foi devolvido ao cliente.

**Regra de Negócio:** Todas as transições de estado são controladas por uma máquina de estados (FSM) e auditadas. Isso impede, por exemplo, que um pagamento falho seja marcado como bem-sucedido.

### 3.3. Integração com Gateways de Pagamento
O PayFlow é desacoplado de provedores específicos (Stripe, MercadoPago, etc.).
- **Flexibilidade:** Permite adicionar novos gateways sem alterar a lógica de negócio principal.
- **Orquestração:** O sistema seleciona o gateway apropriado com base no método de pagamento e na configuração da organização.

### 3.4. Reembolsos (Refunds)
- **Total ou Parcial:** A plataforma suporta a devolução total ou parcial de um pagamento já efetuado.
- **Regra de Negócio:** Um reembolso só pode ser iniciado para pagamentos no estado `succeeded`. O sistema atualiza o status do pagamento e registra a operação no ledger financeiro.

---

## 4. Faturamento e Conformidade Fiscal

A plataforma garante que as transações comerciais sejam documentadas de forma fiscalmente compatível.

### 4.1. Modelo de Fatura (Invoice)
A **Fatura** é o documento comercial que detalha uma transação.
- **Snapshot de Dados:** No momento da emissão, a fatura "congela" os dados do cliente e os itens vendidos. Essa imutabilidade é crucial para auditoria e conformidade fiscal, garantindo que o registro histórico permaneça intacto mesmo que os dados originais (do cliente ou do plano) mudem.
- **Tipos:** Suporta faturamento para transações nacionais e internacionais.

### 4.2. Integração com Sistemas Fiscais (ERP)
O PayFlow foi projetado para se integrar a sistemas de emissão de notas fiscais (NFe/NFSe).
- **Configuração Flexível:** Cada organização pode configurar suas próprias credenciais e parâmetros fiscais.
- **Fluxo Automatizado:**
    1.  Uma fatura é gerada no PayFlow.
    2.  Um worker de sincronização envia os dados para o sistema fiscal.
    3.  O sistema aguarda a confirmação da emissão e atualiza o status da fatura.
- **Auditoria:** Todo o processo de comunicação com o sistema fiscal é registrado em logs detalhados, incluindo tentativas, erros e sucessos.

---


## 5. Carteiras e Ledger Financeiro

Este módulo oferece uma estrutura de contabilidade interna para gestão de saldos.

### 5.1. Carteira (Wallet)
Uma **Wallet** representa um saldo financeiro.
- **Casos de Uso:**
    1.  **Saldo da Organização:** Acumula os recebíveis que a organização pode sacar (payout).
    2.  **Crédito do Cliente:** Funciona como um saldo pré-pago que o cliente pode usar para abater futuras cobranças.
- **Saldos:** A carteira distingue entre `saldo disponível` e `saldo bloqueado` (ex: valores em disputa ou aguardando liberação).

### 5.2. Ledger Financeiro
O **Ledger** é um livro-razão imutável que registra cada movimentação financeira.
- **Atomicidade:** Cada entrada é um registro de débito ou crédito que não pode ser alterado.
- **Categorias:** As movimentações são categorizadas para clareza contábil (`pagamento`, `reembolso`, `comissão`, `saque`, `ajuste manual`).
- **Rastreabilidade:** Cada entrada no ledger está vinculada à sua origem (um pagamento, uma assinatura, etc.), garantindo uma trilha de auditoria completa.

### 5.3. Comissões e Payouts
- **Comissões:** O sistema suporta a configuração de regras de comissionamento (fixas ou percentuais) que são aplicadas automaticamente sobre os pagamentos.
- **Payouts (Saques):** As organizações podem solicitar a retirada de seu saldo disponível. O sistema gerencia o fluxo desde a solicitação até a confirmação da transferência bancária.

---

## 6. Cupons e Descontos (Planejado)

Este módulo gerenciará a aplicação de descontos e cupons promocionais às assinaturas e pagamentos, oferecendo flexibilidade total nas estratégias de promoção.

### 6.1. Tipos de Desconto
O sistema suportará dois modelos de desconto:
- **Percentual:** Redução de um percentual do valor (ex: 10% de desconto).
- **Valor Fixo:** Redução de um valor absoluto em centavos (ex: R$ 50,00 de desconto).

### 6.2. Cupons (Promocional)
Um **Cupom** é um código reutilizável que clientes podem aplicar para obter desconto.
- **Identificador:** Código único e amigável (ex: `SUMMER2026`, `WELCOME10`).
- **Tipo de Desconto:** Percentual ou valor fixo.
- **Validade:** Data de início e data de expiração.
- **Restrições:**
    - Máximo de utilizações totais (ex: 100 cupons disponíveis).
    - Máximo de utilizações por cliente (ex: um cupom por cliente).
    - Aplicável apenas a planos ou produtos específicos.
    - Valor mínimo de compra requerido.
- **Regra de Negócio:** Um cupom expirado ou esgotado não pode ser aplicado a novas assinaturas. Assinaturas já criadas com o cupom mantêm o desconto até o fim de sua validade ou conforme configurado.

### 6.3. Descontos Manuais (Ad-hoc)
Um **Desconto Manual** é aplicado diretamente a uma assinatura ou pagamento por um administrador.
- **Durabilidade:** 
    - **Único:** Aplicado apenas ao próximo pagamento.
    - **Recorrente:** Aplicado a todos os pagamentos até uma data de término.
    - **Permanente:** Aplicado indefinidamente.
- **Aprovação:** Requer permissão específica no RBAC (ex: `discount_apply`).

### 6.4. Desconto de Período de Prova
Permite oferecer um período inicial com desconto (ex: 50% de desconto nos primeiros 3 meses).
- **Configuração:** Vinculado à assinatura na criação.
- **Término Automático:** Desconto expira automaticamente após o período configurado.

### 6.5. Impacto no Faturamento
- **Cálculo:** O desconto é aplicado ao valor total da fatura antes da emissão.
- **Snapshot:** A fatura registra o valor original, o desconto aplicado e o valor final.
- **Rastreabilidade:** O ledger financeiro registra o desconto como uma categoria separada, permitindo análise de impacto de promoções.

### 6.6. Regras de Negócio Críticas
- **Não Cumulativo:** Apenas um cupom ou desconto por assinatura por período.
- **Limite de Valor:** Um desconto nunca pode resultar em valor negativo ou zero.
- **Auditoria:** Todos os descontos são registrados com quem os aplicou e quando.

---


## 7. Segurança e Conformidade

A segurança é um pilar fundamental em operações financeiras.

- **Controle de Acesso (RBAC):** A plataforma implementa uma hierarquia de papéis (proprietário, administrador, membro) e um sistema de permissões granulares para controlar o acesso a funcionalidades críticas, como reembolsos ou alteração de configurações.
- **Conformidade PCI-DSS:** Dados sensíveis de cartão de crédito **não são armazenados** no PayFlow. Essa responsabilidade é delegada aos gateways de pagamento, que são certificados.
- **Auditoria Completa:** Todas as ações importantes (login, alterações de configuração, operações financeiras) são registradas em logs detalhados, identificando "quem", "o quê", "quando" e "de onde".
- **Webhooks Assinados:** As notificações enviadas pelo PayFlow são assinadas digitalmente, permitindo que os sistemas parceiros verifiquem sua autenticidade e integridade.