---
data:
  priority: high
  status: todo
  tags:
    - storage
  type: bug
id: 8zzj1
title: Reject frontmatter values that YAML does not round-trip
---
Found in the codebase review of 4c63f85 (confirmed by reproduction through the public API).

The round-trip guard in `collection/markdown.go:74-77` only checks that the encoded frontmatter parses, not that it decodes to the same value. Two known divergences:

- a string containing U+0085 (NEL) is stored as a space, including in titles;
- `json.Number("1e400")` is stored as `1e400` and reads back as the string `"1e400"`.

`Create` and `Update` return the unmodified input value together with a hash for a file whose content differs.

Acceptance criteria:
- A value that would not read back identically is either encoded so that it does (for example double-quoted strings for NEL) or rejected with a clear error; nothing is silently altered.
- The response from create/update always matches what a subsequent read returns.
- A table or fuzz test covers Create-then-Read equality for edge values, including the two cases above.
