# Using means

This guide covers the file format, detailed workflows, and implementation limits.
`means <command> --help` and `means help <command>` show the same command help,
including nested commands such as `schema set`. Adding `--json` returns the same
description and usage in structured form; command syntax has one shared reference.

Run `means` for help pointers and a command list grouped into documents,
collection management, and help/integrations. `means help` and `means --help` show
the same short introduction. `means help init` covers setup; `means help mcp`
covers MCP integration. Successful initialization points to `means help create`.

`means help agents` serves a concise collection authoring guide embedded in the
installed binary. It covers collection structure, `MEANS.md`, schema and workflow
design, and agent discovery, without requiring a collection or user configuration.
Its JSON uses `command`, `description`, and `usage`, like other help; `description`
contains the same guide as text output. `means agents` is a hidden alias for this
guide. Both agents routes still accept a command to show its canonical help.
The hidden `means help humans` alias shows the overview.

## Collections

A collection is one directory with a `MEANS.md` configuration file:

```yaml
---
title: Project memory
description: Decisions and facts for this project.
schema:
  type: object
  properties:
    tags:
      type: array
      items:
        type: string
  additionalProperties: false
---
# Working with project memory

Search for existing knowledge before creating a document.
```

The schema validates only document `frontmatter.data`; means owns each document's
`id` and `title`. It remains JSON Schema, expressed as YAML. Draft 2020-12 is the
default. Internal references work; external files and remote schemas do not.

The required `schema` field accepts an object or boolean. Optional `title` and
`description` must be strings. Optional `app.views` defines experimental TUI views
(see below). Embedding settings belong in the machine-local
user config, described below; unknown collection configuration fields are rejected.
Missing or blank title falls back to the directory name; description defaults to
empty. Schema annotations do not define collection identity. The Markdown body
provides self-description and workflow and may be empty. Configuration uses the
same JSON-compatible YAML subset as documents: string keys, exact numbers, no
aliases, merge keys, custom tags, or non-finite numbers.

Initialize with a starter, complete configuration, or JSON Schema:

```sh
mise means init --path ./memory
mise means init --path ./memory --input ./template/MEANS.md --json
mise means init --path ./memory --input - --json < ./template/MEANS.md
mise means init --path ./memory --schema ./memory.schema.json --json
mise means init --path ./memory --schema - --json < ./memory.schema.json
```

The starter includes a directory-derived title, editable guidance, and a schema
accepting optional `tags` while rejecting other application fields.
`--input` preserves a complete file's bytes. `--schema` accepts JSON and writes YAML.
These flags are mutually exclusive. Init JSON returns `path`, `config` (the absolute
MEANS.md path), and `next_steps`. `init` validates configuration and schema before
creating files and never overwrites an existing `MEANS.md`. It also creates `.gitignore` with `.means/`, or
appends that rule while preserving existing content and avoiding duplicates.
The search cache is created only when first used. Use `schema set` to validate a replacement, or edit
the file directly to evolve a collection. Means
loads the selected directory exactly; it does not search parent directories.
Schema edits take effect on the next command or MCP call. Run `doctor` after a
schema change to find existing documents that need attention.

The root `MEANS.md` must be a regular UTF-8 file, not a symlink. It is configuration,
not a document, and is excluded from lists and reads. The server does not load `AGENTS.md`.

Run `means info --path ./memory` to inspect collection identity, schema, and
workflow. `info --json` returns `path`, `title`, `description`, `schema`, and
`instructions`. Instructions contain the exact Markdown body of `MEANS.md`.
Info requires a valid collection configuration and schema.

For agent use, put collection purpose, path, and when-to-use guidance in project
instructions or a discoverable skill, or register the collection with an MCP client.
Write schema-specific rules and workflow in MEANS.md. Creating that file alone does
not make a CLI-based agent discover it; means does not load project instructions.
Verify the intended collection with `info` and check representative data with `doctor`.

### Changing the schema

Use `schema set` to check a proposed schema against existing documents before
installing it:

```sh
mise means schema set --path ./memory --input new-schema.json --json
mise means schema set --path ./memory --input - --force --json < new-schema.json
mise means doctor --path ./memory --json
```

Input is the JSON Schema itself. Means parses and compiles it with the same
rules as `init`; invalid or unsupported schemas always fail. The command requires
an existing regular `MEANS.md` file. It can replace an invalid or missing schema
when the YAML and other configuration fields are valid. Malformed YAML or invalid
identity fields require direct editing.
It checks visible documents against the proposal, excluding nested collections.
This tests existing data, not abstract compatibility with the old schema or
future callers. A stricter schema can pass on an empty collection.

By default, data violations, malformed envelopes, and duplicate IDs prevent
installation. The command exits 1 and reports IDs, paths, and errors on stderr;
the schema stays unchanged. `--force` allows these document diagnostics and
installs the schema, returning exit 0 with the diagnostics on stdout. It never
bypasses schema validation or filesystem errors. JSON results include `installed`,
`healthy`, `checked`, and `diagnostics`; rejected compatibility checks also have
an `error` field. Other errors use the usual `{"error":"..."}` format.

Only the embedded schema changes. It is serialized as YAML with exact numbers.
Other configuration values, exact Markdown body bytes, and file permission bits
are preserved. Frontmatter formatting may change. Installation uses a same-directory
temporary replacement. There are no locks: concurrent document edits can invalidate
the check, and configuration edits can race replacement. Observed changes to any
part of MEANS.md abort installation.

After a forced change, the agent or user migrates documents: read each affected
ID, patch its data using the returned hash, and run `doctor` again. Reads and
searches remain available for schema-invalid data; each update must leave the
whole document valid. For staged migrations, first accept both old and new
forms, migrate documents, then tighten the schema. Means does not rewrite data
automatically. MCP calls load the new schema; restart MCP to refresh advertised
collection metadata. MCP provides the same operation through `schema_set`:

