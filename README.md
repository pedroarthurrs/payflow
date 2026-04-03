# i9pay

API de gerenciamento de pagamentos e assinaturas para plataformas SaaS multi-tenant.

## Visão Geral

i9pay é uma plataforma backend desenvolvida em Go 1.26.1 com Gin Framework, projetada para gerenciar organizations, clientes, assinaturas, faturas e transações. A arquitetura é modular e pragmática.

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
│   │   ├── middleware/       — auth, logging, rate limiting, errors
│   │   ├── dto/              — Data Transfer Objects compartilhados
│   │   ├── err/              — tipos de erro e tradução de validação
│   │   └── router/           — roteador e inicialização
│   ├── auth/token/           — JWT service
│   ├── service/              — lógica de negócio
│   ├── domain/               — modelos e validações
│   ├── config/               — configurações
│   └── database/             — conexão e queries (SQLC)
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
- Arquitetura multi-organization com isolamento de dados
- Health check com status de dependências (monitor em background)
- Rate limiting com 3 tiers (IP, login, organization)
- Logging estruturado com slog (JSON em produção, texto em desenvolvimento)
- Validação de entrada com mensagens traduzidas
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
