---
data:
  priority: normal
  status: todo
  tags:
    - storage
    - envelope
  type: feature
id: ygty4
title: Record created and updated timestamps on documents
---
Nothing records when a document was created or changed. The issues collection's "Recently created" view stays empty because no document has `created_at`, and datetime filters have little to work on.

Proposal: means-owned envelope fields (e.g. `created`, `updated` as RFC 3339) set by create/update/append/edit, alongside `id`/`title`/`data`.

Questions to settle:
- Envelope fields vs. conventional `data` fields; how filters/sort/views address them.
- Handling of existing documents without timestamps (absent vs. backfill from git/mtime).
- Whether direct edits are expected to maintain `updated`.
- Body-only edits currently preserve frontmatter text exactly; decide how `updated` interacts with that.
