---
data:
  priority: normal
  status: done
  tags:
    - search
  type: task
id: 5bge8
title: Improve search excerpts and measure warm-search parsing costs
---
Prefer excerpts that cover more distinct query terms, add a representative relevance fixture, and measure parsing cost before introducing a parsed-document cache. Preserve literal mode and full-content hash freshness.

Outcome:
- Full-text excerpts select a bounded title/body passage with the greatest number of distinct complete query terms. Repetitions do not increase coverage; ties prefer title, then earliest passage. Up to 180 Unicode characters plus ellipses, with collapsed whitespace. Literal excerpts and result ranking are unchanged.
- Added a fixed issue-like relevance set covering index refresh, schema validation, deletion, and deployment timeout. Added excerpt tests for body-vs-title coverage, late matches, repeated terms, ties, split fields, Unicode, partial terms at window boundaries, and widely spaced terms.
- Added BenchmarkSearchScanCosts. On Apple M4 Max with 1,000 synthetic ~2 KB documents, 10 iterations: read/hash/parse 24.3 ms and 22.8 MB allocated; read/hash-only lower bound 14.9 ms and 7.8 MB. The ~9.4 ms difference is an upper bound on potential savings, not a measured cache speedup: cache decoding, envelope checks, duplicate detection and invalidation remain to be accounted for. No parsed-document cache was introduced; full-file hash freshness remains unchanged.
- Updated command help and usage guide. Real issue searches now show passages containing search and index together.

Validation: mise test passed across all packages, git diff --check passed, and the real issues search returned fresh multi-term excerpts without diagnostics.
