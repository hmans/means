---
data:
  priority: low
  status: done
  tags:
    - search
  type: task
id: taarx
title: Evaluate persistent derived indexes when needed
---
Implemented at the user's request as part of ranked full-text search (4fx7d). The index is derived; Markdown and means.schema.json remain authoritative.

Decision:
- Keep a Bleve index under each collection's .means/search/ and an OS lock at .means/search.lock. Full-text searches wait up to five seconds, scan and hash every document, reconcile changes and removals, query, and close. Literal and data-only searches bypass the cache.
- Hashes catch edits with unchanged size/timestamps. Duplicate, malformed, deleted, and newly excluded documents are removed from the index. Adding/removing nested schemas changes scan membership. Cache metadata and document changes commit in one batch. Missing, obsolete, or unopenable indexes rebuild; permission errors and cache symlinks fail. No source files are rewritten.
- Full scans remain necessary without a watcher or filesystem journal. Results reflect observed files, not an atomic snapshot. Ordinary commands need no index writes, and MCP does not retain an open index between searches.
- Init creates or preserves .gitignore and ensures a .means/ rule. Cache storage is never authoritative and can be discarded when searches are idle.

Benchmark (local Apple M4 Max, synthetic ~2 KB documents, three iterations per case):
- 100 documents: warm persistent 18.2 ms / 4.0 MB allocated; in-memory rebuild 6.35 ms / 18.1 MB.
- 1,000 documents: warm persistent 36.3 ms / 23.3 MB; in-memory rebuild 50.9 ms / 185.9 MB.
- This preliminary sample is not a general speed guarantee. Persistence reduces analysis/allocation work for larger collections but startup makes it slower for small ones. Direct literal scanning remains available; Bleve provides ranked retrieval without a managed service. Embeddings and vector search remain deferred.

Validation:
- mise test passed across all packages; race-enabled tests passed for collection, CLI, and MCP.
- Tests cover multi-term/title/body ranking, Unicode, retained stop words, no stemming, exact filters across result pages, literal mode, same-size/timestamp edits, renames, deletes, additions, duplicates, malformed files, collection boundaries, cache removal/corruption/version changes, symlink refusal, lock timeouts, concurrent goroutines/processes, and unclean process exit with index and lock open.
- Init tests cover .gitignore preservation, duplicate rules, missing newline/CRLF, reinitialization, and symlink refusal. MCP tests confirm updates on a live connection. BenchmarkFullTextIndex compares warm persistence with rebuilding Bleve in memory.
- Real issues search succeeded with the cache ignored by Git; doctor reported 22 healthy documents. git diff --check passed.
