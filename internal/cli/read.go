package cli

import (
	"github.com/spf13/cobra"
)

func newReadCommand(opts *options) *cobra.Command {
	return &cobra.Command{
		Use:    "read <id>",
		Hidden: true,
		Short:  "Read a stored document by its frontmatter ID",
		Long: `Read exact stored Markdown by ID, including YAML frontmatter.
Use --json for the structured document and content hash. Use show for rendered
Markdown or show --raw for the same raw output. Missing/duplicate IDs and malformed
documents fail; current data-schema violations do not prevent reading.
Use stored IDs, not filenames. JSON returns id, hash, frontmatter, body. Errors exit 1.`,
		Example: `  means read <id> --path ./memory
  means read <id> --path ./memory --json`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := opts.loadCollection()
			if err != nil {
				return err
			}
			doc, err := c.Read(args[0])
			if err != nil {
				return err
			}
			return opts.write(cmd.OutOrStdout(), doc, doc.Markdown)
		},
	}
}
