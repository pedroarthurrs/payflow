# i9pay

API de gerenciamento de pagamentos e assinaturas para plataformas SaaS multi-tenant.

## Visão Geral

i9pay é uma plataforma backend desenvolvida em Go 1.26.1 com Gin Framework, projetada para gerenciar organizations, clientes, assinaturas, faturas e transações. A arquitetura é modular em camadas orientada a princípios de Clean Architecture, com contratos no domínio e implementações de infraestrutura em repositórios (abordagem pragmática de Ports and Adapters).

Stack: Go 1.26.1 | Gin | PostgreSQL | SQLC | JWT | Docker

## Início Rápido

```bash
# 1. Clone o repositório
git clone <repository-url>
cd i9pay

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
i9pay/
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
│   ├── monitor/              — integração com Sentry
│   ├── queue/                — interface de fila e implementação in-memory
│   ├── service/              — lógica de negócio
│   ├── domain/               — modelos, contratos e erros de domínio
│   ├── worker/               — webhook worker com retry e backoff
│   └── database/
│       ├── repository/       — implementações de repositórios
│       └── dbgen/            — código gerado pelo SQLC
├── Makefile                  — comandos de build e desenvolvimento
├── go.mod / go.sum           — dependências
└── sqlc.yml                  — configuração SQLC
```

---

## Funcionalidades

- Autenticação JWT com access e refresh tokens (email/password e API Key)
- RBAC com hierarquia de roles (owner > admin > member)
- Gerenciamento de usuários (CRUD com controle de roles)
- Gerenciamento de clientes (CRUD com soft/hard delete para LGPD/GDPR)
- Gerenciamento de endereços (CRUD vinculado a clientes)
- Gerenciamento de produtos (CRUD com soft/hard delete)
- Gerenciamento de credenciais (API Keys: criar, rotacionar, revogar)
- Audit trail de credenciais (log de requisições autenticadas via API Key)
- Webhooks (endpoints CRUD, entrega assíncrona com retry, logs de entrega)
- Arquitetura multi-organization com isolamento de dados
- Health check com status de dependências (monitor em background)
- Rate limiting com 3 tiers (IP, login, organization)
- Feature flags plugáveis (env vars por padrão, extensível para AppConfig/etc.)
- Monitoramento de erros com Sentry (opcional)
- Logging estruturado com slog (JSON em produção, texto em desenvolvimento)
- Validação de entrada com mensagens traduzidas
- Criptografia centralizada em `internal/crypto` (bcrypt, SHA-256, HMAC-SHA256)
- Queries type-safe geradas pelo SQLC
- Migrations versionadas com golang-migrate
- Docker para desenvolvimento e produção

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
