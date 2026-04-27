# desafio-backend

Serviço de notificações em tempo real para cidadãos acompanharem o estado dos chamados de manutenção urbana (aberto, em análise, em execução, concluído).

O sistema externo da prefeitura envia eventos (webhook com assinatura HMAC), a API expõe listagem e leitura de notificações ao cidadão autenticado (JWT) e liga o cliente por WebSocket para entrega imediata.

**Esta versão** inclui **`POST /webhook`** (HMAC, idempotência, fingerprint, persistência) e **REST `/notifications`** com **JWT** (`preferred_username` = CPF). WebSocket (`/ws`) ainda é stub (**501**).

- Webhook: [`docs/webhook.md`](docs/webhook.md)
- Notificações: [`docs/notifications.md`](docs/notifications.md)

## Requisitos

- Go 1.24 ou superior
- Docker e Docker Compose
- [just](https://github.com/casey/just) (opcional, para atalhos de comando)

## Configuração

Copia [`.env.example`](.env.example) para `.env` e ajusta. Variáveis principais:

- `HTTP_ADDR` — endereço de escuta (ex. `:8080`)
- `DATABASE_URL` — PostgreSQL
- `REDIS_ADDR` — endereço do Redis (ex. `localhost:6379`)
- `WEBHOOK_SECRET` — segredo HMAC do corpo bruto do webhook (**obrigatório** no arranque)
- `CPF_PEPPER` — segredo para derivar `citizens.fingerprint` a partir do CPF (**obrigatório**; distinto do webhook)
- `JWT_SECRET` — segredo HS256 para validar JWT da API de notificações (**obrigatório**)
- `JWT_ISS` / `JWT_AUD` — opcionais; se definidos, o token tem de conter `iss` / `aud` compatíveis

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
- Stub (501): `GET /ws`

Collection Postman: [`postman/desafio-backend.postman_collection.json`](postman/desafio-backend.postman_collection.json).

## Comandos úteis (just)

| Comando | Descrição |
|---------|-----------|
| `just` | Lista as receitas |
| `just up` | Sobe a stack (compose) |
| `just build` | Compila o binário |
| `just test` | `go test ./...` (unitários) |
| `just migrate-up` | `go run ./cmd/migrate -up` (aplica migrações; usa `DATABASE_URL` do ambiente) |
| `just test-integration` | `go test -tags=integration ./...` (requer `DATABASE_URL`, `REDIS_ADDR`; as migrações correm no início se `DATABASE_URL` estiver definida) |

## Fase de implementação

- **Feito:** webhook, REST de notificações com JWT e isolamento por cidadão, migrations, testes de integração opcionais
- **Seguinte:** WebSocket em `/ws` e broadcast após eventos

## Estrutura (resumo)

- `cmd/server` — entrada do processo; `cmd/migrate` — CLI de migrações (opcional em dev)
- `internal/migrate` — aplicação das migrações embebidas (também no arranque do servidor)
- `internal/config`, `internal/db`, `internal/rdb` — configuração e ligações
- `internal/identity` — fingerprint do CPF (partilhado webhook + JWT)
- `internal/authjwt` — middleware JWT
- `internal/webhook`, `internal/repo`, `internal/httpapi` — webhook, SQL, rotas HTTP
- `migrations`, `docs`, `postman`, `Dockerfile`

## Decisões de design

- SQL com `pgx`, sem ORM
- CPF nunca em texto no banco; JWT usa o mesmo fingerprint que o webhook
- `just test-integration` para Postgres + Redis reais

## Stack

- Go, Gin, PostgreSQL, Redis, WebSocket (a completar)
- `docker compose up` com zero dependências fora de Docker, para avaliação do repositório