```json
{"schema":{"type":"object","required":["status"]},"force":false}
```

Pass a schema object or boolean, not a JSON string. `force` defaults to false.
Rejected compatibility checks return a tool error with structured diagnostics;
forced installations return success with those diagnostics. Error responses
retain explanatory text. Schema numbers retain
their precision when stored as YAML. A running server can repair an invalid or
missing embedded schema in otherwise valid configuration; malformed YAML needs
a direct edit. Starting a server requires valid configuration.

### Subdirectories and collection boundaries

Use the filename `MEANS.md` as written. Means uses ordinary filesystem lookup;
case sensitivity depends on the filesystem. It does not search for other case
variants. Neither `schema.json` nor the former `means.schema.json` marks a collection.

Ordinary subdirectories belong to the selected collection. A subdirectory with
its own `MEANS.md` starts a separate collection: all document operations stop
at that directory, including list, search, doctor, ID lookup, updates, deletion, and ID
allocation. The presence of the configuration path establishes the boundary, even if
the configuration is invalid, a directory, or a broken symlink.

For example, with `foo/MEANS.md` and `foo/bar/MEANS.md`, `--path foo`
excludes everything under `foo/bar`. Use `--path foo/bar` to work with that
collection. IDs are unique within each collection; parent and child collections
can reuse an ID. Schemas and `MEANS.md` guidance do not inherit from parents.

Adding or removing a nested `MEANS.md` changes which collection owns that
subtree on the next operation. Moving a document across a collection boundary
changes the schema that applies to it and can introduce an ID collision in the
destination. Moving it within one collection preserves its identity.

## Documents

For a document with empty application data, supply a title and optional body file:

```sh
means create --path ./memory --title "Deployment notes" --body-file notes.md
printf 'Notes\n' | means create --path ./memory --title "Notes" --body-file -
```

The schema must accept `{}` as application data. The body defaults to empty.
To supply application fields, pass a JSON envelope from a file or stdin instead;
`--input` cannot be combined with `--title` or `--body-file`:

```sh
mise means create --path ./memory --input - --json <<'JSON'
{
  "frontmatter": {
    "title": "Use Go for means",
    "data": {"tags": ["decision"]}
  },
  "body": "We chose Go for single-binary distribution.\n"
}
JSON
```

The stored file is ordinary Markdown:

```markdown
---
id: k7m4p
title: Use Go for means
data:
  tags:
    - decision
---
We chose Go for single-binary distribution.
```

Means generates a random five-character ID and writes an
`<id>-<title-slug>.md` filename. Clients use the stored ID, not the filename.
Renames and moves into visible subdirectories preserve identity; title updates
leave filenames alone. Discovery ignores hidden paths and symlinks. Direct edits
can introduce duplicate IDs; reads, updates, and deletions reject ambiguous IDs, while
lists report each affected file. Creation checks for collisions and reserves a
candidate ID temporarily so concurrent means creators cannot allocate it twice.

On creation, `title` is required, omitted `data` defaults to `{}`, and omitted
`body` defaults to an empty string. Explicit `null` is invalid for either.
Application fields belong under `data`, including fields named `id` or `title`.
Creation and updates validate the data against the current schema. Reads and
lists check the means envelope but allow schema-invalid application data to be
retrieved after schema changes.

Creation is not idempotent: retrying a successful call creates another document.

## Interactive browser

Run `means tui --path ./issues` to browse a collection in your terminal. The browser
is read-only and shows all valid documents in filename order. Single-row Search,
Filter, and optional Views buttons sit above the search field and framed document and
preview panes. Idle buttons have a subdued fill; active controls use the accent
color. Labels precede dimmed keyboard shortcuts. Clear appears in the top button
row when a search or data filter is active. Search alone adds no status row.
View, Diagnostics, Back, and Quit
are clickable footer controls.
The selected row is highlighted, the focused pane has an accent border, and pane
footers show the visible range. Wide terminals show both panes side by side;
below 90 columns, the View button or Tab switches panes. The preview includes
syntax-highlighted JSON application data below a separator after the body, with
the ID at the upper right beside the wrapping title.
Body text has no extra outer margin. Wrapped list items currently follow Glamour's
default alignment. Smaller terminals use compact toolbar labels;
below 32 columns or 12 rows, resize the terminal to browse.

In terminals that support mouse reporting, click toolbar buttons or the search
field, click a document to select it, and click the preview to focus it. Hovered
documents have a subtle gray background; the selected document has a blue background.
The empty data-filter row is hidden. The wheel
scrolls the pane under the pointer without changing the selected document or
keyboard focus. Drag the gap or adjacent borders between the document list and
preview to resize the panes. Each pane stays at least 32 columns wide; the chosen
list width is kept for the session, including after a terminal resize.
Click a filter field and then a value to apply it; the first value
removes that field's filter. The wheel also scrolls filter choices and diagnostics.
Clicking a document or another toolbar action while searching keeps the live query;
Back restores the previous query. Browsing actions remain available from the keyboard.

| Key | Action |
| --- | --- |
| Up/Down or j/k | Select a document, or scroll the focused preview |
| Tab or Enter | Switch between list selection and preview scrolling |
| Page Up/Page Down, Home/End | Move within the list or preview |
| / | Filter title/body text immediately while typing; Enter keeps the search |
| f | Choose a top-level data field, then a schema or stored value |
| v | Choose a collection-defined view (type to find, Enter to select) |
| x | Clear all data filters and the text search |
| r | Reload documents and collection configuration from disk |
| d | Open or close diagnostics; Up/Down scrolls them |
| Esc | Cancel the current input or picker, return to the list, or leave the active view when browsing documents |
| : | Open the command bar outside search and pickers |
| :q or :quit, then Enter | Quit; bare q no longer quits |
| Ctrl+C | Quit immediately |

