---
data:
  priority: normal
  status: done
  tags:
    - search
    - embeddings
  type: feature
id: 9rxzp
title: Add opt-in automatic OpenAI embeddings and semantic search
---
Implement the first embedding-backed search feature. Collections explicitly opt in. Create/update/delete maintain vectors automatically through the shared collection package, for both CLI and MCP. Store vectors in a separate local JSONL file without SQLite. Use OpenAI embedding models, with configurable OpenAI-compatible endpoint support for OpenRouter. Include document titles and bodies in embedding input. The user confirmed that descriptions need no support. Semantic searches reconcile direct file edits and retain filtering, diagnostics, and collection boundaries. Preserve ordinary offline behavior when disabled. Verify lifecycle failure handling, freshness, API validation, concurrency, and CLI/MCP contracts with a mock API; document configuration and limitations. Related completed search work: 4fx7d and taarx.

Outcome:
- Added validated MEANS.md embeddings configuration (disabled by default), OpenAI small/large models, optional dimensions, and process-only endpoint/key overrides for OpenRouter.
- Create/update cache a title/body vector before writing Markdown. API/cache failures leave source documents unchanged. Data-only edits reuse vectors. Delete removes the cached vector locally before the file. Separate cache and source writes are not one transaction; refresh reconciles failed writes and crashes.
- Store float vectors and input hashes in a separate versioned, checksummed .means/embeddings.jsonl with atomic replacement and an OS lock. Semantic search and embed reconcile external edits, removed/invalid/duplicate files, and nested boundaries. No SQLite, daemon, chunking, quantization, or shared vectors in this milestone.
- Added CLI embed and search --semantic, MCP embed and semantic search, cosine scores, exact pre-limit filters, diagnostics, and optional CLI bodies. MCP requests propagate cancellation to embedding requests and lock waits.
- Updated CLI help, guide, README, project instructions, and release note.

Validation:
- mise test passed across all packages.
- Race-enabled tests passed across collection, CLI, and MCP packages.
- Mock HTTP tests cover lifecycle, offline defaults, data-only reuse, title/body refresh, failures preserving source files, post-API conflicts including no-op updates, direct edits/deletes, filtering, diagnostics, duplicate/nested boundaries, cache/profile recovery, schema preservation, symlink rejection, API vector validation, cancellation, concurrent creates, and CLI/MCP integration.
- JSON embed help and git diff --check passed. No live paid API requests were made; OpenRouter routing is documented from its official endpoint reference rather than verified with credentials.

OpenRouter follow-up: OPENROUTER_API_KEY is now selected automatically for the openrouter.ai endpoint. OPENAI_API_KEY is not required and is never used as a fallback for that endpoint. MEANS_EMBEDDING_API_KEY remains an explicit override. Updated CLI help, usage guide, and release note. Endpoint credential-selection tests and mise test passed; git diff --check passed.

Provider configuration follow-up: Require embeddings.provider (openai or openrouter). It selects the default endpoint, model name, and credential variable; model and dimensions remain optional. OpenRouter now needs only provider: openrouter and OPENROUTER_API_KEY. Endpoint/key environment overrides remain supported; credential selection follows the configured provider rather than the endpoint hostname, superseding the earlier endpoint-based selection. Wrong provider/model combinations fail. Provider changes participate in the cache profile. Updated help, guide, project instructions, and release note. mise test passed, including provider defaults, strict config validation, credential isolation, and a mock OpenRouter request through a custom endpoint.

Issues collection setup: Configure issues/MEANS.md with provider: openrouter and forward OPENROUTER_API_KEY through the repository MCP registration. Initial backfill is pending because that variable is unavailable in this session. Run mise means embed --path ./issues after supplying it; opted-in creates and updates also require credentials when a new vector is needed.

Embedding progress: CLI create/update, embed, semantic search, and the stdio MCP server now report progress on stderr through an optional collection callback. Messages show document IDs and provider/model before API requests, query embedding without query text, and generated/reused/removed counts after refresh. Collection operations remain silent without a callback. No document/query text, vectors, or credentials appear in progress; stdout retains JSON results or MCP messages. Updated help and guide. mise test and CLI/MCP race tests passed; tests verify progress counts and clean structured responses.

Batching and vector-size follow-up: Refreshes now collect missing/stale documents and send sequential batches of at most 32, with one JSONL replacement per completed batch. Response indices map out-of-order vectors to inputs; missing, duplicate, out-of-range indices and invalid vectors reject the whole batch. Completed batches survive later failures and are reused on retry. Progress reports completed/total changed documents. Model and dimensions remain configurable; omitted dimensions now default to 512 with the provider's OpenAI text-embedding-3-small model. Existing explicit dimensions remain unchanged. Concurrent batches remain deferred.

Validation: mise test and race-enabled collection/CLI/MCP tests passed. A 65-document fixture used three requests (32/32/1), with no requests on an unchanged refresh; failed-second-batch tests retained and reused the first 32 vectors. Invalid batch responses never partially updated cache records. Live OpenRouter backfill completed all 34 issues at 512 dimensions in two requests, resolving the earlier pending backfill. Updated help, usage guide, project instructions, and release note.


Forced refresh follow-up: Added means embed --force and MCP embed force:true. Clear existing vectors atomically under the cache lock before regenerating all valid documents, including unchanged ones. Failures retain only batches completed in the forced run; ordinary embed resumes from those checkpoints. Markdown remains unchanged. Updated CLI help, guide, project rules, and release note. mise test passes, including first/later batch failures, resumability, all-document regeneration, source preservation, and CLI/MCP flag propagation.


MCP scope correction: Removed the embed tool, including force, from MCP. Backfill, refresh, and forced rebuilds are CLI-only via means embed. MCP retains semantic search, embedding configuration in info, and automatic maintenance through create/update/delete. Updated advertised instructions, guide, project rules, release notes, and MCP tool discovery tests. This supersedes the earlier MCP embed additions.


Final cleanup: Init now persists omitted model and dimensions for enabled embeddings, preserves explicit settings and body bytes, and may reformat frontmatter only when adding defaults. Disabled/default initialization remains opt-in. Issues config explicitly sets openai/text-embedding-3-small and 512 dimensions. CLI JSON progress is suppressed to preserve structured errors. Added init tests for both providers, partial/complete/disabled configs, schema number precision, and body preservation; mise test passes.
