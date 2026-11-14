---
data:
  priority: high
  status: todo
  tags:
    - embeddings
    - storage
  type: bug
id: 6br4n
title: Keep stored vectors when a write runs under a different embedding profile
---
Found in the codebase review of 4c63f85 (confirmed by reproduction: 5 records before, 0 after a single delete under another endpoint).

On a profile mismatch (provider, model, dimensions or endpoint), `collection/embeddings.go:184-187` loads the cache as empty and marks it dirty. Any later delete or update then saves it (`:285-294`, `collection/update.go:136-141`), rewriting `.means/embeddings.jsonl` with no records. One invocation with a different `MEANS_EMBEDDING_BASE_URL` followed by an unrelated write destroys every paid vector. AGENTS.md says profile changes invalidate vectors; it does not say an unrelated write deletes them.

Acceptance criteria:
- Create, update, append and delete never discard vectors that belong to another profile. Only `embed` (including `embed --force`) may replace a mismatched cache.
- Search and doctor keep treating mismatched vectors as unusable/stale.
- A test checks that the old profile vectors survive an unrelated write under a different profile.
- AGENTS.md and `docs/guide.md` state the behaviour.
