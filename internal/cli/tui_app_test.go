package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/hmans/means/internal/collection"
)

func writeTUIApp(t *testing.T, path, config string) {
	t.Helper()
	content := "---\nschema: true\napp:\n  " + strings.ReplaceAll(strings.TrimSpace(config), "\n", "\n  ") + "\n---\n"
	if err := os.WriteFile(filepath.Join(path, collection.ConfigFilename), []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
}

func loadTUIApp(path string) (*tuiApp, error) {
	c, err := collection.Load(path)
	if err != nil {
		return nil, err
	}
	return compileTUIApp(c.App, nil), nil
}

func TestTUIAppViewsAndReload(t *testing.T) {
	c := tuiTestCollection(t)
	a := tuiCreate(t, c, "Open bug", map[string]any{"status": "todo", "type": "bug", "tags": []any{"cli"}, "priority": "high"})
	tuiCreate(t, c, "Closed bug", map[string]any{"status": "done", "type": "bug", "tags": []any{"cli"}})
	tuiCreate(t, c, "Feature", map[string]any{"status": "todo", "type": "feature"})
	config := "views:\n  - title: Open bugs\n    filter: data.type == 'bug' and data.status == 'todo' and 'cli' in (data.tags ?? [])\n    columns: [title, data.priority]\n"
	writeTUIApp(t, c.Path, config)
	m := newTUIModel(c.Path)
	tuiRun(t, m, m.Init())
	if len(m.docs) != 3 {
		t.Fatal("startup should use All documents")
	}
	tuiPress(m, 'v')
	m.Update(tea.PasteMsg{Content: "bugs"})
	if len(m.choices) != 1 || m.mode != "views" {
		t.Fatal("view type-ahead failed")
	}
	tuiRun(t, m, tuiPress(m, tea.KeyEnter))
	if len(m.docs) != 1 || m.docs[0].ID != a.ID || m.view != "Open bugs" {
		t.Fatal("view filter failed")
	}
	plain := ansi.Strip(m.View().Content)
	if !strings.Contains(plain, "high") || !strings.Contains(plain, "Open bugs") {
		t.Fatal("configured columns were not rendered")
	}
	if strings.Contains(plain, "data.priority") || m.geometry().status {
		t.Fatal("view should use the pane title without column names or an extra status row")
	}
	m.filters["priority"] = "low"
	tuiRun(t, m, m.filterData())
	if len(m.docs) != 0 {
		t.Fatal("view did not combine with data filters")
	}
	tuiRun(t, m, m.action("c"))
	if len(m.docs) != 1 || m.view != "Open bugs" {
		t.Fatal("clear lost selected view")
	}
	tuiPress(m, '/')
	m.Update(tea.PasteMsg{Content: "Open"})
	writeTUIApp(t, c.Path, strings.ReplaceAll(config, "data.status == 'todo'", "data.status == 'done'"))
	tuiPollNow(t, m)
	if len(m.docs) != 0 || m.input != "Open" || m.mode != "search" {
		t.Fatal("live config reload lost active search or view")
	}
	tuiPress(m, tea.KeyEsc)
	if len(m.docs) != 1 || m.docs[0].ID == a.ID {
		t.Fatal("edited view did not apply")
	}
	writeTUIApp(t, c.Path, "views: [broken")
	_, cmd := m.Update(tuiChanged{})
	m.Update(cmd())
	if m.err == nil || len(m.docs) != 1 || m.view != "Open bugs" {
		t.Fatal("invalid app did not retain last usable state")
	}
	if err := os.WriteFile(filepath.Join(c.Path, collection.ConfigFilename), []byte("---\nschema: true\n---\n"), 0600); err != nil {
		t.Fatal(err)
	}
	tuiPollNow(t, m)
	if m.err != nil || m.view != "" || len(m.docs) != 3 {
		t.Fatal("app removal did not restore All documents")
	}
}

func TestTUIAppDateAndNumberExpressions(t *testing.T) {
	path := t.TempDir()
	writeTUIApp(t, path, "views:\n  - title: Recent\n    filter: data.created_at != nil and date(data.created_at) > now() - duration('168h') and data.number == 9007199254740993\n")
	app, err := loadTUIApp(path)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 10, 28, 12, 0, 0, 0, time.UTC)
	for _, tc := range []struct {
		date        any
		match, fail bool
	}{
		{"2026-10-28T13:00:00+02:00", true, false},
		{"2026-10-01T00:00:00Z", false, false},
		{nil, false, false},
		{"not-a-date", false, true},
	} {
		doc := collection.SearchHit{ID: "abc12", Frontmatter: map[string]any{"title": "Test", "data": map[string]any{"created_at": tc.date, "number": json.Number("9007199254740993")}}}
		match, err := app.Views[0].matches(doc, now)
		if match != tc.match || (err != nil) != tc.fail {
			t.Errorf("date %v: match=%v err=%v", tc.date, match, err)
		}
	}
	if _, err := tuiExprValue(json.Number("9223372036854775808")); err == nil {
		t.Fatal("oversized integer was silently rounded")
	}
}

