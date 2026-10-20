package cli

import (
	"encoding/json"
	"fmt"
	"strings"

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
IDs come from frontmatter.id, not filenames. Results are sorted by relative
filename, not creation time. Root instructions.md is excluded.
--limit defaults to 20; use 0 for all documents. Negative limits are rejected.

--json returns {"documents":[{"id":"...","frontmatter":{...}}],"has_more":false}.
Bodies and hashes are omitted; use read --json before updating a document.
An empty collection returns an empty documents array.
If more results exist, has_more is true; increase --limit or use 0 to retrieve all.
Text output contains one ID and its JSON metadata per line, plus a notice when
the limit hides results. Read a full document with "means read <id>".

Malformed documents and duplicate IDs have null frontmatter, an error, and a
relative path for diagnosis. id is empty when it cannot be extracted. These
entries count toward the limit. Duplicate detection covers the whole collection,
even when --limit hides later matches. Listing checks the means envelope but does
not validate data against schema.json; application schema violations remain readable. Per-document errors still produce exit 0. Collection-level failures
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
			var text strings.Builder
			for _, doc := range result.Documents {
				if doc.Error != "" {
					fmt.Fprintf(&text, "%s\t%s\tERROR: %s\n", doc.ID, doc.Path, doc.Error)
					continue
				}
				metadata, err := json.Marshal(doc.Frontmatter)
				if err != nil {
					return err
				}
				fmt.Fprintf(&text, "%s\t%s\n", doc.ID, metadata)
			}
			if result.HasMore {
				text.WriteString("More documents available; increase --limit or use --limit 0.\n")
			}
			return opts.write(cmd.OutOrStdout(), result, text.String())
		},
	}
	cmd.Flags().IntVar(&limit, "limit", 20, "Maximum documents to return; 0 returns all")
	return cmd
}
