# Estratégia de testes

## Unitários (default)

```bash
just test
# ou: go test ./...
```

Correm **sem** Postgres ou Redis. Pacotes como `internal/webhook`, `internal/wsbus` ou clientes HTTP usam entradas controladas ou `httptest`.

- Rápidos, adequados a CI mínimo e pré-commit.
- Não provam corrida real com `pgx`, nem Pub/Sub Redis entre processos.

## Integração (`integration` build tag)

```bash
export DATABASE_URL='postgres://notif:notif@localhost:5432/notif?sslmode=disable'
export REDIS_ADDR='localhost:6379'
just test-integration
```

Ficheiros com `//go:build integration` (por exemplo `internal/httpapi/ws_integration_test.go`, `notifications_integration_test.go`, `internal/webhook/webhook_integration_test.go`).

- **Pesam mais**: levantam schema via golang-migrate, usam bases e Redis reais, exercitam fluxos completos (webhook → outbox → Redis → WS em alguns cenários).
- **Valor**: detetam regressões em SQL, tipos, e ordem de operações que mocks raramente captam.
- **Skip**: se `DATABASE_URL` (ou `REDIS_ADDR` onde necessário) não estiver definido, os testes fazem `t.Skip` em vez de falhar — útil em máquinas sem Docker.

## Verificação alargada sem serviços

```bash
just test-all
```

Inclui `go vet`, `go build ./...`, testes unitários sem cache, validação do `docker-compose.yml` e manifests Kubernetes com kubeconform (via Docker). Não inicia Postgres/Redis.

## Carga (k6)

Não substitui testes Go; mede latência e taxa de erro sob concorrência. Ver [k6-load-testing.md](k6-load-testing.md).

## O que pesa mais na revisão

| Tipo | Custo | Quando priorizar |
|------|--------|------------------|
| Integração real | Alto (I/O, tempo) | Fluxos críticos: webhook, migrações, WS + Redis |
| Unitário | Baixo | Regras puras, parsers, HMAC, hub em memória |
| k6 | Médio (ambiente, interpretação) | Regressões de performance ou de HMAC sob carga |

Regra prática: **mudanças em `internal/repo` ou `migrations/`** devem incluir ou atualizar testes de integração quando possível.
