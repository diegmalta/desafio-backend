# desafio-backend

Serviço de notificações em tempo real para cidadãos acompanharem o estado dos chamados de manutenção urbana (aberto, em análise, em execução, concluído).

O sistema externo da prefeitura envia eventos (webhook com assinatura HMAC), a API expõe listagem e leitura de notificações ao cidadão autenticado (JWT) e liga o cliente por WebSocket para entrega imediata.

**Esta versão** contém a stack mínima (Gin, PostgreSQL, Redis, Docker) e **rotas placeholder**; a lógica de negócio (HMAC, idempotência, CPF, JWT e WebSocket real) vem a seguir.

## Requisitos

- Go 1.24 ou superior
- Docker e Docker Compose
- [just](https://github.com/casey/just) (opcional, para atalhos de comando)

## Configuração

Copia [`.env.example`](.env.example) para `.env` e ajusta. Variáveis principais:

- `HTTP_ADDR` — endereço de escuta (ex. `:8080`)
- `DATABASE_URL` — PostgreSQL
- `REDIS_ADDR` — endereço do Redis (ex. `localhost:6379`)

## Como subir

**Com Docker (recomendado):** sobe a API, Postgres (com SQL inicial) e Redis.

```bash
just up
# ou: docker compose up --build
```

Na primeira carga, o Postgres aplica o SQL em `migrations/`. Se já tiveres volume antigo e mudares o schema, usa `docker compose down -v` (apaga dados locais) antes de subir de novo.

- API: <http://localhost:8080>
- `GET /health` — liveness
- `GET /ready` — PostgreSQL e Redis
- Stubs (501, Fase 2): `POST /webhook`, `GET /notifications`, `PATCH /notifications/:id/read`, `GET /notifications/unread-count`, `GET /ws`

## Comandos úteis (just)

| Comando   | Descrição              |
|----------|-------------------------|
| `just`   | Lista as receitas       |
| `just up`   | Sobe a stack (compose) |
| `just build` | Compila o binário      |
| `just test`  | `go test ./...`        |

## Fase de implementação

- **Agora:** ficheiro SQL de schema inicial, ligação a DB/Redis, rotas stub (501) para webhook, notificações e WebSocket
- **Seguinte:** HMAC, JWT (`preferred_username`), idempotência, privacidade do CPF, WebSocket a sério, testes de integração

## Estrutura (resumo)

- `cmd/server` — entrada do processo
- `internal/config`, `internal/db`, `internal/redis` — configuração e ligações
- `migrations` — esquema base (também montado no container Postgres)
- `Dockerfile` — imagem da aplicação

## Decisões de design (a documentar com mais detalhe na implementação completo)

- SQL com driver `pgx`, sem ORM, conforme enunciado
- CPF: não ser armazenado em texto; na implementação, usar derivada (HMAC/ hash com segredo) para associar cidadãos
- `just test` a invocar `go test ./...` no repositório

## Stack

- Go, Gin, PostgreSQL, Redis, WebSocket (a completar)
- `docker compose up` com zero dependências fora de Docker, para avaliação do repositório
