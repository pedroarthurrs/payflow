# i9pay

API de gerenciamento de pagamentos e assinaturas para plataformas SaaS multi-tenant.

## Visão Geral

i9pay é uma plataforma backend desenvolvida em **Go 1.26.1** com **Gin Framework**, projetada para gerenciar tenants, clientes, assinaturas, faturas e transações. A arquitetura é **modular** e pragmática.

**Stack:**
- Go 1.26.1 | Gin | PostgreSQL | SQLC | JWT | Docker

## Quick Start

```bash
# 1. Clone
git clone <repository-url>
cd i9pay

# 2. Suba o stack completo
make build-dev

# 3. Teste
curl http://localhost:8080/v1/health
```

Pronto! API rodando em `http://localhost:8080`.

---

## Documentação

- **[SETUP.md](/docs/SETUP.md)** — Como instalar e configurar localmente com Docker
- **[API.md](/docs/API.md)** — Documentação completa de endpoints e exemplos cURL
- **[ARCHITECTURE.md](/docs/ARCHITECTURE.md)** — Padrões e decisões de arquitetura
- **[DEVELOPMENT.md](/docs/DEVELOPMENT.md)** — Como contribuir e adicionar features
- **[COMMANDS.md](/docs/COMMANDS.md)** — Referência de comandos disponíveis

---

## Estrutura de Pastas

```
i9pay/
├── cmd/api/                  # Ponto de entrada
├── docker/                   # Dockerfiles (dev/prod)
├── docs/                     # Documentação
├── internal/
│   ├── api/                  # Camada HTTP (router, handlers, middleware)
│   ├── service/              # Lógica de negócio
│   ├── domain/               # Modelos e validações
│   ├── config/               # Configurações
│   ├── database/             # Conexão e queries (SQLC)
│   └── token/                # JWT service
├── migrations/               # SQL migrations versionadas
├── Makefile                  # Comandos úteis
├── go.mod / go.sum           # Dependências
└── sqlc.yml                  # Configuração SQLC
```



---

## Features

✅ Autenticação JWT com access e refresh tokens
✅ Multi-tenant com isolamento de data
✅ Health check com status de dependências
✅ Arquitetura modular e escalável
✅ Migrations versionadas com Go Migrate
✅ Type-safe queries com SQLC
✅ Docker para dev e produção

---

## Próximos Steps

**Novo no projeto?**
1. Leia [SETUP.md](/docs/SETUP.md) para setup local
2. Teste com `curl` os exemplos em [API.md](/docs/API.md)
3. Explore a arquitetura em [ARCHITECTURE.md](/docs/ARCHITECTURE.md)

**Quer contribuir?**
1. Veja [DEVELOPMENT.md](/docs/DEVELOPMENT.md) para adicionar features
2. Consulte [COMMANDS.md](/docs/COMMANDS.md) para todos os comandos
3. Execute `make help` para lista rápida

---

## Support

Dúvidas? Abra uma [issue](../../issues) no repositório.

**Sociais:**  
[GitHub](https://github.com) • [Docs](https://docs)

Licença: MIT
