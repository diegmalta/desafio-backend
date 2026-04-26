# Webhook `POST /webhook`

Implementação do recebimento de eventos do sistema externo, com assinatura HMAC, idempotência e armazenamento sem CPF em texto.

## Contrato HTTP

| Item | Detalhe |
|------|---------|
| Método e path | `POST /webhook` |
| Corpo | JSON UTF-8 (objeto no formato abaixo). Tamanho máximo aceite: **512 KiB**. |
| Assinatura | Cabeçalho `X-Signature-256` com valor `sha256=<hex>`, onde `<hex>` são 64 caracteres hexadecimais do **HMAC-SHA256** do corpo **bruto** (bytes exatos recebidos), com chave `WEBHOOK_SECRET`. |
| Comparação | Comparação em tempo constante (`crypto/subtle`). |

### Códigos de resposta

| HTTP | Situação |
|------|----------|
| 200 | Evento aceite e gravado (`{"ok":true}`) ou reenvio idempotente (`{"ok":true,"duplicate":true}`). |
| 400 | JSON inválido, campos obrigatórios em falta, CPF ou timestamp inválidos, ou campos desconhecidos no JSON (`DisallowUnknownFields`). |
| 401 | Cabeçalho de assinatura ausente ou assinatura incorreta. |
| 413 | Corpo acima do limite. |
| 500 | Erro ao persistir (detalhe não exposto ao cliente). |

## Payload JSON

Campos esperados (todos obrigatórios exceto onde indicado; `status_anterior` pode ser string vazia):

```json
{
  "chamado_id": "CH-2024-001234",
  "tipo": "status_change",
  "cpf": "12345678901",
  "status_anterior": "em_analise",
  "status_novo": "em_execucao",
  "titulo": "Buraco na Rua — Atualização",
  "descricao": "Equipe designada para reparo na Rua das Laranjeiras, 100",
  "timestamp": "2024-11-15T14:30:00Z"
}
```

- `cpf`: exatamente **11 dígitos** (sem pontuação). Não é persistido; ver secção Privacidade.
- `timestamp`: `RFC3339` ou `RFC3339Nano` parseável por Go.

## Idempotência

Reenvios com o mesmo significado de evento não criam segunda linha.

- **Chave canónica** (texto UTF-8): `v1|{chamado_id}|{status_novo}|{timestamp}|{tipo}` usando os **mesmos** valores de string após `json.Unmarshal` (incluindo o `timestamp` literal no JSON).
- **Valor guardado** (`notifications.idempotency_key`): `SHA256` desse texto, codificado em **hexadecimal minúsculo** (64 caracteres).
- Persistência: `INSERT ... ON CONFLICT (idempotency_key) DO NOTHING`. Se nada for inserido, a API responde **200** com `"duplicate":true`.

## Privacidade (CPF)

- O CPF **não** é escrito em colunas nem em logs de aplicação.
- O cidadão é identificado internamente por `citizens.fingerprint` (`BYTEA`, 32 bytes): **HMAC-SHA256** com chave `CPF_PEPPER` e mensagem os 11 dígitos do CPF (`CPF_PEPPER` é **independente** de `WEBHOOK_SECRET`).
- `WEBHOOK_SECRET` autentica o emissor do webhook; `CPF_PEPPER` deriva identificadores estáveis de cidadão.

## Persistência (PostgreSQL)

- `citizens`: `INSERT ... ON CONFLICT (fingerprint) DO NOTHING` seguido de `SELECT id` para obter `citizen_id`.
- `notifications`: `chamado_id`, `title` (`titulo`), `body` (`descricao`), metadados opcionais da migration `002` (`status_anterior`, `status_novo`, `event_type`, `source_timestamp`).

## Exemplo `curl` (gerar assinatura)

Com GNU coreutils e OpenSSL (Linux/macOS/WSL), com body em `body.json`:

```bash
BODY_FILE=body.json
SECRET='seu-WEBHOOK_SECRET'
SIG="sha256=$(openssl dgst -sha256 -hmac "$SECRET" -binary "$BODY_FILE" | xxd -p -c 256)"
curl -sS -X POST "http://localhost:8080/webhook" \
  -H "Content-Type: application/json" \
  -H "X-Signature-256: $SIG" \
  --data-binary "@$BODY_FILE"
```

Em PowerShell podes usar o mesmo ficheiro e calcular HMAC com .NET ou WSL; o importante é que o **ficheiro binário** enviado no `--data-binary` seja **idêntico** ao usado no HMAC.

## Variáveis de ambiente

| Variável | Uso |
|----------|-----|
| `WEBHOOK_SECRET` | Chave HMAC do body (obrigatória no arranque). |
| `CPF_PEPPER` | Chave HMAC para fingerprint do cidadão (obrigatória no arranque). |

Ver [`.env.example`](../.env.example). O `docker-compose.yml` define valores **apenas para desenvolvimento**; não reutilizar em produção.

## Testes

- Unitários: `go test ./...` (inclui `internal/webhook`).
- Integração (Postgres real, schema com `001` + `002` aplicados):

```bash
export DATABASE_URL='postgres://notif:notif@localhost:5432/notif?sslmode=disable'
just test-integration
```

Garante que o Postgres está acessível com esse URL (ex.: `docker compose up -d postgres` e, na **primeira** criação do volume, as migrations em `migrations/`). Se alterares o schema e já existir volume antigo: `docker compose down -v` antes de subir de novo.

## Limitações conhecidas

- As migrations em `migrations/` correm só na **inicialização** do volume do container Postgres (`docker-entrypoint-initdb.d`). Alterações ao SQL exigem novo volume ou aplicação manual em bases já existentes.
- Redis não participa nesta fatia do webhook (reservado para WebSocket/pub-sub mais tarde).

## Código relevante

- Handler e serviço: [`internal/webhook/`](../internal/webhook/)
- SQL transacional: [`internal/repo/webhook.go`](../internal/repo/webhook.go)
- Rotas: [`internal/httpapi/router.go`](../internal/httpapi/router.go)
