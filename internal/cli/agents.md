# Collections for agents

This guide ships with the installed means binary. Use `means help <command>`
for command syntax and input formats.

## Collection structure

A collection is a directory containing `MEANS.md` and Markdown documents.
Files are the source of truth and can be edited directly. Select the directory
with `--path`; means does not search parent directories.

Visible subdirectories belong to the collection until a nested `MEANS.md` starts
another collection, even if that configuration is invalid. Nested collections
have independent schemas, IDs, and guidance; nothing inherits from the parent.
Hidden paths and symlinks are excluded from document scans. `.means/` contains
disposable local caches, not source documents.

## Write MEANS.md

Use YAML frontmatter for configuration and the Markdown body for workflow:

```yaml
---
title: Project decisions
description: Decisions and their rationale for this project.
schema:
  type: object
  properties:
    status:
      type: string
      enum: [proposed, accepted, superseded]
      description: The current stage of the decision.
  required: [status]
  additionalProperties: false
---
# Working with project decisions

Search for an existing decision before creating one.
Start new decisions as proposed. Mark accepted only after agreement.
Explain context, alternatives, and rationale in the body.
When replacing a decision, mark it superseded and reference its successor by ID.
```

`title` and `description` are optional strings. A missing or blank title defaults
to the directory name. `schema` is required: a JSON Schema object or boolean,
expressed in YAML. Draft 2020-12 is the default; references must stay within the
schema. Schema annotations do not set collection identity.

The schema validates only each document's `data`, not its title, ID, or body.
Choose fields that support the application's queries and rules. Use descriptions
to explain meaning, enums for fixed choices, and `required` for mandatory fields.
Schema defaults do not fill in data. Put process rules and body requirements in
the Markdown workflow; means does not enforce that prose.

Optional `app.views` configures experimental TUI views. Unknown configuration
fields are rejected. Embedding settings and credentials belong in machine-local
configuration and the environment, never in `MEANS.md`.

## Document structure

A stored document for the example collection could be:

```yaml
---
id: a1b2c
title: Use a single binary
data:
  status: proposed
---
A single binary would simplify installation. Compare it with a managed service
before accepting this decision.
```

Means generates the immutable five-character ID on creation; do not supply the
example ID. The title must be a nonempty string. All application fields belong
under `data`, which must be an object. Omitted creation data defaults to `{}`;
explicit null is invalid. The body is free-form Markdown.

Creation writes `<id>-<title-slug>.md`. Use stored IDs to reference documents:
renames and moves within a collection preserve identity, and title edits preserve
filenames. IDs must be unique within the collection. `MEANS.md` is configuration,
not a document; it is excluded from document lists and reads.

## Establish and maintain a collection

1. Inspect an existing collection with `means info --path DIR` before changing it.
   Read its schema and workflow, then list or search existing documents.
2. For a new collection, define its purpose, schema, and workflow. Use
   `means init --path DIR --input FILE` to install a complete `MEANS.md` without
   overwriting an existing one. See `means help init` for other inputs.
3. Make the collection discoverable. Add its path, purpose, and when-to-use rules
   to project instructions or a skill, or register it through MCP. Creating
   `MEANS.md` alone does not make agents discover it. Means does not load `AGENTS.md`.
4. Check representative documents with `means doctor --path DIR`. Creation and
   updates validate data; reads and lists remain available after schema changes.
   Inspect diagnostics instead of assuming every stored document is valid.
5. Use `schema set` to check a proposed schema against existing documents before
   installing it. It does not migrate documents. For a staged migration, accept
   both forms, migrate data, then tighten the schema.

Read a document before updating or deleting it and supply its current hash.
On a conflict, read again and reconcile changes. Keep collection-specific rules
in `MEANS.md`; use command help for operation details and `means help mcp` for
integration. MCP `info` supplies the current collection schema and guidance.
