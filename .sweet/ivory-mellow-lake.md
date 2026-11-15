---
type: fix
breaking: true
---

Means no longer loads `.env` files. Previously a collection's `.env` could set any environment variable, so a cloned or shared collection could choose the program `means edit` runs or redirect embedding requests, along with your API key, to another host. If you kept `OPENAI_API_KEY`, `OPENROUTER_API_KEY`, `MEANS_EMBEDDING_API_KEY`, `MEANS_EMBEDDING_BASE_URL`, or other variables in a `.env` file, set them in your shell or MCP host environment instead, or load the file with a tool such as mise or direnv.
