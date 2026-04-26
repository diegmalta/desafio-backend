# desafio-backend

Serviço de notificações em tempo real para cidadãos acompanharem o estado dos chamados de manutenção urbana (aberto, em análise, em execução, concluído).

O sistema externo da prefeitura envia eventos (webhook com assinatura HMAC), a API expõe listagem e leitura de notificações ao cidadão autenticado (JWT) e liga o cliente por WebSocket para entrega imediata.

**Esta versão** inclui **`POST /webhook` implementado** (HMAC, idempotência, fingerprint do CPF, persistência em PostgreSQL). REST de notificações, JWT e WebSocket continuam em desenvolvimento.

Documentação do webhook: [`docs/webhook.md`](docs/webhook.md).

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

## Como subir

**Com Docker (recomendado):** sobe a API, Postgres (com SQL inicial) e Redis.

```bash
just up
# ou: docker compose up --build
```

Na primeira carga, o Postgres aplica os ficheiros em `migrations/` por ordem (`001_init.sql`, `002_webhook_metadata.sql`, …). Se já tiveres volume antigo e mudares o schema, usa `docker compose down -v` (apaga dados locais) antes de subir de novo.

- API: <http://localhost:8080>
- `GET /health` — liveness
- `GET /ready` — PostgreSQL e Redis
- `POST /webhook` — evento assinado (ver [`docs/webhook.md`](docs/webhook.md))
- Stubs (501): `GET /notifications`, `PATCH /notifications/:id/read`, `GET /notifications/unread-count`, `GET /ws`

## Comandos úteis (just)

| Comando | Descrição |
|---------|-----------|
| `just` | Lista as receitas |
| `just up` | Sobe a stack (compose) |
| `just build` | Compila o binário |
| `just test` | `go test ./...` (unitários; sem Postgres obrigatório) |
| `just test-integration` | Testes com `-tags=integration` (requer `DATABASE_URL` e schema migrado) |

## Fase de implementação

- **Feito:** webhook HMAC, idempotência, privacidade do CPF (fingerprint), migrations e testes de integração opcionais
- **Seguinte:** JWT (`preferred_username`), REST de notificações, WebSocket e ligação em tempo real

## Estrutura (resumo)

- `cmd/server` — entrada do processo
- `internal/config`, `internal/db`, `internal/rdb` — configuração e ligações
- `internal/webhook`, `internal/repo` — webhook e SQL transacional
- `migrations` — esquema (montado no container Postgres na primeira inicialização)
- `docs` — contratos e operação (ex.: webhook)
- `Dockerfile` — imagem da aplicação

## Decisões de design

- SQL com driver `pgx`, sem ORM, conforme enunciado
- CPF nunca em texto no banco: `citizens.fingerprint` = HMAC-SHA256 com `CPF_PEPPER`
- `just test` mantém-se rápido; integração explícita em `just test-integration`

## Stack

- Go, Gin, PostgreSQL, Redis, WebSocket (a completar)
- `docker compose up` com zero dependências fora de Docker, para avaliação do repositório
