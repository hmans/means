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
- [ ] Implement document updates and deletion.
- [x] Validate document creation against the collection schema.
- [ ] Add basic text search and frontmatter filtering.
- [ ] Add `doctor` to report documents that do not satisfy the schema.
- [x] Document and test the initial end-to-end workflow.

## Exercise real use cases

- [x] Add an issue-tracking example collection and AGENTS.md usage instructions.
- [ ] Integrate means with ChattoBot through the structured CLI interface.
- [ ] Use those integrations to decide what SDK or MCP support is needed.

## Later / under consideration

- A `fix` command for document repair.
- Persistent derived indexes, if needed.
- SDK and MCP interfaces that preserve the short-lived process model.

Embeddings and vector search are outside the initial scope.
