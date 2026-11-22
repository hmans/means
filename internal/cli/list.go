package cli

import (
	"fmt"

	"github.com/hmans/means/collection"
	"github.com/spf13/cobra"
)

func newListCommand(opts *options) *cobra.Command {
	var offset, limit int
	var withBody bool
	var filters filterFlags
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List document IDs and titles, or metadata as JSON",
		Long: `List documents in filename order. Use --filter KEY=VALUE to select application
data, datetime flags to compare RFC 3339 timestamps, or --where for an Expr
expression (same language as MEANS.md views). Filters combine with AND
before --offset (default 0) and --limit (default 20; 0 means all remaining).
--sort KEY[,KEY...] orders by id, title, or data.FIELD (dots for nested fields)
before paging; prefix - to descend. Missing/null values sort last either way;
mixed types order booleans, numbers, strings, arrays, objects; ID breaks ties.
Fields with a schema enum sort in enum order (other values follow).
Offset skips valid matching documents; both values must be zero or greater.
Filter values can be strings or JSON; quote shell arguments containing JSON.
--json includes metadata; --with-body adds bodies. Malformed documents and duplicate
IDs appear in diagnostics regardless of offset or limit.
Use doctor for schema validation. View a result with show <id>.
JSON returns documents, diagnostics, has_more; hashes are omitted. has_more means
valid matches remain after this page. Each call scans current files; changes between
calls can shift pages. Per-document diagnostics still exit 0; collection-level
errors exit 1. Nested MEANS.md directories
are separate collections; hidden paths and symlinks are ignored.`,
		Example: `  means list --path ./memory --json
  means list --path ./memory --offset 20 --limit 20 --json
  means list --path ./memory --limit 0 --json
  means list --path ./issues --where 'data.status in ["todo", "in_progress"]' --json
  means list --path ./issues --sort data.status,-data.created_at,title`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if withBody && !opts.json {
				return fmt.Errorf("--with-body requires --json")
			}
			if limit < 0 {
				return fmt.Errorf("--limit must be zero or greater")
			}
			if offset < 0 {
				return fmt.Errorf("--offset must be zero or greater")
			}
			c, err := opts.loadCollection()
			if err != nil {
				return err
			}
			filter, datetime, err := filters.parse()
			if err != nil {
				return err
			}
			result, err := c.List(cmd.Context(), collection.ListOptions{Filter: filter, DateTime: datetime, Where: filters.where, Sort: filters.sort, Offset: offset, Limit: limit, WithBody: withBody})
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
	cmd.Flags().IntVar(&limit, "limit", 20, "Maximum documents to return; 0 returns all remaining matches")
	cmd.Flags().IntVar(&offset, "offset", 0, "Skip this many valid matching documents before applying the limit")
	cmd.Flags().BoolVar(&withBody, "with-body", false, "Include Markdown bodies in JSON documents (requires --json)")
	filters.bind(cmd)
	return cmd
}
