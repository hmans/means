# TODO

## Define the first version

- [x] Define collection boundaries and the schema file's name and format.
- [x] Choose the initial frontmatter types, constraints, and validation behavior.
- [x] Decide how documents are identified and how filenames relate to identity.
- [ ] Define the basic CLI commands and structured input, output, and errors.
- [x] Scaffold Cobra with inherited `--path` and `--json` flags.
- [x] Establish an example `schema.json` through `init`; consumers read and edit it directly.
- [x] Accept agent-authored schemas from a file or stdin during initialization.
- [x] Document the schema-authoring workflow in text and JSON CLI help.
- [ ] Decide how schema changes and directly edited invalid documents are handled.

## Build the core

- [x] Initialize the Go module and a minimal CLI entry point.
- [x] Initialize collections and load and validate their JSON Schemas.
- [x] Parse Markdown documents with YAML frontmatter.
- [x] Implement document creation, reading, and limited metadata listing.
- [x] Implement patch updates with required content hashes and stale-write protection.
- [x] Establish means-owned IDs/titles, nested application data, and readable filenames.
- [ ] Implement document deletion.
- [x] Validate document creation against the collection schema.
- [ ] Add basic text search and frontmatter filtering.
- [ ] Add `doctor` to report documents that do not satisfy the schema.
- [x] Document and test the initial end-to-end workflow.

## Exercise real use cases

- [x] Add an issue-tracking example collection and AGENTS.md usage instructions.
- [ ] Integrate means with ChattoBot through the structured CLI interface.
- [x] Expose collection information, creation, reading, listing, and patch updates through stdio MCP.
- [x] Configure the example for Codex with MCP usage instructions.
- [x] Describe each MCP collection using schema annotations and optional instructions.md.
- [x] Make the example's issue workflow available through MCP independently of AGENTS.md.
- [x] Add a plain-memory example collection alongside issues, with a separate MCP registration.
- [ ] Use integrations to decide what SDK support is needed.

## Later / under consideration

- A `fix` command for document repair.
- Persistent derived indexes, if needed.
- SDK integration; ordinary CLI operations remain short-lived.

Embeddings and vector search are outside the initial scope.
