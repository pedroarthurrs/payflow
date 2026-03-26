# --- ESTÁGIO 1: Compilação ---
FROM golang:1.26.1-alpine AS builder

RUN apk add --no-cache git

WORKDIR /api

# Cache de dependências
COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o main ./cmd/server/main.go

# --- ESTÁGIO 2: Execução ---
FROM alpine:latest

# Instala certificados CA
RUN apk --no-cache add ca-certificates

WORKDIR /usr/bin/

# Copia apenas o binário
COPY --from=builder /api/main .

CMD ["./main"]