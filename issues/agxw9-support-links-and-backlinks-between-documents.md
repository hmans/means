---
data:
  priority: normal
  status: todo
  tags:
    - schema
    - links
    - doctor
  type: feature
id: agxw9
title: Validate document ID references with an x-means-ref schema keyword
---
Issue tracking and agent memory need relationships ("parent", "blocked by", "supersedes"). Links should stay application data, not a means primitive: the schema marks which data fields hold document IDs, and means verifies them.

## Design

A custom JSON Schema keyword, `x-means-ref`. JSON Schema cannot add types, and `format` describes syntax, not references. Draft 2020-12 allows unknown keywords and other validators ignore them, so collections stay portable. Register it as a custom vocabulary in `santhosh-tekuri/jsonschema/v6`.

```yaml
$defs:
  docId:
    type: string
    x-means-ref: true
properties:
  parent: { $ref: "#/$defs/docId" }
  blocked_by:
    type: array
    items: { $ref: "#/$defs/docId" }
```

Reuse through `$defs`/`$ref` gives type-like ergonomics.

## Checks

1. Shape (schema validation, every write): value must be a string with a valid means ID shape.
2. Existence (collection-wide): the ID must belong to a document in the same collection. `doctor` reports dangling references as diagnostics. Decide whether create/update also reject unknown IDs (extra scan per write); deletes can still leave dangling refs, so doctor remains authoritative.

## Acceptance criteria

- `x-means-ref: true` compiles in collection schemas, including via `$ref`; invalid keyword values (non-boolean) fail schema compilation.
- Shape violations fail writes like other schema errors.
- Doctor reports dangling references with document, field path, and missing ID.
- Documented in `means help agents` and docs/guide.md.

## Out of scope / later

- `[[id]]` references in Markdown bodies.
- Cross-collection references (the keyword can later accept an object, e.g. a target collection, without breaking `true`).
- Backlinks: expressible as a query once Expr filters (mmed0) land; TUI could make ref fields navigable.
