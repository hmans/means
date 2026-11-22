---
created_at: "2026-11-22T16:31:40Z"
data:
  priority: normal
  status: done
  tags:
    - cli
    - mcp
    - search
  type: feature
id: h3brd
title: Sort list and search results by metadata
updated_at: "2026-11-22T16:55:44Z"
---
`list` returns documents in filename order, which with random IDs is effectively arbitrary. Data-only searches have the same problem.

Proposal: `--sort FIELD[,-FIELD...]` (e.g. `title`, `data.priority`, `-data.created_at`) for list and data-only search; MCP `sort` array. Ranked full-text/semantic search keeps relevance order unless a sort is given.

Acceptance criteria:
- Stable, deterministic ordering with ID as the final tie-breaker.
- Defined behavior for missing fields and mixed types.
- Sorting applies before offset/limit.
- Consider schema enum order for fields like priority.

## Outcome

Implemented `Sort []string` on `collection.ListOptions` and `SearchOptions` (logic in collection/sort.go), CLI `--sort KEY[,KEY...]` (comma-separated or repeatable) on list/search, and MCP `sort` string arrays on list/search.

- Keys: `id`, `title`, `data.FIELD` (dots nest), `-` prefix descends; invalid, empty, or duplicate keys fail.
- Sorting runs after all filters and the semantic min_score threshold, before offset/limit/has_more.
- Missing, null, and paths through non-objects sort last in either direction. Mixed types order booleans, numbers (exact), strings (code points), arrays, objects.
- Ties keep relevance order (then ID) for full-text/semantic search, otherwise ID. Sort replaces relevance; semantic hits keep their score.
- Schema enum order deferred to gs8qf; strings compare literally.

Validation: collection, CLI, and MCP tests (list/search/literal/data-only/full-text/semantic, paging, validation); `mise test` passes; checked against ./issues.
