# means Changelog

## 0.1.1

### Fixes

- Include Sweet Release notes in GitHub release drafts. ([#5](https://github.com/hmans/means/pull/5))

## 0.1.0

### Features

- Introduce means, a single Go binary for agentic memory stored as Markdown
  documents with typed YAML frontmatter and collection-local JSON Schemas.

  - Create, read, update, and delete documents with stable IDs and hash-checked writes.
  - Search collections with ranked full-text queries, literal matches, and data filters.
  - Validate schemas and documents, and inspect collection diagnostics.
  - Use structured CLI input and output or a collection-bound stdio MCP server.
  - Read agent onboarding guidance with `means agents`.

  ([#3](https://github.com/hmans/means/pull/3))
