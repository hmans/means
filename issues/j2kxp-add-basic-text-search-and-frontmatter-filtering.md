---
data:
  priority: normal
  status: done
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

Outcome: Added collection search, CLI `search`, and MCP `search`. The optional query is a case-insensitive literal phrase in title/body; data filters combine with AND, nested objects match supplied keys, arrays match exactly, and typed JSON values retain precision. Results include frontmatter and a match field/excerpt without bodies or hashes. The default limit is 20 matching documents; 0 returns all. Malformed files and duplicate IDs appear as diagnostics outside the result limit; schema-invalid application data remains searchable. Search does not write files. Updated CLI/MCP help, the README, and issue workflow instructions. The README now explains that means tracks this repository's own issues.

Validation: `GOCACHE=/tmp/means-go-cache mise test` passed. `mise means search --path ./issues --filter priority=normal --filter type=feature --json` returned four issues with no diagnostics. A bounded text search returned excerpts and `has_more: true`. `git diff --check` passed.
