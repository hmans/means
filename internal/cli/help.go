package cli

var humanCommandExamples = map[string]string{
	"means init":   "  means init --path ./memory\n  means init --path ./issues --schema schema.json",
	"means create": "  means create --title \"Deployment notes\" --body-file notes.md\n  means create --input document.json --json",
	"means update": "  means show abc12 --json\n  means update abc12 --input patch.json --json",
	"means delete": "  means show abc12 --json\n  means delete abc12 --hash <hash-from-show>",
	"means search": "  means search \"deployment timeout\"\n  means search deployment --filter status=todo\n  means search \"why Go\" --semantic",
	"means list":   "  means list --filter status=todo\n  means list --limit 0 --json",
}

// Command Long descriptions remain the full contract for agent and JSON help.
// Human help focuses on inputs, useful behavior, and safety constraints.
var humanCommandHelp = map[string]string{
	"means init": `Create a collection in --path (default: current directory).
Use the example schema, --schema FILE|- for JSON Schema, or --input FILE|- for
a complete MEANS.md. The two input flags are exclusive. Existing MEANS.md files
are never overwritten. Run "means help agents" for agent setup guidance.`,
	"means create": `Create a document with --title TITLE and optional --body-file FILE|-.
This uses empty application data. If the schema requires data fields, use
--input FILE|- with {"frontmatter":{"title":"Title","data":{}},"body":"Markdown"}.
Do not combine these input modes. A dash reads stdin. Means validates the schema
and generates the ID. Text returns the ID; --json returns the document and hash.
Enabled embeddings send title/body to the provider; failures prevent creation.`,
	"means update": `Read the document with show --json or read --json, then patch it with --input FILE|-:
{"hash":"<current hash>","frontmatter":{"data":{"status":"done"}}}.
Omitted fields stay; null deletes metadata keys; arrays replace. A supplied body
replaces the full body. Stale hashes and invalid data fail without saving.
Enabled embedding failures also prevent saving. Use edit for an editor workflow.`,
	"means delete": `Permanently delete a document using its ID and current hash from show --json.
Pass --hash HASH or --input FILE|- containing {"hash":"<current hash>"}.
These options are exclusive. Stale hashes fail. There is no undo, force, or
confirmation prompt. Directories and collection configuration are kept.`,
	"means show": `Display a document by ID with a rendered Markdown body.
Use --raw for exact source, including YAML frontmatter, or --json for metadata,
body, and the hash required for updates/deletion. --raw and --json are exclusive.
Missing/duplicate IDs and malformed documents fail. Current data-schema violations
do not prevent reading. read --json returns the same structured document.`,
	"means read": `Read exact stored Markdown by ID, including YAML frontmatter.
Use --json for the structured document and content hash. Use show for rendered
Markdown or show --raw for the same raw output. Missing/duplicate IDs and malformed
documents fail; current data-schema violations do not prevent reading.`,
	"means list": `List documents in filename order. Use --filter KEY=VALUE to select application
data, or datetime flags to compare RFC 3339 timestamps. Filters combine with AND
before --limit (default 20; 0 means all). Values can be strings or JSON; quote
shell arguments containing JSON. --json includes metadata; --with-body adds bodies.
Malformed documents and duplicate IDs appear in diagnostics, even past the limit.
Use doctor for schema validation. View a result with show <id>.`,
	"means search": `Find documents by words in their title/body, ranked by relevance.
Use --literal for a case-insensitive substring or --semantic for embedding similarity;
these modes are exclusive. Semantic search requires configured embeddings and a
nonempty query, sends the query to the provider, and skips stale/missing vectors.
Run embed to refresh vectors. --min-score applies only to semantic search.
Combine --filter KEY=VALUE and RFC 3339 datetime flags with AND before --limit.
Omit the query to filter only by data. Default limit is 20; 0 returns all.
--json includes metadata and excerpts; --with-body adds bodies. Inspect diagnostics
for malformed/duplicate documents; finding no matches still exits successfully.`,
	"means schema set": `Validate and install JSON Schema from --input FILE|- into MEANS.md.
Check existing documents before replacing the schema. --force allows document
violations, but never invalid schemas or filesystem errors. Documents are not
migrated. Malformed MEANS.md YAML requires direct editing. Run doctor afterward.
Text and JSON report installation, checked counts, and diagnostics; rejection exits 1.`,
	"means doctor": `Check all visible documents for malformed frontmatter, duplicate IDs, and schema
violations. Report optional embedding-cache warnings offline. No files are changed.
Document errors exit 1 with the report on stdout; embedding warnings alone exit 0.
Collection-level failures go to stderr. --json returns the structured report.`,
	"means embed": `Refresh missing/stale embeddings and remove obsolete vectors.
Enable embeddings in the machine-local user config and set provider credentials
first. Title/body content is sent to the provider and may incur charges.
--force discards existing vectors before rebuilding; failed runs keep completed
batches. Ordinary runs resume. Search does not backfill missing vectors.
Use "means help agents embed" for configuration fields and provider setup.`,
	"means mcp": `Serve the collection selected by --path over stdio for an MCP client.
Initialize the collection first and register each collection separately.
The server exposes identity/workflow, document operations, search, and schema changes.
Stdout is reserved for MCP messages; --json only affects help and startup errors.
Restart after changing machine settings or startup guidance.`,
}
