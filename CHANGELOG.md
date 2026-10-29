# means Changelog

## 0.4.2

### Fixes

- Move the TUI Clear button into the top toolbar and remove the empty status row during text search. ([#35](https://github.com/hmans/means/pull/35))

## 0.4.1

### Fixes

- Remove macOS quarantine during Homebrew installation so means can run without a manual workaround. ([#33](https://github.com/hmans/means/pull/33))

## 0.4.0

### Features

- ⚠️ **BREAKING CHANGE:** Use one command help reference for people and agents, with the same content in text and JSON. Bare means shows a banner, pointers to means help and means help mcp, and grouped commands. Legacy audience-specific help routes remain hidden aliases; their JSON uses the standard command/description/usage response instead of guide. ([#20](https://github.com/hmans/means/pull/20))
- ⚠️ **BREAKING CHANGE:** Move embedding settings from MEANS.md to the machine-local Means user config.
  On first use, create a commented, disabled config at
  $XDG_CONFIG_HOME/means/config.yaml (default ~/.config/means/config.yaml) without
  overwriting existing files. Settings apply to all collections; CLI reloads them
  per invocation and MCP loads them at startup. Move legacy collection embedding
  blocks into the user config and remove them from MEANS.md. ([#19](https://github.com/hmans/means/pull/19))
- Install means with Homebrew using `brew install --cask hmans/tap/means`. ([#29](https://github.com/hmans/means/pull/29))
- Add `means help agents` with an embedded, version-matched guide to collection
  structure, MEANS.md authoring, schemas, workflows, and agent discovery. The guide
  works without a collection or user configuration and supports text and JSON help. ([#26](https://github.com/hmans/means/pull/26))
- Add timezone-aware datetime range filters to CLI and MCP search, including exclusive and inclusive before/after comparisons in every search mode. ([#17](https://github.com/hmans/means/pull/17))
- Use `:` in the TUI to open a command pane with help, visible errors, and Tab
  completion for commands and view names. Quit with `:quit`, `:q`, or Ctrl+C;
  bare `q` no longer exits accidentally. Use `:home` or Esc while browsing to
  leave a view without clearing search and filters, and `:clear` or the `x`
  keyboard shortcut to clear them without leaving the view. Active views now
  use their name as the document pane title, without an extra heading row. ([#32](https://github.com/hmans/means/pull/32))
- Browse collections with `means tui`: Markdown previews, live text search,
  metadata filters, and document diagnostics. Use the keyboard or mouse, resize
  the panes, and filter array fields by individual items with document counts.
  External file and schema changes refresh the browser automatically while
  preserving the current search, selection, and scroll position. ([#21](https://github.com/hmans/means/pull/21))
- Load optional collection and working-directory .env files for CLI and MCP startup, preserving existing environment variables. ([8031312](https://github.com/hmans/means/commit/8031312d3624d20588599c2feb6ca1c923bd94a0))
- Add collection info, editor-based document updates, title/body creation, direct delete hashes, raw viewing, list filters, grouped commands, and focused command help. Informational commands no longer read or create user configuration; existing structured document inputs remain supported. ([#20](https://github.com/hmans/means/pull/20))
- Define experimental TUI views in `MEANS.md` under `app.views`, with Expr filters
  and configurable columns. Switch views with a searchable picker; edits reload
  automatically, and an invalid view does not prevent other views from working. ([#23](https://github.com/hmans/means/pull/23))

## 0.3.0

### Features

- Extend means doctor with offline warnings for missing or stale document embeddings, orphaned vectors, and corrupt or incompatible embedding caches. Reports include a means embed repair hint and preserve all files. Embedding findings appear as WARNING in text and in a separate JSON warnings array; warnings alone exit 0. Document validation errors and failures to complete the check still exit 1.

  Style text reports with a colored status summary, grouped errors and warnings with counts, and readable paths and messages. Disable colors for pipes and NO_COLOR; preserve JSON output.

  ([#12](https://github.com/hmans/means/pull/12))

## 0.2.0

### Features

- Add opt-in semantic search through the CLI and MCP, with OpenAI embeddings accessed directly or through OpenRouter. Configure the provider, model, and dimensions per collection; defaults use text-embedding-3-small and 512 dimensions. Provider selection supplies the default endpoint and credential variable, with environment overrides available.

  When embeddings are enabled, init writes omitted model and dimensions explicitly into MEANS.md. Explicit values and Markdown body bytes are preserved; frontmatter may be reformatted. CLI JSON mode suppresses embedding progress so errors remain valid JSON.

  Embed each document's title and body together and store vectors in a local JSONL cache. Create/update maintain vectors automatically; delete removes them without an API call. Existing full-text and literal search remain offline and are still available without embeddings.

  Semantic search embeds only the query and ranks current stored vectors by cosine similarity. It skips missing or stale vectors without modifying the cache and reads the cache without waiting for the cache lock. The minimum score defaults to 0.3, configurable through CLI --min-score or MCP min_score from -1 to 1. Apply the threshold before result limits, display scores in semantic search tables, and omit search progress messages.

  Use CLI means embed to backfill or refresh vectors after direct file edits, or means embed --force to clear stored vectors and regenerate every valid document. Refreshes process batches of up to 32 documents, checkpoint each completed batch, and report progress on stderr. Failed runs retain completed batches; ordinary embed resumes from that progress. Refresh and forced rebuilds are CLI-only, with no MCP embed tool.

  ([#10](https://github.com/hmans/means/pull/10))

## 0.1.1

### Fixes

- Include Sweet Release notes in GitHub release drafts. ([#5](https://github.com/hmans/means/pull/5))

## 0.1.0

### Features

- Introduce means, a single Go binary for agentic memory stored as Markdown
  documents with typed YAML frontmatter and collection-local JSON Schemas.

  - Create, read, update, and delete documents with stable IDs and hash-checked writes.
  - Search collections with ranked full-text queries, literal matches, and data filters.
  - Validate schemas and documents, and inspect collection diagnostics.
  - Use structured CLI input and output or a collection-bound stdio MCP server.
  - Read agent onboarding guidance with `means agents`.

  ([#3](https://github.com/hmans/means/pull/3))
