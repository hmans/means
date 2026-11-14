---
type: fix
---

Keep stored embeddings when a document is created, updated, appended to, or deleted
while the embedding provider, model, dimensions, or endpoint differ from the ones the
cache was built with. Previously one such write, for example under a temporary
`MEANS_EMBEDDING_BASE_URL`, emptied `.means/embeddings.jsonl` and every vector had to
be regenerated. These writes now leave the cache untouched and make no embedding
request; only `means embed` replaces vectors from another profile. Run `means embed`
afterwards to embed documents written in the meantime.
