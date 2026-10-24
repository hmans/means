---
data:
  priority: normal
  status: done
  tags:
    - embeddings
    - review
  type: bug
id: 31adj
title: Release the embedding cache lock before query-only API requests
---
Adversarial review of 9rxzp. Severity P2. internal/collection/embeddings.go:515 searchSemantic defers closing the store until after embedding the query. Even with a fully current cache, a slow query API request holds the exclusive lock needed by create/update/delete. Reproduced by blocking the query endpoint: deletion of an unchanged document (which needs no API call) fails after 5 seconds with lock embeddings: context deadline exceeded. Release the lock once reconciliation and the in-memory snapshot are complete, before calling the query endpoint. Preserve snapshot consistency and serialize actual cache mutations. Add a concurrency regression test.

Outcome: Semantic search now releases the cache lock immediately after loading its independent vector snapshot, before scanning documents or embedding the query. It never saves that snapshot. A regression test deletes a document during the query HTTP request and checks that deletion succeeds and its cache change remains intact.

Validation: mise test and race tests for collection, CLI, and MCP packages pass. CLI/MCP help and the guide describe the new contract.
