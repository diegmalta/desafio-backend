# Windows: usar PowerShell (evita depender de `sh`, que não vem no PATH por defeito).
set windows-shell := ["powershell.exe", "-NoLogo", "-Command"]

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

# Migrações (DATABASE_URL; por defeito a do .env se exportado, ou a local do projecto)
migrate-up:
    go run ./cmd/migrate -up

# Integração (Postgres acessível; ex.: DATABASE_URL do compose na porta local)
test-integration:
    go test -tags=integration -count=1 ./...

# Formatação (opcional)
fmt:
    go fmt ./...