The experimental command pane temporarily replaces the search field and takes
keyboard focus; document panes keep their state with muted borders. It uses Cobra
for command dispatch and Bubbles for input.
Commands are `quit` (alias `q`), `views` (alias `view`), `filters` (alias `filter`),
`clear`, `home`, `diagnostics`, and `help`. `views` without an argument opens the
picker; `views "Open work"` selects a named view. `views "All documents"` resets it.
`home` leaves the current view while keeping search and data filters.
`clear` clears search and data filters while keeping the current view.
`filters` opens the existing field/value picker; filter arguments are not yet supported.
Dim text previews a completion. Tab accepts a match and repeated Tab cycles matches;
Shift+Tab cycles backwards. Completion is available at the end of the input;
view-name completion inserts quotes and a `--` separator when needed.
Enter executes the actual input, while Esc or Backspace on empty input dismisses
the pane. Errors appear below
the input with a red border and clear when the input changes. Long errors are
limited to the space available in the pane (at most three lines).
Input supports quoted arguments, but does not run shell expansions or pipelines.
Collection-defined commands and scripting are not part of this prototype.

The field picker includes the schema's direct top-level `properties`, even in an
empty collection, plus fields found in stored documents. Property `title` and
`description` annotations supply labels and guidance. The value picker shows
`enum` choices in schema order, or a `const` value, followed by additional stored
values in sorted order. Additional values are marked `(stored)` when the schema
supplies choices. It uses all valid documents, including those outside the current
results. Data filters use the same loaded snapshot as choices and counts.
Filesystem notifications and Refresh load external changes. Ordinary string labels appear
without quotes. JSON notation is retained
for other types and ambiguous strings such as `"true"`, `"42"`, or `"null"`, as
well as empty strings and strings with significant whitespace. Select
the first value option to remove that field's filter. Filters on different fields
combine with AND. Scalar filters use the same matching rules as `list` and
`search`, including partial object matches and distinguishing missing fields from null.
For arrays, the picker offers deduplicated individual items rather than complete
combinations. Selecting `cli` for `tags` matches any document whose tags contain
that item, regardless of order or other tags. The summary shows `tags contains cli`.
String items display a document count, such as `cli (15)`, across all valid documents
in the loaded collection, independent of active filters. Repeated strings within
one array count once. Unused schema choices show zero, and refresh updates the counts.
Type-ahead searches item labels, not their displayed counts.
Select one item per field; selecting another replaces it. Empty arrays and missing
fields do not match. Item matching uses exact JSON equality, including types and
complete objects or nested arrays. If a field also stores scalar values, those
choices are marked `(value)` and use ordinary field equality. CLI and MCP exact
array filters retain their existing whole-array behavior.
Existing data that no longer matches the schema remains readable and filterable.
Both picker steps support immediate, case-insensitive type-ahead over their labels:
type a field name or title, then type a value to narrow its choices. Paste works too.
Use arrow keys or Page Up/Page Down to navigate, Enter or a click to choose,
Backspace to edit, and Ctrl+U to clear the picker query. Letters including `j`, `k`,
and `q` are text in the picker; Ctrl+C or the Quit footer control still exits.
Each new picker step starts with an empty query. Esc cancels the picker without
applying its highlighted choice. Picker searches do not change the document text
search or data filters until a value is chosen. Refresh preserves the picker query.
Refresh reloads schema hints as well as documents. This first version reads direct
property definitions; it does not derive choices through `$ref`, schema composition,
conditions, numeric ranges, or string patterns. Direct `items.enum` and `items.const`
supply array item choices even before those items are stored. Whole-array enum/const
values also supply their individual items. Other open-ended fields use observed values.

Text search is a case-insensitive literal match against the loaded title and body.
Typing, pasting, Backspace, and Ctrl+U (clear input) update the list immediately,
within the active data filters. Search does not rescan files per keystroke; `r`
refreshes the snapshot from disk. Enter keeps the search and returns to navigation;
Esc restores the search that was active when `/` was pressed.

Malformed documents and duplicate IDs are excluded and reported in diagnostics,
even when filters yield no results. Normal collection boundaries apply. The
browser loads document bodies into memory without a limit; it is intended for
collections that fit in memory. While the browser is open, a filesystem watcher
refreshes the collection after 200 ms without further change events. Added,
changed, removed, and renamed files, schema changes, diagnostics, and nested
collection boundaries update automatically. Write bursts share one refresh;
changes during a scan queue a follow-up. Directories are watched to support atomic
editor saves. Hidden paths and nested collection contents are ignored. Watches
are closed when the TUI exits; no daemon is installed. Press `r` to check immediately.

Background checks do not block typing. Search input, filters, picker input and
selection, and document selection are retained where possible. Scroll positions
are retained and clamped to the available content. Unchanged snapshots leave the
preview alone. A failed collection load keeps the last usable snapshot, shows the
error, and retries on the next filesystem change or manual refresh. Watcher errors
are shown; filesystems without notification support may require manual refresh.
Individual malformed files follow the usual
diagnostics rules and are excluded until repaired. There is no editing UI or
action execution.

Both input and output must be terminals. `NO_COLOR` disables content styling;
terminal controls are still required for navigation. `tui --json` returns an error;
JSON help remains available through `tui --help --json`. Use `list` or `search` for
pipes and structured output. Browsing stays offline and does not build a search
index or request embeddings. Collection startup follows the normal user-config
loading and bootstrap rules.

