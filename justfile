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

# OpenTelemetry (pacote telemetry + span Gin em /health)
test-telemetry:
    go test -count=1 ./internal/telemetry/... ./internal/httpapi -run "TestOTel|TestInitTracesExporterNone|TestInitEmptyServiceName"

# Testes sem cache de resultados (útil para CI / verificação completa)
test-nocache:
    go test -count=1 ./...

# Sumário de dependências e checksums do módulo
mod-verify:
    go mod verify

# Valida sintaxe do docker-compose.yml (requer Docker; saída descartada para não imprimir segredos do .env)
compose-validate:
    docker compose config | Out-Null

# Análise estática Go
vet:
    go vet ./...

# Compila todos os pacotes e comandos (sem gravar binário obrigatório em cmd/)
build-check:
    go build ./...

# Só renderiza Kustomize (rápido; não exige cluster)
k8s-kustomize:
    kubectl kustomize k8s/ | Out-Null

# Manifests Kubernetes sem cluster: Kustomize + kubeconform (schemas K8s; requer Docker para a imagem)
k8s-validate: k8s-kustomize
    kubectl kustomize k8s/ | docker run --rm -i ghcr.io/yannh/kubeconform:v0.6.7 -summary -strict

# Simula apply no cliente (pode contactar o API server para merge; útil com permissões no namespace)
k8s-apply-dry-run-client:
    kubectl apply -k k8s/ --dry-run=client

# Valida manifests no servidor (requer permissões de apply/dry-run no namespace)
k8s-validate-server:
    kubectl apply -k k8s/ --dry-run=server

# Tudo o que corre sem Postgres/Redis: módulo, compose, vet, build, testes unitários, K8s (kustomize + kubeconform)
test-all: mod-verify compose-validate vet build-check test-nocache k8s-validate
    @Write-Host 'test-all: OK (mod, compose, vet, build, testes, k8s kubeconform)'

# Igual a test-all mais integração com Postgres e Redis reais (DATABASE_URL, REDIS_ADDR)
test-full: test-all test-integration
    @Write-Host 'test-full: OK (inclui integracao)'

# Migrações (DATABASE_URL; por defeito a do .env se exportado, ou a local do projecto)
migrate-up:
    go run ./cmd/migrate -up

# Integração (Postgres acessível; ex.: DATABASE_URL do compose na porta local)
test-integration:
    go test -tags=integration -count=1 ./...

# Carga (k6) via Docker
# Define WEBHOOK_SECRET no ambiente (igual ao .env do servidor). Opcional: K6_CPF, BASE_URL (padrão aponta à API na máquina anfitriã).
k6-webhook:
    docker run --rm -v "{{justfile_directory()}}:/src" -w /src -e WEBHOOK_SECRET -e "BASE_URL={{env_var_or_default('BASE_URL', 'http://host.docker.internal:8080')}}" -e K6_CPF grafana/k6:latest run ./k6/webhook-load.js

# Carga (k6) via Docker. Define K6_JWT (token completo, sem prefixo Bearer). Opcional: BASE_URL.
k6-notifications:
    docker run --rm -v "{{justfile_directory()}}:/src" -w /src -e K6_JWT -e "BASE_URL={{env_var_or_default('BASE_URL', 'http://host.docker.internal:8080')}}" grafana/k6:latest run ./k6/notifications-read.js

# Mesmos scripts com binário k6 nativo (PATH). BASE_URL padrão no script: http://localhost:8080
k6-webhook-native:
    k6 run ./k6/webhook-load.js

k6-notifications-native:
    k6 run ./k6/notifications-read.js

# k6: novos endpoints (K6_JWT; WEBHOOK_SECRET opcional para seed CH-k6-ext). BASE_URL padrão host.docker.internal:8080
k6-api-extensions:
    docker run --rm -v "{{justfile_directory()}}:/src" -w /src -e K6_JWT -e WEBHOOK_SECRET -e K6_CPF -e "BASE_URL={{env_var_or_default('BASE_URL', 'http://host.docker.internal:8080')}}" grafana/k6:latest run ./k6/api_extensions.js

k6-api-extensions-native:
    k6 run ./k6/api_extensions.js

# Build da imagem do mock HTTP (chamados, mapas, push)
integrations-mock-build:
    docker build -f Dockerfile.integrations-mock -t desafio-integrations-mock "{{justfile_directory()}}"

fmt:
    go fmt ./...
