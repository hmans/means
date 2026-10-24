package cli

import (
	"errors"
	"fmt"
	"strings"

	"github.com/hmans/means/internal/collection"
	"github.com/spf13/cobra"
)

// ErrUnhealthy signals that doctor wrote a complete diagnostic result but found
// invalid documents. Execute returns it without writing a second error response.
var ErrUnhealthy = errors.New("collection has invalid documents")

func newDoctorCommand(opts *options) *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "Check every document against the current collection schema",
		Long: `Inspect every visible Markdown document in the collection selected by --path.

Doctor checks Markdown and YAML frontmatter, the means-owned id/title/data
envelope, duplicate IDs, and frontmatter.data against the current MEANS.md.
It scans visible subdirectories, ignoring hidden paths, symlinks, and the root
MEANS.md. It only reads files; it never repairs or rewrites them.
Subdirectories containing MEANS.md are separate collections and are skipped,
even if their schema is invalid. Run doctor with their --path to inspect them.

Doctor accepts no positional arguments or structured input; it checks all documents
without a result limit.

--json writes {"healthy":true,"checked":0,"diagnostics":[]} on an empty collection.
checked counts all inspected document files, including malformed and duplicate
files, not the number of diagnostics. healthy is true when diagnostics is empty.
Each diagnostic has id, relative path, and an actionable
error. id is empty if the file has no readable ID. A file can have multiple
diagnostics. Text output shows the checked count and each affected path.

Healthy collections exit 0. Invalid documents produce the complete report on
stdout and exit 1, with no stderr error response. Collection-level failures
(such as an invalid MEANS.md or unreadable directory) exit 1 and write an
error to stderr, formatted as {"error":"..."} with --json. Reads and lists
continue to allow schema-invalid application data for retrieval.`,
		Example: `  means doctor --path ./memory
  means doctor --path ./memory --json`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := collection.Load(opts.path)
			if err != nil {
				return err
			}
			result, err := c.Doctor()
			if err != nil {
				return err
			}
			var output strings.Builder
			fmt.Fprintf(&output, "Checked %d documents.\n", result.Checked)
			if result.Healthy {
				output.WriteString("Collection is healthy.\n")
			} else {
				for _, diagnostic := range result.Diagnostics {
					fmt.Fprintf(&output, "%s\t%s\tERROR: %s\n", diagnostic.ID, diagnostic.Path, diagnostic.Error)
				}
			}
			if err := opts.write(cmd.OutOrStdout(), result, output.String()); err != nil {
				return err
			}
			if !result.Healthy {
				return ErrUnhealthy
			}
			return nil
		},
	}
}
