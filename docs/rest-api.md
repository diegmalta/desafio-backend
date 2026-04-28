# Mapa da API REST (além de `/notifications`)

Todos os endpoints abaixo (exceto onde indicado) assumem **`Authorization: Bearer <JWT>`** igual ao descrito em [notifications.md](notifications.md).

## Notificações

| Método | Path | Notas |
|--------|------|--------|
| `GET` | `/notifications` | Paginação `limit` / `offset`; sem cidadão ainda criado → lista vazia |
| `GET` | `/notifications/:id` | Detalhe; 404 se UUID inválido, outro cidadão, ou sem auth válido em rotas que exigem recurso |
| `GET` | `/notifications/unread-count` | `{ "count": N }` |
| `PATCH` | `/notifications/:id/read` | 204; 404 se não aplicável |
| `PATCH` | `/notifications/read-all` | `{ "updated": n }`; pode acionar callbacks Chamados em segundo plano |

## Cidadão

| Método | Path | Notas |
|--------|------|--------|
| `GET` | `/citizens/me` | Dados do perfil ligados ao JWT |

## Dispositivos (push HTTP opcional)

| Método | Path | Notas |
|--------|------|--------|
| `POST` | `/devices` | Registo de token para `PUSH_WEBHOOK_URL` |
| `DELETE` | `/devices` | Remoção de token |

## Chamados (proxy opcional)

| Método | Path | Notas |
|--------|------|--------|
| `GET` | `/chamados/:chamado_id/summary` | Só se existir notificação para esse `chamado_id` **e** cidadão; depois proxy a `CHAMADOS_API_BASE_URL`. Sem cliente configurado → 503 com `reason`. Circuito aberto → `chamados_api_unavailable`. Erro de rede/5xx a montante → 502 `chamados_upstream` |

## Mapas (estado do cliente)

| Método | Path | Notas |
|--------|------|--------|
| `GET` | `/mapas/status` | **Sempre** 200 com JSON do circuito / último ping se o cliente Mapas existir; não é proxy de negócio |

## Webhook (sem JWT)

| Método | Path | Notas |
|--------|------|--------|
| `POST` | `/webhook` | HMAC; ver [webhook.md](webhook.md) |

## WebSocket (não REST)

| Método | Path | Notas |
|--------|------|--------|
| `GET` | `/ws` | Upgrade; Bearer no handshake; ver [websocket.md](websocket.md) |

## Saúde (sem JWT)

| Método | Path | Notas |
|--------|------|--------|
| `GET` | `/health` | Liveness |
| `GET` | `/ready` | DB + Redis |

## Stubs internos (opcional)

Com `INTERNAL_UPSTREAM_STUBS=1`, rotas sob `/_upstream/...` simulam fornecedores externos (sem Bearer). Usadas com `CHAMADOS_API_BASE_URL` / `MAPAS_API_BASE_URL` apontando para o próprio processo. Ver README e `internal/upstream/`.

## Códigos de erro comuns

| HTTP | `error` / corpo | Contexto típico |
|------|-----------------|-----------------|
| 401 | `unauthorized` | JWT em falta ou inválido |
| 404 | `not_found` | Recurso inexistente ou não visível para o cidadão (por desenho) |
| 400 | `invalid_id`, `invalid_limit`, … | Parâmetros |
| 503 | `reason` | Cliente externo não configurado ou circuit breaker |
| 502 | `chamados_upstream` | Falha ao contactar API de chamados |
