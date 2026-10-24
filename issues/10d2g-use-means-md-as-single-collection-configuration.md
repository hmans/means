---
data:
  priority: normal
  status: done
  tags:
    - storage
    - schema
    - cli
    - mcp
  type: feature
id: 10d2g
title: Use MEANS.md as single collection configuration
---
Implement the agreed single-file configuration plan: YAML title, description, and JSON Schema in MEANS.md frontmatter; Markdown guidance body. Clean break from means.schema.json. Add init --input, preserve schema-only commands, migrate the issues collection, update documentation, and verify parsing, CLI, MCP, boundaries, and schema replacement with mise test and doctor.

Outcome:
- MEANS.md is the sole collection configuration and nested boundary marker. Identity, schema, and guidance load from one snapshot. Configuration validation rejects unknown fields and invalid types while preserving JSON number precision.
- init accepts complete MEANS.md through --input FILE|- or JSON Schema through --schema FILE|-, validates before writes, and returns the configuration path as config. Complete-file input preserves exact bytes.
- Schema replacement retains identity values, exact body bytes, and permissions. It rejects observed whole-file changes and permits invalid or missing schemas to be repaired when the remaining configuration is valid. Malformed YAML requires direct editing.
- Review follow-ups: schema input from init --schema, schema set, and MCP schema_set keeps the supplied key order and exact numbers; JSON escapes YAML rejects (such as \/) are normalized and duplicate keys are rejected. Frontmatter encoding quotes << keys and values and refuses to write output it cannot parse back. Collection.Schema returned by info is normalized after reload.
- CLI/MCP contracts, agent guidance, README, usage guide, and project instructions were updated. The issues collection was migrated; manual migration is documented with no legacy fallback.

Validation: mise test passed for all packages; git diff --check passed; doctor reported 31 healthy issue documents with no diagnostics. Added tests for complete-file CLI initialization, configuration validation and snapshots, exact numbers and references, schema preservation and conflict checks, MCP repair limits, and old-format rejection, plus key order (CLI and MCP), escape and merge-key round-trips, duplicate-key rejection, and JSON-in-YAML configuration. Updated nested-boundary and search reconciliation coverage.