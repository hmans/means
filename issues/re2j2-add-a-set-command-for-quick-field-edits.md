---
data:
  priority: high
  status: todo
  tags:
    - cli
    - mcp
    - ux
  type: feature
id: re2j2
title: Add a set command for quick field edits
---
Changing one field today requires `show --json`, copying the hash, and composing a JSON merge patch. This is the most common edit and should be one command.

Proposal: `means set <id> KEY=VALUE...` (values may be JSON, like `--filter`), reading the current hash internally and applying the same validated, hash-checked update path. Consider `KEY+=VALUE` / `KEY-=VALUE` for array add/remove and `--title`.

Acceptance criteria:
- Sets/deletes top-level and nested data fields without the caller supplying a hash.
- Uses the shared update path: schema validation, embeddings, round-trip checks.
- An optional `--hash` keeps strict stale-write protection for agents.
- Decide whether MCP needs an equivalent or `update` with optional hash suffices.
