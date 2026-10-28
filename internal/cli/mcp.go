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
		Long: `Serve the collection selected by --path over stdio for an MCP client.
Initialize the collection first and register each collection separately.
Call info to verify identity, schema, and workflow. Other tools are list, search,
read, create, update, delete, and schema_set. Successful calls return structuredContent;
errors include explanations. Embed is CLI-only. Register the executable and
absolute collection path in your client's stdio MCP configuration.
Stdout is reserved for MCP protocol framing; --json only affects help and startup errors.
The client owns the process; no daemon or network listener is needed. Schemas
reload per call. Loads --path/.env, then working-directory .env; existing environment wins.
Restart after changing machine settings or startup guidance.
Use project instructions or a skill to explain when agents should use the collection.`,
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
