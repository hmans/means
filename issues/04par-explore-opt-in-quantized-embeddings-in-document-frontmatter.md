---
data:
  priority: normal
  status: canceled
  tags:
    - embeddings
    - storage
  type: feature
id: 04par
title: Explore opt-in quantized embeddings in document frontmatter
---
Selected for an experimental implementation on 2026-10-26.

Goal: Make embeddings portable with Markdown documents so a fresh checkout can reuse vectors without regenerating them through the API. Storage in documents must be opt-in; the existing local JSONL storage remains the default.

Proposed direction (subject to evaluation):
- Add an explicit collection storage setting, for example embeddings.storage: frontmatter.
- Store vectors in a means-owned top-level embedding field, separate from application data.
- Evaluate int8 quantization with base64 encoding. A 512-dimensional int8 payload would take about 684 base64 characters, excluding metadata.
- Include format/encoding version, model identity, dimensions, quantization parameters, and a hash of title/body input so stale or incompatible vectors can be rejected. Define compatibility across providers/endpoints without persisting credentials.
- Maintain stored vectors on create/update and CLI means embed; searches remain read-only and embed only the query. Keep refresh CLI-only.

Acceptance criteria and design questions:
- Compare float and quantized retrieval rankings and scores on representative queries before selecting a format; measure size and quality impact, including the default similarity threshold.
- Define strict validation and behavior for missing, stale, malformed, or incompatible stored vectors, with useful doctor reporting.
- Define optimistic concurrency for means embed, since it would now modify Markdown and invalidate previously read content hashes. Preserve document identity, application data, and exact body bytes; detect observed concurrent edits and never overwrite them blindly.
- Define the role of the local cache and how a fresh checkout reuses valid frontmatter vectors without document embedding API calls.
- Specify safe behavior when enabling/disabling or switching storage modes. Automatic bulk conversion can remain deferred until the format and write behavior are established.
- Add tests and document opt-in configuration, portability, failure recovery, and file/hash changes.

Related foundation: 9rxzp (opt-in embeddings and semantic search).

## Outcome: canceled

We built and tried the frontmatter quantization prototype, then decided to remove
it. Keeping Means' configuration and document format small is more valuable than
sharing embeddings between systems: embeddings are cheap enough to regenerate
on each system configured to use them.

The prototype implementation, storage setting, documentation, tests, and unreleased
feature note were removed. Generated embedding fields were stripped from the
issues while preserving their metadata and bodies. Means retains the existing
opt-in local float cache and semantic search. No portable storage, quantization,
or migration API is being shipped. Real-model quantization evaluation is no
longer planned.
