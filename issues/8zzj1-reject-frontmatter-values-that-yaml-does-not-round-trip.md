---
created_at: "2026-11-15T00:11:52Z"
data:
  priority: high
  status: done
  tags:
    - storage
  type: bug
id: 8zzj1
title: Reject frontmatter values that YAML does not round-trip
updated_at: "2026-11-15T00:11:52Z"
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

## Outcome

`encodeMarkdown` (shared by create, update, and MEANS.md rewrites) now parses what it encoded and compares it with the input value, numbers by exact text, instead of only checking that it parses.

- Preserved: strings containing NEL, LS, or PS, and strings starting with a tab, are written as double-quoted scalars. DEL, C1 controls, U+FFFE, and U+FFFF, which the YAML reader previously refused with "control characters are not allowed", are escaped and stored exactly.
- Rejected: numbers outside the float64 range (`1e400`, `1e309`, a 400-digit integer). YAML resolves them as strings, so the write fails with `frontmatter.<path> cannot be stored as YAML without changing its value; numbers must fit the float64 range`. Large integers, long decimals, and underflowing exponents within range keep their exact text.

Validation: `collection/roundtrip_test.go` covers Create/Update-then-Read equality for about 110 edge values as titles, keys, and values, the rejections through Create and Update, and a fuzz target for arbitrary strings and numbers (40s, about 300k executions, no failures). `mise test` passes.
