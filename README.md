# desafio-backend

Serviço de notificações em tempo real para cidadãos acompanharem o estado dos chamados de manutenção urbana (aberto, em análise, em execução, concluído).

O sistema externo da prefeitura envia eventos (webhook com assinatura HMAC), a API expõe listagem e leitura de notificações ao cidadão autenticado (JWT) e liga o cliente por WebSocket para entrega imediata.

**Esta versão** inclui **`POST /webhook`** (HMAC, idempotência, fingerprint, persistência, **outbox** + **DLQ** em falha de persistência), **REST `/notifications`** (lista, detalhe `GET /notifications/:id`, `PATCH /notifications/read-all`, contagens) e **`GET /citizens/me`**, **`POST/DELETE /devices`** (tokens para entrega HTTP opcional), **`GET /chamados/:id/summary`** (proxy opcional a um sistema de chamados), **`GET /mapas/status`**, integração **JWT** (`preferred_username` = CPF) e **WebSocket `/ws`** com broadcast via **Redis Pub/Sub** (`notif:citizen:<uuid>`) e hub in-memory por processo. Marcar notificações como lidas **persiste em PostgreSQL**; chamadas HTTP para sistemas externos são **opcionais** (`CHAMADOS_API_BASE_URL`, `MAPAS_API_BASE_URL`, `PUSH_WEBHOOK_URL`).

- Webhook: [`docs/webhook.md`](docs/webhook.md)
- Notificações: [`docs/notifications.md`](docs/notifications.md)
- WebSocket: [`docs/websocket.md`](docs/websocket.md)

## Requisitos

