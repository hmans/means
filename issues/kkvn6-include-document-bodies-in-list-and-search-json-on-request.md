---
created_at: "2026-10-22T09:58:44Z"
data:
  priority: normal
  status: done
  tags:
    - cli
  type: feature
id: kkvn6
title: Include document bodies in list and search JSON on request
updated_at: "2026-10-22T09:58:44Z"
---
Added --with-body to list/search with --json. Valid documents include body, including empty strings. Diagnostics omit body; hashes remain omitted. Existing scans supply bodies without per-result reads. Defaults, ordering, filters and limits are preserved. Text mode rejects --with-body without --json. Updated CLI help and guide.

Validation: mise test and git diff --check passed. Regression tests cover list, data-only search, full-text search, literal search, limits, diagnostics, empty and Unicode Markdown bodies, unchanged default JSON and the JSON-only flag requirement.