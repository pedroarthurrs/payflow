# Diagrama Relacional do Banco de Dados

## Visão Geral

O banco de dados do i9pay é estruturado em torno da entidade `organizations`, que isola todos os dados por tenant. Cada tabela possui `organization_id` como chave de particionamento lógico, garantindo isolamento multi-tenant.

## Diagrama de Entidade-Relacionamento

```
┌────────────────────────────────────────────────────────────────────────────────────────────┐
│                                       "user"                                               │
│  id (PK) · email (UNIQUE) · name · password_hash · created_at · updated_at · deleted_at   │
└────────────────────────────────────────────┬───────────────────────────────────────────────┘
                                             │ 1
              ┌──────────────────────────────┼──────────────────────────────────┐
              │ N                            │ N                                │ N
┌─────────────▼───────────────────────┐  ┌───▼────────────────────────────┐  ┌──▼──────────────────────┐
│       ORGANIZATION_MEMBERS          │  │      MEMBER_PERMISSIONS        │  │       MEMBER_LOGS       │
│  id (PK)                            │  │  id (PK)                       │  │  id (PK)                │
│  organization_id (FK)               │  │  organization_id (denorm.)     │  │  organization_id        │
│  user_id (FK)                       │  │  member_id (FK)                │  │    (denorm.)            │
│  role                               │  │  permission                    │  │  member_id (FK)         │
│  invited_by (FK, nullable)          │  │  granted (bool)                │  │  action                 │
│  joined_at                          │  │  created_at · updated_at       │  │  method · path          │
│  created_at · updated_at            │  └────────────────────────────────┘  │  status_code · latency  │
└─────────────────────────────────────┘                                      │  ip_address · user_agent│
                                                                             │  metadata               │
                                                                             │  created_at · updated_at†│
                                                                             └─────────────────────────┘


┌─────────────────────────────────────────────────────────────────────────────────────────────┐
│                                      ORGANIZATIONS                                          │
│  id (PK) · name · webhook_secret · status · created_at · updated_at · deleted_at            │
└────────────────────────────────────────────┬────────────────────────────────────────────────┘
                                             │ 1
              ┌──────────────────────────────┼───────────────────────────────┐
              │                              │                               │
              │ N                            │ N                             │ N
┌─────────────▼──────────────┐  ┌────────────▼────────────┐  ┌───────────────▼────────────────┐
│     ORGANIZATION_CONFIGS   │  │  ORGANIZATION_MEMBERS   │  │         CREDENTIALS            │
│  id (PK)                   │  │  id (PK)                │  │  id (PK)                       │
│  organization_id (FK)      │  │  organization_id (FK)   │  │  organization_id (FK)          │
│  support_email             │  │  user_id (FK)           │  │  name                          │
│  logo_url                  │  │  role                   │  │  key_hash (UNIQUE)             │
│  primary_color             │  │  invited_by (FK,        │  │  key_prefix                    │
│  created_at · updated_at   │  │    nullable)            │  │  environment                   │
└────────────────────────────┘  │  joined_at              │  │  last_used_at · expires_at     │
                                │  created_at · updated_at│  │  created_at · updated_at       │
                                │  deleted_at             │  │  deleted_at                    │
                                └─────────────────────────┘                 │ 1
                                                                            │ N
                                                             ┌──────────────▼─────────────────┐
                                                             │        CREDENTIAL_LOGS         │
                                                             │  id (PK)                       │
                                                             │  organization_id (FK)          │
                                                             │  credential_id (FK, nullable)  │
                                                             │  method · path                 │
                                                             │  request_body · response_body  │
                                                             │  status_code · latency_ms      │
                                                             │  ip_address · user_agent       │
                                                             │  created_at · updated_at       │
                                                             └────────────────────────────────┘


┌──────────────────────────────────────────────────────────────────────────────────────────────┐
│                                        CUSTOMERS                                             │
│  id (PK) · organization_id (FK) · name · email · phone · document · document_type            │
│  external_reference · status · metadata · created_at · updated_at · deleted_at               │
└───────────────────┬───────────────────────┬────────────────────────────────┬─────────────────┘
                    │ 1                     │ 1                              │ 1
                    │ N                     │ N                              │ N
       ┌────────────▼──────────┐  ┌─────────▼───────────────┐  ┌─────────────▼────────────────┐
       │       ADDRESSES       │  │     SUBSCRIPTIONS       │  │      COMMISSION_SETTINGS     │
       │  id (PK)              │  │  id (PK)                │  │  id (PK)                     │
       │  organization_id (FK) │  │  organization_id (FK)   │  │  organization_id (FK)        │
       │  customer_id (FK)     │  │  customer_id (FK)       │  │  target_customer_id (FK)     │
       │  type                 │  │  status                 │  │  type · value                │
       │  postal_code · street │  │  current_period_start   │  │  conditions                  │
       │  number · complement  │  │  current_period_end     │  │  is_active                   │
       │  neighborhood · city  │  │  cancel_at_period_end   │  │  created_at · updated_at     │
       │  state · country      │  │  billing_locked_until   │  │  deleted_at                  │
       │  is_main              │  │  payment_method         │  └──────────────────────────────┘
       │  created_at · updated │  │  metadata               │
       │  deleted_at           │  │  created_at · updated_at│
       └───────────────────────┘  └──────────┬──────────────┘
                                             │ 1
                                             │
              ┌──────────────────────────────├─────────┬────────────────────────────────┐
              │ N                                      │ N                              │ N
       ┌──────────────────────────────┐   ┌────────────▼──────────────┐  ┌──────────────▼──────────────┐
       │   SUBSCRIPTION_SCHEDULES     │   │   SUBSCRIPTION_ITEMS      │  │  SUBSCRIPTION_STATUS_HISTORY│
       │  id (PK)                     │   │  id (PK)                  │  │  id (PK)                    │
       │  organization_id (FK)        │   │  organization_id (FK)     │  │  organization_id (FK)       │
       │  subscription_id (FK)        │   │  subscription_id (FK)     │  │  subscription_id (FK)       │
       │  scheduled_at                │   │  plan_id (FK)             │  │  old_status · new_status    │
       │  target_status               │   │  quantity                 │  │  reason · metadata          │
       │  target_plan_id              │   │  created_at · updated_at  │  │  event_source_id            │
       │  applied_at                  │   └───────────────────────────┘  │  created_at · updated_at†   │
       │  created_at · updated_at     │                                  └─────────────────────────────┘
       └──────────────────────────────┘


┌─────────────────────────────────────────────────────────────────────────────────────────────┐
│                                         PRODUCTS                                            │
│  id (PK) · organization_id (FK) · name · active · created_at · updated_at · deleted_at      │
└─────────────────────────────────────────────┬───────────────────────────────────────────────┘
                                              │ 1
                                              │ N
                              ┌───────────────▼────────────────┐
                              │             PLANS              │
                              │  id (PK)                       │
                              │  organization_id (FK)          │
                              │  product_id (FK)               │
                              │  name · slug                   │
                              │  amount_cents · currency       │
                              │  interval · interval_count     │
                              │  active · version              │
                              │  created_at · updated_at       │
                              │  deleted_at                    │
                              └────────────────────────────────┘


┌─────────────────────────────────────────────────────────────────────────────────────────────┐
│                                         INVOICES                                            │
│  id (PK) · organization_id (FK) · customer_id (FK) · subscription_id (FK, nullable)         │
│  invoice_type · operation_type · amount_cents · currency · status · erp_reference           │
│  customer_snapshot · items_snapshot · last_sync_log · created_at · updated_at               │
└───────────────────┬──────────────────────────┬───────────────────────────────┬──────────────┘
                    │ 1                        │ 1                             │ 1
                    │ N                        │ N                             │ N
       ┌────────────▼──────────┐  ┌────────────▼────────────┐  ┌───────────────▼───────────────┐
       │     INVOICE_ITEMS     │  │      INVOICE_LOGS       │  │           PAYMENTS            │
       │  id (PK)              │  │  id (PK)                │  │  id (PK)                      │
       │  organization_id (FK) │  │  organization_id (FK)   │  │  organization_id (FK)         │
       │  invoice_id (FK)      │  │  invoice_id (FK)        │  │  invoice_id (FK, nullable)    │
       │  description          │  │  event_type             │  │  customer_id (FK, nullable)   │
       │  quantity             │  │  invoice_status         │  │  gateway_name                 │
       │  unit_price_cents     │  │  erp_reference          │  │  gateway_reference            │
       │  total_price_cents    │  │  erp_error_code         │  │  payment_method               │
       │  metadata             │  │  erp_error_message      │  │  amount_cents · status        │
       │  created_at · updated │  │  http_status            │  │  currency                     │
       └───────────────────────┘  │  request_url            │  │  idempotency_key (UNIQUE)     │
                                  │  request_payload        │  │  error_log                    │
                                  │  response_body          │  │  created_at · updated_at      │
                                  │  attempt_number         │  └───────────────┬───────────────┘
                                  │  success · message      │                  │ 1
                                  │  created_at · updated_at│                  │ N
                                  └─────────────────────────┘  ┌───────────────▼───────────────┐
                                                               │   PAYMENT_STATUS_HISTORY      │
                                                               │  id (PK)                      │
                                                               │  organization_id (FK)         │
                                                               │  payment_id (FK)              │
                                                               │  old_status · new_status      │
                                                               │  event_source                 │
                                                               │  event_source_id              │
                                                               │  reason                       │
                                                               │  created_at · updated_at†     │
                                                               └───────────────────────────────┘


┌─────────────────────────────────────────────────────────────────────────────────────────────┐
│                                         WALLETS                                             │
│  id (PK) · organization_id (FK) · organization_owner_id (FK) · customer_owner_id (FK)       │
│  balance · currency · status · created_at · updated_at                                      │
│  [CHECK: exatamente um dono — organization_owner_id XOR customer_owner_id]                  │
└───────────────────┬──────────────────────────┬──────────────────────────────────────────────┘
                    │ 1                        │ 1
                    │ N                        │ N
       ┌────────────▼──────────┐  ┌────────────▼────────────┐
       │     LEDGER_ENTRIES    │  │         PAYOUTS         │
       │  id (PK)              │  │  id (PK)                │
       │  organization_id (FK) │  │  organization_id (FK)   │
       │  wallet_id (FK)       │  │  wallet_id (FK)         │
       │  type (credit/debit)  │  │  amount_cents           │
       │  amount_cents         │  │  fee_cents              │
       │  category             │  │  status                 │
       │  balance_after        │  │  bank_info              │
       │  subscription_id (FK) │  │  created_at · updated_at│
       │  invoice_id (FK)      │  └─────────────────────────┘
       │  payment_id (FK)      │
       │  metadata             │
       │  created_at · updated†│
       └───────────────────────┘


┌─────────────────────────────────────────────────────────────────────────────────────────────┐
│                                    WEBHOOK_ENDPOINTS                                        │
│  id (PK) · organization_id (FK) · url · description · enabled_events · environment          │
│  is_active · created_at · updated_at · deleted_at                                           │
└─────────────────────────────────────────────┬───────────────────────────────────────────────┘
                                              │ 1
                                              │ N
                              ┌───────────────▼────────────────┐
                              │         WEBHOOK_LOGS           │
                              │  id (PK)                       │
                              │  organization_id (FK)          │
                              │  endpoint_id (FK)              │
                              │  event_type                    │
                              │  payload                       │
                              │  response_body                 │
                              │  response_status               │
                              │  duration_ms                   │
                              │  attempt_number                │
                              │  success                       │
                              │  created_at · updated_at       │
                              └────────────────────────────────┘


┌─────────────────────────────────────────────────────────────────────────────────────────────┐
│                                    INVOICE_SETTINGS                                         │
│  id (PK) · organization_id (FK) · type · erp_api_key · erp_api_secret · erp_account_id      │
│  nfse_service_code · nfse_item_list_service · tax_rate · invoice_series                     │
│  iso_currency · export_regime · issue_on · is_active · created_at · updated_at              │
└─────────────────────────────────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────────────────────────────────┐
│                                    IDEMPOTENCY_KEYS                                         │
│  id (PK) · organization_id (FK) · idempotency_key · request_path                            │
│  response_code · response_body · expires_at · created_at · updated_at                       │
└─────────────────────────────────────────────────────────────────────────────────────────────┘
```

