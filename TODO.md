# TODO

Active work has moved to the [issues](issues/) means collection. Follow its
[workflow](issues/MEANS.md) and use means to manage issues:

```sh
mise means list --path ./issues --limit 0 --json
```

The completed milestones below are historical. Open tasks and deferred ideas
were migrated to the collection; add new work there.

## Define the first version

- [x] Define collection boundaries and the schema file's name and format.
- [x] Choose the initial frontmatter types, constraints, and validation behavior.
- [x] Decide how documents are identified and how filenames relate to identity.
- [x] Scaffold Cobra with inherited `--path` and `--json` flags.
- [x] Establish an example `means.schema.json` through `init`; consumers read and edit it directly.
- [x] Accept agent-authored schemas from a file or stdin during initialization.
- [x] Document the schema-authoring workflow in text and JSON CLI help.

## Build the core

- [x] Initialize the Go module and a minimal CLI entry point.
- [x] Initialize collections and load and validate their JSON Schemas.
- [x] Parse Markdown documents with YAML frontmatter.
- [x] Implement document creation, reading, and limited metadata listing.
- [x] Implement patch updates with required content hashes and stale-write protection.
- [x] Establish means-owned IDs/titles, nested application data, and readable filenames.
- [x] Validate document creation against the collection schema.
- [x] Document and test the initial end-to-end workflow.

## Exercise real use cases

- [x] Add an issue-tracking example collection and AGENTS.md usage instructions.
- [x] Expose collection information, creation, reading, listing, and patch updates through stdio MCP.
- [x] Configure the example for Codex with MCP usage instructions.
- [x] Describe each MCP collection using schema annotations and optional MEANS.md.
- [x] Make the example's issue workflow available through MCP independently of AGENTS.md.
- [x] Add a plain-memory example collection alongside issues, with a separate MCP registration.

Embeddings and vector search are outside the initial scope.
