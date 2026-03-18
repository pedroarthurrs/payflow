# Comandos Disponíveis

## Make Commands

Todos os comandos estão no [Makefile](Makefile). Execute:

```bash
make help
```

Para ver lista completa com descrições.

---

## Docker

### `make dev`
Sobe containers de desenvolvimento com hot-reload (Air).

```bash
make dev
# Equivalente a: docker compose -f docker/dev/docker-compose.yml up
```

**Inicializa:**
- PostgreSQL em `localhost:5433`
- API em `http://localhost:8080`

### `make down`
Para os containers sem removê-los.

```bash
make down
```

### `make remove`
Para e remove containers, volumes e networks.

```bash
make remove
```

Use quando quiser limpar completamente e recomeçar.

### `make build-dev`
Force rebuild das imagens Docker.

```bash
make build-dev
```

Use quando mudar Dockerfile ou dependências.

---

## SQL & Code Generation

### `make sqlc`
Gera código Go a partir das queries SQL em `internal/database/queries.sql`.

```bash
make sqlc
```

**Output:**
- Atualiza `internal/database/dbgen/*.go`
- Type-safe queries prontas para uso

Execute sempre que:
- Escrever nova query SQL
- Mudar schema do banco

---

## Go Management

### `make tidy`
Limpa dependências e atualiza `go.mod` / `go.sum`.

```bash
make tidy
```

Execute regularmente para manter dependências limpas.

### `make test`
Roda todos os testes.

```bash
make test
# Equivalente a: go test -v ./...
```

---

## Database Migrations

### `make migrate-up`
Aplica todas as migrations pendentes.

```bash
make migrate-up
```

**Requer:**
- PostgreSQL rodando
- `DATABASE_URL` configurada

### `make migrate-down`
Reverte a última migration aplicada.

```bash
make migrate-down
```

Use com cuidado! Desfaz dados.

### `make migrate-create`
Cria nova migration vazia.

```bash
make migrate-create name=add_products_table
```

**Output:**
- `migrations/000004_add_products_table.up.sql`
- `migrations/000004_add_products_table.down.sql`

Edite os arquivos com seu SQL.

---

## Útil para Development

### Compilar sem rodar
```bash
go build ./cmd/api/main.go
```

### Run direto (sem docker)
```bash
go run ./cmd/api/main.go
```

### Format código
```bash
go fmt ./...
```

### Lint
```bash
golangci-lint run
```

### Vet
```bash
go vet ./...
```

### Dependency tree
```bash
go mod graph
```

### Upgrade dependências
```bash
go get -u github.com/gin-gonic/gin
go mod tidy
```

---

## Debugging

### Delve (debugger)
```bash
go install github.com/go-delve/delve/cmd/dlv@latest
dlv debug ./cmd/api/main.go
```

### Logs estruturados

A aplicação usa `log/slog` nativo do Go 1.21+. Em desenvolvimento os logs saem em texto;
em produção saem em JSON. O logger é inicializado em `cmd/api/main.go` via `SetupLogger(env)`.

---

## Full Workflow Example

```bash
# 1. Inicie o stack
make dev

# 2. Em outro terminal, crie migration
make migrate-create name=add_products_table

# 3. Edite migrations/000004_add_products_table.up.sql

# 4. Aplique migrations
make migrate-up

# 5. Escreva query em internal/database/queries.sql
# -- name: GetProduct :one
# SELECT * FROM products WHERE id = $1;

# 6. Gere código Go
make sqlc

# 7. Implemente service e handler

# 8. Teste
make test

# 9. Format
go fmt ./...

# 10. Se tudo ok, commit
git add .
git commit -m "feat: add products feature"

# 11. Limpe antes de parar
make remove
```

---

## CI/CD Integration

Para integração com Bitbucket Pipelines, use:

```bash
# Lint
golangci-lint run ./...

# Test with coverage
go test -v -coverprofile=coverage.out ./...
go tool cover -func=coverage.out

# Build
go build -o build/api ./cmd/api
```

---

## Troubleshooting Comandos

### Erro: "make: command not found"

**Linux:**
```bash
sudo apt install make
```

**macOS:**
```bash
brew install make
```

**Windows:**
Use [WSL2](https://docs.microsoft.com/en-us/windows/wsl/install) ou Git Bash.

### Erro: "migrate: command not found"

```bash
# Instale migrate
go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest

# Ou via package manager
brew install migrate  # macOS
choco install migrate # Windows
```

### Erro: "sqlc: command not found"

```bash
go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
```

### Docker command refused

```bash
# Adicione seu user ao grupo docker (Linux)
sudo usermod -aG docker $USER

# Reinicie
newgrp docker
```

---

## Performance Tips

- **go build** é rápido, use `make dev` com Air para hot-reload
- **go test** com `-short` para testes rápidos: `go test -short ./...`
- **go test -race** detecta race conditions: `go test -race ./...`

---

## Próximos Steps

Após dominar os comandos:
1. Leia [SETUP.md](SETUP.md) para setup detalhado
2. Leia [ARCHITECTURE.md](ARCHITECTURE.md) para entender fluxo
3. Leia [API.md](API.md) para rotas disponíveis
4. Leia [DEVELOPMENT.md](DEVELOPMENT.md) para contribuir
