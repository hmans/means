---
data:
  priority: normal
  status: todo
  tags:
    - search
  type: feature
id: 4fx7d
title: Improve plaintext search beyond literal matching
---
The current `search` command matches one case-insensitive literal phrase in document titles and bodies. Explore more useful plaintext retrieval, such as multi-term queries, tokenization, ranking, and better excerpts. Evaluate Bleve or other full-text approaches; consider vector embeddings only if a concrete use case calls for semantic matching.

Acceptance criteria:
- Define the user-facing query and result semantics, including how they coexist with the current literal search and data filters.
- Compare a direct scan with candidate search libraries and optional semantic retrieval using representative collections and queries; record the choice and its tradeoffs.
- Implement the agreed behavior in the shared collection layer, CLI, and MCP, with documentation and tests for relevance, empty results, and malformed documents.
- If persistent indexes are introduced, keep Markdown and schema files authoritative and define rebuilding and invalidation after direct file edits. Ordinary commands remain short-lived, with no managed daemon.

Related: `j2kxp` delivered basic literal search; `taarx` tracks derived indexes and remains deferred until there is an observed need.
