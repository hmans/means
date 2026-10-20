---
data:
  priority: normal
  status: todo
  tags:
    - search
  type: feature
id: j2kxp
title: Add basic text search and frontmatter filtering
---
Consumers currently list metadata and filter results themselves. Add a basic way to find relevant documents by text and metadata.

Acceptance criteria:
- Define text matching and frontmatter filter semantics, including result limits.
- Expose the agreed behavior through CLI and MCP with useful result context.
- Document and test matching, empty results, and malformed-document behavior.

Embeddings and vector search are outside the initial scope. Persistent indexes are a separate, optional consideration.

Migrated from TODO.md.
