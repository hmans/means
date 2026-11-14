---
type: fix
breaking: true
---

Collection and working-directory `.env` files now supply only API keys: `OPENAI_API_KEY`, `OPENROUTER_API_KEY`, and `MEANS_EMBEDDING_API_KEY`. Every other variable in a `.env` file is ignored, so a cloned or shared collection can no longer choose the program `means edit` runs or redirect embedding requests to another host. If you set `MEANS_EMBEDDING_BASE_URL`, `VISUAL`, `EDITOR`, or other variables in a `.env` file, move them to your shell or MCP host environment.
