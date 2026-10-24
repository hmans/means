---
data:
  priority: normal
  status: done
  tags:
    - embeddings
    - review
  type: bug
id: zfs3y
title: Avoid embedding excluded documents during filtered semantic search
---
Adversarial review of 9rxzp. Severity P2. internal/collection/embeddings.go:524 reconciles the entire collection before applying opts.Filter at line 531. Reproduced with a current wanted document and an excluded document missing a vector: a filtered search sends the excluded document to the provider and fails when that embedding request returns HTTP 400. Thus an over-limit or otherwise unembeddable excluded document can block searches for healthy eligible documents, and narrow searches unnecessarily pay for unrelated refresh work. Apply eligibility filtering before generating vectors for semantic queries while preserving collection-wide malformed/duplicate diagnostics and safe pruning. Explicit embed should still refresh the entire collection. Add a regression test for failure on an excluded document.

Outcome: Per the clarified contract, semantic search never generates document embeddings, even for matching documents. It embeds only the query and uses stored vectors matching current title/body hashes and configuration. Missing or stale vectors are skipped; deleted, malformed, duplicate, and nested documents are excluded without cache pruning. Only create/update and explicit embed generate document vectors; delete removes its existing vector. Regression tests cover filtered and unfiltered searches with missing vectors, unchanged cache bytes, direct edits/deletions, corrupt/missing/incompatible caches, and explicit refresh.

Validation: mise test and race tests for collection, CLI, and MCP packages pass. CLI/MCP help and the guide describe the new contract.
