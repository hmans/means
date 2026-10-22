---
data:
  priority: normal
  status: done
  tags:
    - search
  type: feature
id: 4fx7d
title: Improve plaintext search beyond literal matching
---
The current `search` command matches one case-insensitive literal phrase in document titles and bodies. Implement more useful plaintext retrieval, such as multi-term queries, tokenization, ranking, and better excerpts. Evaluate Bleve or other full-text approaches; consider vector embeddings only if a concrete use case calls for semantic matching.

Acceptance criteria:
- Define the user-facing query and result semantics, including how they coexist with the current literal search and data filters.
- Compare a direct scan with candidate search libraries and optional semantic retrieval using representative collections and queries; record the choice and its tradeoffs.
- Implement the agreed behavior in the shared collection layer, CLI, and MCP, with documentation and tests for relevance, empty results, and malformed documents.
- If persistent indexes are introduced, keep Markdown and schema files authoritative and define rebuilding and invalidation after direct file edits. Ordinary commands remain short-lived, with no managed daemon.

Related: `j2kxp` delivered basic literal search; `taarx` records the collection-local persistent index selected for this implementation.

Outcome:
- Added Bleve v2.6.1 ranked full-text search as the default in CLI and MCP. All lowercase Unicode tokens must match across title/body; title receives a boost of 3, score ties sort by ID. No stemming, stop-word removal, query operators, or semantic search. Literal substring mode remains available with --literal / literal:true; data-only and literal queries retain direct scans in filename order.
- Exact application-data filters apply before limits, including over multiple result pages. Excerpts and metadata come from the files observed during the scan. Malformed and duplicate documents remain diagnostics; schema-invalid data remains searchable.
- Refresh .means/search/ on each full-text operation using full file hashes, batch updates/deletes, and a versioned manifest committed with the index changes. No watcher or daemon; MCP closes the index after each call.
- Updated help, guide, project instructions, and init to preserve/create .gitignore with .means/. Added the rule to the repository issues collection.

Validation:
- mise test passed across all packages; race-enabled tests passed for collection, CLI, and MCP.
- Tests cover multi-term/title/body ranking, Unicode, retained stop words, no stemming, exact filters across result pages, literal mode, same-size/timestamp edits, renames, deletes, additions, duplicates, malformed files, collection boundaries, cache removal/corruption/version changes, symlink refusal, lock timeouts, concurrent goroutines/processes, and unclean process exit with index and lock open.
- Init tests cover .gitignore preservation, duplicate rules, missing newline/CRLF, reinitialization, and symlink refusal. MCP tests confirm updates on a live connection. BenchmarkFullTextIndex compares warm persistence with rebuilding Bleve in memory.
- Real issues search succeeded with the cache ignored by Git; doctor reported 22 healthy documents. git diff --check passed.
