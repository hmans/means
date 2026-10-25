---
title: means issues
description: "The means project backlog: bugs, features, design decisions, and planned work on means itself. Use this collection to track priorities, acceptance criteria, and outcomes."
embeddings:
  provider: openrouter
  model: openai/text-embedding-3-small
  dimensions: 512
schema:
  $schema: https://json-schema.org/draft/2020-12/schema
  type: object
  properties:
    type:
      type: string
      enum: [task, bug, feature]
      description: Task for general work, bug for incorrect behavior, or feature for new functionality.
    status:
      type: string
      enum: [todo, in_progress, done, canceled]
      description: Todo before work starts, in_progress while working, done after acceptance criteria are met, or canceled when intentionally abandoned.
    priority:
      type: string
      enum: [low, normal, high, urgent]
      description: Relative importance, ordered urgent, high, normal, then low.
    tags:
      type: array
      items:
        type: string
        minLength: 1
      uniqueItems: true
      description: Optional labels for grouping related issues.
  required: [type, status, priority]
  additionalProperties: false
---
# Working with means issues

This collection is the means project's own backlog. It contains actionable work,
bugs, and design decisions for this repository.

Use means to create, read, list, search, and update issues. From the repository root:

```sh
mise means list --path ./issues --limit 0 --json
mise means search --path ./issues --filter status=todo --filter priority=normal --json
mise means read <id> --path ./issues --json
mise means create --path ./issues --input - --json
mise means update <id> --path ./issues --input - --json
```

Read the schema in this file's frontmatter before creating issues. When accessed through MCP, use `info`
for the current schema and these instructions; the tools are `list`, `search`,
`read`, `create`, `update`, `delete`, and `schema_set`.

This collection uses OpenRouter embeddings. Set `OPENROUTER_API_KEY` in the CLI
or MCP host environment, or put `OPENROUTER_API_KEY=...` in the repository-root
`.env` file. Mise loads this optional, Git-ignored file for CLI tasks and the
MCP server launched through mise. Restart an existing MCP connection after
changing the file. Run `mise means embed --path ./issues` to backfill
existing issues. Use `search --semantic` (MCP: `semantic: true`) for semantic
search. Create/update maintain title/body embeddings automatically and require
the key when a new vector is needed; list/read and ordinary search remain offline.

## Issue content

Means generates `frontmatter.id` and requires a nonempty `frontmatter.title`.
The collection schema validates only `frontmatter.data`: required fields are
`type` (`task`, `bug`, or `feature`), `status` (`todo`, `in_progress`, `done`, or
`canceled`), and `priority` (`low`, `normal`, `high`, or `urgent`). Optional `tags`
are unique, nonempty strings. Supply required data fields explicitly.

For creation, use:

```json
{"frontmatter":{"title":"Add a feature","data":{"type":"feature","status":"todo","priority":"normal"}},"body":"Context and acceptance criteria..."}
```

Never supply `id` on creation or inside a frontmatter patch. Use the body for
context, acceptance criteria, reproduction steps, and outcomes. IDs are stored in
frontmatter; they are independent of filenames and remain valid after moves.

## Workflow

1. List all issues and read relevant bodies before proposing or creating work.
   Check for duplicates and inspect any per-document errors.
2. Follow the user's scope and priorities. The `data.tags` value `deferred` marks ideas that
   remain out of scope until the user chooses to pursue them.
3. For agreed work, update its data.status to `in_progress` when starting. Set `done`
   only when acceptance criteria are met, and record the outcome and validation.
   Use `canceled` for intentionally abandoned work.
4. Read before each update and supply the returned `hash`. On conflict, read
   again and reconcile the patch with the new content before retrying.
5. Report relevant issue IDs. Preserve filenames; never create replacement
   issues to bypass conflicts or blindly retry a successful creation.

For example, after reading an issue, update it with:

```json
{
  "hash": "<hash from read>",
  "frontmatter": {"data": {"status": "in_progress"}}
}
```

Include the issue ID as the CLI argument or MCP `id` field. Metadata objects
merge recursively, null deletes a field, and arrays replace entirely. Omit `body`
to preserve it; when supplied, it replaces the full body, so retain useful context.
Updates validate the means envelope and resulting data. Search can match issue
titles/bodies and filter `frontmatter.data` fields such as `type`, `status`, and
`priority`; inspect its diagnostics for malformed files. Doctor checks all
documents against the current schema. Delete permanently removes a document by
ID and current hash. Prefer status `done` or `canceled` to retain issue history.
