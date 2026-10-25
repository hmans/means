---
type: feature
---

Add opt-in semantic search through the CLI and MCP, with OpenAI embeddings accessed directly or through OpenRouter. Configure the provider, model, and dimensions per collection; defaults use text-embedding-3-small and 512 dimensions. Provider selection supplies the default endpoint and credential variable, with environment overrides available.

When embeddings are enabled, init writes omitted model and dimensions explicitly into MEANS.md. Explicit values and Markdown body bytes are preserved; frontmatter may be reformatted. CLI JSON mode suppresses embedding progress so errors remain valid JSON.

Embed each document's title and body together and store vectors in a local JSONL cache. Create/update maintain vectors automatically; delete removes them without an API call. Existing full-text and literal search remain offline and are still available without embeddings.

Semantic search embeds only the query and ranks current stored vectors by cosine similarity. It skips missing or stale vectors without modifying the cache and reads the cache without waiting for the cache lock. The minimum score defaults to 0.3, configurable through CLI --min-score or MCP min_score from -1 to 1. Apply the threshold before result limits, display scores in semantic search tables, and omit search progress messages.

Use CLI means embed to backfill or refresh vectors after direct file edits, or means embed --force to clear stored vectors and regenerate every valid document. Refreshes process batches of up to 32 documents, checkpoint each completed batch, and report progress on stderr. Failed runs retain completed batches; ordinary embed resumes from that progress. Refresh and forced rebuilds are CLI-only, with no MCP embed tool.
