---
type: fix
breaking: true
---

Means no longer reads a `.env` file inside a collection, and the working-directory `.env` now supplies only API keys: `OPENAI_API_KEY`, `OPENROUTER_API_KEY`, and `MEANS_EMBEDDING_API_KEY`. A cloned or shared collection can therefore no longer choose the program `means edit` runs, redirect embedding requests to another host, or supply credentials. The `.env` is skipped when the working directory or one of its parents contains `MEANS.md`, and the `--path` directory is not consulted. If you kept keys in a collection's `.env`, move them to a `.env` in a directory outside the collection that you run means from (for example a project root, with `--path`), or set them in your shell or MCP host environment. Set `MEANS_EMBEDDING_BASE_URL`, `VISUAL`, `EDITOR`, and other variables in the environment as well.
