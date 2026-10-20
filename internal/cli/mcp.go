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
  The schema's top-level title and description identify the collection in server
  instructions and every tool description, before an agent calls any tool.
  Missing or blank title falls back to the directory name; description is optional.
  These schema annotations are separate from document frontmatter fields.
  Add optional instructions.md beside schema.json for collection workflow guidance.
  This plain UTF-8 Markdown file needs no frontmatter and is reserved configuration,
  excluded from document lists and reads. It must be a regular file, not a symlink.
  Its contents are included in server instructions and info results.
  Startup instructions give concise means usage guidance; info returns the
  complete usage guidance with the current collection workflow.

Tools (all bound to --path; no per-call path argument):
  info {}                            Title, description, schema, path, instructions.
  list {"limit":20}                  IDs and metadata; 0 returns all, no bodies.
  search {"query":"schema","filter":{"priority":"normal"},"limit":20}
                                     Text/data matches, excerpts, and diagnostics.
  read {"id":"..."}                 ID, hash, frontmatter, and body.
  create {"frontmatter":{"title":"...","data":{}},"body":"..."}
                                     Validate and create; body defaults to empty.
  update {"id":"...","hash":"...","frontmatter":{...},"body":"..."}
                                     Patch metadata and/or replace body.

Call info before creating documents: its schema validates only frontmatter.data.
Means owns frontmatter.id (generated, immutable) and frontmatter.title (required,
nonempty string). Omitted data defaults to {} on creation; explicit null is invalid.
IDs are stored values, not filenames; duplicate IDs fail lookup and appear as list
errors. Read and update follow IDs after file renames or moves into subdirectories.
Patch title and/or data; never include id inside the frontmatter patch. Each tool reloads schema.json so
schema edits take effect immediately. Reads and lists do not validate existing
data against the schema; list entries may contain individual errors. Search
matches a case-insensitive literal phrase in title/body and partially matches
frontmatter.data using JSON values. It reports malformed files separately.
Creation is not idempotent: retrying a success creates another document.
Read before update and pass its hash. Metadata uses JSON Merge Patch: objects merge
recursively, null deletes keys, and arrays replace. Omitted frontmatter/body stays
unchanged. Body must be a string when supplied; "" clears it. A stale hash fails
without writing; read again and review the changes before retrying. The means envelope and resulting
data are validated. Title edits preserve filenames. See "means update --help" for concurrency and storage details.
Deletion, initialization, and schema editing are not MCP tools yet.
info reads current title, description, and instructions.md on each call.
Restart the server/client to refresh advertised tool descriptions and connection
instructions after editing these files. AGENTS.md is not loaded by the server.

Output is always MCP JSON-RPC, one compact message per line on stdout. Successful
tool results include structuredContent and a compact JSON text equivalent by
default. --json omits the text copy for clients that consume structuredContent.
Use it when every client of this registration supports structured results.
Tool errors still include explanatory text.
Tool failures use MCP isError results; the server stays available for later calls.
Startup failures and diagnostics go to stderr. --json also formats CLI help and
startup errors; it never changes MCP message framing.`,
		Example: `  means mcp --path ./example/issues
  means mcp --path ./issues --json
  # Register separately to give the same agent another collection
  means mcp --path ./decisions
  means mcp --help --json`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			server, err := mcpserver.New(opts.path, version, opts.json)
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
