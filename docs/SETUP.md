# Setup e Instalação

## Pré-requisitos

- Go 1.26.1+ ([download](https://go.dev/dl/))
- Docker e Docker Compose ([instalação](https://docs.docker.com/get-docker/))
- Make
  - Linux: `sudo apt install make`
  - macOS: `brew install make`
  - Windows: use WSL2 ou `choco install make`

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

---

## Configuração Detalhada

### Variáveis de Ambiente

O arquivo `.env` é único para todos os ambientes. O controle de ambiente é feito pela variável `GO_ENV`.

**Valores possíveis para `GO_ENV`:**
- `development`
- `staging`
- `production`

Em produção, as variáveis devem vir via sistema de secrets do orquestrador (Kubernetes, ECS, etc.) em vez do arquivo `.env`.

### Executando com Docker

A execução local deve ser sempre feita com Docker para padronizar o ambiente.

**Build e execução:**
```bash
make build-dev
```

Este comando constrói as imagens Docker, sobe os containers (API + PostgreSQL) e aplica as migrations automaticamente.

- API: `http://localhost:8080`
- Banco de dados: `localhost:5433`

**Parar os containers:**
```bash
make down
```

**Remover containers e volumes:**
```bash
make remove
```

Use quando quiser limpar completamente e recomeçar do zero.

---

## Verificando o Ambiente

### Health Check

```bash
# Liveness probe
curl http://localhost:8080/v1/live
# Resposta: {"status": "alive"}

# Readiness probe
curl http://localhost:8080/v1/ready
# Resposta: {"status": "ready"} ou 503 {"status": "unavailable"}
```

### Teste de Login

```bash
curl -X POST http://localhost:8080/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email": "test@example.com", "password": "test123"}'
```

---

## Ferramentas de Desenvolvimento

### Git Hooks

Configure os hooks do projeto para garantir que verificações sejam aplicadas localmente:

```bash
make setup-hooks
```

Isso configura o Git para usar os hooks em `.githooks/`.

### Verificação do Projeto

Para rodar formatação, tidy, vet, lint e testes em sequência:

```bash
make check
```

### Ferramentas Recomendadas

**air** — hot reload:
```bash
go install github.com/air-verse/air@latest
```

**golangci-lint** — linter estático:
```bash
curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin
```

O projeto inclui `.golangci.yml` com regras configuradas. Execute `make lint` para rodar.

**sqlc** — geração de código a partir de SQL:
```bash
go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
```

**golang-migrate** — migrations de banco:
```bash
go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
```

**mockery** — geração de mocks para testes:
```bash
go install github.com/vektra/mockery/v2@latest
```

**delve** — debugger:
```bash
go install github.com/go-delve/delve/cmd/dlv@latest
```

**gopls** — language server:
```bash
go install golang.org/x/tools/gopls@latest
```

### Integração com VSCode

Extensões recomendadas:

| Extensão | ID | Propósito |
|----------|----|-----------|
| Go | `golang.go` | Suporte completo à linguagem |
| REST Client | `humao.rest-client` | Testar requisições HTTP no editor |
| Docker | `ms-azuretools.vscode-docker` | Gerenciar containers |
| GitLens | `eamodio.gitlens` | Histórico e blame no editor |
| Error Lens | `usernamehw.errorlens` | Erros inline no código |
| Database Client | `cweijan.vscode-database-client2` | Explorar o PostgreSQL local |
| DotENV | `mikestead.dotenv` | Syntax highlight em arquivos `.env` |

Configurações recomendadas em `.vscode/settings.json`:
- `go.lintTool: "golangci-lint"` — integra o linter ao editor
- `go.formatTool: "goimports"` — formata e organiza imports ao salvar

---

## Troubleshooting

### Porta 5433 já em uso

```bash
# Verifique qual processo está usando a porta
lsof -i :5433
```

Ou altere a porta em `deploy/docker/dev/docker-compose.yml`: `"5433:5432"` para `"5434:5432"`.

### Database connection refused

```bash
# Verifique se o container está rodando
docker ps

# Se não aparecer, suba o stack
make dev

# Verifique os logs
docker logs $(docker ps -q --filter "ancestor=postgres:15")
```

### JWT_SECRET não configurada

Certifique-se que `.env` existe e contém a variável:
```env
JWT_SECRET=your-secret-key-change-me
```

Em produção, use um valor gerado de forma segura e armazene via sistema de secrets.

### Containers não iniciam

```bash
# Verifique os logs
docker compose -f deploy/docker/dev/docker-compose.yml logs

# Limpe e tente novamente
make remove
make build-dev
```

---

## Integração com CI/CD

Exemplo para Bitbucket Pipelines:

```yaml
image: golang:1.26.1

pipelines:
  default:
    - step:
        name: Build & Test
        script:
          - go mod download
          - golangci-lint run
          - go test -v ./...
        services:
          - postgres

services:
  postgres:
    image: postgres:15
    variables:
      POSTGRES_DB: i9pay_test
      POSTGRES_PASSWORD: password
```

---

## Deploy em Produção

### Variáveis de Produção

**Kubernetes:**
```bash
kubectl create secret generic i9pay-secrets \
  --from-literal=DB_USER=prod_user \
  --from-literal=DB_PASSWORD=secure_password \
  --from-literal=JWT_SECRET=$(openssl rand -base64 32)
```

**ECS/Fargate:** use AWS Secrets Manager para armazenar `DB_PASSWORD` e `JWT_SECRET`.

### Migrations em Produção

Execute antes do deploy:

```bash
docker run --rm \
  -v "$(pwd)/deploy/migrations:/migrations" \
  migrate/migrate \
  -path /migrations \
  -database "postgres://user:pass@db:5432/i9pay" \
  up
```

---

## Referências

- [API.md](API.md) — documentação de endpoints
- [ARCHITECTURE.md](ARCHITECTURE.md) — estrutura e fluxo
- [DEVELOPMENT.md](DEVELOPMENT.md) — guia de contribuição
- [COMMANDS.md](COMMANDS.md) — referência de comandos
