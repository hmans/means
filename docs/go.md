# Use means from Go

The `github.com/hmans/means/collection` package lets Go applications manage means
collections directly. The CLI, TUI, and MCP server use this same implementation.
The library and binary share one Go module and release version.

Add the dependency:

```sh
go get github.com/hmans/means/collection
```

Use a release that includes the public package. The required Go version is in
the module's `go.mod`. The API is pre-1.0 and can change in minor releases; pin
the module version in your application.

## Open or create a collection

`Load(path)` opens the exact directory supplied. It does not search parents or
create files. `Init(path, schema)` creates a collection and its `.gitignore` rule
for `.means/`. Pass JSON Schema bytes, or `nil` for the example schema (optional
`tags`). `InitConfig(path, content)` accepts a complete `MEANS.md` and preserves
its bytes. Both initializers validate before creating files and refuse to
overwrite an existing `MEANS.md`.

```go
import (
    "context"

    "github.com/hmans/means/collection"
)

c, err := collection.Load("./notes")
if err != nil {
    return err
}

ctx := context.Background()
doc, err := c.Create(ctx, collection.CreateOptions{
    Title: "First note",
    Body:  "Written directly from Go.",
})
if err != nil {
    return err
}
```

`CreateOptions.Data` holds application fields under `frontmatter.data`. Nil means
an empty object; supply fields required by your schema. Means generates the ID.
Creation is not idempotent: each successful call creates another document.
The library normalizes metadata to JSON-compatible values without changing the
input map. Use `encoding/json.Number` to supply exact large integers or decimals;
floating-point values have already lost any precision absent from that value.

`Document` contains the ID, frontmatter, exact body, content hash, and complete
Markdown source. IDs remain valid after file moves or renames. `Path()`,
`Schema()`, `App()`, and `Describe()` expose collection configuration. Returned
schema bytes and views are copies.

## Read and change documents

Read by ID before a change and use the returned hash:

```go
current, err := c.Read(ctx, doc.ID)
if err != nil {
    return err
}
body := "Replacement body."
updated, err := c.Update(ctx, current.ID, current.Hash, collection.Patch{
    Frontmatter: map[string]any{
        "data": map[string]any{"tags": []string{"reviewed"}},
    },
    Body: &body,
})
if err != nil {
    return err
}
_, err = c.Delete(ctx, updated.ID, updated.Hash)
```

`Patch.Frontmatter` merges title/data recursively. Null deletes nested keys;
arrays replace. A nil frontmatter or body leaves that part unchanged. A pointer
to an empty body clears it. IDs cannot be patched. `ReplaceMarkdown` accepts a
complete edited document and replaces metadata instead of merge-patching it,
preserving literal null values. It requires the original ID and content hash.

Updates and deletion reject observed content changes with `ErrConflict`.
Read again and reconcile the change before retrying. These checks do not lock
external editors out: a write can still race the final replacement or deletion.
Deletion permanently removes the document file.

## List, search, and validate

Use `List(ctx, ListOptions{...})` for filename-ordered documents and
`Search(ctx, SearchOptions{...})` for full-text, literal, semantic, or data-only
search. A zero limit returns all matches. Filters apply before the limit;
`ListOptions.Offset` skips valid matching documents. Set `WithBody` to include
bodies; list/search results omit content hashes, so read before changing a file.

Both return per-document `Diagnostics` separately from fatal errors and report
`HasMore` after filtering and limits. Malformed files and duplicate IDs are
diagnosed even outside a requested list page. Full-text search updates disposable
caches under `.means/`; it needs write access there.

Creation, validation, updates, and list/search filters accept JSON-compatible Go
values such as `int` and `[]string`. Public operations normalize them to the same
representation used by stored documents. `MatchesFilter` is a lower-level helper:
its inputs must already use `json.Number`, `[]any`, and `map[string]any`.

Creation and updates validate the means envelope and application data. Reads,
lists, and searches check the envelope but allow data that no longer satisfies
the schema. `ValidateData` checks an application object; `Validate` checks a
complete frontmatter envelope, including ID. `Doctor(ctx)` checks all documents
against the loaded schema. Its warnings about embedding caches do not make the
collection unhealthy.

Use `errors.Is` with `collection.ErrConflict`, `collection.ErrDuplicateID`,
`collection.ErrValidation`, and `fs.ErrNotExist`. Validation errors retain the
underlying schema error for `errors.As`. Invalid arguments, filesystem failures,
and embedding failures return errors rather than document diagnostics.

## Configuration snapshots

A loaded collection holds a snapshot of `MEANS.md`. Documents are scanned afresh
for each operation, but schemas, identity, instructions, and views are fixed until
you call `Load` again. Reapply host embedding settings and callbacks after reload.

`SetSchema(ctx, path, schema, force)` validates a proposal against existing
documents before installing it. If documents fail, it returns both
`ErrSchemaIncompatible` and a non-nil result with diagnostics. `force` permits
those document violations, but never invalid schemas or filesystem failures.
It can repair a missing or invalid schema in an otherwise valid configuration.
It does not migrate documents or update existing collection snapshots.

`SetInstructions(ctx, hash, instructions)` replaces the entire workflow body.
Use the complete configuration hash from `Describe()`. An empty string clears
the body. Frontmatter bytes and permissions are preserved. Reload afterward to
observe the new instructions through the collection snapshot.

## Host settings and cancellation

The library does not read or bootstrap user configuration, load `.env` files,
or print output. Embeddings are disabled by default. Opt in explicitly:

```go
c = c.WithEmbeddings(&collection.EmbeddingConfig{Provider: "openai"})
result, err := c.Embed(ctx, collection.EmbedOptions{})
```

`WithEmbeddings` copies the settings. They are validated and defaulted when used.
`EmbeddingConfig.WithDefaults()` validates and returns resolved settings upfront.
Model and dimension defaults are `text-embedding-3-small` and 512; OpenRouter
requires provider `openrouter` and the `openai/` model prefix. Credentials and
endpoint overrides come from the process environment, as described in the
[embedding guide](guide.md#opt-in-semantic-search). Configured embedding
maintenance can make creation and title/body updates fail if the API fails.
`EmbedOptions.Force` clears previous vectors before rebuilding; completed batches
remain if a later batch fails. Semantic search embeds only the query.

All document operations, schema installation, and instruction changes take a
non-nil `context.Context`. Cancellation is checked before work starts and during
embedding requests and cache lock waits. Synchronous filesystem work and schema
validation may finish before cancellation is observed. `Load`, `Init`, and
`InitConfig` do not take a context.

A loaded collection can be shared by concurrent callers. Each operation has its
own context. Methods that attach settings or progress callbacks return copies.
Callbacks supplied through `WithEmbeddingProgress` can run concurrently; protect
shared callback state. Do not change caller-owned input maps while an operation
uses them. File updates retain the optimistic concurrency limits described above.
