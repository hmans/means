package cli

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/hmans/means/collection"
)

const tuiAppLocation = "MEANS.md app.views"

// View selection validates the view and resets focus to the document list.
func (m *tuiModel) selectView(name string) (tea.Cmd, error) {
	if name == "All documents" {
		name = ""
	}
	if name != "" {
		found := false
		if m.app != nil {
			for _, view := range m.app.Views {
				if view.Title != name {
					continue
				}
				if view.compileErr != nil {
					return nil, fmt.Errorf("view %q is invalid; see Diagnostics", name)
				}
				found = true
				break
			}
		}
		if !found {
			return nil, fmt.Errorf("unknown view %q", name)
		}
	}
	m.view, m.mode, m.previewFocus = name, "documents", false
	return m.filterData(), nil
}

type tuiApp struct {
	Views  []tuiAppView
	source string
	errors []collection.Diagnostic
}

type tuiAppView struct {
	collection.AppView
	program    *collection.Expression
	compileErr error
}

func compileTUIApp(config collection.AppConfig, previous *tuiApp) *tuiApp {
	var app tuiApp
	data, _ := json.Marshal(config)
	if previous != nil && previous.source == string(data) {
		return previous
	}
	for _, view := range config.Views {
		app.Views = append(app.Views, tuiAppView{AppView: view})
	}
	for i := range app.Views {
		view := &app.Views[i]
		if len(view.Columns) == 0 {
			view.Columns = []string{"title"}
		}
		program, err := collection.CompileExpression(view.Filter)
		if err != nil {
			view.compileErr = err
			app.errors = append(app.errors, collection.Diagnostic{Path: tuiAppLocation, Error: fmt.Sprintf("view %q: %v", view.Title, err)})
		}
		view.program = program
	}
	app.source = string(data)
	return &app
}

func (v *tuiAppView) matches(doc collection.SearchHit, now time.Time) (bool, error) {
	body := ""
	if doc.Body != nil {
		body = *doc.Body
	}
	return v.program.Match(doc.Frontmatter, body, now)
}

func (m *tuiModel) currentView() *tuiAppView {
	if m.app != nil {
		for i := range m.app.Views {
			if m.app.Views[i].Title == m.view {
				return &m.app.Views[i]
			}
		}
	}
	return nil
}

func (m *tuiModel) openViews() {
	m.choices = []tuiChoice{{label: "All documents", value: ""}}
	if m.app != nil {
		for _, view := range m.app.Views {
			label := view.Title
			if view.compileErr != nil {
				label += " (invalid; see Diagnostics)"
			}
			m.choices = append(m.choices, tuiChoice{label: label, value: view.Title})
		}
	}
	m.mode, m.choice, m.choiceTop = "views", 0, 0
	m.allChoices = m.choices
	m.filterChoices("")
	for i, choice := range m.choices {
		if choice.value == m.view {
			m.choice = i
			break
		}
	}
	m.choiceTop = tuiKeepVisible(m.choice, 0, m.pickerHeight(), len(m.choices))
}

func tuiAppRow(doc collection.SearchHit, columns []string, selected, hovered bool, width int) string {
	prefix := "  "
	rowStyle := lipgloss.NewStyle()
	if selected {
		prefix = "> "
	}
	if selected || hovered {
		rowStyle = rowStyle.Background(tuiDocumentBackground(selected))
	}
	remaining := max(0, width-2-2*(len(columns)-1))
	weight := len(columns)
	for _, column := range columns {
		if column == "title" {
			weight++
		}
	}
	parts := make([]string, 0, len(columns))
	for _, column := range columns {
		style, value, units := tuiMuted, any(nil), 1
		switch column {
		case "id":
			value, style = doc.ID, tuiAccent
		case "title":
			value, style, units = doc.Frontmatter["title"], tuiTitle, 2
		default:
			value = doc.Frontmatter["data"].(map[string]any)[strings.TrimPrefix(column, "data.")]
		}
		text := "—"
		if value != nil {
			text = tuiValueLabel(value)
		}
		space := remaining * units / weight
		remaining, weight = remaining-space, weight-units
		if selected || hovered {
			style = style.Background(tuiDocumentBackground(selected))
		}
		parts = append(parts, style.Render(tuiFit(terminalCell(text), space)))
	}
	return tuiFillDocumentRow(rowStyle.Render(prefix)+strings.Join(parts, rowStyle.Render("  ")), rowStyle, width)
}
