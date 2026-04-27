# WebSocket — `/ws`

O app mantém uma ligação WebSocket para receber notificações **assim que** um evento do webhook é persistido e publicado (via **outbox** + **Redis Pub/Sub** + hub in-memory no processo).

## Autenticação

Header **`Authorization: Bearer <jwt>`** no pedido de upgrade (igual ao REST). Postman e clientes HTTP enviam este header no handshake WebSocket sem precisar de query string.

O JWT é o mesmo da API REST (**HS256**, `JWT_SECRET`), com claim **`preferred_username`** = CPF 11 dígitos e `exp` válido. Opcionais `JWT_ISS` / `JWT_AUD` aplicam-se tal como em [`notifications.md`](notifications.md).

- Token inválido, expirado, ou cidadão **sem** linha em `citizens` para esse fingerprint: **401** `{"error":"unauthorized"}` (sem upgrade WebSocket).
- Histórico de notificações continua a vir de `GET /notifications`; o WS **não** faz replay ao conectar.

## Formato da mensagem

Cada evento novo (após insert idempotente bem-sucedido) publica um JSON com `type` e o mesmo conjunto de campos que a REST usa para um item de notificação:

```json
{
  "type": "notification",
  "id": "uuid",
  "chamado_id": "CH-2024-001234",
  "title": "…",
  "body": "…",
  "created_at": "2024-11-15T14:30:00Z",
  "status_anterior": "em_analise",
  "status_novo": "em_execucao",
  "event_type": "status_change",
  "source_timestamp": "2024-11-15T14:30:00Z"
}
```

O payload **não** inclui CPF.

## Exemplo com wscat

Substitui o token JWT válido:

```bash
wscat -c "ws://localhost:8080/ws" -H "Authorization: Bearer SEU_JWT_AQUI"
```

## Código

- Handler: [`internal/httpapi/ws.go`](../internal/httpapi/ws.go)
- Hub e cliente: [`internal/wsbus/`](../internal/wsbus/)
- Outbox + worker + subscriber Redis: [`internal/notify/`](../internal/notify/)

## Testes de integração

Com `DATABASE_URL`, `REDIS_ADDR` e migrações aplicadas (`migrator.Up` nos testes), ver `internal/httpapi/ws_integration_test.go` (`go test -tags=integration ./...`).
