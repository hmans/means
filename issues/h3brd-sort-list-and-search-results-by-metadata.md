---
data:
  priority: normal
  status: todo
  tags:
    - cli
    - mcp
    - search
  type: feature
id: h3brd
title: Sort list and search results by metadata
---
`list` returns documents in filename order, which with random IDs is effectively arbitrary. Data-only searches have the same problem.

Proposal: `--sort FIELD[,-FIELD...]` (e.g. `title`, `data.priority`, `-data.created_at`) for list and data-only search; MCP `sort` array. Ranked full-text/semantic search keeps relevance order unless a sort is given.

Acceptance criteria:
- Stable, deterministic ordering with ID as the final tie-breaker.
- Defined behavior for missing fields and mixed types.
- Sorting applies before offset/limit.
- Consider schema enum order for fields like priority.
