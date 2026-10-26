package cli

import (
	"github.com/spf13/cobra"
)

func newReadCommand(opts *options) *cobra.Command {
	return &cobra.Command{
		Use:   "read <id>",
		Short: "Read a stored document by its frontmatter ID",
		Long: `Read a document by its stored frontmatter.id from --path (default: current directory).

Pass the ID returned by create or list, not a filename or path. Lookup scans
visible .md files recursively, stopping at subdirectories containing MEANS.md.
Renaming or moving a file within the collection preserves its identity.
IDs contain 1–128 letters, digits, - and _, starting with a letter or digit.
Missing and duplicate IDs fail instead of choosing an arbitrary document.
Symlinks and hidden paths are ignored. The root MEANS.md is collection
configuration, not a document; read it directly or use MCP info.

Text output is the exact stored Markdown, including YAML frontmatter.
--json returns {"id":"...","hash":"...","frontmatter":{...},"body":"..."} on stdout.
hash is lowercase SHA-256 of the exact stored file bytes, including frontmatter,
whitespace, and line endings. Supply it to update to prevent stale writes.
Success exits 0; missing or malformed documents fail with exit 1 and an error on
stderr (as {"error":"..."} with --json).

Read checks the means envelope (id, nonempty title, object data), but does not
enforce the current data schema, so documents remain readable after schema changes. The collection still needs a valid MEANS.md.
Frontmatter must be a mapping between --- delimiter lines. LF and CRLF work.
Use JSON-compatible YAML values; duplicate keys, aliases, merge keys, custom tags,
and non-finite numbers are rejected. Plain YAML dates are returned as strings.`,
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
