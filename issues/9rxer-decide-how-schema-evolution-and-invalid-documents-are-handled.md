---
data:
  priority: normal
  status: done
  tags:
    - schema
    - design
  type: task
id: 9rxer
title: Decide how schema evolution and invalid documents are handled
---
Consumers can edit means.schema.json directly. Reads and lists parse existing documents without validating against the current schema; creation and updates validate resulting frontmatter. Decide how consumers discover and resolve invalid documents after schema changes or direct edits.

Acceptance criteria:
- Document the schema-evolution and repair workflow.
- Define how invalid documents are reported without preventing access to their contents.
- Coordinate the decision with doctor; automatic repair remains deferred.

Migrated from TODO.md.

Outcome:
- Added CLI schema set --input FILE|- with --force, backed by collection.SetSchema. Every proposal must compile as a supported JSON Schema.
- Validate current documents against the proposal using doctor semantics and collection boundaries. Reject diagnostics by default without changing the schema; force installs and retains diagnostics. No abstract old/new schema compatibility proof or automatic document migration.
- Preserve supplied bytes and permissions through temporary replacement. Reject observed schema edits; concurrent edits can still race the final replacement. Existing malformed schemas can be replaced, but symlinks, directories, and missing schemas are rejected.
- Added tests for rejection, forced changes, valid migrations, schema validation, malformed/duplicate documents, nested collections, empty collections, exact bytes, permissions, CLI input/output/errors/help, and invalid-schema recovery.
- Documented staged migrations and agent-led repairs after forced installation.

Validation: mise test passed for all packages; git diff --check passed; the issues collection passed doctor with 21 documents and no diagnostics.
