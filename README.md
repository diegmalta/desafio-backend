# desafio-backend

Serviço de notificações (desafio técnico back-end Pleno — Go, Gin, PostgreSQL, Redis, WebSocket). Este repositório ainda contém a estrutura de projeto e configuração do Cursor; a implementação da aplicação será adicionada em commits seguintes.

## Requisitos locais (Windows 11)

| Ferramenta   | Uso |
|-------------|-----|
| Go 1.24+    | Linguagem (instalado: 1.26.x via `winget install GoLang.Go`) |
| Docker      | `docker compose` (Docker Desktop) |
| just        | Task runner: `just test`, etc. (`winget install Casey.Just`) |

Confirmação: `go version`, `just --version`, `docker --version`.

## Estrutura

- [desafio-backend.code-workspace](desafio-backend.code-workspace) — workspace do VS Code/Cursor
- [.cursor/](.cursor/) — regras, agentes, skills, `repos/app.yaml` e [mcp.json](.cursor/mcp.json)
- `tasks/` — planos e documentos de pipeline (refino, code, review, QA), quando usados

## Cursor e MCP

O ficheiro [.cursor/mcp.json](.cursor/mcp.json) regista o servidor **team-memory** (`npx @arvoretech/memory-mcp`). Requer **Node.js** (para `npx`) se quiseres este MCP ativo. Os embeddings usam a pasta local `./memories` (ver [.gitignore](.gitignore)).

## Como clonar e desenvolver (após o código existir)

Instruções detalhadas (`just`, `docker compose up`, variáveis de ambiente) serão documentadas quando a aplicação e o `Justfile` estiverem no repositório.

## Git e GitHub

O histórico inicial mostra: configuração Cursor, `.gitignore`, e este README.

Comando sugerido para repo privado (GitHub CLI autenticado: `gh auth login`):

```text
gh repo create desafio-backend --private --source=. --remote=origin --push
```

(ajusta o nome do repositório se necessário)
