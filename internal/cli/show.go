package cli

import (
	"fmt"
	"github.com/spf13/cobra"
)

func newShowCommand(opts *options) *cobra.Command {
	var raw bool
	cmd := &cobra.Command{
		Use:   "show <id>",
		Short: "Show a document with rendered Markdown",
		Long: `Display a document by ID with a rendered Markdown body.
Use --raw for exact source, including YAML frontmatter, or --json for metadata,
body, and the hash required for updates/deletion. --raw and --json are exclusive.
Missing/duplicate IDs and malformed documents fail. Current data-schema violations
do not prevent reading. read --json returns the same structured document.`,
		Example: `  means show <id> --path ./issues
  means show <id> --path ./issues --json`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if raw && opts.json {
				return fmt.Errorf("--raw and --json are mutually exclusive")
			}
			c, err := opts.loadCollection()
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
			if raw {
				return opts.write(cmd.OutOrStdout(), doc, doc.Markdown)
			}
			text, err := renderDocument(cmd.OutOrStdout(), doc)
			if err != nil {
				return err
			}
			return writeTerminal(cmd.OutOrStdout(), text)
		},
	}
	cmd.Flags().BoolVar(&raw, "raw", false, "Print exact stored Markdown, including frontmatter")
	return cmd
}