- Go 1.24 ou superior
- Docker e Docker Compose
- [just](https://github.com/casey/just) (opcional, para atalhos de comando)
- [Docker](https://docs.docker.com/get-docker/) — para `just k6-webhook` / `just k6-notifications` (k6), `just test-all` (kubeconform + compose) e validação K8s offline
- [`kubectl`](https://kubernetes.io/docs/tasks/tools/) — para `just k8s-*` (renderização Kustomize e, opcionalmente, dry-run contra o cluster)
- [Grafana k6](https://grafana.com/docs/k6/latest/set-up/install-k6/) (opcional, só se quiseres `just k6-webhook-native` / `just k6-notifications-native` no PATH)

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
- Tracing: `OTEL_SERVICE_NAME` (default `desafio-backend`), `OTEL_TRACES_EXPORTER` (`stdout` ou `none` para desativar exportação)
- Integrações HTTP opcionais: `CHAMADOS_API_BASE_URL`, `MAPAS_API_BASE_URL`, `PUSH_WEBHOOK_URL`, `HTTP_CLIENT_TIMEOUT`, `MAPAS_PING_INTERVAL` (ping a `{MAPAS_API_BASE_URL}/health` quando mapas está configurado; circuit breaker nos clientes HTTP)
- `INTERNAL_UPSTREAM_STUBS` — `1` ou `true` regista rotas `/_upstream/...` no mesmo processo (respostas JSON embebidas; **sem** Bearer). Para forçar falhas e testar circuit breaker: `?fail=1` nos GET de stub ou cabeçalho `X-Simulate-Fail: 1` nos POST de stub

## Como subir

**Com Docker (recomendado):** sobe a **API**, **Postgres** e **Redis**. O serviço **`app` lê o ficheiro `.env`** (`env_file`). O `docker-compose.yml` **só** força **`DATABASE_URL`** e **`REDIS_ADDR`** para os serviços `postgres` e `redis` no contentor; **não** sobrescreve `CHAMADOS_API_BASE_URL`, `MAPAS_API_BASE_URL`, `PUSH_WEBHOOK_URL` nem `INTERNAL_UPSTREAM_STUBS` — isso vem **só** do teu `.env` (ex.: loopback `http://127.0.0.1:8080/_upstream` com stubs, ou URLs reais). O arranque do `app` aplica o schema com **[golang-migrate](https://github.com/golang-migrate/migrate)** a partir dos ficheiros em `migrations/` (incluídos no binário via `go:embed`).

```bash
just up
# ou: docker compose up --build
```

Se o `app` falhar com **`lookup postgres ... no such host`** (DNS interno do Docker), os serviços partilham a rede **`backend`** no `docker-compose.yml`. Recria tudo: `docker compose down && docker compose up --build` (ou `docker compose up --force-recreate`).

Para correr as migrações **sem** subir o servidor: `go run ./cmd/migrate -up` (ou `just migrate-up`). Bases vazias são preenchidas; para uma base com schema de um fluxo antigo (só `docker-entrypoint-initdb.d`) e sem tabela `schema_migrations`, usa `docker compose down -v` ou ajusta a versão com a CLI [`migrate force`](https://github.com/golang-migrate/migrate/blob/master/GETTING_STARTED.md#forcing-your-database-version).

- API: <http://localhost:8080>
- `GET /health` — liveness
- `GET /ready` — PostgreSQL e Redis
- `POST /webhook` — evento assinado ([`docs/webhook.md`](docs/webhook.md))
- `GET /notifications`, `PATCH /notifications/:id/read`, `GET /notifications/unread-count` — JWT Bearer ([`docs/notifications.md`](docs/notifications.md))
- `GET /ws` — WebSocket com JWT ([`docs/websocket.md`](docs/websocket.md))


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

**Validar manifests sem cluster (recomendado em CI):** `just k8s-validate` — corre `kubectl kustomize` e [kubeconform](https://github.com/yannh/kubeconform) (`ghcr.io/yannh/kubeconform:v0.6.7`) em Docker sobre o YAML renderizado; **não** exige permissões no API server. Se tiveres contexto kubectl e quiseres simular `kubectl apply` no cliente (pode falhar por RBAC), usa `just k8s-apply-dry-run-client`; validação no servidor: `just k8s-validate-server`.

## Comandos úteis (just)

| Comando | Descrição |
|---------|-----------|
| `just` | Lista as receitas |
| `just up` | Sobe a stack (compose) |
| `just build` | Compila o binário |
| `just test` | `go test ./...` (unitários) |
| `just test-nocache` | `go test -count=1 ./...` (sem cache de resultados) |
| `just test-all` | Verificação ampla **sem** Postgres/Redis: `go mod verify`, `docker compose config` (saída suprimida), `go vet`, `go build ./...`, testes unitários sem cache, `just k8s-validate` |
| `just test-full` | `just test-all` seguido de `just test-integration` (exige `DATABASE_URL` e `REDIS_ADDR`, ex.: com `just up`) |
| `just mod-verify` / `just vet` / `just build-check` / `just compose-validate` | Passos isolados usados por `test-all` |
| `just k8s-kustomize` | Só `kubectl kustomize k8s/` (falha se o YAML estiver inválido) |
| `just k8s-validate` | Kustomize + kubeconform em Docker (schemas K8s; offline) |
| `just k8s-apply-dry-run-client` | `kubectl apply -k k8s/ --dry-run=client` (pode contactar o API server) |
| `just k8s-validate-server` | `kubectl apply -k k8s/ --dry-run=server` (requer permissões no namespace) |
| `just migrate-up` | `go run ./cmd/migrate -up` (aplica migrações; usa `DATABASE_URL` do ambiente) |
| `just test-integration` | `go test -tags=integration ./...` (requer `DATABASE_URL`, `REDIS_ADDR`; as migrações correm no início se `DATABASE_URL` estiver definida) |
| `just k6-webhook` | Carga no webhook via **Docker** (`grafana/k6`) — exige `WEBHOOK_SECRET` no ambiente; `BASE_URL` opcional (padrão `http://host.docker.internal:8080` para alcançar a API no anfitrião) |
| `just k6-notifications` | Carga em `GET /notifications` via Docker — exige `K6_JWT`; `BASE_URL` opcional (mesmo padrão) |
| `just k6-api-extensions` | Carga nos novos endpoints (`citizens/me`, `read-all`, `chamados`, `mapas/status`, `devices`) — exige `K6_JWT`; `WEBHOOK_SECRET` opcional (seed no setup); `BASE_URL` como nos outros |
| `just k6-webhook-native` / `just k6-notifications-native` | Igual, mas com o binário `k6` no PATH (`BASE_URL` padrão `http://localhost:8080` nos scripts) |
| `just k6-api-extensions-native` | Variante nativa de `k6-api-extensions` |
## Testes de carga (k6)

Usa apenas em **ambiente local** ou staging; não apontar para produção sem acordo.

1. Sobe a stack (`just up`) com a API acessível na porta publicada (ex.: 8080 no anfitrião).
2. **Webhook:** no PowerShell, o mesmo segredo que no `.env` do servidor: `$env:WEBHOOK_SECRET = '…'; just k6-webhook`. O `just` usa **Docker** (`grafana/k6`); por padrão `BASE_URL` é `http://host.docker.internal:8080` para o contentor alcançar a API no Windows/macOS. Se a API estiver doutro host, define `$env:BASE_URL = 'http://…'`. Opcional: `K6_CPF` (11 dígitos; o script tem padrão `12345678901`).
3. **REST:** gera um JWT (ver [docs/notifications.md](docs/notifications.md)). `$env:K6_JWT = '<token>'; just k6-notifications`.
4. Se tiveres o binário **k6** instalado e quiseres falar com `http://localhost:8080` sem Docker, usa `just k6-webhook-native` / `just k6-notifications-native` com as mesmas variáveis.

Os scripts estão em [`k6/webhook-load.js`](k6/webhook-load.js), [`k6/notifications-read.js`](k6/notifications-read.js) e [`k6/api_extensions.js`](k6/api_extensions.js). Carga em **WebSocket** fica fora deste diferencial (extensões k6 ou testes de integração Go).

## Verificação completa (local / CI)

- **`just test`** — o mínimo exigido pelo desafio (`go test ./...`).
- **`just test-telemetry`** — testes de OpenTelemetry (`internal/telemetry`, span `otelgin` em `/health`).
- **`just test-all`** — inclui análise estática, compilação de todos os pacotes, testes unitários sem cache, validação do `docker-compose.yml` e manifests Kubernetes **sem** precisar de cluster acessível (kubeconform). Requer **Docker** e **kubectl** no PATH.
- **`just test-full`** — `just test-all` seguido de `just test-integration`. Sem `DATABASE_URL` (e `REDIS_ADDR` onde aplicável) os testes de integração **fazem skip** e o comando ainda termina com sucesso; para realmente exercitar a stack, exporta essas variáveis ou usa `just up` e o mesmo `DATABASE_URL` / `REDIS_ADDR` que o compose expõe no host.

## Fase de implementação

- **Feito:** webhook (com `webhook_dlq` em falha de persistência após HMAC válido), outbox transacional + worker + Redis Pub/Sub, entrega HTTP opcional por dispositivo (`PUSH_WEBHOOK_URL`), WebSocket `/ws`, REST com JWT (detalhe, read-all, citizens/me, devices, chamados summary e mapas status com clientes HTTP opcionais e circuit breaker), migrations, testes de integração opcionais, k6 (incl. `api_extensions`), manifests `k8s/`, tracing OpenTelemetry (Gin + `otelhttp` nos clientes externos)
- **Seguinte (exemplos):** exportador OTLP, contratos adicionais com fornecedores de push

## Estrutura (resumo)

- `cmd/server` — entrada do processo; `cmd/migrate` — CLI de migrações (opcional em dev)
- `internal/migrate` — aplicação das migrações embebidas (também no arranque do servidor)
- `internal/config`, `internal/db`, `internal/rdb` — configuração e ligações
- `internal/identity` — fingerprint do CPF (partilhado webhook + JWT)
- `internal/authjwt` — middleware JWT
- `internal/webhook`, `internal/repo`, `internal/httpapi` — webhook, SQL, rotas HTTP
- `internal/telemetry` — inicialização do TracerProvider (stdout; `OTEL_TRACES_EXPORTER=none` desativa)
- `internal/wsbus`, `internal/notify` — WebSocket local e fan-out Redis / outbox
- `migrations`, `docs`, `postman`, `Dockerfile`, `k8s`, `k6`

## Decisões de design

- SQL com `pgx`, sem ORM
- CPF nunca em texto no banco; JWT usa o mesmo fingerprint que o webhook
- `just test-integration` para Postgres + Redis reais
- Outbox na mesma transacção que `INSERT` da notificação; worker publica em Redis com retry e estado `dead`; `webhook_dlq` grava corpo bruto e assinatura quando a persistência falha após HMAC válido
- Clientes HTTP opcionais (`CHAMADOS_API_BASE_URL`, `MAPAS_API_BASE_URL`, `PUSH_WEBHOOK_URL`): ausentes = sem chamadas externas; marcar lidas e listagens dependem só de PostgreSQL
- Com `INTERNAL_UPSTREAM_STUBS=1`, o binário expõe `/_upstream/...` alinhado a esses clientes (fixtures em [`internal/upstream/fixtures/`](internal/upstream/fixtures/))
- k6: HMAC do corpo UTF-8 enviado no POST (alinhado a [`docs/webhook.md`](docs/webhook.md)); cenários com rampa de VUs e limiar de `http_req_failed`
- K8s offline: `just k8s-validate` com kubeconform evita depender de RBAC no `kubectl apply --dry-run=client`

## Stack

- Go, Gin, PostgreSQL, Redis, WebSocket (gorilla/websocket)
- `docker compose up` com zero dependências fora de Docker, para avaliação do repositório
