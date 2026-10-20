# Issue tracking with means

This directory demonstrates an issue tracker built on means. Its collection is
`example/issues/`; the issues are fictional examples, not the means project's
backlog. Work on them when the user's task concerns this example.

## Use the CLI

Use means to create, list, and read issues. Run these commands from the repository
root so the collection path is unambiguous:

```sh
mise means list --path ./example/issues --limit 0 --json
mise means read <id> --path ./example/issues --json
mise means create --path ./example/issues --input - --json
```

Read `issues/schema.json` before creating or editing issues. Use
`mise means <command> --help` for input and output details. This collection is
already initialized; do not run `init` over it.

The issue ID is the Markdown filename stem returned by means. Use the ID exactly
as returned; do not add an `id` field to the frontmatter or rename an issue file.
JSON responses are pretty-printed; parse the whole response. A failed command
returns exit code 1 and a JSON `error` on stderr.

## Issue schema

Required fields:

- `title`: a short, nonempty description of the work.
- `type`: `task`, `bug`, or `feature`.
- `status`: `todo`, `in_progress`, `done`, or `canceled`.
- `priority`: `low`, `normal`, `high`, or `urgent`.

Optional `tags` are unique, nonempty strings. Other fields are rejected. Supply
all required fields explicitly; schema descriptions do not populate values.

Use the Markdown body for context, acceptance criteria, reproduction steps, and
progress notes. Keep machine-queryable fields in frontmatter.

## Workflow

1. List the collection before proposing or creating work. Check for existing
   issues with the same purpose, and read relevant issue bodies by ID.
2. When asked to choose work, prefer `todo` issues by priority: urgent, high,
   normal, then low. Consider the user's instructions and context in the body.
3. Create focused issues with `status: todo` unless the user specifies otherwise.
   Use concrete acceptance criteria. Do not retry a successful creation: each
   successful call creates a new issue.
4. For work the user asks you to perform, record `in_progress` when starting and
   `done` only after the acceptance criteria are met. Record the outcome in the
   body. Use `canceled` for intentionally abandoned work.
5. Report relevant issue IDs in your response so the user can find the records.

Example creation (adapt the content to the user's request):

```sh
mise means create --path ./example/issues --input - --json <<'JSON'
{
  "frontmatter": {
    "title": "Explain how to create the first issue",
    "type": "task",
    "status": "todo",
    "priority": "normal",
    "tags": ["documentation"]
  },
  "body": "## Context\nNew users need a concrete starting point.\n\n## Acceptance criteria\n- Include a working create command.\n- Explain how to read the returned issue ID.\n"
}
JSON
```

## Current limitations

Means currently supports creation, reading, and listing. Filtering, updates,
deletion, and `doctor` are not implemented. Filter list results yourself and
inspect any per-document `error` entries instead of treating them as valid issues.

For a requested update, first read the issue through means, then edit its existing
Markdown file directly, preserving its filename and schema-compatible metadata.
Read it again through means to verify the result. Reads check parsing, not schema
compliance. Do not create a replacement issue to simulate an update. Prefer
`canceled` to deleting an abandoned issue so its context remains available.
