package cli

import (
	"github.com/hmans/means/internal/collection"
	"github.com/spf13/cobra"
)

func newReadCommand(opts *options) *cobra.Command {
	return &cobra.Command{
		Use:   "read <id>",
		Short: "Read a stored document by its filename ID",
		Long: `Read <id>.md from the collection selected by --path (default: current directory).

The ID is the filename stem. Pass the ID returned by create or list, without
the .md extension. IDs may contain
letters, digits, - and _, and must start with a letter or digit. Paths, symlinks,
and directories are not accepted. Manually created files using these names work.
The reserved instructions.md is collection configuration, not a document; read it
directly or retrieve its contents through the MCP info tool.

Text output is the exact stored Markdown, including YAML frontmatter.
--json returns {"id":"...","frontmatter":{...},"body":"..."} on stdout.
Success exits 0; missing or malformed documents fail with exit 1 and an error on
stderr (as {"error":"..."} with --json).

Read does not enforce the current frontmatter schema, so documents remain
retrievable after a schema change. The collection still needs a valid schema.json.
Frontmatter must be a mapping between --- delimiter lines. LF and CRLF work.
Use JSON-compatible YAML values; duplicate keys, aliases, merge keys, custom tags,
and non-finite numbers are rejected. Plain YAML dates are returned as strings.`,
		Example: `  means read <id> --path ./memory
  means read <id> --path ./memory --json`,
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
			return opts.write(cmd.OutOrStdout(), doc, doc.Markdown)
		},
	}
}
