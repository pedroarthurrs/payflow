# --- ESTÁGIO 1: Build Frontend ---

FROM node:22-alpine AS web-builder

WORKDIR /web

COPY web/package.json web/pnpm-lock.yaml ./ 

RUN npm install -g pnpm && pnpm install

COPY web/ .

RUN pnpm build

# --- ESTÁGIO 2: Build Backend ---
FROM golang:1.26.1-alpine AS builder

RUN apk add --no-cache git

WORKDIR /api

# Cache de dependências
COPY go.mod go.sum ./
RUN go mod download

# Copia o backend
COPY . .

# Copia a pasta dist gerada no estágio 1 para dentro do pacote router (onde está o embed)
COPY --from=web-builder /web/dist ./internal/api/router/web-dist

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o main ./cmd/server/main.go

# --- ESTÁGIO 3: Execução ---
FROM alpine:latest

# Instala certificados CA
RUN apk --no-cache add ca-certificates tzdata

# Roda como user
RUN adduser -D -u 10001 appuser
USER appuser

WORKDIR /app

# Copia apenas o binário
COPY --from=builder /api/main .

ENTRYPOINT ["./main"]