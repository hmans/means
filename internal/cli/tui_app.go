package cli

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"charm.land/lipgloss/v2"
	"github.com/expr-lang/expr"
	"github.com/expr-lang/expr/vm"
	"github.com/hmans/means/internal/collection"
)

const tuiAppLocation = "MEANS.md app.views"

type tuiApp struct {
	Views  []tuiAppView
	source string
	errors []collection.Diagnostic
}

type tuiAppView struct {
	collection.AppView
	program    *vm.Program
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
		if strings.TrimSpace(view.Filter) == "" {
			view.Filter = "true"
		}
		program, err := expr.Compile(view.Filter, expr.Env(map[string]any{
			"id": "", "title": "", "body": "", "data": map[string]any{}, "now": func() time.Time { return time.Time{} },
		}), expr.AsBool(), expr.MaxNodes(1000), expr.Optimize(false))
		if err != nil {
			view.compileErr = err
			app.errors = append(app.errors, collection.Diagnostic{Path: tuiAppLocation, Error: fmt.Sprintf("view %q: %v", view.Title, err)})
		}
		view.program = program
	}
	app.source = string(data)
	return &app
}

// Adapt a copy for Expr; collection filters retain their exact JSON semantics.
func tuiExprValue(value any) (any, error) {
	switch value := value.(type) {
	case json.Number:
		if n, err := value.Int64(); err == nil {
			return n, nil
		}
		if !strings.ContainsAny(string(value), ".eE") {
			return nil, fmt.Errorf("integer %s exceeds Expr's int64 range", value)
		}
		return value.Float64()
	case map[string]any:
		out := make(map[string]any, len(value))
		for k, v := range value {
			converted, err := tuiExprValue(v)
			if err != nil {
				return nil, err
			}
			out[k] = converted
		}
		return out, nil
	case []any:
		out := make([]any, len(value))
		for i, v := range value {
			converted, err := tuiExprValue(v)
			if err != nil {
				return nil, err
			}
			out[i] = converted
		}
		return out, nil
	default:
		return value, nil
	}
}

func (v *tuiAppView) matches(doc collection.SearchHit, now time.Time) (bool, error) {
	data, err := tuiExprValue(doc.Frontmatter["data"])
	if err != nil {
		return false, err
	}
	body := ""
	if doc.Body != nil {
		body = *doc.Body
	}
	machine := vm.VM{MemoryBudget: 100000}
	result, err := machine.Run(v.program, map[string]any{"id": doc.ID, "title": doc.Frontmatter["title"], "data": data, "body": body, "now": func() time.Time { return now }})
	if err != nil {
		return false, err
	}
	match, ok := result.(bool)
	if !ok {
		return false, fmt.Errorf("filter returned %T, expected boolean", result)
	}
	return match, nil
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
