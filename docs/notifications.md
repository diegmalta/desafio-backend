# API REST — notificações (`/notifications`)

Endpoints protegidos por **JWT** (HS256). O CPF do cidadão vem na claim **`preferred_username`** (string com **11 dígitos**, sem formatação), alinhado ao enunciado do desafio.

O fingerprint do cidadão é o mesmo que no webhook: `HMAC-SHA256(CPF_PEPPER, preferred_username)` — ver [`internal/identity/fingerprint.go`](../internal/identity/fingerprint.go).

## Autenticação

| Item | Valor |
|------|--------|
| Header | `Authorization: Bearer <jwt>` |
| Algoritmo | **HS256** |
| Segredo | Variável de ambiente **`JWT_SECRET`** (obrigatória no arranque) |
| Claims obrigatórias | `preferred_username` (11 dígitos), `exp` (expiração) |
| Opcionais | `iss` — validado se **`JWT_ISS`** estiver definido; `aud` — validado se **`JWT_AUD`** estiver definido (token deve incluir esse valor na lista `aud`) |

Sem `Authorization`, token inválido, expirado, ou CPF inválido na claim: **401** `{"error":"unauthorized"}`.

Se o JWT é válido mas **ainda não existe** linha em `citizens` com esse fingerprint, as listagens devolvem **lista vazia** e contagem **0**; `PATCH …/read` devolve **404** para qualquer id.

## Endpoints

### `GET /notifications`

Query:

| Parâmetro | Default | Notas |
|-----------|---------|--------|
| `limit` | 20 | Máximo **100** (valores acima são truncados). Mínimo **1**; inválido → **400**. |
| `offset` | 0 | Não negativo; inválido → **400**. |

Resposta **200**:

```json
{
  "items": [
    {
      "id": "uuid",
      "chamado_id": "…",
      "title": "…",
      "body": "…",
      "read_at": "2024-11-15T14:30:00Z",
      "created_at": "2024-11-15T14:00:00Z",
      "status_anterior": "…",
      "status_novo": "…",
      "event_type": "…",
      "source_timestamp": "2024-11-15T14:30:00Z"
    }
  ],
  "limit": 20,
  "offset": 0,
  "total": 42
}
```

`read_at` e `source_timestamp` podem ser omitidos se `null`.

Ordenação: `created_at DESC`, `id DESC`.

### `PATCH /notifications/:id/read`

Marca a notificação como lida se pertencer ao cidadão autenticado e ainda não tiver `read_at`.

- **204** sem corpo — sucesso.
- **404** `{"error":"not_found"}` — id inválido (UUID), notificação inexistente, já lida, ou **de outro cidadão** (mesma resposta para não expor existência).
- **400** `{"error":"invalid_id"}` — `:id` não é UUID.

### `GET /notifications/unread-count`

Resposta **200**: `{"count": N}` com número de notificações com `read_at` nulo.

## Gerar JWT de teste (HS256)

Com **Node.js** (substitui `SEU_JWT_SECRET` e o CPF):

```bash
node -e "const crypto=require('crypto');const secret='SEU_JWT_SECRET';const h=JSON.stringify({alg:'HS256',typ:'JWT'});const p=JSON.stringify({preferred_username:'12345678901',exp:Math.floor(Date.now()/1000)+3600});const b=(o)=>Buffer.from(o).toString('base64url');const sig=crypto.createHmac('sha256',secret).update(b(h)+'.'+b(p)).digest('base64url');console.log(b(h)+'.'+b(p)+'.'+sig);"
```

Coloca a string no Postman na variável `access_token` (sem o prefixo `Bearer `).

## Testes de integração

Requer **`DATABASE_URL`**, **`REDIS_ADDR`** (o router de integração regista `/ready`), e os segredos alinhados com o servidor (`JWT_SECRET`, `CPF_PEPPER`, `WEBHOOK_SECRET` — ver `internal/httpapi/notifications_integration_test.go`). Com `DATABASE_URL` definido, o `TestMain` aplica as migrações embebidas (golang-migrate) antes dos testes.

```bash
export DATABASE_URL='postgres://notif:notif@localhost:5432/notif?sslmode=disable'
export REDIS_ADDR='localhost:6379'
just test-integration
```

## Código

- Middleware: [`internal/authjwt/middleware.go`](../internal/authjwt/middleware.go)
- Handlers: [`internal/httpapi/notifications.go`](../internal/httpapi/notifications.go)
- SQL: [`internal/repo/notifications.go`](../internal/repo/notifications.go)
