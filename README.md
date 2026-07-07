# PayFlow

API de gerenciamento de pagamentos e assinaturas para plataformas SaaS multi-tenant.

## Visão Geral

PayFlow é uma plataforma backend desenvolvida em Go 1.26.1 com Gin Framework, projetada para gerenciar organizations, clientes, assinaturas, pagamentos, carteiras e transações. A arquitetura é modular em camadas orientada a princípios de Clean Architecture, com contratos no domínio e implementações de infraestrutura em repositórios (abordagem pragmática de Ports and Adapters).

Stack: Go 1.26.1 | Gin | PostgreSQL | SQLC | JWT | Vue 3 | TypeScript | Docker

## Início Rápido

```bash
# 1. Clone o repositório
git clone <repository-url>
cd payflow

# 2. Configure os Git hooks do projeto
make setup-hooks

# 3. Suba o stack completo
make build-dev
```

A API estará disponível em `http://localhost:8080`.

```bash
# Verifique se está rodando
curl http://localhost:8080/v1/live
```

---

## Documentação

- [SETUP.md](docs/SETUP.md) — instalação e configuração local com Docker
- [API.md](docs/API.md) — endpoints, exemplos e referência completa
- [ARCHITECTURE.md](docs/ARCHITECTURE.md) — padrões e decisões de arquitetura
- [SECURITY_OBSERVABILITY.md](docs/SECURITY_OBSERVABILITY.md) — rate limiting, logging e validação
- [DEVELOPMENT.md](docs/DEVELOPMENT.md) — guia de contribuição e adição de features
- [COMMANDS.md](docs/COMMANDS.md) — referência de comandos disponíveis

---

## Estrutura de Pastas

```
payflow/
├── cmd/server/               — ponto de entrada (main.go)
├── deploy/
│   ├── docker/               — Dockerfiles (dev/prod)
│   └── migrations/           — SQL migrations versionadas
├── docs/                     — documentação
├── internal/
│   ├── api/
│   │   ├── handler/          — handlers consolidados (um arquivo por recurso)
│   │   ├── middleware/       — auth, logging, rate limiting, errors, maintenance
│   │   ├── dto/              — Data Transfer Objects e query filter helpers
│   │   ├── err/              — tipos de erro e tradução de validação
│   │   └── router/           — roteador e composition root
│   ├── auth/token/           — JWT service
│   ├── config/               — configurações e feature flags
│   ├── crypto/               — utilitários criptográficos centralizados
│   ├── gateway/              — adaptadores de gateway de pagamento
│   ├── monitor/              — integração com Sentry
│   ├── queue/                — interface de fila e implementação in-memory
│   ├── service/              — lógica de negócio
│   ├── domain/               — modelos, contratos e erros de domínio
│   ├── worker/               — webhook, recorrência e cleanup workers
│   └── database/
│       ├── queries/          — SQL (input do SQLC, um arquivo por entidade)
│       ├── repository/       — implementações de repositórios
│       └── dbgen/            — código gerado pelo SQLC
├── Makefile                  — comandos de build e desenvolvimento
├── go.mod / go.sum           — dependências Go
├── sqlc.yml                  — configuração SQLC
└── web/                      — frontend Vue 3 + TypeScript
    └── src/
        ├── views/            — páginas (customers, users, dashboard, login)
        ├── components/       — componentes reutilizáveis
        ├── stores/           — Pinia stores (auth, status)
        ├── services/         — cliente HTTP Axios
        ├── composables/      — composables (useAuth, useCrud, useToast)
        ├── types/            — TypeScript DTO types
        └── router/           — Vue Router com guards de autenticação
```

---

## Funcionalidades

**Autenticação e Acesso**
- JWT com access/refresh tokens (email/password e API Key)
- RBAC com hierarquia de roles: `owner > admin > developer > support`
- 30 permissões granulares por membro com overrides individuais e cache de 5min
- Idempotência via `X-Idempotency-Key` (24h TTL, POST/PATCH)
- Rate limiting em 3 tiers: IP, login e organization

**Gestão de Usuários e Organizations**
- Gerenciamento de membros (convite, roles, remoção; owner único por org)
- Audit trail de membros (log de ações autenticadas via JWT)
- Gerenciamento de credenciais (API Keys: criar, rotacionar, revogar)
- Audit trail de credenciais (log de requisições autenticadas via API Key)

**Clientes e Produtos**
- Gerenciamento de clientes (CRUD, soft/hard delete, LGPD/GDPR)
- Gerenciamento de endereços (CRUD vinculado a clientes)
- Gerenciamento de produtos (CRUD, soft/hard delete)
- Gerenciamento de planos (CRUD, versionamento automático em mudanças de preço)

**Pagamentos e Assinaturas**
- Gerenciamento de pagamentos com FSM de 8 estados e `SELECT FOR UPDATE`
- Motor de cobrança recorrente (cron scheduler + worker + idempotência por período)
- Gerenciamento de assinaturas (FSM, `cancel_at_period_end`, substituição de itens, schedules)

**Ledger e Carteiras**
- Carteiras por organization e por cliente com saldo e saldo reservado
- Ledger imutável: toda movimentação de saldo gera um `ledger_entry`
- Saques com reserva de saldo (`WalletLock`) e débito efetivo na liquidação
- Ajustes manuais de saldo (owner only)
- Configurações de comissão por cliente (percentual ou fixo, creditadas automaticamente no pagamento)

**Faturamento**
- Notas fiscais com FSM de 5 estados (`pending → sent → paid/error/canceled`)
- Snapshots imutáveis de cliente e itens capturados via trigger no momento da emissão
- Configurações de emissão fiscal por tipo (nacional/internacional) com integração ERP
- Audit trail de operações ERP via `invoice_logs`
- Geração automática de invoices a partir de assinaturas
- Eventos de webhook: `invoice.created`, `invoice.paid`, `invoice.payment_failed`

**Infraestrutura**
- Webhooks (endpoints CRUD, entrega assíncrona com HMAC-SHA256 e retry exponencial)
- Health check com monitor em background (liveness/readiness separados)
- Modo manutenção via feature flag
- Monitoramento de erros com Sentry (opcional)
- Logging estruturado com slog (JSON em produção, texto em desenvolvimento)
- Cleanup agendado de logs e chaves de idempotência expiradas
- Queries type-safe geradas pelo SQLC
- Migrations versionadas com golang-migrate
- Docker para desenvolvimento (hot reload via Air) e produção

---

## Referências

Novo no projeto:
1. Leia [SETUP.md](docs/SETUP.md) para configurar o ambiente local
2. Teste os endpoints com os exemplos em [API.md](docs/API.md)
3. Explore a arquitetura em [ARCHITECTURE.md](docs/ARCHITECTURE.md)

Para contribuir:
1. Veja [DEVELOPMENT.md](docs/DEVELOPMENT.md) para adicionar features
2. Consulte [COMMANDS.md](docs/COMMANDS.md) para todos os comandos disponíveis
3. Execute `make help` para a lista rápida de comandos
