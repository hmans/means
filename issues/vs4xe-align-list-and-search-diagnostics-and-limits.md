---
data:
  priority: normal
  status: done
  tags:
    - cli
    - mcp
  type: bug
id: vs4xe
title: Align list and search diagnostics and limits
---
Aligned list with search: responses contain documents, diagnostics and has_more. Malformed files and every duplicate occurrence are excluded from documents. Limits count valid documents only; scanning continues to report all diagnostics. Empty responses use arrays, and has_more only indicates further valid documents. CLI and MCP share this behavior; --with-body applies only to valid documents.

Updated CLI/MCP help, project instructions and guide, including migration guidance for the breaking list JSON contract change. Corrected stale list summary, body-output descriptions and README claim about generated titles.

Validation: mise test and git diff --check passed. Regression tests cover broken files before/after valid results, duplicates across paths (including invalid envelopes), all-invalid and empty collections, limit boundaries, diagnostics parity with search, schema-invalid data remaining readable, body inclusion, CLI table behavior and MCP JSON results. Installed the updated local binary with mise deploy.