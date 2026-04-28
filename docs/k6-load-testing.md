# Testes de carga com k6

Scripts em [`k6/`](../k6/) complementam os testes Go com tráfego HTTP concorrente. Usar só em **local** ou staging acordado — nunca contra produção sem controlo.

## Pré-requisitos

- API acessível (por exemplo `just up` com porta 8080 no anfitrião).
- Para cenários Docker: Docker instalado (`grafana/k6` é puxado pelo `just`).
- Variáveis de ambiente descritas abaixo.

## Comandos `just` (k6 dentro de Docker)

O `justfile` monta o repositório em `/src` dentro do contentor. O **valor por defeito** de `BASE_URL` é `http://host.docker.internal:8080` para o contentor alcançar a API no Windows/macOS.

| Receita | Variáveis obrigatórias | Ficheiro |
|---------|------------------------|----------|
| `just k6-webhook` | `WEBHOOK_SECRET` (igual ao servidor) | `k6/webhook-load.js` |
| `just k6-notifications` | `K6_JWT` (token completo, sem prefixo `Bearer`) | `k6/notifications-read.js` |
| `just k6-api-extensions` | `K6_JWT`; `WEBHOOK_SECRET` opcional (seed de chamado) | `k6/api_extensions.js` |

Opcionais comuns:

- `BASE_URL` — URL base da API (ex. `http://host.docker.internal:8080` ou IP do serviço).
- `K6_CPF` — 11 dígitos para o corpo do webhook nos scripts que geram eventos (default `12345678901` no webhook-load).

### PowerShell (exemplo)

```powershell
$env:WEBHOOK_SECRET = 'o-mesmo-do-env-do-servidor'
just k6-webhook

$env:K6_JWT = 'eyJ...'
just k6-notifications
```

## Comandos nativos (`k6` no PATH)

`just k6-webhook-native`, `just k6-notifications-native`, `just k6-api-extensions-native` executam os mesmos scripts com o binário local. O default de `BASE_URL` nos scripts é tipicamente `http://localhost:8080` — adequado quando k6 corre no mesmo host que a API.

## O que cada script faz

### `webhook-load.js`

- Gera corpos JSON únicos por iteração (`chamado_id` com VU/iteração/tempo).
- Calcula **HMAC-SHA256** do **string UTF-8** enviado no POST (alinhado ao servidor: corpo idêntico ao assinado).
- Cabeçalhos: `Content-Type: application/json; charset=utf-8`, `X-Signature-256: sha256=<hex>`.
- Cenário em rampa (ex.: 5 VUs) e limiar `http_req_failed` &lt; 2%.

### `notifications-read.js`

- `GET /notifications` com Bearer do `K6_JWT`.
- Útil para stress na listagem e no middleware JWT.

### `api_extensions.js`

- Exercita vários endpoints adicionais (`/citizens/me`, `read-all`, `chamados`, `mapas/status`, `devices`, etc.) conforme definido no script.
- Pode enviar webhook de seed se `WEBHOOK_SECRET` estiver definido.

## Falhas frequentes

| Sintoma | Causa provável |
|---------|----------------|
| `WEBHOOK_SECRET tem de estar definido` | Exportar o segredo antes de `just k6-webhook` |
| 401 no webhook | Segredo diferente entre k6 e servidor, ou corpo não coincide com o HMAC |
| Connection refused no contentor Docker | Usar `BASE_URL=http://host.docker.internal:8080` (ou IP da máquina), não `localhost` visto **de dentro** do contentor |
| Threshold `http_req_failed` falhou | API sobrecarregada, base lenta, ou erros 5xx — rever logs do servidor |

## WebSocket

Carga contínua em WebSocket não faz parte destes scripts; o projeto cobre WS com testes de integração Go (`-tags=integration`). Extensões k6 para WebSocket existem, mas exigem cenário separado.
