# Using means as an agent

means stores shared knowledge as Markdown files: issues, decisions, research, or
memory. Files are the source of truth. Use the CLI or optional stdio MCP tools.

**Agent setup is complete when the collection has a schema, a workflow, and a
discovery path for future agents. Creating files alone is not enough.**

## Set up a collection

1. Inspect existing collections and project instructions. Reuse a suitable
   collection before creating another.
2. Design `MEANS.md` for the purpose. Its YAML frontmatter has optional string
   `title` and `description` fields and a required `schema` field containing JSON
   Schema expressed in YAML. The schema validates document `data` only.
   Put filterable attributes in `data` and evidence in document bodies.
3. Initialize with your complete configuration:

       means init --path ./issues --input ./template/MEANS.md

   Alternatively, `--schema FILE` accepts a standalone JSON Schema. Without either
   flag, init writes a directory-derived title, an example schema with optional
   tags, and editable guidance. See `means help agents init` for stdin input.
4. Refine the Markdown body of `issues/MEANS.md`: when to create/reuse documents,
   field meanings, workflow transitions, and outcomes to record. Means has no
   built-in issue or memory workflow.
5. Establish discovery through project instructions, a skill, or MCP below.
6. Verify that the chosen interface reaches the intended collection and exposes
   its schema/workflow. Run `means doctor --path ./issues`. Validate representative
   data too: an empty collection passing doctor does not test schema suitability.
   Keep disposable test documents out of the real collection.

## Make future agents find it

**CLI:** add guidance to the project instruction file agents actually load, or a
discoverable skill. Preserve existing instructions; follow the host's conventions.
Include the purpose, collection path, when to use it, and the workflow location:

> Track project work in `issues/` using means. Read `issues/MEANS.md`
> before use. Search existing issues before creating
> new ones; record progress and outcomes according to the workflow.
> Run `means help agents` for usage guidance. Resolve paths from the project root.

Adapt this to the purpose. `MEANS.md` alone is not automatically discovered by CLI
agents. Means does not load `AGENTS.md`.

**MCP:** configure the client's stdio server command:

    means mcp --path /absolute/path/to/issues

Initialize first; register each collection separately. Ensure the executable and
path work in the client's environment. Call `info` to confirm identity, schema,
and workflow. Clients must consume structured JSON results; `--json` is unnecessary.
Restart after code or startup-guidance changes. See `means help agents mcp`.

MCP supplies access; project instructions or a skill explain when to use it.
Neither a skill nor MCP is required if project instructions plus the CLI suffice.

## Use it correctly

- Read the configuration and guidance with `means info --json`, directly from
  `MEANS.md`, or through MCP `info`. Search before
  creating. Use `--json` for CLI results, `--with-body` for list/search bodies,
  and `--limit 0` for all results. Inspect diagnostics, not only matches.
- Documents have YAML frontmatter `{id, title, data}` and a Markdown body.
  Means generates the ID; supply the title and schema-valid data. Custom fields
  belong under `data`. Use stored IDs, never filenames.
- Create/update/delete accept `--input FILE` or `--input -` for stdin. Consult
  `means help agents <command>` for JSON shapes. Retrying a successful create duplicates it.
- Read with `means read <id> --path ./issues --json` before update/delete; pass its
  current `hash`. On conflict, reread and reconcile. Updates merge metadata:
  omitted keys stay, null deletes, arrays replace. A supplied body replaces it.
  Delete is permanent; prefer archival status when history matters.
- Use `schema set` to check and install schema changes, then doctor to find
  documents needing repair. Agents perform migrations; means does not fix data.
- Always select the intended `--path`; means does not search parent directories.
  Nested directories containing `MEANS.md` are separate collections. Input file paths are
  relative to the working directory, not the collection.

`means help agents <command>` and `means <command> --help --json` provide full
contracts and limitations. Ordinary `--help` is concise. `means help agents`
prints this guide and installs no integrations. Bare invocation, help, and version
do not read or create user config. Machine-local embedding
settings belong in `$XDG_CONFIG_HOME/means/config.yaml` (default
`~/.config/means/config.yaml`), never in `MEANS.md`. See `means help agents embed`.

## CLI reference

A collection is a directory containing MEANS.md with YAML configuration (title,
description, and a JSON Schema under schema) and Markdown workflow guidance. The files are authoritative and can be edited
directly. Means owns frontmatter.id (generated and immutable) and frontmatter.title
(required); application fields live under frontmatter.data. Filenames are storage
details. Read and update use the stored ID, even after file renames or moves.

Start with "means help agents init" for the complete schema-authoring workflow,
file and stdin input examples, validation rules, and response formats.
Then use "means help agents create" to store a validated document, "means list" to
discover IDs and titles, and "means show <id>" to view rendered Markdown.
Use "means read <id>" to retrieve the exact stored Markdown.
Use "means help agents mcp" to expose a collection to an MCP client over stdio.
Use "means help agents update" to patch a document using the hash from read --json.
Use "means help agents delete" to permanently remove a document using its content hash.
Use "means help agents schema set" to validate and install a replacement schema.
Use "means doctor" to check all documents
against the current schema, or "means search" to find documents by text/data.
Use "means help agents embed" for opt-in OpenAI embeddings, automatic maintenance,
and semantic search with "means search --semantic".

Machine configuration:
  On the first collection command, create $XDG_CONFIG_HOME/means/config.yaml (default
  ~/.config/means/config.yaml) with a commented embedding example. Existing files
  are never overwritten. Edit it manually to enable embeddings for every collection
  on this machine; credentials belong in environment variables. No setup command.
  Failure to create a missing config leaves embeddings disabled and reports a
  warning on successful commands. Existing invalid configs fail commands; help
  and version remain available. CLI reloads settings each invocation; restart MCP
  after editing settings. See "means help agents embed" for fields and migration.

Every command accepts --path and --json before or after the command name.
--path defaults to the current working directory; relative paths resolve there.
Means works with that exact directory and does not search parent directories.
--json returns two-space-indented JSON, including help and errors.
The mcp command always uses MCP message framing and structured JSON tool results;
--json only affects its CLI help and startup errors.
Successful responses go to stdout (exit 0); errors go to stderr (exit 1) as
{"error":"..."} in JSON mode. Doctor reports invalid documents on stdout and
exits 1. Schema set rejects incompatible documents with a diagnostic report on
stderr (exit 1), including an error field in JSON mode; see "means help agents schema set".
Parse the entire JSON response, not individual lines.
Use "means help COMMAND" or "means COMMAND --help" to discover each command.
