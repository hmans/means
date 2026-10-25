# means Changelog

## 0.3.0

### Features

- Extend means doctor with offline warnings for missing or stale document embeddings, orphaned vectors, and corrupt or incompatible embedding caches. Reports include a means embed repair hint and preserve all files. Embedding findings appear as WARNING in text and in a separate JSON warnings array; warnings alone exit 0. Document validation errors and failures to complete the check still exit 1.

  Style text reports with a colored status summary, grouped errors and warnings with counts, and readable paths and messages. Disable colors for pipes and NO_COLOR; preserve JSON output.

  ([#12](https://github.com/hmans/means/pull/12))

## 0.2.0

### Features

- Add opt-in semantic search through the CLI and MCP, with OpenAI embeddings accessed directly or through OpenRouter. Configure the provider, model, and dimensions per collection; defaults use text-embedding-3-small and 512 dimensions. Provider selection supplies the default endpoint and credential variable, with environment overrides available.

  When embeddings are enabled, init writes omitted model and dimensions explicitly into MEANS.md. Explicit values and Markdown body bytes are preserved; frontmatter may be reformatted. CLI JSON mode suppresses embedding progress so errors remain valid JSON.

  Embed each document's title and body together and store vectors in a local JSONL cache. Create/update maintain vectors automatically; delete removes them without an API call. Existing full-text and literal search remain offline and are still available without embeddings.

  Semantic search embeds only the query and ranks current stored vectors by cosine similarity. It skips missing or stale vectors without modifying the cache and reads the cache without waiting for the cache lock. The minimum score defaults to 0.3, configurable through CLI --min-score or MCP min_score from -1 to 1. Apply the threshold before result limits, display scores in semantic search tables, and omit search progress messages.

  Use CLI means embed to backfill or refresh vectors after direct file edits, or means embed --force to clear stored vectors and regenerate every valid document. Refreshes process batches of up to 32 documents, checkpoint each completed batch, and report progress on stderr. Failed runs retain completed batches; ordinary embed resumes from that progress. Refresh and forced rebuilds are CLI-only, with no MCP embed tool.

  ([#10](https://github.com/hmans/means/pull/10))

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