> **†  Tabelas imutáveis** — `subscription_status_history`, `payment_status_history`, `member_logs` e `ledger_entries` são registros append-only. A coluna `updated_at` existe exclusivamente para rastreabilidade de alterações manuais diretas no banco (DBA / hotfix) e não deve ser atualizada pela aplicação.

## Grupos Funcionais

### Identidade e Acesso
| Tabela | Descrição |
|--------|-----------|
| `organizations` | Tenant raiz. Todos os dados são isolados por `organization_id`. |
| `organization_configs` | Configurações visuais e de suporte por organização (1:1). |
| `"user"` | Identidade global de um usuário, independente de organização. Email único global. |
| `organization_members` | Vínculo entre `"user"` e `organization`. Carrega o `role` (`owner`/`admin`/`developer`/`support`). |
| `member_permissions` | Overrides de permissão granular por membro. FK direto em `organization_members`. `granted=true` concede; `granted=false` revoga. |
| `member_logs` | Audit trail assíncrono de ações de membros autenticados via JWT. Sempre vinculado a um `organization_member` ativo. |
| `credentials` | API Keys para autenticação de sistema. Armazena hash SHA-256 + prefixo de ambiente. |
| `credential_logs` | Audit trail assíncrono de requisições autenticadas via API Key. |

