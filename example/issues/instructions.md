# Issue tracking workflow

This collection contains issues for a fictional example application. Work on them
when the user's request concerns this example. Do not treat them as the means
project's backlog.
Use the separate `memory` collection for durable facts, preferences, and context
that do not represent actionable work.

## Schema and content

Read the current schema through `info` before creating or editing an
issue. Required frontmatter fields:

- `title`: a short, nonempty description of the work.
- `type`: `task`, `bug`, or `feature`.
- `status`: `todo`, `in_progress`, `done`, or `canceled`.
- `priority`: `low`, `normal`, `high`, or `urgent`.

Optional `tags` are unique, nonempty strings. Other fields are rejected. Supply
all required fields explicitly; schema descriptions do not populate values.
Use the Markdown body for context, acceptance criteria, reproduction steps, and
progress notes.

## Working with issues

1. List before proposing or creating work. Use `list` with `limit: 0`
   to see all issues. Check for duplicates and read relevant bodies by ID.
2. When asked to choose work, prefer `todo` issues by priority: urgent, high,
   normal, then low. Consider the user's instructions and the issue's context.
3. Create focused issues with `status: todo` unless the user specifies otherwise.
   Include concrete acceptance criteria. Do not retry a successful creation:
   every successful call creates a new issue.
4. For work the user asks you to perform, record `in_progress` when starting and
   `done` only after the acceptance criteria are met. Record the outcome in the
   body. Use `canceled` for intentionally abandoned work.
5. Report relevant issue IDs so the user can find the records. IDs are filename
   stems, without `.md`; preserve them and do not inject an `id` metadata field.

## Current capabilities

Means exposes creation, reading, listing, and patch updates. Filtering, deletion, and
`doctor` are not implemented. Filter list results yourself and inspect any
per-document `error` entries instead of treating them as valid issues.

For a requested update, call `read` first and use its `hash` in `update`, along
with the issue ID and only the fields that need changing. For example, set
`frontmatter` to `{"status":"in_progress"}` to start work. Metadata objects merge
recursively, null removes a field, and arrays replace entirely. The optional
`body` replaces the whole body; omit it to preserve existing context.

The resulting frontmatter must satisfy the schema. If the hash is stale, read
again, review the intervening changes, and reapply the intended patch. Do not
blindly retry or create a replacement issue. Prefer `canceled` to deleting an
abandoned issue so its context remains available.
