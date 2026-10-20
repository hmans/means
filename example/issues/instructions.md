# Issue tracking workflow

This collection contains issues for a fictional example application. Work on them
when the user's request concerns this example. Do not treat them as the means
project's backlog.

## Schema and content

Read the current schema through `collection_info` before creating or editing an
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

1. List before proposing or creating work. Use `list_documents` with `limit: 0`
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

Means exposes creation, reading, and listing. Filtering, updates, deletion, and
`doctor` are not implemented. Filter list results yourself and inspect any
per-document `error` entries instead of treating them as valid issues.

For a requested update, first read the issue through means. If filesystem access
is available, edit its existing Markdown file under the collection path returned
by `collection_info`, preserving its filename and schema-compatible metadata.
Read it again through means to verify parsing. Reads do not check schema compliance.
If filesystem access is unavailable, explain that updates require it; do not
create a replacement issue to simulate an update. Prefer `canceled` to deleting
an abandoned issue so its context remains available.
