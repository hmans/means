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

Scan regular .md files directly inside the directory; do not recurse. Ignore
symlinks and filenames whose stem is not a valid ID (letters, digits, - and _,
starting with a letter or digit). Results are sorted by filename, not creation time.
--limit defaults to 20; use 0 for all documents. Negative limits are rejected.

--json returns {"documents":[{"id":"...","frontmatter":{...}}],"has_more":false}.
Bodies are omitted. An empty collection returns an empty documents array.
If more results exist, has_more is true; increase --limit or use 0 to retrieve all.
Text output contains one ID and its JSON metadata per line, plus a notice when
the limit hides results. Read a full document with "means read <id>".

Malformed documents appear as {"id":"...","frontmatter":null,"error":"..."} so
one bad file does not hide the others. These entries count toward the limit.
Listing does not validate metadata against the schema; schema violations remain
readable. Per-document errors still produce exit 0. Collection-level failures
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
					fmt.Fprintf(&text, "%s\tERROR: %s\n", doc.ID, doc.Error)
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
