# desafio-backend

Serviço de notificações em tempo real para cidadãos acompanharem o estado dos chamados de manutenção urbana (aberto, em análise, em execução, concluído).

O sistema externo da prefeitura envia eventos (webhook com assinatura HMAC), a API expõe listagem e leitura de notificações ao cidadão autenticado (JWT) e liga o cliente por WebSocket para entrega imediata.

**Esta versão** inclui **`POST /webhook`** (HMAC, idempotência, fingerprint, persistência, **outbox** + **DLQ** em falha de persistência), **REST `/notifications`** com **JWT** (`preferred_username` = CPF) e **WebSocket `/ws`** com broadcast via **Redis Pub/Sub** (`notif:citizen:<uuid>`) e hub in-memory por processo.

- Webhook: [`docs/webhook.md`](docs/webhook.md)
- Notificações: [`docs/notifications.md`](docs/notifications.md)
- WebSocket: [`docs/websocket.md`](docs/websocket.md)

## Requisitos

- Go 1.24 ou superior
- Docker e Docker Compose
- [just](https://github.com/casey/just) (opcional, para atalhos de comando)
- [Grafana k6](https://grafana.com/docs/k6/latest/set-up/install-k6/) (opcional, só para `just k6-webhook` / `just k6-notifications`)

## Configuração

Copia [`.env.example`](.env.example) para `.env` e ajusta. Variáveis principais:

- `HTTP_ADDR` — endereço de escuta (ex. `:8080`)
- `DATABASE_URL` — PostgreSQL
- `REDIS_ADDR` — endereço do Redis (ex. `localhost:6379`)
- `WEBHOOK_SECRET` — segredo HMAC do corpo bruto do webhook (**obrigatório** no arranque)
- `CPF_PEPPER` — segredo para derivar `citizens.fingerprint` a partir do CPF (**obrigatório**; distinto do webhook)
- `JWT_SECRET` — segredo HS256 para validar JWT da API de notificações (**obrigatório**)
- `JWT_ISS` / `JWT_AUD` — opcionais; se definidos, o token tem de conter `iss` / `aud` compatíveis
- Opcionais (defaults seguros): `OUTBOX_BATCH_SIZE`, `OUTBOX_POLL_INTERVAL`, `OUTBOX_MAX_ATTEMPTS`, `OUTBOX_BACKOFF_BASE`, `WS_WRITE_TIMEOUT`, `WS_PING_INTERVAL`, `WS_PONG_WAIT`, `WS_READ_LIMIT`

## Como subir

**Com Docker (recomendado):** sobe a API, Postgres e Redis. O serviço **`app` lê o ficheiro `.env`** na raiz do projeto (`env_file`). Tens de ter um `.env` (por exemplo copiado de `.env.example`). Os valores **`DATABASE_URL` e `REDIS_ADDR` dentro do container** são definidos pelo `docker-compose.yml` para apontar aos serviços `postgres` e `redis` (os do `.env` com `localhost` servem para `go run` na máquina anfitriã). O arranque do `app` aplica o schema com **[golang-migrate](https://github.com/golang-migrate/migrate)** a partir dos ficheiros em `migrations/` (incluídos no binário via `go:embed`).

```bash
just up
# ou: docker compose up --build
```

Para correr as migrações **sem** subir o servidor: `go run ./cmd/migrate -up` (ou `just migrate-up`). Bases vazias são preenchidas; para uma base com schema de um fluxo antigo (só `docker-entrypoint-initdb.d`) e sem tabela `schema_migrations`, usa `docker compose down -v` ou ajusta a versão com a CLI [`migrate force`](https://github.com/golang-migrate/migrate/blob/master/GETTING_STARTED.md#forcing-your-database-version).

- API: <http://localhost:8080>
- `GET /health` — liveness
- `GET /ready` — PostgreSQL e Redis
- `POST /webhook` — evento assinado ([`docs/webhook.md`](docs/webhook.md))
- `GET /notifications`, `PATCH /notifications/:id/read`, `GET /notifications/unread-count` — JWT Bearer ([`docs/notifications.md`](docs/notifications.md))
- `GET /ws` — WebSocket com JWT ([`docs/websocket.md`](docs/websocket.md))

Collection Postman: [`postman/desafio-backend.postman_collection.json`](postman/desafio-backend.postman_collection.json).

## Kubernetes (opcional)

Manifests em [`k8s/`](k8s/) sobem **namespace** `desafio-notif`, **Postgres**, **Redis** e a **API** com as mesmas credenciais de base que o `docker-compose.yml` (`notif` / `notif`). O Secret de aplicação usa o exemplo [`k8s/app-secret.example.yaml`](k8s/app-secret.example.yaml) (valores só para desenvolvimento; em produção usa `kubectl create secret` ou um gestor de segredos).

**Pré-requisitos:** cluster local (por exemplo [kind](https://kind.sigs.k8s.io/) ou minikube), `kubectl`, imagem da API construída a partir do [`Dockerfile`](Dockerfile):

```bash
docker build -t desafio-backend:latest .
```

Com **kind**, carrega a imagem para o nó do cluster (senão o kubelet não encontra `imagePullPolicy: IfNotPresent`):

```bash
kind load docker-image desafio-backend:latest
```

Aplica tudo (Kustomize inclui o Secret de exemplo; substitui os valores antes de qualquer ambiente partilhado):

```bash
kubectl apply -k k8s/
```

Expõe a API localmente:

```bash
kubectl port-forward -n desafio-notif svc/desafio-backend 8080:8080
```

Abre <http://localhost:8080/health>. O `Deployment` da API usa **probes** em `GET /health` (liveness) e `GET /ready` (readiness e startup), init containers à espera de Postgres e Redis, e `DATABASE_URL` / `REDIS_ADDR` internos ao cluster.

**Produção:** não versionar passwords ou segredos reais; preferir Secrets geridos fora do Git (External Secrets, Sealed Secrets, etc.). O Postgres no manifest usa **emptyDir** (dados perdem-se ao remover o pod); para dados persistentes, substituir por PVC ou StatefulSet e `storageClassName` adequado ao cluster.

## Comandos úteis (just)

| Comando | Descrição |
|---------|-----------|
| `just` | Lista as receitas |
| `just up` | Sobe a stack (compose) |
| `just build` | Compila o binário |
| `just test` | `go test ./...` (unitários) |
| `just migrate-up` | `go run ./cmd/migrate -up` (aplica migrações; usa `DATABASE_URL` do ambiente) |
| `just test-integration` | `go test -tags=integration ./...` (requer `DATABASE_URL`, `REDIS_ADDR`; as migrações correm no início se `DATABASE_URL` estiver definida) |
| `just k6-webhook` | `k6 run ./k6/webhook-load.js` — exige `WEBHOOK_SECRET` no ambiente (igual ao servidor) |
| `just k6-notifications` | `k6 run ./k6/notifications-read.js` — exige `K6_JWT` (token HS256; ver abaixo) |

## Testes de carga (k6)

Usa apenas em **ambiente local** ou staging; não apontar para produção sem acordo.

1. Instala o [k6](https://grafana.com/docs/k6/latest/set-up/install-k6/) e sobe a stack (`just up`).
2. **Webhook:** exporta o segredo usado pelo API (o mesmo `WEBHOOK_SECRET` do `.env`). No PowerShell: `$env:WEBHOOK_SECRET = '…'; just k6-webhook`. Opcional: `BASE_URL` (default `http://localhost:8080`), `K6_CPF` (11 dígitos, default `12345678901`).
3. **REST:** gera um JWT com `preferred_username` = CPF de 11 dígitos e `exp` no futuro, assinado com `JWT_SECRET` (e `iss` / `aud` se o servidor tiver `JWT_ISS` / `JWT_AUD`). Exemplo com Node: [docs/notifications.md](docs/notifications.md). Depois: `$env:K6_JWT = '<token>'; just k6-notifications`.

Os scripts estão em [`k6/webhook-load.js`](k6/webhook-load.js) e [`k6/notifications-read.js`](k6/notifications-read.js). Carga em **WebSocket** fica fora deste diferencial (extensões k6 ou testes de integração Go).

## Fase de implementação

- **Feito:** webhook (com `webhook_dlq` em falha de persistência após HMAC válido), outbox transacional + worker + Redis Pub/Sub, WebSocket `/ws`, REST com JWT, migrations (golang-migrate), testes de integração opcionais, testes de carga k6, manifests Kubernetes em `k8s/`
- **Seguinte (exemplos):** circuit breaker, OpenTelemetry

## Estrutura (resumo)

- `cmd/server` — entrada do processo; `cmd/migrate` — CLI de migrações (opcional em dev)
- `internal/migrate` — aplicação das migrações embebidas (também no arranque do servidor)
- `internal/config`, `internal/db`, `internal/rdb` — configuração e ligações
- `internal/identity` — fingerprint do CPF (partilhado webhook + JWT)
- `internal/authjwt` — middleware JWT
- `internal/webhook`, `internal/repo`, `internal/httpapi` — webhook, SQL, rotas HTTP
- `internal/wsbus`, `internal/notify` — WebSocket local e fan-out Redis / outbox
- `migrations`, `docs`, `postman`, `Dockerfile`, `k8s`, `k6`

## Decisões de design

- SQL com `pgx`, sem ORM
- CPF nunca em texto no banco; JWT usa o mesmo fingerprint que o webhook
- `just test-integration` para Postgres + Redis reais
- Outbox na mesma transacção que `INSERT` da notificação; worker publica em Redis com retry e estado `dead`; `webhook_dlq` grava corpo bruto e assinatura quando a persistência falha após HMAC válido
- k6: HMAC do corpo UTF-8 enviado no POST (alinhado a [`docs/webhook.md`](docs/webhook.md)); cenários com rampa de VUs e limiar de `http_req_failed`

## Stack

- Go, Gin, PostgreSQL, Redis, WebSocket (gorilla/websocket)
- `docker compose up` com zero dependências fora de Docker, para avaliação do repositório
