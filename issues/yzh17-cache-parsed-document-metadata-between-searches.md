---
data:
  priority: normal
  status: done
  tags:
    - search
    - storage
  type: feature
id: yzh17
title: Cache parsed document metadata between searches
---
Add .means/documents.jsonl with document paths, SHA-256 checksums and parsed metadata. Hash all source files on each search; reuse parsing only for unchanged content. Preserve precise numbers, diagnostics, duplicate detection and collection boundaries. Recover from missing, corrupt or obsolete caches; write atomically under the search lock. Test filesystem changes and measure warm scan costs.

Outcome:
- Added .means/documents.jsonl for full-text search: path, SHA-256, parsed frontmatter and body offset per valid document. Versioned header checksums records to detect corruption and truncation. JSON decoding preserves number precision.
- Every scan still reads and hashes source files and discovers collection boundaries. Invalid documents are reparsed; duplicate detection remains per scan. Other commands use direct scans.
- Cache refresh uses atomic replacement under search.lock and skips unchanged writes. Missing/obsolete/corrupt caches rebuild; nonregular paths fail.
- Added source-equivalence tests for warm scans, exact numbers, CRLF, diagnostics, same-size/same-mtime edits, moves, repairs, duplicates, deletes, corruption, truncation and symlink/directory rejection. Existing search lifecycle tests also pass with the metadata cache.
- Apple M4 Max, 1000 synthetic documents, 10 iterations: full parsing 21.4 ms / 22.8 MB allocated; cached metadata 17.5 ms / 15.9 MB; read/hash lower bound 13.2 ms / 7.8 MB. Small local sample, not a general performance guarantee.

Validation: mise test; go test -race for collection, CLI and MCP; git diff --check; real issues full-text search without diagnostics. Updated guide, CLI help and project instructions.

Cleanup follow-up:
- Track cache hits and valid changes during scanning; skip JSON encoding as well as writes when records are unchanged. Missing old records still trigger refresh for deletion, boundary changes, read failures and invalidated documents.
- Pass each already-computed content hash into parsing instead of hashing cache misses twice.
- Extract reconcileSearchIndex while retaining the independent Bleve manifest and atomic index/manifest batch.
- Added empty-cache and valid-to-invalid transition tests. mise test and git diff --check passed.
- Warm metadata scan allocations fell from about 15.9 MB / 59,218 allocations to 14.5 MB / 49,178 allocations for 1,000 documents. Latest isolated 10-iteration sample: full parse 23.1 ms, cached 19.7 ms, read/hash baseline 14.6 ms. Timing varies between samples; no demonstrated latency improvement over the earlier cached run.
