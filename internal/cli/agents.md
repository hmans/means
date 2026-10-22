# Using means as an agent

means stores shared knowledge as Markdown files: issues, decisions, research, or
memory. Files are the source of truth. Use the CLI or optional stdio MCP tools.

**Agent setup is complete when the collection has a schema, a workflow, and a
discovery path for future agents. Creating files alone is not enough.**

## Set up a collection

1. Inspect existing collections and project instructions. Reuse a suitable
   collection before creating another.
2. Design `means.schema.json` for the purpose. Its title/description identify the
   collection; its JSON Schema validates document `data` only. Use a small set of
   useful fields, constraints, and descriptions. Put filterable attributes in
   `data` and explanations or evidence in the Markdown body.
3. Initialize with your schema:

       means init --path ./issues --schema ./issues.schema.json

   Without `--schema`, init provides only an example with optional tags.
   See `means init --help` for schema requirements and stdin input.
4. Write `issues/MEANS.md`: when to create/reuse documents, field meanings,
   workflow transitions, and outcomes to record. Means has no built-in issue or
   memory workflow. Init does not write this file.
5. Establish discovery through project instructions, a skill, or MCP below.
6. Verify that the chosen interface reaches the intended collection and exposes
   its schema/workflow. Run `means doctor --path ./issues`. Validate representative
   data too: an empty collection passing doctor does not test schema suitability.
   Keep disposable test documents out of the real collection.

## Make future agents find it

**CLI:** add guidance to the project instruction file agents actually load, or a
discoverable skill. Preserve existing instructions; follow the host's conventions.
Include the purpose, collection path, when to use it, and the workflow location:

> Track project work in `issues/` using means. Read `issues/MEANS.md` and
> `issues/means.schema.json` before use. Search existing issues before creating
> new ones; record progress and outcomes according to the workflow.
> Run `means agents` for usage guidance. Resolve paths from the project root.

Adapt this to the purpose. `MEANS.md` alone is not automatically discovered by CLI
agents. Means does not load `AGENTS.md`.

**MCP:** configure the client's stdio server command:

    means mcp --path /absolute/path/to/issues

Initialize first; register each collection separately. Ensure the executable and
path work in the client's environment. Call `info` to confirm identity, schema,
and workflow. Clients must consume structured JSON results; `--json` is unnecessary.
Restart after code or startup-guidance changes. See `means mcp --help`.

MCP supplies access; project instructions or a skill explain when to use it.
Neither a skill nor MCP is required if project instructions plus the CLI suffice.

## Use it correctly

- Read the schema and `MEANS.md`, directly or through MCP `info`. Search before
  creating. Use `--json` for CLI results, `--with-body` for list/search bodies,
  and `--limit 0` for all results. Inspect diagnostics, not only matches.
- Documents have YAML frontmatter `{id, title, data}` and a Markdown body.
  Means generates the ID; supply the title and schema-valid data. Custom fields
  belong under `data`. Use stored IDs, never filenames.
- Create/update/delete accept `--input FILE` or `--input -` for stdin. Consult
  the command's `--help` for JSON shapes. Retrying a successful create duplicates it.
- Read with `means read <id> --path ./issues --json` before update/delete; pass its
  current `hash`. On conflict, reread and reconcile. Updates merge metadata:
  omitted keys stay, null deletes, arrays replace. A supplied body replaces it.
  Delete is permanent; prefer archival status when history matters.
- Use `schema set` to check and install schema changes, then doctor to find
  documents needing repair. Agents perform migrations; means does not fix data.
- Always select the intended `--path`; means does not search parent directories.
  Nested collections with their own schema are separate. Input file paths are
  relative to the working directory, not the collection.

`means <command> --help` provides full contracts and limitations. `means agents`
only prints this guide; it creates no files and installs no integrations.
