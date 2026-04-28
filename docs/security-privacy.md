# Segurança e privacidade

Resumo das garantias implementadas e dos limites do repositório enquanto exemplo de desenvolvimento.

## Webhook (`POST /webhook`)

| Aspeto | Implementação |
|--------|----------------|
| Autenticação do emissor | HMAC-SHA256 do corpo **bruto** com `WEBHOOK_SECRET`; cabeçalho `X-Signature-256: sha256=<hex>`; comparação em tempo constante |
| Integridade | Qualquer alteração de byte no corpo invalida a assinatura |
| Abuso / DoS | Limite de leitura do corpo (512 KiB); resposta 413 se exceder |
| Erros | Assinatura inválida ou ausente: **401**; não se distingue “formato errado” de “segredo errado” no detalhe exposto |

O segredo do webhook é **independente** do `CPF_PEPPER` e do `JWT_SECRET`.

## API REST e WebSocket (cidadão)

| Aspeto | Implementação |
|--------|----------------|
| Autenticação | `Authorization: Bearer <JWT>`; algoritmo HS256; `JWT_SECRET` |
| Claims | `preferred_username` com CPF normalizado (11 dígitos); `exp` obrigatório; `iss` / `aud` opcionais via `JWT_ISS` / `JWT_AUD` |
| Mapeamento | Mesmo fingerprint que o webhook (`HMAC-SHA256` com `CPF_PEPPER` sobre o CPF) → `citizens.id` |
| Isolamento | Consultas SQL incluem `citizen_id` do contexto; recursos de outro cidadão tendem a **404** para não revelar existência |

### WebSocket

- O upgrade exige o mesmo Bearer que o REST (`internal/httpapi/ws.go`).
- Mensagens enviadas ao cliente **não** incluem CPF; incluem identificadores de notificação e `chamado_id`.
- **Origem (`Origin`)**: o `Upgrader` usa `CheckOrigin: func(...) bool { return true }`, o que aceita qualquer origem. Isto simplifica testes com ferramentas de linha de comando e Postman. Para aplicação browser em produção, deve restringir-se a origens conhecidas.

## Privacidade de dados pessoais

- O CPF do webhook **não** é persistido em coluna de texto; apenas o fingerprint (32 bytes) em `citizens.fingerprint`.
- Logs de aplicação não devem imprimir CPF; o código de produção deve manter essa disciplina em alterações futuras.
- DLQ (`webhook_dlq`) guarda o **corpo bruto** do webhook — que contém CPF no JSON do contrato atual. Acesso à tabela deve ser tratado como dados sensíveis e retenção limitada conforme política.

## Integrações HTTP saíntes

- TLS, timeouts e circuit breaker reduzem cascata de falhas; não substituem lista de permissões de rede.
- Respostas de erro a montante podem ser devolvidas em parte ao cliente (`502` com `detail` em chamados) — útil em desenvolvimento; em produção avaliar sanitização.

## Kubernetes e segredos

- `k8s/app-secret.example.yaml` contém valores de exemplo; não são segredos reais, mas **não** devem ser reutilizados em ambientes partilhados sem rotação.
- Preferir `kubectl create secret generic` ou operador de secrets fora do Git.

## Checklist rápido antes de produção

- [ ] Restringir `CheckOrigin` no WebSocket  
- [ ] Rotação e armazenamento seguro de `WEBHOOK_SECRET`, `JWT_SECRET`, `CPF_PEPPER`  
- [ ] TLS na borda (ingress) e entre serviços conforme política  
- [ ] Política de retenção e acesso à `webhook_dlq`  
- [ ] Redis e Postgres com autenticação e rede privada  