### Experimental collection views

Add an optional `app.views` block to the frontmatter of `MEANS.md` to define TUI
views without writing Go. Keep the existing `schema` and workflow body in that
same file. Collection commands and MCP validate the app structure; only the TUI
compiles and evaluates filters. This prototype format may change as the app model evolves.

```yaml
app:
  views:
    - title: Open bugs
      filter: data.type == "bug" and data.status in ["todo", "in_progress"]
      columns: [title, data.priority]

    - title: CLI work
      filter: '"cli" in (data.tags ?? [])'
      columns: [title, data.status]

    - title: Recently created
      filter: data.created_at != nil and date(data.created_at) > now() - duration("168h")
      columns: [title, data.created_at]
```

Click **Views** or press `v`, type a view name, and press Enter or click to select.
The dropdown opens below the toolbar button while documents remain visible.
The active view has a check mark. Arrow keys and the mouse wheel navigate the
menu; Escape or clicking outside dismisses it without changing the view.
The browser starts in **All documents**; select that entry to leave a custom view.
Search and metadata filters combine with the view. Clear removes those extra
filters, keeping the view. The active view name is the document pane title.
Columns appear in their configured order, with extra space for titles; narrow
columns truncate their displayed text. The preview still shows the full document.
The issues collection includes example views in `issues/MEANS.md` to edit and try.

