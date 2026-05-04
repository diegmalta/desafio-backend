# Documentação do serviço

Ponto de entrada para contratos HTTP, fluxos internos, segurança e operação (incluindo k6 e Kubernetes).

## Índice

| Documento | Conteúdo |
|-----------|----------|
| [architecture.md](architecture.md) | Fluxo webhook → PostgreSQL → outbox → Redis → WebSocket; componentes e limites |
| [security-privacy.md](security-privacy.md) | HMAC, JWT, isolamento por cidadão, dados sensíveis, notificações para produção |
| [webhook.md](webhook.md) | `POST /webhook`: assinatura, payload, idempotência, códigos HTTP |
| [notifications.md](notifications.md) | JWT, paginação, marcar lidas, gerar token de teste |
| [rest-api.md](rest-api.md) | Mapa de todas as rotas autenticadas e integrações opcionais |
| [websocket.md](websocket.md) | `GET /ws`, formato das mensagens, autenticação |
| [testing-strategy.md](testing-strategy.md) | Testes unitários vs integração (`-tags=integration`), o que exige Docker |
| [k6-load-testing.md](k6-load-testing.md) | Scripts em `k6/`, variáveis de ambiente, Docker vs k6 nativo |
| [kubernetes.md](kubernetes.md) | Manifests em `k8s/`, probes, secrets, limitações do exemplo |

O [README principal](../README.md) resume requisitos, `.env`, `just` e decisões de alto nível.

**Comportamento funcional (webhook, REST, WebSocket)**  
O webhook valida HMAC sobre o corpo bruto, rejeita JSON inválido ou campos extra (`DisallowUnknownFields`), persiste com idempotência e encadeia outbox na mesma transação quando há insert. O worker drena o outbox, publica em `notif:citizen:<uuid>` e o subscritor Redis entrega ao hub WebSocket só para esse `citizen_id`. A REST filtra sempre por `citizen_id` derivado do JWT. Isto está alinhado a um modelo “notificação por cidadão” com entrega em tempo real.

**Segurança e privacidade**  
CPF não é guardado em claro: usa-se fingerprint com `CPF_PEPPER`. O payload WebSocket não inclui CPF. Listagens e `GET /notifications/:id` não vazam existência de notificações de outros cidadãos (404 genérico onde aplicável). Pontos a ter em conta em mente: `CheckOrigin` no WebSocket está permissivo (`true`) — adequado para testes; em browsers expostos à Internet deve restringir-se origens. Segredos vêm de ambiente; manifests K8s de exemplo usam credenciais de desenvolvimento.

**Código Go**  
Separação habitual: `cmd/`, `internal/httpapi`, `internal/repo`, `internal/webhook`, `internal/notify`, `internal/wsbus`, `internal/integrations`. Erros HTTP seguem padrões Gin (`gin.H{"error": ...}`) com pequenas variações semânticas (por exemplo 404 quando não há `citizen_id` em rotas que exigem recurso).

**Testes**  
Integração com Postgres e Redis reais pesa mais que mocks, mas cobre migrações, transações e Pub/Sub de ponta a ponta. Unitários cobrem lógica isolada (webhook, hub, integrações com `httptest`).

**Onboarding**  
Com `README`, `.env.example`, `docker compose` e este diretório `docs/`, deve ser possível subir o projeto e percorrer os fluxos; integrações externas opcionais (`CHAMADOS_API_BASE_URL`, etc.) continuam a exigir contexto de ambiente.
