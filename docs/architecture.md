# Arquitetura

Visão em camadas do serviço de notificações e do caminho de um evento desde o sistema externo até ao cliente.

## Visão geral

```text
[Sistema externo] --HMAC POST--> [/webhook]
                                     |
                                     v
                            PostgreSQL (transação)
                            - citizens (fingerprint)
                            - notifications (idempotency_key)
                            - event_outbox (payload JSON)
                                     |
                                     v
                            [Worker] (polling outbox)
                                     |
                                     v
                            Redis PUBLISH notif:citizen:<uuid>
                                     |
                                     v
                            [Subscriber] -> Hub in-memory -> WebSocket /ws
```

Paralelamente, o mesmo worker pode enviar **push HTTP** opcional (`PUSH_WEBHOOK_URL`) por token de dispositivo.

## Componentes

| Área | Pacote / local | Função |
|------|------------------|--------|
| Entrada HTTP | `internal/httpapi` | Gin: rotas, middleware JWT, WebSocket upgrade |
| Webhook | `internal/webhook` | Leitura limitada do corpo, verificação HMAC, validação JSON, orquestração da persistência |
| Identidade | `internal/identity` | Normalização de CPF e HMAC para fingerprint |
| Persistência | `internal/repo` | SQL com `pgx` (sem ORM): notificações, outbox, DLQ, dispositivos |
| Autenticação API | `internal/authjwt` | JWT HS256, mapeamento `preferred_username` → `citizen_id` |
| Entrega em tempo real | `internal/notify` | Worker de outbox, publicação Redis, subscritor, nome do canal |
| WebSocket | `internal/wsbus` | Hub por processo, um ou mais clientes por `citizen_id` |
| Integrações | `internal/integrations` | Chamados, Mapas, push HTTP; circuit breaker (`gobreaker`) |
| Stubs locais | `internal/upstream` | Rotas `/_upstream/...` quando `INTERNAL_UPSTREAM_STUBS` está ativo |
| Configuração | `internal/config` | Variáveis de ambiente e defaults |
| Migrações | `migrations/`, `internal/migrate` | `golang-migrate`; também no arranque do servidor |

## Transação do webhook

Numa única transação típica (quando o evento é novo):

1. Garantir linha em `citizens` para o fingerprint do CPF do payload.
2. `INSERT` em `notifications` com `ON CONFLICT (idempotency_key) DO NOTHING`.
3. Se houve insert, `INSERT` em `event_outbox` com o JSON que o cliente WebSocket deve receber.

Se a persistência falhar **depois** de HMAC válido, o corpo pode ser gravado em `webhook_dlq` para análise (sem expor detalhe ao emissor do webhook).

## Redis e canais

- Prefixo fixo: `notif:citizen:` + UUID em texto canónico.
- Apenas o worker publica nesses canais após marcar o outbox como enviado (após `PUBLISH` bem-sucedido).
- O subscritor usa `PSUBSCRIBE` no padrão `notif:citizen:*`, extrai o UUID do nome do canal e chama `Hub.Dispatch(citizenID, payload)`.
- Não há autenticação Redis no exemplo local; em produção usar ACL ou rede isolada.

## WebSocket e escala horizontal

O hub é **em memória no processo**. Com **uma réplica**, todos os clientes ligados a esse pod recebem as mensagens daquele processo. Com **várias réplicas**, cada instância subscreve Redis; qualquer processo que tenha clientes para um dado `citizen_id` recebe a mensagem — desde que o subscritor esteja ativo nesse processo. O Redis funciona como barramento entre instâncias; não substitui persistência (histórico continua em `GET /notifications`).

## Integrações opcionais

- **Chamados:** proxy de summary após verificar que o cidadão tem notificação para aquele `chamado_id` (evita enumeração arbitrária).
- **Mapas:** ping periódico a `{MAPAS_API_BASE_URL}/health` com circuit breaker; `GET /mapas/status` apenas expõe estado (HTTP 200 com corpo JSON).
- **Push:** `POST` para URL externa por token, após publicação Redis bem-sucedida no worker.

## Onde ler código

- Arranque e wiring: `cmd/server/main.go`
- Registo de rotas: `internal/httpapi/router.go`
- Contrato de migrações: ficheiros numerados em `migrations/`