### Clientes e Endereços
| Tabela | Descrição |
|--------|-----------|
| `customers` | Clientes finais do SaaS. Suporta soft delete e metadados customizados. |
| `addresses` | Endereços vinculados a clientes (`billing`, `shipping`, `home`). Flag `is_main` para endereço padrão. |

### Catálogo
| Tabela | Descrição |
|--------|-----------|
| `products` | Produtos comercializados pela organização. |
| `plans` | Regras de preço e recorrência dos produtos. Suporta múltiplos intervalos (`day`, `week`, `month`, `year`). |

### Assinaturas
| Tabela | Descrição |
|--------|-----------|
| `subscriptions` | Contratos ativos de assinatura com ciclo de vida completo. Campo `billing_locked_until` previne double processing pelo motor de recorrência. |
| `subscription_items` | Planos contratados dentro de uma assinatura (N:N entre subscription e plan). |
| `subscription_status_history` | Histórico automático de mudanças de status (via trigger). |
| `subscription_schedules` | Agendamento de alterações futuras (mudança de plano, cancelamento). |

### Faturamento
| Tabela | Descrição |
|--------|-----------|
| `invoices` | Notas fiscais emitidas. Contém snapshots imutáveis de cliente e itens (via trigger). |
| `invoice_items` | Itens de linha da nota fiscal. Atualiza `items_snapshot` na invoice automaticamente. |
| `invoice_logs` | Auditoria técnica das operações com o ERP externo. |
| `invoice_settings` | Configurações de emissão fiscal por organização (NFS-e nacional ou internacional). |

