package cli

import (
	"fmt"

	"github.com/hmans/means/internal/collection"
	"github.com/spf13/cobra"
)

func newListCommand(opts *options) *cobra.Command {
	var limit int
	var withBody bool
	var filters filterFlags
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List document IDs and titles, or metadata as JSON",
		Long: `List documents in filename order. Use --filter KEY=VALUE to select application
data, or datetime flags to compare RFC 3339 timestamps. Filters combine with AND
before --limit (default 20; 0 means all). Values can be strings or JSON; quote
shell arguments containing JSON. --json includes metadata; --with-body adds bodies.
Malformed documents and duplicate IDs appear in diagnostics, even past the limit.
Use doctor for schema validation. View a result with show <id>.
JSON returns documents, diagnostics, has_more; hashes are omitted. Per-document
diagnostics still exit 0; collection-level errors exit 1. Nested MEANS.md directories
are separate collections; hidden paths and symlinks are ignored.`,
		Example: `  means list --path ./memory --json
  means list --path ./memory --limit 0 --json`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if withBody && !opts.json {
				return fmt.Errorf("--with-body requires --json")
			}
			if limit < 0 {
				return fmt.Errorf("--limit must be zero or greater")
			}
			c, err := opts.loadCollection()
			if err != nil {
				return err
			}
			filter, datetime, err := filters.parse()
			if err != nil {
				return err
			}
			result, err := c.Search(collection.SearchOptions{Filter: filter, DateTime: datetime, Limit: limit, WithBody: withBody})
			if err != nil {
				return err
			}
			if opts.json {
				return opts.write(cmd.OutOrStdout(), result, "")
			}
			var rows [][]string
			for _, doc := range result.Documents {
				rows = append(rows, []string{doc.ID, terminalCell(doc.Frontmatter["title"].(string))})
			}
			return writeTerminal(cmd.OutOrStdout(), documentTable(cmd.OutOrStdout(), rows, result.Diagnostics, result.HasMore, "documents"))
		},
	}
	cmd.Flags().IntVar(&limit, "limit", 20, "Maximum documents to return; 0 returns all")
	cmd.Flags().BoolVar(&withBody, "with-body", false, "Include Markdown bodies in JSON documents (requires --json)")
	filters.bind(cmd)
	return cmd
}
