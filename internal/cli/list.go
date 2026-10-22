package cli

import (
	"fmt"

	"github.com/hmans/means/internal/collection"
	"github.com/spf13/cobra"
)

func newListCommand(opts *options) *cobra.Command {
	var limit int
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List document IDs and metadata without their bodies",
		Long: `List documents in the collection selected by --path (default: current directory).

Scan regular .md files recursively, ignoring symlinks and hidden paths.
Stop at subdirectories containing means.schema.json, even if their schema is invalid.
IDs come from frontmatter.id, not filenames. Results are sorted by relative
filename, not creation time. Root MEANS.md is excluded.
--limit defaults to 20; use 0 for all documents. Negative limits are rejected.

--json returns {"documents":[{"id":"...","frontmatter":{...}}],"has_more":false}.
Bodies and hashes are omitted; use read --json before updating a document.
An empty collection returns an empty documents array.
If more results exist, has_more is true; increase --limit or use 0 to retrieve all.
Text output is a colored ID/title table, followed by diagnostics and a notice when
the limit hides results. Pipes and NO_COLOR disable styling. Use --json for metadata.
View a full document with "means show <id>" or its source with "means read <id>".

Malformed documents and duplicate IDs have null frontmatter, an error, and a
relative path for diagnosis. id is empty when it cannot be extracted. These
entries count toward the limit. Duplicate detection covers the whole collection,
even when --limit hides later matches. Listing checks the means envelope but does
not validate data against means.schema.json; application schema violations remain readable. Per-document errors still produce exit 0. Collection-level failures
produce exit 1 and stderr, formatted as {"error":"..."} when --json is set.`,
		Example: `  means list --path ./memory --json
  means list --path ./memory --limit 0 --json`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if limit < 0 {
				return fmt.Errorf("--limit must be zero or greater")
			}
			c, err := collection.Load(opts.path)
			if err != nil {
				return err
			}
			result, err := c.List(limit)
			if err != nil {
				return err
			}
			if opts.json {
				return opts.write(cmd.OutOrStdout(), result, "")
			}
			var rows [][]string
			var diagnostics []collection.Diagnostic
			for _, doc := range result.Documents {
				if doc.Error != "" {
					diagnostics = append(diagnostics, collection.Diagnostic{ID: doc.ID, Path: doc.Path, Error: doc.Error})
					continue
				}
				rows = append(rows, []string{doc.ID, terminalCell(doc.Frontmatter["title"].(string))})
			}
			return writeTerminal(cmd.OutOrStdout(), documentTable(cmd.OutOrStdout(), rows, diagnostics, result.HasMore, "documents"))
		},
	}
	cmd.Flags().IntVar(&limit, "limit", 20, "Maximum documents to return; 0 returns all")
	return cmd
}
