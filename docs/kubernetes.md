# Kubernetes (manifests de exemplo)

O diretório [`k8s/`](../k8s/) contém um **exemplo** para correr Postgres, Redis e a API no namespace `desafio-notif`. Serve para validar probes, init containers e Kustomize em CI; **não** é um blueprint completo de produção.

## Estrutura

| Ficheiro | Conteúdo |
|----------|----------|
| `namespace.yaml` | Namespace `desafio-notif` |
| `kustomization.yaml` | Agrega recursos e o Secret de exemplo |
| `postgres.yaml` | Deployment + Service Postgres 16 (dados em `emptyDir` no exemplo) |
| `redis.yaml` | Deployment + Service Redis 7 |
| `app.yaml` | Deployment da API: probes, recursos, `env` de ligação |
| `app-service.yaml` | Service `ClusterIP` na porta 8080 |
| `app-secret.example.yaml` | `Secret` com `WEBHOOK_SECRET`, `CPF_PEPPER`, `JWT_SECRET`, opcionais OTEL e URLs — **valores de laboratório** |

## Fluxo típico (cluster local)

1. Construir a imagem a partir do [`Dockerfile`](../Dockerfile):

   ```bash
   docker build -t desafio-backend:latest .
   ```

2. Com **kind**, carregar a imagem para os nós (evita `ImagePullBackOff` com `imagePullPolicy: IfNotPresent`):

   ```bash
   kind load docker-image desafio-backend:latest
   ```

3. Aplicar Kustomize:

   ```bash
   kubectl apply -k k8s/
   ```

4. Port-forward para testar no portátil:

   ```bash
   kubectl port-forward -n desafio-notif svc/desafio-backend 8080:8080
   ```

5. `curl http://localhost:8080/health` e `http://localhost:8080/ready`.

## Probes (Deployment da API)

| Probe | Path | Função |
|-------|------|--------|
| `startupProbe` | `/ready` | Arranque lento: até 30 × 5s aguardando DB/Redis |
| `readinessProbe` | `/ready` | Retira tráfego se DB ou Redis falharem |
| `livenessProbe` | `/health` | Reinicia o contentor se o processo HTTP morrer |

`/ready` verifica ping a Postgres e Redis com timeout curto (ver `internal/httpapi/router.go` e `cmd/server`).

## Init containers

- `wait-for-postgres`: `pg_isready` contra o serviço `postgres`.
- `wait-for-redis`: `redis-cli ping` contra o serviço `redis`.

Reduzem corridas onde a API arranca antes dos dados estarem aceitadores de ligação.

## Secrets e configuração

- Credenciais de aplicação vêm de `envFrom.secretRef: app-secrets` (gerado a partir do exemplo no `kustomization.yaml`).
- `DATABASE_URL` e `REDIS_ADDR` no Deployment apontam para os serviços internos (`postgres`, `redis`).
- **Rotação e gestão**: em produção, substituir o Secret versionado por gestor externo (External Secrets, Vault, etc.).

## Limitações do exemplo

- Postgres usa **`emptyDir`**: dados perdem-se ao apagar o pod; sem backup, sem HA.
- Uma réplica da API: escalar réplicas implica vários consumidores Redis (ver [architecture.md](architecture.md)).
- Sem Ingress/TLS no repositório: expor só em laboratório ou acrescentar camada à parte.

## Validação sem cluster (CI)

```bash
just k8s-validate
```

Executa `kubectl kustomize k8s/` e pipe para **kubeconform** em Docker, validando o YAML contra schemas Kubernetes — não precisa de API server com RBAC.

## Dry-run

- Cliente: `just k8s-apply-dry-run-client` (pode contactar o API server para merge).
- Servidor: `just k8s-validate-server` (requer permissões no namespace).