### Pagamentos
| Tabela | Descrição |
|--------|-----------|
| `payments` | Registros financeiros brutos. Vinculados a invoice e/ou customer diretamente. |
| `payment_status_history` | Histórico de transições de status do pagamento. |

### Carteira e Ledger
| Tabela | Descrição |
|--------|-----------|
| `wallets` | Carteira digital. Pertence a uma organização OU a um customer (constraint XOR). |
| `ledger_entries` | Movimentações financeiras imutáveis (crédito/débito) com saldo pós-operação. |
| `payouts` | Saques da carteira para conta bancária. |
| `commission_settings` | Regras de comissão por customer (percentual ou valor fixo). |

### Webhooks
| Tabela | Descrição |
|--------|-----------|
| `webhook_endpoints` | Endpoints registrados para receber eventos. |
| `webhook_logs` | Log de entregas com suporte a retentativas. |

### Infraestrutura
| Tabela | Descrição |
|--------|-----------|
| `idempotency_keys` | Cache de respostas para blindagem contra requisições duplicadas. TTL via `expires_at`. |

## Tipos Enumerados

| Tipo | Valores |
|------|---------|
| `environment_type` | `test`, `production` |
| `address_type` | `billing`, `shipping`, `home` |
| `subscription_status_type` | `trialing`, `active`, `past_due`, `unpaid`, `canceled`, `paused`, `incomplete` |
| `payment_status_type` | `pending`, `authorized`, `succeeded`, `failed`, `canceled`, `disputed`, `refunded`, `partially_refunded` |

## Triggers

| Trigger | Tabela | Evento | Função |
|---------|--------|--------|--------|
| `tr_*_updated_at` | Todas as tabelas com `updated_at` | `BEFORE UPDATE` | Atualiza `updated_at` automaticamente |
| `tr_subscriptions_status_history` | `subscriptions` | `AFTER UPDATE` | Registra mudança de status em `subscription_status_history` |
| `tr_invoices_customer_snapshot` | `invoices` | `BEFORE INSERT` | Captura snapshot do customer no momento da emissão |
| `tr_invoice_items_snapshot` | `invoice_items` | `AFTER INSERT/UPDATE/DELETE` | Recalcula `items_snapshot` na invoice pai |

## Padrões de Design

**Multi-tenancy via `organization_id`**
Todas as tabelas carregam `organization_id`. Chaves estrangeiras compostas `(organization_id, entity_id)` garantem que referências cruzadas nunca vazem entre tenants.

**Soft Delete**
Tabelas com dados sensíveis usam `deleted_at` em vez de remoção física: `organizations`, `"user"`, `credentials`, `customers`, `addresses`, `products`, `plans`, `webhook_endpoints`, `commission_settings`.

**Snapshots Imutáveis**
`invoices` armazena `customer_snapshot` e `items_snapshot` em JSONB para garantir integridade histórica independente de alterações futuras nos dados originais.

**Idempotência**
`payments` possui `idempotency_key` único. A tabela `idempotency_keys` provê cache de resposta para qualquer endpoint da API.

**Auditoria Automática**
`subscription_status_history` e `payment_status_history` registram todas as transições de estado. `credential_logs` registra todas as requisições autenticadas via API Key. `member_logs` registra todas as ações de membros autenticados via JWT (`auth_type=user`); por ser vinculado a `organization_members`, os logs são automaticamente isolados por organização.

**Tabelas Imutáveis**
`subscription_status_history`, `payment_status_history`, `member_logs` e `ledger_entries` são registros append-only — a aplicação nunca executa `UPDATE` nessas tabelas. A coluna `updated_at` está presente exclusivamente para rastreabilidade de alterações manuais diretas (DBA / hotfix de produção), disparada pelo trigger padrão `tr_*_updated_at`.
