package cli

import (
	"io"

	"github.com/hmans/means/internal/mcpserver"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/spf13/cobra"
)

func newMCPCommand(opts *options, version string) *cobra.Command {
	return &cobra.Command{
		Use:   "mcp",
		Short: "Serve one collection over MCP on stdin/stdout",
		Long: `Expose the existing collection selected by --path through MCP over stdio.
The default path is the current working directory. Initialize the collection
before starting; a missing or invalid schema fails startup with exit code 1.

Configure your MCP client to launch "means mcp --path /absolute/collection/path".
The client owns this subprocess. It serves requests until stdin closes, then exits.
No network listener, background daemon, or separate service is required.

Multiple collections:
  Register this command once per collection, each with its own --path. Choose
  registration names in the client, e.g. issues, decisions, or memories.
  MEANS.md frontmatter title and description identify the collection in server
  instructions and every tool description, before an agent calls any tool.
  Missing or blank title falls back to the directory name; description is optional.
  Its required schema field is JSON Schema expressed in YAML. Schema annotations
  do not define collection identity. The Markdown body supplies workflow guidance.
  MEANS.md is reserved configuration, excluded from document lists and reads.
  It must be a regular UTF-8 file, not a symlink.
  The body is included in server instructions and info results.
  Startup instructions give concise means usage guidance; info returns the
  complete usage guidance with the current collection workflow.

Tools (all bound to --path; no per-call path argument):
  schema_set {"schema":{"type":"object"},"force":false}
                                     Validate and install; report document diagnostics.
  info {}                            Title, description, schema, path, instructions.
  list {"limit":20}                  IDs and metadata; 0 returns all, no bodies.
  search {"query":"schema","filter":{"priority":"normal"},"limit":20}
                                     Text/data matches, excerpts, and diagnostics.
  read {"id":"..."}                 ID, hash, frontmatter, and body.
  create {"frontmatter":{"title":"...","data":{}},"body":"..."}
                                     Validate and create; body defaults to empty.
  update {"id":"...","hash":"...","frontmatter":{...},"body":"..."}
                                     Patch metadata and/or replace body.
  delete {"id":"...","hash":"..."}   Permanently remove one document; returns {"id":"..."}.

Call info before creating documents: its schema validates only frontmatter.data.
Means owns frontmatter.id (generated, immutable) and frontmatter.title (required,
nonempty string). Omitted data defaults to {} on creation; explicit null is invalid.
IDs are stored values, not filenames; duplicate IDs fail lookup and appear as list
diagnostics. Read and update follow IDs after file renames or moves into subdirectories.
Patch title and/or data; never include id inside the frontmatter patch. Each tool reloads MEANS.md so
schema edits take effect immediately. Reads and lists do not validate existing
data against the schema. List returns documents, diagnostics, and has_more;
its limit counts valid documents only, while diagnostics cover the whole collection. Search
matches all lowercase Unicode query tokens across title/body in any order, with
a title boost and ID order for score ties. No stemming, stop-word removal, or
query-language operators are used. literal:true selects the original substring
search in filename order; omitting query also uses filename order. Data filters
retain exact JSON values and apply before the limit. Diagnostics include all
malformed or duplicate documents regardless of filters and limits.
Full-text search reads and hashes visible documents on every call and refreshes
.means/search/. It closes the index after the call, with no watcher or daemon.
Concurrent searches wait up to five seconds for .means/search.lock. The index is
disposable and needs write access; literal and data-only searches need no index.
Results reflect a scan, not an atomic snapshot of concurrent filesystem edits.
Creation is not idempotent: retrying a success creates another document.
Read before update and pass its hash. Metadata uses JSON Merge Patch: objects merge
recursively, null deletes keys, and arrays replace. Omitted frontmatter/body stays
unchanged. Body must be a string when supplied; "" clears it. A stale hash fails
without writing; read again and review the changes before retrying. The means envelope and resulting
data are validated. Title edits preserve filenames. See "means update --help" for concurrency and storage details.
Delete requires the hash from read. Missing or duplicate IDs, malformed envelopes,
and stale hashes fail. Existing application data need not match the current schema.
Only the document file is removed; directories and configuration remain. There
is no undo or force option. A repeated deletion fails. Concurrent writes can race
the final removal; see "means delete --help" for details.
schema_set accepts a complete JSON Schema object or boolean, not a string.
force defaults to false. Document violations reject installation with isError
and structured {error,installed:false,healthy,checked,diagnostics}. With force:true,
installation succeeds and retains diagnostics. Invalid schemas and filesystem
errors cannot be forced. Checks cover current data, not every future value.
Documents are never rewritten. Schema JSON is stored as YAML without losing number
precision. Other configuration values and exact Markdown body bytes are preserved;
frontmatter formatting may change. Concurrent edits can race installation.
A running server can repair an invalid or missing embedded schema using schema_set
if the YAML and other configuration fields are valid. Malformed YAML requires a
direct edit. Startup requires valid configuration. Initialization is not an MCP tool yet.
info reads current title, description, and MEANS.md on each call.
Restart the server/client to refresh advertised tool descriptions and connection
instructions after editing MEANS.md. AGENTS.md is not loaded by the server.

Output is always MCP JSON-RPC, one compact message per line on stdout. Successful
tool results always use structuredContent without a redundant text copy.
Clients must consume structured results. --json only formats CLI help and startup
errors; it does not change protocol results.
Tool errors still include explanatory text.
MCP bootstraps the same user config as CLI: $XDG_CONFIG_HOME/means/config.yaml
(default ~/.config/means/config.yaml), with a commented example if missing.
Embedding settings apply to all collections on this machine; MEANS.md cannot
configure embeddings. Settings are read at startup; restart after editing the
user config or changing credentials. Collection schemas still reload per call.
Missing-config bootstrap failures leave embeddings disabled and warn on stderr;
existing malformed configs fail startup. See means embed --help for migration.
Embedding progress goes to stderr; stdout remains exclusively MCP messages.
Tool failures use MCP isError results; the server stays available for later calls.
Startup failures and diagnostics go to stderr. --json also formats CLI help and
startup errors; it never changes MCP message framing.`,
		Example: `  means mcp --path ./issues
  # Register separately to give the same agent another collection
  means mcp --path ./decisions
  means mcp --help --json`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			server, err := mcpserver.NewWithEmbeddingProgress(opts.path, version, embeddingProgress(cmd))
			if err != nil {
				return err
			}
			return server.Run(cmd.Context(), &mcp.IOTransport{
				Reader: io.NopCloser(cmd.InOrStdin()),
				Writer: nopWriteCloser{cmd.OutOrStdout()},
			})
		},
	}
}

type nopWriteCloser struct{ io.Writer }

func (nopWriteCloser) Close() error { return nil }
