FROM golang:1.26.1-alpine

WORKDIR /api

# Instala o Air
RUN go install github.com/air-verse/air@latest

COPY go.mod go.sum ./
RUN go mod download

CMD ["air", "-c", ".air.toml"]