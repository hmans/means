---
data:
  priority: normal
  status: in_progress
  tags:
    - cli
    - sdk
  type: feature
id: x1xw3
title: Append text to a document body
---
Add means append to append exact UTF-8 text to a document body by ID. Accept inline text, a body file or stdin, and structured JSON. Require the current content hash, preserve frontmatter and file permissions, validate the resulting document, and use existing embedding maintenance. Document the command and verify success, stale hashes, invalid input, and embedding failures.

Outcome: Added means append and Collection.Append. Inline text, --body-file FILE|-, and --input FILE|- are supported. Appends insert no separators or newlines. Updated command help, the usage guide, Go API guide, and embedded agent guidance; added a feature release note.

Verification: mise test passed for all packages. Tests cover exact body and frontmatter bytes, permissions, empty input, stale and malformed hashes, UTF-8 errors, schema violations, duplicate IDs, cancellation, embedding refresh and provider failures, and external edits during embedding calls. CLI tests cover all input modes, text/JSON output, invalid inputs, and replay rejection. git diff --check passed.