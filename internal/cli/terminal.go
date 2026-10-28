package cli

import (
	"io"
	"os"
	"strings"
	"unicode"

	"charm.land/glamour/v2"
	"charm.land/glamour/v2/styles"
	"charm.land/lipgloss/v2"
	"charm.land/lipgloss/v2/table"
	"github.com/charmbracelet/colorprofile"
	"github.com/charmbracelet/x/ansi"
	"github.com/charmbracelet/x/term"
	"github.com/hmans/means/internal/collection"
)

func terminalWidth(w io.Writer) int {
	if f, ok := w.(interface{ Fd() uintptr }); ok {
		if width, _, err := term.GetSize(f.Fd()); err == nil && width > 0 {
			return width
		}
	}
	return 80
}

func writeTerminal(w io.Writer, text string) error {
	profile := colorprofile.NoTTY
	if f, ok := w.(interface{ Fd() uintptr }); ok && term.IsTerminal(f.Fd()) && os.Getenv("NO_COLOR") == "" {
		profile = colorprofile.Detect(w, os.Environ())
	}
	_, err := io.WriteString(&colorprofile.Writer{Forward: w, Profile: profile}, text)
	return err
}

// Titles and diagnostic paths are plain text, even when stored with newlines
// or terminal escape sequences. Keep them from changing the table layout.
func terminalCell(text string) string {
	return strings.Join(strings.Fields(strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return ' '
		}
		return r
	}, ansi.Strip(text))), " ")
}

func documentTable(w io.Writer, rows [][]string, diagnostics []collection.Diagnostic, more bool, noun string, headers ...string) string {
	if len(headers) == 0 {
		headers = []string{"ID", "TITLE"}
	}
	var out strings.Builder
	if len(rows) == 0 {
		out.WriteString("No " + noun + ".\n")
	} else {
		idWidth := 2
		for _, row := range rows {
			idWidth = max(idWidth, ansi.StringWidth(row[0]))
		}
		idWidth = min(idWidth, max(2, terminalWidth(w)/3))
		t := table.New().Headers(headers...).Rows(rows...).Width(terminalWidth(w)).
			Border(lipgloss.RoundedBorder()).BorderColumn(false).
			BorderStyle(lipgloss.NewStyle().Foreground(lipgloss.Color("240"))).
			StyleFunc(func(row, col int) lipgloss.Style {
				style := lipgloss.NewStyle().Padding(0, 1)
				if col == 0 {
					style = style.Width(idWidth + 2)
				}
				if headers[col] == "SCORE" {
					style = style.Width(8).Align(lipgloss.Right)
				}
				if row == table.HeaderRow {
					return style.Bold(true).Foreground(lipgloss.Color("99"))
				}
				if col == 0 {
					return style.Foreground(lipgloss.Color("37"))
				}
				return style
			})
		out.WriteString(t.String() + "\n")
	}
	if len(diagnostics) > 0 {
		out.WriteString("\n" + lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("203")).Render("Diagnostics") + "\n")
		for _, d := range diagnostics {
			out.WriteString(terminalCell(d.Path) + ": " + terminalCell(d.Error) + "\n")
		}
	}
	if more {
		out.WriteString("More " + noun + " available; increase --limit or use --limit 0.\n")
	}
	return out.String()
}

func renderDocument(w io.Writer, doc *collection.Document) (string, error) {
	return renderDocumentWidth(terminalWidth(w), doc)
}

func renderPreviewData(width int, data []byte) string {
	width = max(1, width)
	theme := styles.DarkStyleConfig
	margin := uint(0)
	theme.Document.Margin = &margin
	theme.Document.BlockPrefix = ""
	theme.Document.BlockSuffix = ""
	theme.CodeBlock.Margin = &margin
	rendered := string(data)
	renderer, err := glamour.NewTermRenderer(glamour.WithStyles(theme), glamour.WithWordWrap(width))
	if err == nil {
		if highlighted, err := renderer.Render("```json\n" + string(data) + "\n```\n"); err == nil {
			rendered = strings.Trim(highlighted, "\n")
		}
	}
	separator := lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render(strings.Repeat("─", width))
	label := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("37")).Render("Data")
	return separator + "\n" + label + "\n" + rendered
}

func renderDocumentWidth(width int, doc *collection.Document) (string, error) {
	width = max(1, min(80, width))
	return renderDocumentLayout(width, doc, false)
}

func renderDocumentLayout(width int, doc *collection.Document, preview bool) (string, error) {
	width = max(1, width)
	theme := styles.DarkStyleConfig
	if preview {
		margin := uint(0)
		theme.Document.Margin = &margin
	}
	// BlockPrefix inherits the paragraph color. Prefix uses the item's own
	// style, so the bullet can be colored without coloring the whole list.
	accent := "37"
	theme.Item.BlockPrefix = ""
	theme.Item.Prefix = "• "
	theme.Item.Color = &accent
	renderer, err := glamour.NewTermRenderer(glamour.WithStyles(theme), glamour.WithWordWrap(width))
	if err != nil {
		return "", err
	}
	body, err := renderer.Render(ansi.Strip(doc.Body))
	if err != nil {
		return "", err
	}
	title := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("99")).Width(width).Render(terminalCell(doc.Frontmatter["title"].(string)))
	id := lipgloss.NewStyle().Foreground(lipgloss.Color("37")).Render(doc.ID)
	if preview {
		id = ansi.Truncate(id, width, "")
		titleWidth := max(0, width-ansi.StringWidth(id)-1)
		title = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("231")).Render(
			ansi.Truncate(terminalCell(doc.Frontmatter["title"].(string)), titleWidth, "…"))
		return title + strings.Repeat(" ", max(0, width-ansi.StringWidth(title)-ansi.StringWidth(id))) + id + "\n" + body, nil
	}
	return title + "\n" + id + "\n" + body, nil
}
