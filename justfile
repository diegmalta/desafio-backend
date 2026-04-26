default:
    @just --list

# Sobe postgres, redis e a API
up:
    docker compose up --build

# Compila o servidor localmente
build:
    go build -o bin/server ./cmd/server

# Testes
test:
    go test ./...

# Formatação (opcional)
fmt:
    go fmt ./...
