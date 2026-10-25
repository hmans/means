package cli

import (
	"errors"
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/hmans/means/internal/collection"
	"github.com/spf13/cobra"
)

// ErrUnhealthy signals that doctor wrote a complete diagnostic result but found
// invalid documents. Execute returns it without a second error response.
var ErrUnhealthy = errors.New("collection has invalid documents")

func newDoctorCommand(opts *options) *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "Check documents and optional embedding cache health",
		Long: `Inspect every visible Markdown document in the collection selected by --path.

Doctor checks Markdown and YAML frontmatter, the means-owned id/title/data
envelope, duplicate IDs, and frontmatter.data against the current MEANS.md.
It scans visible subdirectories, ignoring hidden paths, symlinks, and the root
MEANS.md. It only reads files; it never repairs or rewrites them.
Subdirectories containing MEANS.md are separate collections and are skipped,
even if their schema is invalid. Run doctor with their --path to inspect them.

When embeddings are enabled, also report documents with missing or stale vectors
and orphaned vectors with no unique valid document in this collection. Corrupt or
incompatible caches are reported too. Schema-invalid documents still count as
embedding owners if their envelope and ID are valid and unique. Embedding checks
use only local data, require no API key, take no cache lock, and never create or
change cache files. Run means embed to refresh or prune the reported vectors.
Disabled collections skip embedding checks. Embedding findings are warnings:
they do not change healthy or the exit code. Cache read failures still fail the check.

Doctor accepts no positional arguments or structured input; it checks all documents
without a result limit.

--json writes {"healthy":true,"checked":0,"diagnostics":[]} on an empty collection.
checked counts all inspected document files, including malformed and duplicate
files, not the number of diagnostics. healthy is true when diagnostics is empty.
Each diagnostic has id, relative path, and an actionable
error. id is empty if the file has no readable ID. A file can have multiple
diagnostics. Text output shows the checked count and each affected path.
Terminal reports use colored status and grouped errors and warnings. Pipes and
NO_COLOR disable colors. Stored paths and messages are rendered as plain text.
JSON includes warnings when present, using the same id/path/error fields as
diagnostics. Text output labels them WARNING.

Healthy collections, including those with only warnings, exit 0. Invalid documents produce the complete report on
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
			if opts.json {
				err = opts.write(cmd.OutOrStdout(), result, "")
			} else {
				err = writeTerminal(cmd.OutOrStdout(), renderDoctor(result, terminalWidth(cmd.OutOrStdout())))
			}
			if err != nil {
				return err
			}
			if !result.Healthy {
				return ErrUnhealthy
			}
			return nil
		},
	}
}

func renderDoctor(result *collection.DoctorResult, width int) string {
	var out strings.Builder
	heading := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("99"))
	muted := lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	out.WriteString(heading.Render("Collection check") + "\n")
	out.WriteString(muted.Render(fmt.Sprintf("Checked %d documents.", result.Checked)) + "\n")
	status, color := "Collection is healthy.", "42"
	if !result.Healthy {
		status, color = "Document errors need attention.", "203"
	} else if len(result.Warnings) > 0 {
		status, color = "Documents are valid; embedding warnings follow.", "214"
	}
	out.WriteString(lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(color)).Render(status) + "\n")
	for _, group := range []struct {
		label, color string
		items        []collection.Diagnostic
	}{
		{"ERROR", "203", result.Diagnostics},
		{"WARNING", "214", result.Warnings},
	} {
		if len(group.items) == 0 {
			continue
		}
		style := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(group.color))
		fmt.Fprintf(&out, "\n%s\n", style.Render(fmt.Sprintf("%s: %d", group.label, len(group.items))))
		for _, item := range group.items {
			location := terminalCell(item.Path)
			if item.ID != "" {
				location += " (" + terminalCell(item.ID) + ")"
			}
			out.WriteString("  " + lipgloss.NewStyle().Bold(true).Width(max(1, width-2)).Render(location) + "\n")
			out.WriteString(lipgloss.NewStyle().PaddingLeft(4).Width(max(5, width)).Render(terminalCell(item.Error)) + "\n")
		}
	}
	return out.String()
}
