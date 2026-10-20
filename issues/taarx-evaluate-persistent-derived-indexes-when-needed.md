---
data:
  priority: low
  status: todo
  tags:
    - deferred
    - search
  type: task
id: taarx
title: Evaluate persistent derived indexes when needed
---
Later / under consideration: means may store derived data locally if collection size or search performance warrants it. Markdown files and schemas remain authoritative. Storage format is undecided.

Before implementation:
- Demonstrate a need using actual workloads.
- Define rebuilding and invalidation after direct filesystem edits.
- Keep ordinary commands short-lived and avoid a managed daemon.

Embeddings and vector search are outside the initial scope. Do not build indexing infrastructure speculatively.

Migrated from TODO.md.