Filters use [Expr](https://expr-lang.org/docs/language-definition) and must return
a boolean. Available variables are `id`, `title`, `body`, and `data`; missing data
keys evaluate to `nil`. Guard optional dates as shown above and use `?? []` for
optional arrays. `now()` returns one shared instant per view evaluation. Relative
dates are reevaluated when the view is selected, filters change, or the collection
refreshes; time passing alone does not trigger a refresh. Date strings must be
accepted by Expr's `date()` function. No shell commands or document writes are exposed.

Omitted filters match all documents. Omitted columns default to `[title]`.
Columns accept `id`, `title`, or `data.FIELD` for a direct data key, with missing
or null values shown as a dash. View names must be unique; **All documents** is
reserved. The prototype accepts up to 32 views and six columns per view, limits
the JSON-encoded `app` object to 64 KiB, and rejects unknown configuration keys.

Config edits reload through the filesystem watcher. Invalid YAML or app structure
shows an error and retains the previous snapshot and app; fix the file to recover.
Invalid filter expressions disable only their own views, which are marked invalid
in the dropdown and explained in Diagnostics. All documents and valid views remain
usable. If the selected view becomes invalid, the browser returns to All documents.
Compiled expressions are reused until app configuration changes.

View evaluation runs in the background while navigation and typing remain available.
The previous results remain visible with an Updating view indicator until evaluation
finishes. Results superseded by another view, metadata filter, or snapshot are
discarded; superseded work stops between documents. The latest text search is
applied when results arrive. Removing `app` or the selected view returns to All documents. Runtime errors,
such as malformed dates, exclude the affected documents from that view and appear
in Diagnostics. Expr receives integers as signed 64-bit values and fractional or
exponent-form numbers as float64; oversized integer literals in document data are
reported as errors. Existing metadata filters retain their exact JSON semantics.
Schema replacement preserves `app` and the workflow body. If you used the earlier
`MEANS.app.yaml` prototype, move its `views` block under `app` in `MEANS.md` and
remove the separate file; it is no longer read.

## Finding and updating documents

```sh
mise means list --path ./memory --limit 0 --json
mise means search decision --path ./memory --json
mise means search --path ./memory --filter 'tags=["decision"]' --json
mise means read <id> --path ./memory --json
mise means show <id> --path ./memory
```

Without `--json`, `list` and `search` show colored tables of ID and title, followed
by diagnostics and limit notices. Tables wrap to the terminal width (80 columns
when piped). Piped output and `NO_COLOR` disable terminal styling. Use `--json`
for complete metadata and search excerpts.
Add `--with-body` to `list --json` or `search --json` to include each document's
Markdown body. Empty bodies appear as `"body": ""`; diagnostic entries have no
body. The flag requires `--json` and does not add hashes or change limits or order.

`show <id>` prints the title, ID, and Markdown-rendered body, including styled
headings, accented bullet markers, and code blocks. Prose wraps to at most 80
columns, shrinking to fit narrower terminals. `show --json` returns the same document and
content hash as `read --json`. Use `show --raw` for the exact source,
including YAML frontmatter; `--raw` and `--json` are exclusive. The hidden `read`
compatibility command keeps its exact-source default and existing JSON behavior.
Neither command enforces the current application data
schema, so documents remain readable after schema changes.

`list --json` returns `documents`, `diagnostics`, and `has_more`, like search.
Documents contain IDs and frontmatter; bodies require `--with-body`, and hashes
are omitted. Its default limit is 20 valid documents; `--limit 0` returns all.
List accepts the same exact-data and datetime filters as search, combined with AND
before the limit. For example, `means list --filter status=todo` selects open work.
List uses filename order and performs no text indexing or embedding API calls.
Malformed documents and every duplicate occurrence are excluded from documents
and reported in diagnostics with ID, path, and error, regardless of the limit.
`search` uses full-text ranking by default: all
query terms must occur, in any order, across the title and body. Title matches
receive a boost; equal scores sort by stored ID. Queries use lowercase Unicode
tokens with no stemming, stop-word removal, or query-language operators.
For example, `search "deployment timeout"` finds "timeout during deployment".
A nonempty query with no tokens matches nothing. An omitted query or empty string
uses data-only search; without filters, all valid documents are eligible.

Use `search "deployment timeout" --literal` for the original case-insensitive
substring search. Literal searches and searches with no query use filename order.
Repeated `--filter KEY=VALUE` flags combine with AND and apply before the result
limit. JSON numbers retain their precision, objects match recursively, and arrays
must match in full. Search JSON returns `documents`, `diagnostics`, and `has_more`.
Each document has `id` and `frontmatter`; text matches also have a `match` object
with `field` (`title` or `body`) and `excerpt`. Data-only results omit `match`.
Bodies require `--with-body`, and hashes are omitted. The default limit is 20;
0 returns all matches, and negative limits fail. No matches returns an empty
`documents` array. Search exits 0 even with diagnostics; invalid arguments or
collection-level failures exit 1 with an error on stderr.
Full-text excerpts prefer the title or body passage containing the most distinct
query terms. Repeated copies of one term do not outweigh several different terms.
Ties prefer the title, then the earliest passage. Passages contain at most 180
Unicode characters plus ellipses; whitespace is collapsed. One excerpt cannot
always show terms that occur in different fields or far apart. Literal-mode
excerpts retain their original behavior.
Datetime fields are stored as strings. A schema can describe them with
`type: string` and `format: date-time`; search does not require this annotation.
Use RFC 3339 timestamps with an explicit timezone, such as
`2026-10-26T09:00:00Z` or `2026-10-26T10:00:00+01:00`. These represent the same
instant. Quoted and unquoted YAML timestamps retain their string values.

```sh
means search --path ./memory \
  --on-or-after created_at=2026-10-01T00:00:00Z \
  --before created_at=2026-11-01T00:00:00Z --json
```

`--before` and `--after` exclude the boundary; `--on-or-before` and
`--on-or-after` include it. All four accept repeatable `KEY=TIMESTAMP` arguments.
Keys identify literal fields directly under `data` (not nested paths). Datetime
comparisons combine with each other and `--filter` using AND, before limits and
`has_more`, in data-only, literal, full-text, and semantic searches. Ordinary
`--filter` still compares exact string values, without timezone normalization.

Timestamps support up to nine fractional second digits. Date-only values,
timezone-free values, leap seconds, and longer fractions are unsupported.
Invalid filter timestamps fail the request. Missing, non-string, and invalid
document timestamps do not match; they do not introduce additional diagnostics.
Search does not rewrite or normalize stored values.

MCP `search` exposes the same comparisons through `datetime`:

```json
{"datetime":[{"field":"created_at","op":"on_or_after","value":"2026-10-01T00:00:00Z"},{"field":"created_at","op":"before","value":"2026-11-01T00:00:00Z"}]}
```

Supported operators are `before`, `after`, `on_or_before`, and `on_or_after`.
Each comparison requires `field`, `op`, and `value`.

Malformed documents and every occurrence of duplicate IDs are excluded and reported
as diagnostics, regardless of query, filter, or limit. Schema-invalid data remains
searchable. MCP offers the same behavior with the optional `literal` boolean.

### Search cache

Each collection stores a disposable Bleve index in `.means/search/`, for example
`issues/.means/search/`. Every full-text search scans and hashes the visible
documents, then updates changed index entries and removes absent or invalid ones.
Content hashes detect changes even when file size and timestamps stay the same.
Renames, deletions, and changes to nested collection boundaries are reflected in
the next search. Nested collections keep independent caches. Source documents
and schemas remain authoritative; create, update, and delete need no cache writes.

Full-text search also keeps parsed metadata in `.means/documents.jsonl`. Each
record contains a relative path, SHA-256 content hash, frontmatter, and the byte
offset where the body starts. Unchanged files reuse their parsed frontmatter;
bodies come from the source files. Every file is still read and hashed. Invalid
documents are parsed again so diagnostics stay current. Application data is not
validated against the schema during search.

The JSONL file starts with a format version and a checksum of the remaining
records. Missing, corrupt, truncated, or obsolete files are rebuilt. Refreshes
use a temporary file and atomic replacement under the search lock; unchanged
records skip both JSON encoding and writing. This is a disposable internal format, not an
interchange contract. Removing `.means/documents.jsonl` while no search is running
forces metadata to be parsed again. Other commands, including literal and
data-only searches, continue to read documents directly.

Search acquires `.means/search.lock`, opens and refreshes the index, queries it,
then closes it and releases the lock. Concurrent searches wait at most five
seconds before failing. OS locks are released when a process exits. MCP uses this
same per-call lifecycle. There is no background watcher or daemon. External editors
do not take the lock: results reflect files observed during the scan, not an atomic
filesystem snapshot. Concurrent edits may appear on the next search.

Missing indexes, obsolete index formats, and indexes that cannot be opened are
rebuilt. Permission failures and symlinked cache paths produce errors. To force
a rebuild, remove `.means/search/` while no search is running. Do not remove the
lock file while searches are active. Full-text search needs a writable cache;
literal and data-only searches use direct scans and create no cache. For older
collections, add `.means/` to their `.gitignore` manually.

### Opt-in semantic search

Embedding settings belong to the machine running Means, not the collection.
On the first collection command or MCP startup, Means creates
`$XDG_CONFIG_HOME/means/config.yaml`, defaulting to `~/.config/means/config.yaml`.
`XDG_CONFIG_HOME` must be absolute. The starter contains a commented example;
embeddings remain disabled until you edit it manually. There is no setup command.
Existing files, including empty or malformed ones, are never overwritten. Concurrent
first runs publish a complete file without replacing one another. The starter
file has mode 0600 and newly created configuration directories have mode 0700.

If a missing file cannot be created, Means continues with embeddings disabled.
Successful CLI commands report a warning on stderr (a JSON warning object with
`--json`); failed commands retain a single structured error. MCP reports bootstrap
warnings on stderr after successful startup, never on protocol stdout. Existing
unreadable or invalid configs fail commands/startup instead of silently changing
behavior; CLI help and version remain available to aid recovery.
Bare invocation, all help forms, and version do not read or create this file.

The user config is plain YAML, with an optional `embeddings` field. Unknown or
duplicate fields, multiple YAML documents, aliases, merge keys, and custom tags
are rejected. Empty/comment-only files and `{}` disable embeddings. Add:

```yaml
embeddings:
  provider: openrouter
```

The provider is required; an empty object is invalid. Omit `embeddings` or set it
to `false` to disable embedding operations. Providers supply these defaults:

| Provider | Model | Endpoint | Credential |
| --- | --- | --- | --- |
| `openai` | `text-embedding-3-small` | `https://api.openai.com/v1` | `OPENAI_API_KEY` |
| `openrouter` | `openai/text-embedding-3-small` | `https://openrouter.ai/api/v1` | `OPENROUTER_API_KEY` |

Optional `model` also accepts the corresponding `text-embedding-3-large` model.
OpenRouter requires the `openai/` prefix; direct OpenAI rejects it. Optional
`dimensions` must be positive and at most 1536 for small or 3072 for large;
omitted dimensions default to 512 for either model. For example:

```yaml
embeddings:
  provider: openrouter
  model: openai/text-embedding-3-small
  dimensions: 512
```

Set the selected provider's API key in the process environment, then backfill
existing documents:

```sh
means embed --path ./memory --json
means search "why did we choose Go" --semantic --path ./memory --json
```

To discard existing vectors and regenerate every valid document, including
unchanged documents, use `means embed --force --path ./memory`. This clears the
embedding cache under its lock before calling the API and incurs fresh API
charges. Markdown files are unchanged. If the run fails, only batches completed
during that run remain cached. Resume with `means embed` without `--force` to
reuse that progress. Refresh and forced rebuilds are CLI-only; MCP does not
expose an `embed` tool.

Once enabled, CLI and MCP create/update/delete maintain embeddings automatically
for **every collection** used by that machine, sending changed document title/body
text to the configured provider. CLI reloads settings each invocation; MCP captures
them at startup, so restart it after changing settings or credentials. Collection
schemas and workflow guidance still reload per tool call. Offline collection
operations never require API credentials. Bootstrap does not generate vectors;
run `means embed` explicitly to backfill existing or externally edited documents.
Each document has one vector for `Title: ` plus its title, two newlines, and its
exact Markdown body. Application data is excluded; data-only edits reuse the
vector. A title or body change generates a replacement. Inputs are not chunked
or truncated; inputs exceeding the provider's token limit fail. This version
uses the provider's limit errors rather than a local tokenizer.

Create and update store the vector before writing the document. API or cache-write
errors leave the document unchanged. Delete removes the cached vector before
removing the document, without an API call. Cache and Markdown are separate files,
not an atomic transaction: a later document-write failure or crash can leave a
stale or orphan vector. Search skips stale vectors and documents no longer visible;
`means embed` repairs the cache.
External edits can still race the final hash check and document replacement.

`embed` scans the current collection, refreshes missing/changed vectors, and
removes vectors for deleted, malformed, duplicate, and newly nested documents.
It sends document input to the configured endpoint and may incur charges.
Semantic search sends only the query and never writes the embedding cache.
Each request has a 60-second timeout and
no automatic retries. Refreshes send up to 32 changed documents per request,
process batches sequentially, and replace the JSONL cache once per completed batch.
Responses are matched by index, even if returned out of order. An invalid or
failed batch contributes no vectors; earlier completed batches remain cached.
Over-limit documents fail their whole batch without truncation. Single-document
create/update operations still use one request.
CLI and MCP embedding progress goes to stderr: document IDs for single writes,
completed/total changed documents for batches,
and refresh counts for generated, reused, and removed
vectors. Logs omit document/query text, vectors, and credentials. CLI `--json`
suppresses progress so errors remain valid JSON on stderr. MCP progress remains
on stderr, separate from protocol messages on stdout.
Redirect stderr to suppress progress. Semantic search emits no progress messages.
Read, list, doctor, full-text search, and literal search stay offline. No watcher
or managed service is required; direct file edits are detected on the next refresh.

Semantic search requires a nonempty query and cannot combine with `--literal`.
It uses stored vectors that match the current title/body and embedding configuration.
Documents with missing or stale vectors are skipped. A missing, corrupt, or
incompatible cache produces no semantic hits; run `means embed` to rebuild it.
Search does not prune vectors or repair the cache. If no eligible vectors exist,
it returns without a query API call. Search reads the atomically replaced cache
without taking the lock, so it neither waits for `means embed` or document writes
nor blocks them.
It ranks all eligible documents by cosine similarity, descending, with ID order
for ties. Scores must be at least **0.30** by default. Override this with
`--min-score NUMBER` (requires `--semantic`); values must be finite and between
-1 and 1. Higher thresholds return fewer results. Use `--min-score -1` to include
all scores, or `--min-score 0` to include zero and positive scores. The threshold
is inclusive and applies at full precision before `--limit`, together with exact
data filters. `has_more` counts only results that pass these filters.

The default is a starting point for the default embedding model, not a guarantee
of relevance across models or collections. Inspect scores for representative
queries before choosing a different threshold:

```sh
means search openai --semantic --min-score 0.3 --path ./issues
```

Semantic text tables include a score column rounded to three decimal places.
JSON hits contain the full-precision `score` instead of `match`; semantic results
have no literal-match excerpt. Use `--with-body --json` for bodies. Existing
diagnostics behavior applies. Full-text search remains the default.

The cache is `.means/embeddings.jsonl`, separate from `.means/documents.jsonl`.
It contains float vectors, IDs, input hashes, a version/model/dimension/endpoint
profile, and a record checksum. It is derived local data, excluded by `.means/`
in `.gitignore`. Atomic file replacements under `.means/embeddings.lock` protect
concurrent means operations. Lock waits stop after five seconds with a "busy"
error; semantic search does not take the lock. Missing, corrupt,
or obsolete cache data regenerates on demand, potentially incurring API charges;
permission errors and symlinked cache paths fail. Do not delete cache files while
embedding operations run. There is no SQLite dependency. Quantized vectors,
frontmatter sharing, chunking, and hybrid ranking are not implemented yet.

#### Migrating collection embedding settings

Move the `embeddings` block from each `MEANS.md` into this machine's user config,
then remove it from `MEANS.md`. Even legacy `embeddings: false` must be removed.
Means rejects legacy collection settings with migration guidance; it never copies
settings automatically or enables external API calls just because a collection
contains them. Choose one provider/model/dimension profile for this machine.
There are no collection overrides or named profiles. Keep credentials in the
process environment, not in either configuration file.

Existing compatible `.means/embeddings.jsonl` caches remain usable: moving the
settings does not change their profile identity. Other machines generate their
own vectors after configuration; documents and MEANS.md remain portable. Run
`means embed --path ./collection` to populate or reconcile each local cache.

#### OpenRouter

[OpenRouter supports embedding requests](https://openrouter.ai/docs/api/api-reference/embeddings/submit-an-embedding-request)
for OpenAI models. Use `embeddings: {provider: openrouter}` in the user config and set
`OPENROUTER_API_KEY`. Neither `OPENAI_API_KEY` nor a base URL override is required.

`MEANS_EMBEDDING_BASE_URL` overrides the selected provider's endpoint for proxies
or testing. `MEANS_EMBEDDING_API_KEY` overrides its credential. Changing the URL
does not change the provider, model, or credential selection. Missing credentials
fail; means never falls back to another provider's key. Endpoint overrides and
credentials are never loaded from collection files. Changing the provider, endpoint, model, or dimensions
invalidates the local vector profile and causes regeneration on the next refresh.

### Updating documents

For an interactive edit, set an editor executable and run:

```sh
EDITOR=vi means edit <id> --path ./memory
```

Means uses `$VISUAL`, falling back to `$EDITOR`. The value must be an executable
name or path; use a wrapper script for editor arguments. The editor must wait
until editing is complete. Means opens a private temporary Markdown copy and
keeps the original file unchanged until the editor exits successfully, the
document passes validation against the freshly loaded schema, and the original
hash still matches. Enabled embeddings use the same maintenance as `update`.
Editor output goes to stderr so JSON results stay on stdout.

Keep the ID unchanged. Edited metadata replaces the old metadata, including
removing omitted keys and retaining literal null values. Body-only edits preserve
original frontmatter formatting; metadata changes may reformat YAML. Editor or
save failures retain the temporary copy and report its path for recovery. Successful
saves remove it. Hash checks have the same external-write race limitation as update.

For automation, keep using the structured merge-patch interface below.

Read before updating. JSON read/create/update results include an ID, frontmatter,
body, and a SHA-256 hash of the exact stored file bytes. Supply that hash with
only the fields to change:

```sh
mise means update <id> --path ./memory --input - --json <<'JSON'
{
  "hash": "<hash from read>",
  "frontmatter": {"title": "Corrected title", "data": {"tags": ["decision"]}}
}
JSON
```

Metadata uses JSON Merge Patch: omitted keys remain, nested objects merge,
`null` deletes a key, and arrays replace. This patch format cannot set a literal
null metadata value. Supply `body` to replace the whole Markdown body; omit it
to keep the existing body or use `""` to clear it. IDs cannot be patched.
Updates validate the resulting document and reject stale hashes. On conflict,
read again and review the intervening changes before retrying. Updates use a
same-directory temporary replacement, but do not lock the document against a
write racing the final replacement.

`mise means doctor --path ./memory --json` checks every visible document against
the current schema without changing files. It reports malformed frontmatter,
invalid means fields, duplicate IDs, and application schema violations. It takes
no positional arguments or structured input and has no result limit. JSON reports
contain `healthy`, `checked`, and `diagnostics`. `checked` counts inspected files,
including malformed files and duplicate occurrences. Each diagnostic has `id`
(empty when unreadable), a collection-relative `path`, and `error`; a file can
have multiple diagnostics. Text reports show a status summary, grouped errors
and warnings with counts, and affected paths with indented messages. Interactive
terminals use green for healthy status, amber for warnings, and red for errors.
Pipes and `NO_COLOR` disable colors; JSON is unchanged.
Healthy collections exit 0; an unhealthy report goes to stdout and exits 1 with
no stderr response. Collection-level failures, such as an invalid schema, exit 1
with an error on stderr instead of a report.

When this machine has embeddings enabled, doctor also reports documents without
usable embeddings, stale vectors, and orphaned vectors with no unique valid
document in the collection. Deleted documents, duplicate IDs, malformed envelopes,
and documents moved into a nested collection can leave orphaned vectors.
Schema-invalid data alone does not make a vector orphaned. Cache corruption or
an incompatible embedding profile is reported as a cache diagnostic; documents
then have no usable embeddings. These findings are warnings: text labels them
`WARNING`, and JSON includes a `warnings` array with the same `id`, `path`, and
`error` fields as diagnostics. The array is omitted when empty. Warnings do not
change `healthy` or the exit code; warnings alone exit 0. Document validation
errors still exit 1, as do failures to complete the check, such as an unreadable
cache. `checked` still counts document files,
not vectors. Run `means embed --path COLLECTION` to refresh and prune vectors.
Doctor makes no API calls, requires no API key, takes no cache lock, and does not
create or modify cache files. Machines without embeddings configured skip these checks.

CLI success goes to stdout; errors go to stderr. With `--json`, ordinary commands
return structured responses and errors such as `{"error":"..."}`. `--path`
defaults to the current working directory. Relative collection and input file
paths both resolve against the process working directory; an input path is not
relative to the selected collection.

## Deleting documents

Read the document, review it, then supply its ID and current hash:

```sh
mise means read <id> --path ./memory --json
mise means delete <id> --path ./memory --input - --json <<'JSON'
{"hash":"<hash from read>"}
JSON
```

Alternatively, pass the hash directly with `means delete <id> --hash <hash>`.
`--hash` and `--input` are mutually exclusive; both enforce the same checks.

Deletion permanently removes the document file and its optional cached embedding. It preserves directories,
collection configuration, and unrelated files. There is no trash, undo, force
option, or confirmation prompt. For archival, use an application field such as
`data.status` instead. The JSON result is `{"id":"..."}`; text output is the ID.

Missing or duplicate IDs, malformed document envelopes, and stale or invalid
hashes fail without deletion. A repeated deletion fails because the document is
missing. Data that fails the current application schema can still be deleted;
the selected collection must have a valid schema. Collection boundaries apply.
Opted-in embedding operations share a cache lock; external edits do not take it.
A concurrent external write between the hash check and removal can still be
deleted. With embeddings disabled, updates/deletes take no locks. On conflict,
read again and review the changes.

## MCP

`means mcp --path /absolute/collection/path` serves one collection over stdio.
The client starts and owns the subprocess; there is no network listener or
managed daemon. Register the command once per collection to give one agent
multiple collections. Each server advertises its collection identity and the
Markdown workflow from `MEANS.md` at startup. Call `info` for the current
schema and guidance. Other tools are `list`, `search`, `read`, `create`,
`update`, `delete`, and `schema_set`; they use the same collection behavior as the CLI.
Embedding maintenance on create/update/delete is automatic. Backfills and forced
rebuilds use CLI `means embed`; they are not exposed as MCP tools.
`info` includes the host embedding configuration loaded at startup (or null when disabled).
Use `search` with `semantic: true` for embedding similarity. Optional `min_score`
has the same range and default (0.3) as CLI `--min-score` and requires
`semantic: true`. Give the MCP server
process the same API environment variables as the CLI. Each document call reloads
the schema. Restart the server after changing collection title, description, or
workflow to refresh advertised discovery metadata.

MCP messages use protocol JSON framing regardless of `--json`. Successful tool
results always use `structuredContent` without a redundant text copy; clients
must consume structured results. Tool errors still contain explanatory text.
`--json` only formats CLI help and startup errors; it does not change MCP results.
This repository's
[MCP registration](../.codex/config.toml) serves its
[issues collection](../issues/) by running `mise run --raw means -- mcp --path ./issues`
from the repository root. The MCP registration sets the working directory and
arguments directly; no collection-specific mise task is needed. Raw mode keeps
stdio connected to the MCP client. Each server startup runs the current source;
restart the server to pick up code changes.

## Migrating earlier collections

List responses now separate malformed and duplicate files into `diagnostics`
instead of mixing error entries into `documents`. Update CLI JSON and MCP consumers
to inspect both arrays. Limits count valid documents only; diagnostics are never
limited. `has_more` indicates additional valid documents, not additional errors.

This version requires MEANS.md configuration; there is no compatibility fallback
or migration command. Migrate every collection, including nested collections,
before running parent operations:

1. Put the contents of `means.schema.json` (or older `schema.json`) under a
   `schema:` field in YAML frontmatter at the start of `MEANS.md`.
2. Move the schema's top-level `title` and `description` annotations to frontmatter
   fields beside `schema` to preserve collection identity.
3. Keep existing workflow text as the Markdown body. If it was in `instructions.md`,
   move it here and remove that obsolete file; it is otherwise treated as a document.
4. Remove the old schema file. Run `means doctor --path COLLECTION` and restart MCP.

JSON objects are valid YAML values, so the old schema can initially be placed
inline after `schema:`, then reformatted as YAML. Quote internal references
such as `'#/$defs/status'` when writing YAML.

A directory containing only an old schema file no longer establishes a boundary.
Any nested MEANS.md now establishes one, even with invalid or missing frontmatter.
Update CLI consumers of init JSON: the old `schema` path field is now `config`.
Schema-set commands still accept JSON, but no longer preserve its formatting.

Earlier versions used filename stems as IDs and flat application frontmatter.
Migrate those files explicitly: preserve the old filename stem as
`frontmatter.id`, retain or add `title`, move application fields under `data`,
and remove `title` from the application's schema properties and required list.
An application field named `id` belongs under `data.id`; it does not replace
the document ID. Bodies can stay unchanged. There is no automatic migration or
legacy fallback.

## Environment files

Collection commands and MCP startup load optional `.env` files before user
configuration and credentials. Existing process environment variables win, even
when empty. Means reads the collection directory selected by `--path` first, then
the current working directory for variables still unset. When these directories
are the same, the file is read once. It does not search parent directories.

Use `KEY=value`, optional `export`, comments, and single or double quotes, for example:

```dotenv
OPENROUTER_API_KEY=your-key
```

Missing files are ignored; unreadable or malformed files fail startup without
printing their contents. Help, version, and bare invocation do not load `.env`.
Restart MCP after editing credentials. Keep `.env` files out of version control.
