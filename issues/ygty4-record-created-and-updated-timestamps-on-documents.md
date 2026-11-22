---
created_at: "2026-11-22T16:31:40Z"
data:
  priority: normal
  status: done
  tags:
    - storage
    - envelope
  type: feature
id: ygty4
title: Record created and updated timestamps on documents
updated_at: "2026-11-22T17:17:42Z"
---
Nothing records when a document was created or changed. The issues collection's "Recently created" view stays empty because no document has `created_at`, and datetime filters have little to work on.

Proposal: means-owned envelope fields (e.g. `created`, `updated` as RFC 3339) set by create/update/append/edit, alongside `id`/`title`/`data`.

Questions to settle:
- Envelope fields vs. conventional `data` fields; how filters/sort/views address them.
- Handling of existing documents without timestamps (absent vs. backfill from git/mtime).
- Whether direct edits are expected to maintain `updated`.
- Body-only edits currently preserve frontmatter text exactly; decide how `updated` interacts with that.


## Decision

`created_at` and `updated_at` are core means fields in the envelope (next to id/title/data), not data fields or a schema keyword.

## Outcome

- Creation sets both; every write that changes a document (update, set, append, edit, MCP) bumps `updated_at`. No-op writes leave the file untouched. RFC 3339 UTC, second precision.
- Not client-writable: create/update input with either field fails; `edit` ignores edited values. Envelope validation rejects non-RFC 3339 values.
- Body-only writes now re-encode frontmatter (the old byte-preservation guarantee for body-only edits is dropped); the body stays exact.
- Available as Expr variables (`--where`, views), sort keys (compared as instants), and view columns. Datetime flags stay data-only.
- Legacy documents may lack timestamps; means never invents `created_at` and adds `updated_at` on the next write. Breaking: older means versions reject the new fields.
- This collection was backfilled once from git history (first/last commit per file); views "Recently created" and "Recently updated" use the new fields.

Validation: `mise test` passes; new tests cover create/update/append/set/edit, no-op writes, client rejection, legacy and invalid values, sorting by instant, and Expr access.
