---
created_at: "2026-11-22T16:31:40Z"
data:
  priority: high
  status: done
  tags:
    - cli
    - mcp
    - ux
  type: feature
id: re2j2
title: Add a set command for quick field edits
updated_at: "2026-11-22T16:44:49Z"
---
Changing one field today requires `show --json`, copying the hash, and composing a JSON merge patch. This is the most common edit and should be one command.

Proposal: `means set <id> KEY=VALUE...` (values may be JSON, like `--filter`), reading the current hash internally and applying the same validated, hash-checked update path. Consider `KEY+=VALUE` / `KEY-=VALUE` for array add/remove and `--title`.

Acceptance criteria:
- Sets/deletes top-level and nested data fields without the caller supplying a hash.
- Uses the shared update path: schema validation, embeddings, round-trip checks.
- An optional `--hash` keeps strict stale-write protection for agents.
- Decide whether MCP needs an equivalent or `update` with optional hash suffices.


## Outcome

Added `means set <id> KEY=VALUE|KEY+=VALUE|KEY-=VALUE` with `--title` and optional `--hash`, backed by public `Collection.Set(ctx, id, hash, edits...)` and `FieldEdit`.

- Hashless by default; reads the latest version under the collection edit lock, now shared with `append` (`.means/edit.lock`, formerly `append.lock`). Uses the shared update path: schema validation, embeddings, round-trip checks.
- Dotted keys address nested data; missing parents are created; `null` deletes; `+=` appends unless an equal element exists, `-=` removes equal elements. Metadata is replaced as a whole, so existing literal nulls survive.
- MCP decision: no new tool. MCP clients already read before writing, and `update` with a hash covers it; revisit if agents need hashless edits.

Validation: `mise test` passes; new collection and CLI tests cover edits, errors, and hash conflicts.
