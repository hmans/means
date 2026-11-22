---
type: feature
---

List and search can now sort results by metadata with `--sort` (MCP `sort`): use `id`, `title`, or `data.FIELD` keys, prefix `-` for descending, and combine several keys, e.g. `means list --sort data.status,-data.created_at`. Sorting applies before offset and limit, places missing values last, and keeps results deterministic. In full-text and semantic search, a sort replaces relevance order.
