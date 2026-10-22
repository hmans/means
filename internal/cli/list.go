package cli

import (
	"fmt"

	"github.com/hmans/means/internal/collection"
	"github.com/spf13/cobra"
)

func newListCommand(opts *options) *cobra.Command {
	var limit int
	var withBody bool
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List document IDs and titles, or metadata as JSON",
		Long: `List documents in the collection selected by --path (default: current directory).

Scan regular .md files recursively, ignoring symlinks and hidden paths.
Stop at subdirectories containing means.schema.json, even if their schema is invalid.
IDs come from frontmatter.id, not filenames. Results are sorted by relative
filename, not creation time. Root MEANS.md is excluded.
--limit defaults to 20; use 0 for all documents. Negative limits are rejected.

--json returns {"documents":[{"id":"...","frontmatter":{...}}],"diagnostics":[],"has_more":false}.
Bodies and hashes are omitted by default. --with-body requires --json and adds
the Markdown body to each valid document, including empty bodies as "".
Diagnostic entries have no body. Use read --json before updating a document.
An empty collection returns an empty documents array.
If more results exist, has_more is true; increase --limit or use 0 to retrieve all.
Text output is a colored ID/title table, followed by diagnostics and a notice when
the limit hides results. Pipes and NO_COLOR disable styling. Use --json for metadata.
View a full document with "means show <id>" or its source with "means read <id>".

Malformed documents and every occurrence of duplicate IDs are excluded from
documents and reported in diagnostics with id, path, and error. id is empty when
it cannot be extracted. Limits count valid documents only. Diagnostics cover the
whole collection regardless of the limit. Listing checks the means envelope but does
not validate data against means.schema.json; application schema violations remain readable. Per-document errors still produce exit 0. Collection-level failures
produce exit 1 and stderr, formatted as {"error":"..."} when --json is set.`,
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
			c, err := collection.Load(opts.path)
			if err != nil {
				return err
			}
			result, err := c.ListWithOptions(collection.ListOptions{Limit: limit, WithBody: withBody})
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
	return cmd
}
