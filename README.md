# desafio-backend

Serviço de notificações (desafio técnico back-end Pleno — Go, Gin, PostgreSQL, Redis, WebSocket). A implementação da aplicação será adicionada em commits seguintes.

Ficheiros **não** versionados no Git deste repositório: [`.gitignore`](.gitignore) inclui **`.cursor/`** e qualquer `tasks/` acidental dentro do clone. Os planos e artefatos de agentes ficam em **`E:\Codigos\desafio-tasks\`** (mesmo workspace; ver [desafio-backend.code-workspace](desafio-backend.code-workspace)).

## Requisitos locais (Windows 11)

| Ferramenta   | Uso |
|-------------|-----|
| Go 1.24+    | Linguagem (instalado: 1.26.x via `winget install GoLang.Go`) |
| Docker      | `docker compose` (Docker Desktop) |
| just        | Task runner: `just test`, etc. (`winget install Casey.Just`) |

Confirmação: `go version`, `just --version`, `docker --version`.

## Estrutura

- [desafio-backend.code-workspace](desafio-backend.code-workspace) — abre **duas raízes** no Cursor/VS Code: este repositório e a pasta irmã **desafio-tasks** (planos, refinamentos, saídas dos agentes; **fora** do remoto)
- Código e config da app: repositório `desafio-backend` (Git)

## Cursor e MCP

Se tiveres `desafio-backend/.cursor/mcp.json` (local, não no remoto se ignorado), podes configurar o **team-memory** (`npx @arvoretech/memory-mcp`). Requer **Node.js** (para `npx`). A pasta local `./memories` pode ser ignorada no Git (ver [.gitignore](.gitignore)).

## Como clonar e desenvolver (após o código existir)

Cria a pasta irmã ao lado do clone, por exemplo: `..\desafio-tasks\` (já referenciada no ficheiro `.code-workspace`). Instruções de `just` e `docker compose` entram com a implementação.

## Git e GitHub

O histórico em `main` contém ficheiros do repositório de aplicação (p.ex. `desafio-backend.code-workspace`, `.gitignore`, `README`); **não** inclui `.cursor/`, `tasks/` dentro do repo, nem o conteúdo de `desafio-tasks/`.

**Publicar no GitHub como privado** (após `gh auth login`):

```text
gh repo create desafio-backend --private --source=. --remote=origin --push
```

Repo vazio no site: `git remote add origin <url>` e `git push -u origin main`.

Após reescrever histórico: `git push --force-with-lease`.
