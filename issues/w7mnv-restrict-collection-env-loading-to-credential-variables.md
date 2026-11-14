---
data:
  priority: high
  status: done
  tags:
    - security
    - configuration
  type: bug
id: w7mnv
title: Restrict collection .env loading to credential variables
---
Found in the codebase review of 4c63f85 (confirmed by reproduction).

`LoadEnvironment` (`internal/userconfig/dotenv.go:24-52`, called from `internal/cli/root.go:92`) imports every variable from `<--path>/.env` and the working directory `.env`. A cloned or shared collection can therefore:

- set `VISUAL` or `EDITOR`, so `means edit` executes an arbitrary program (`internal/cli/edit.go:33`);
- set `MEANS_EMBEDDING_BASE_URL`, so the real API key and document title/body are sent to another host (`collection/embeddings.go:130`).

Acceptance criteria:
- `.env` files only supply credential variables: `OPENAI_API_KEY`, `OPENROUTER_API_KEY`, `MEANS_EMBEDDING_API_KEY`. Other variables in `.env` are ignored.
- Endpoint overrides and editor selection come only from the real process environment (or user config), never from `.env`.
- Existing environment variables still take precedence over `.env`.
- Tests assert which variables may and may not load.
- `docs/guide.md` and AGENTS.md describe the restricted scope.

## Outcome

`LoadEnvironment` now copies only `OPENAI_API_KEY`, `OPENROUTER_API_KEY`, and `MEANS_EMBEDDING_API_KEY` (the complete set of key variables means reads) from the collection and working-directory `.env` files. Every other variable is ignored, so `VISUAL`, `EDITOR`, and `MEANS_EMBEDDING_BASE_URL` come only from the process environment. The CLI and MCP startup share this function, so both paths are covered. Precedence is unchanged: process environment (even empty), collection `.env`, then working-directory `.env`. Malformed `.env` files still fail startup.

Updated `docs/guide.md`, AGENTS.md, MCP help, and this collection's workflow text, and added a breaking fix release note.

Validation: `mise test` passes. Tests assert that the three key variables load, that `VISUAL`, `EDITOR`, `MEANS_EMBEDDING_BASE_URL`, `XDG_CONFIG_HOME`, `NO_COLOR`, and an arbitrary variable do not, that process environment values win, and that MCP startup ignores an endpoint override in `.env`.
