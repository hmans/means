package cli

import (
	"github.com/hmans/means/internal/collection"
	"github.com/spf13/cobra"
)

func newShowCommand(opts *options) *cobra.Command {
	return &cobra.Command{
		Use:   "show <id>",
		Short: "Show a document with rendered Markdown",
		Long: `Show a document by its stored ID from --path (default: current directory).

Print the title, ID, and Markdown-rendered body using terminal styling.
Application metadata and the content hash are available with --json. Output wraps
to at most 80 columns, shrinking to fit narrower terminals. Pipes and NO_COLOR
disable styling. Bullet markers have a separate accent color.
Use read for the exact stored Markdown, including YAML frontmatter.

--json returns {"id":"...","hash":"...","frontmatter":{...},"body":"..."},
exactly as read --json does. The hash can be used for update or delete.
Lookup ignores hidden paths and symlinks and stops at nested collection schemas.
Missing IDs, duplicate IDs, malformed documents, and invalid collection schemas
fail with exit 1 and an error on stderr (JSON with --json). Current application
data schema violations do not prevent reading. Success exits 0.`,
		Example: `  means show <id> --path ./issues
  means show <id> --path ./issues --json`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := collection.Load(opts.path)
			if err != nil {
				return err
			}
			doc, err := c.Read(args[0])
			if err != nil {
				return err
			}
			if opts.json {
				return opts.write(cmd.OutOrStdout(), doc, "")
			}
			text, err := renderDocument(cmd.OutOrStdout(), doc)
			if err != nil {
				return err
			}
			return writeTerminal(cmd.OutOrStdout(), text)
		},
	}
}
