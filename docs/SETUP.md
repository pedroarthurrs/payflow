# Setup e Instalação

## Pré-requisitos

- **Go 1.26.1+** ([download](https://go.dev/dl/))
- **Docker & Docker Compose** ([install](https://docs.docker.com/get-docker/))
- **Make** (opcional, facilita comandos)
  - Linux/Mac: `brew install make`
  - Windows: Use WSL2 ou `choco install make`

## Quick Start (Docker)

```bash
# 1. Clone
git clone <repository-url>
cd i9pay

# 2. Configure os Git hooks do projeto
make setup-hooks

# 3. Suba o stack completo
make build-dev

# Pronto! API rodando em http://localhost:8080
```

---

## Setup Detalhado

### 1. Variáveis de Ambiente

O arquivo `.env` existe no repositório e é único para todos os ambientes. As variáveis **não utilizam sufixos** por ambiente — o controle de ambiente é feito apenas pela variável `GO_ENV`.

**Para executar em diferentes ambientes, defina:**
- `GO_ENV=development`
- `GO_ENV=staging`
- `GO_ENV=production`

Em produção, as variáveis devem vir via sistema de secrets do orquestrador (K8s, ECS, etc.) em vez do arquivo `.env`.

### 2. Rodando com Docker

**Importante:** A execução local deve ser sempre feita com Docker para padronizar o ambiente.

#### Build e Execução
```bash
make build-dev
```

Este comando:
- Contrói as imagens Docker
- Sobe os containers (API + PostgreSQL)
- Aplica as migrations automaticamente

A API estará em `http://localhost:8080` e o DB em `localhost:5433`.

#### Parar os containers
```bash
make down
```

#### Remover containers e volumes
```bash
make remove
```

Use quando quiser limpar completamente e recomeçar do zero.

---

## Verificar se está Rodando

### Health Check

O serviço expõe duas rotas de health:

- **Liveness** (`/live`): indica que o processo está rodando.
- **Readiness** (`/ready`): indica que o serviço está pronto para receber tráfego (banco de dados disponível).

```bash
# Liveness probe
curl http://localhost:8080/v1/live
# Resposta: {"status": "alive"}

# Readiness probe
curl http://localhost:8080/v1/ready
# Resposta: {"status": "ready"} ou 503 {"status": "unavailable"}
```

### Test Login
```bash
curl -X POST http://localhost:8080/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email": "test@example.com", "password": "test123"}'
```

---

## Troubleshooting

### Porta 5433 já em uso

```bash
# Verifique qual processo está usando
lsof -i :5433

# Ou, mude a porta em docker-compose.yml
# Altere: "5433:5432" para "5434:5432"
```

### Database connection refused

```bash
# Verifique se o container está rodando
docker ps

# Se não aparecer, rode:
make dev

# Verifique logs
docker logs $(docker ps -q --filter "ancestor=postgres:15")
```

### JWT_SECRET não configurada

Certifique-se que `.env` existe e tem a variável de secret:
```bash
JWT_SECRET=your-secret-key-change-me
```

Mude para um valor seguro. Em produção, as variáveis devem vir via sistema de secrets do orquestrador (K8s, ECS, etc).

### Containers não iniciam

```bash
# Verifique logs
docker compose -f docker/dev/docker-compose.yml logs

# Limpe e tente novamente
make remove
make build-dev
```

---

## Ambiente de Desenvolvimento

### Git Hooks

Configure os hooks do projeto para garantir que convenções de commit e verificações sejam aplicadas localmente:

```bash
make setup-hooks
```

Isso configura o Git para usar os hooks em `.githooks/`.

### Verificar o Projeto

Para rodar formatação, tidy, vet, lint e testes em sequência:

```bash
make check
```

### Adicione tools úteis

**air** - Hot reload:
```bash
go install github.com/air-verse/air@latest
air
```

**golangci-lint** - Linter (multichecker estático):
```bash
curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin
golangci-lint run
```

> O projeto inclui `.golangci.yml` com regras configuradas. Execute `make lint` para rodar.

**sqlc** - Geração de código a partir de SQL:
```bash
go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
```

**golang-migrate** - Migrations de banco:
```bash
go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
```

**mockery** - Geração de mocks para testes:
```bash
go install github.com/vektra/mockery/v2@latest
mockery --all --output internal/mocks
```

**delve** - Debugger:
```bash
go install github.com/go-delve/delve/cmd/dlv@latest
```

**gopls** - Language Server (configure em seu editor):
```bash
go install golang.org/x/tools/gopls@latest
```

### VSCode Integration

Instale as extensões recomendadas (ou aceite quando o VSCode sugerir via `.vscode/extensions.json`):

| Extensão | ID | Propósito |
|----------|----|-----------|
| Go | `golang.go` | Suporte completo à linguagem |
| REST Client | `humao.rest-client` | Testar requisições HTTP no editor |
| Docker | `ms-azuretools.vscode-docker` | Gerenciar containers |
| GitLens | `eamodio.gitlens` | Histórico e blame no editor |
| Error Lens | `usernamehw.errorlens` | Erros inline no código |
| Database Client | `cweijan.vscode-database-client2` | Explorar o PostgreSQL local |
| DotENV | `mikestead.dotenv` | Syntax highlight em arquivos `.env` |

Configurações recomendadas em `.vscode/settings.json` (já incluídas no repo):
- `go.lintTool: "golangci-lint"` — integra o linter ao editor
- `go.formatTool: "goimports"` — formata e organiza imports ao salvar


---

## Integração com CI/CD

### Bitbucket Pipelines

Arquivo `bitbucket-pipelines.yml`:
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
        artifacts:
          - build/**

services:
  postgres:
    image: postgres:15
    variables:
      POSTGRES_DB: i9pay_test
      POSTGRES_PASSWORD: password
```

---

## Deploy em Produção

### Build

```bash
make build-dev
```

O Makefile já contém os targets para build automatizado.

### Variáveis de Produção

Em produção, sempre use um sistema de gerenciamento de secrets:

**Kubernetes:**
```bash
kubectl create secret generic i9pay-secrets \
  --from-literal=DB_USER=prod_user \
  --from-literal=DB_PASSWORD=secure_password \
  --from-literal=JWT_SECRET=$(openssl rand -base64 32)
```

**ECS/Fargate:**
Use AWS Secrets Manager para armazenar `DB_PASSWORD` e `JWT_SECRET`.

**Docker Compose (Produção):**
Use `docker-compose.prod.yml` com variáveis carregadas via `.env.production` ou arquivo separado.

### Database Migrations em Produção

```bash
# Rode antes de deploy
docker run --rm \
  -v "$(pwd)/migrations:/migrations" \
  migrate/migrate \
  -path /migrations \
  -database "postgres://user:pass@db:5432/i9pay" \
  up
```

---

## Próximas Steps

Após setup bem-sucedido:

1. Leia [API.md](API.md) para documentação de endpoints
2. Leia [ARCHITECTURE.md](ARCHITECTURE.md) para entender a estrutura
3. Leia [DEVELOPMENT.md](DEVELOPMENT.md) para contribuir
4. Veja mais comandos em [COMMANDS.md](COMMANDS.md)
