---
data:
  priority: high
  status: done
  tags:
    - embeddings
    - review
  type: bug
id: jkhpe
title: Preserve JSON error output when embedding progress is enabled
---
Adversarial review of 9rxzp. Severity P1. internal/cli/output.go:11 writes progress to stderr unconditionally. Execute later writes the JSON error to that same stream. Reproduced with an embedding endpoint returning HTTP 503 during create --json: stdout empty; stderr starts with means: Embedding document and then a JSON error, so json.Valid(stderr) is false. This violates the structured CLI error contract. Keep human progress, but make JSON error consumption reliable (for example suppress progress by default in JSON mode or route it separately). Add a regression test for an API failure after progress has started; existing tests only checked successful JSON output.

Resolved: CLI JSON mode suppresses embedding progress; human-readable CLI and MCP maintenance retain progress. Added an HTTP 503 regression test proving stderr remains a single valid JSON error and no document is written. mise test passes.