func TestTUIAppValidationAndErrors(t *testing.T) {
	for _, config := range []string{
		"views: [{title: X, typo: true}]",
		"views: [{title: X}, {title: X}]",
		"views: [{title: All documents}]",
		"views: [{title: X, columns: [unknown]}]",
		"views: []\n---\nviews: []",
	} {
		path := t.TempDir()
		writeTUIApp(t, path, config)
		if _, err := loadTUIApp(path); err == nil {
			t.Errorf("accepted invalid app: %s", config)
		}
	}
	c := tuiTestCollection(t)
	tuiCreate(t, c, "Bad date", map[string]any{"created_at": "invalid"})
	writeTUIApp(t, c.Path, "views: [{title: Dates, filter: 'date(data.created_at) < now()', columns: [id, title, data.created_at]}]")
	m := newTUIModel(c.Path)
	tuiRun(t, m, m.Init())
	m.openViews()
	m.choice = 1
	tuiRun(t, m, m.choose())
	if len(m.docs) != 0 || len(m.viewErrors) != 1 || !strings.Contains(ansi.Strip(m.View().Content), "view errors") {
		t.Fatal("runtime errors were hidden")
	}
	m.action("d")
	if !strings.Contains(ansi.Strip(m.View().Content), "MEANS.md app.views") {
		t.Fatal("view error missing from diagnostics")
	}
}

func TestTUIAppMouseAndLayout(t *testing.T) {
	c := tuiTestCollection(t)
	tuiCreate(t, c, "Document", map[string]any{"priority": "high"})
	writeTUIApp(t, c.Path, "views: [{title: Work, columns: [id, title, data.priority]}]")
	m := newTUIModel(c.Path)
	tuiRun(t, m, m.Init())
	for _, b := range m.geometry().buttons {
		if b.key == "v" {
			m.mouse(tea.Mouse{X: b.x, Y: b.y, Button: tea.MouseLeft}, false)
		}
	}
	if m.mode != "views" {
		t.Fatal("Views button did not open picker")
	}
	menu := m.viewMenu()
	tuiRun(t, m, m.mouse(tea.Mouse{X: menu.x + 2, Y: menu.y + 3, Button: tea.MouseLeft}, false))
	if m.view != "Work" {
		t.Fatal("mouse did not choose view")
	}
	for _, width := range []int{32, 60, 80, 120} {
		m.Update(tea.WindowSizeMsg{Width: width, Height: 24})
		m.openViews()
		menu := m.viewMenu()
		if menu.x < 0 || menu.x+menu.w > width || menu.y+menu.h >= m.height {
			t.Fatal("dropdown outside viewport")
		}
		for _, line := range strings.Split(m.View().Content, "\n") {
			if ansi.StringWidth(line) != width {
				t.Fatalf("width %d: overflowing view row", width)
			}
		}
	}
}

func TestTUIAppDropdownInteraction(t *testing.T) {
	c := tuiTestCollection(t)
	tuiCreate(t, c, "Visible document", map[string]any{})
	config := "views:\n"
	for i := 0; i < 15; i++ {
		config += fmt.Sprintf("  - title: View %02d\n", i)
	}
	writeTUIApp(t, c.Path, config)
	m := newTUIModel(c.Path)
	tuiRun(t, m, m.Init())
	m.width = 120
	before := m.geometry().body
	m.openViews()
	if m.geometry().body != before {
		t.Fatal("opening dropdown shifted document panes")
	}
	tuiPress(m, tea.KeyEsc)
	m.view = "View 12"
	m.offset = 3
	m.openViews()
	if m.choices[m.choice].value != "View 12" || m.choiceTop == 0 {
		t.Fatal("dropdown did not reveal active view")
	}
	if !strings.Contains(ansi.Strip(m.View().Content), "Visible document") {
		t.Fatal("dropdown replaced document preview")
	}
	tuiPress(m, tea.KeyEsc)
	if m.mode != "documents" || m.offset != 3 {
		t.Fatal("escape reset background state")
	}
	m.openViews()
	m.mouse(tea.Mouse{X: 0, Y: m.height - 2, Button: tea.MouseLeft}, false)
	if m.mode != "documents" || m.view != "View 12" {
		t.Fatal("outside click changed selection")
	}
	m.openViews()
	m.filterChoices("")
	menu := m.viewMenu()
	m.mouse(tea.Mouse{X: menu.x + 2, Y: menu.y + 2, Button: tea.MouseWheelDown}, true)
	if m.choiceTop != 3 {
		t.Fatal("dropdown wheel did not scroll")
	}
	tuiRun(t, m, m.mouse(tea.Mouse{X: menu.x + 2, Y: menu.y + 2, Button: tea.MouseLeft}, false))
	if m.view != "View 02" || m.mode != "documents" {
		t.Fatal("scrolled dropdown selected wrong row")
	}
	m.openViews()
	m.filterChoices("no such view")
	if !strings.Contains(ansi.Strip(m.View().Content), "No matching views") {
		t.Fatal("missing empty search state")
	}
}
