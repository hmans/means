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
		Long: `Check all visible documents for malformed frontmatter, duplicate IDs, and schema
violations. Report optional embedding-cache warnings offline. No files are changed.
Document errors exit 1 with the report on stdout; embedding warnings alone exit 0.
Collection-level failures go to stderr. --json returns healthy, checked, diagnostics,
and optional warnings. Validation uses MEANS.md; doctor never repairs documents.`,
		Example: `  means doctor --path ./memory
  means doctor --path ./memory --json`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := opts.loadCollection()
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
