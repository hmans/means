# Issue tracking workflow

This collection contains issues for a fictional example application. Work on them
when the user's request concerns this example. Do not treat them as the means
project's backlog.
Use the separate `memory` collection for durable facts, preferences, and context
that do not represent actionable work.

## Schema and content

Read the current data schema through `info` before creating or editing an issue.
Means generates the immutable `frontmatter.id` and requires a nonempty
`frontmatter.title`. The collection schema applies only to `frontmatter.data`:

- `type`: `task`, `bug`, or `feature`.
- `status`: `todo`, `in_progress`, `done`, or `canceled`.
- `priority`: `low`, `normal`, `high`, or `urgent`.
- Optional `tags`: unique, nonempty strings.

Supply required data fields explicitly; schema descriptions do not populate
values. For example:

```json
{"frontmatter":{"title":"Add translations","data":{"type":"feature","status":"todo","priority":"normal"}},"body":"Acceptance criteria..."}
```

Never supply `id` on creation or inside a patch. Use the body for context,
acceptance criteria, reproduction steps, and progress notes.

## Working with issues

1. List before proposing or creating work. Use `list` with `limit: 0`
   to see all issues. Check for duplicates and read relevant bodies by ID.
2. When asked to choose work, prefer `todo` issues by priority: urgent, high,
   normal, then low. Consider the user's instructions and the issue's context.
3. Create focused issues with `data.status: todo` unless the user specifies otherwise.
   Include concrete acceptance criteria. Do not retry a successful creation:
   every successful call creates a new issue.
4. For work the user asks you to perform, record `in_progress` when starting and
   `done` only after the acceptance criteria are met. Record the outcome in the
   body. Use `canceled` for intentionally abandoned work.
5. Report relevant issue IDs so the user can find the records. IDs are
   independent of filenames; preserve the stored `id`.

## Current capabilities

Means exposes creation, reading, listing, and patch updates. Filtering, deletion, and
`doctor` are not implemented. Filter list results yourself and inspect any
per-document `error` entries instead of treating them as valid issues.

For a requested update, call `read` first and use its `hash` in `update`, along
with the issue ID and only the fields that need changing. For example, set
`frontmatter` to `{"data":{"status":"in_progress"}}` to start work. Metadata objects merge
recursively, null removes a field, and arrays replace entirely. The optional
`body` replaces the whole body; omit it to preserve existing context.

The means envelope must stay valid and resulting data must satisfy the schema. If the hash is stale, read
again, review the intervening changes, and reapply the intended patch. Do not
blindly retry or create a replacement issue. Prefer `canceled` to deleting an
abandoned issue so its context remains available.
