---
data:
  priority: normal
  status: done
  tags:
    - cli
    - sdk
  type: feature
id: x1xw3
title: Append text to a document body
---
Add means append to append exact UTF-8 text to a document body by ID. Accept inline text, a body file or stdin, and structured JSON. Hashes are optional; means handles collection-local append locking and optimistic conflict checks internally. Preserve frontmatter and file permissions, validate the resulting document, and use existing embedding maintenance.

Outcome: Added means append and Collection.Append. Inline text, --body-file FILE|-, and --input FILE|- are supported. Appends insert no separators or newlines. An optional explicit hash enforces a specific version. Concurrent appends serialize on .means/append.lock with a context-aware wait of up to five seconds; other commands and direct editors retain existing OCC and external-write race limits. Repeated hashless requests append again. Updated canonical command help, usage/API guides, embedded agent guidance, and one feature release note.

Verification: mise test passed for all packages. Targeted append/help tests passed with the race detector; go vet ./... passed. Tests cover all input modes with and without hashes, exact body/frontmatter bytes, permissions, empty input, stale and malformed hashes, invalid UTF-8, schema violations, duplicate IDs, cancellation, embedding refresh/provider failures, and external edits during API calls. Four separate processes each append five times and preserve all twenty additions. Lock cancellation and symlink rejection are covered. git diff --check passed.