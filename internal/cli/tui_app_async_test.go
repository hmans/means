package cli

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

func TestTUIInvalidViewsAndCompilationCache(t *testing.T) {
	c := tuiTestCollection(t)
	tuiCreate(t, c, "Alpha", map[string]any{})
	config := "views:\n  - title: Broken\n    filter: 'data.('\n  - title: Valid\n    filter: title == 'Alpha'\n"
	writeTUIApp(t, c.Path, config)
	m := newTUIModel(c.Path)
	tuiRun(t, m, m.Init())
	if len(m.docs) != 1 || len(m.diagnostics) != 1 || m.app.Views[0].compileErr == nil {
		t.Fatal("a broken view prevented startup or hid its diagnostic")
	}
	app, program := m.app, m.app.Views[1].program
	tuiCreate(t, c, "Beta", map[string]any{})
	tuiPollNow(t, m)
	if m.app != app || m.app.Views[1].program != program || len(m.docs) != 2 {
		t.Fatal("document refresh did not reuse compiled app")
	}
	m.openViews()
	m.choice = 1
	if cmd := m.choose(); cmd != nil || m.view != "" || m.mode != "views" {
		t.Fatal("invalid view could be selected")
	}
	m.choice = 2
	tuiRun(t, m, m.choose())
	if len(m.docs) != 1 || m.docs[0].Frontmatter["title"] != "Alpha" {
		t.Fatal("valid sibling view failed")
	}
	writeTUIApp(t, c.Path, strings.ReplaceAll(config, "title == 'Alpha'", "title =="))
	tuiPollNow(t, m)
	if m.app == app || m.view != "" || len(m.docs) != 2 || len(m.diagnostics) != 2 {
		t.Fatal("breaking the active view did not fall back to All documents")
	}
	writeTUIApp(t, c.Path, config)
	tuiPollNow(t, m)
	if m.app.Views[1].compileErr != nil || len(m.diagnostics) != 1 {
		t.Fatal("fixed expression did not recover")
	}
}

func TestTUIViewEvaluationRejectsStaleResults(t *testing.T) {
	c := tuiTestCollection(t)
	tuiCreate(t, c, "Alpha", map[string]any{"status": "open"})
	tuiCreate(t, c, "Beta", map[string]any{"status": "closed"})
	writeTUIApp(t, c.Path, "views: [{title: Alpha, filter: title == 'Alpha'}, {title: Beta, filter: title == 'Beta'}]")
	m := newTUIModel(c.Path)
	tuiRun(t, m, m.Init())
	m.openViews()
	m.choice = 1
	oldCommand := m.choose()
	if oldCommand == nil || !m.evaluating || len(m.docs) != 2 {
		t.Fatal("view evaluation ran synchronously")
	}
	oldResult := oldCommand() // Completed work whose delivery is delayed.
	m.openViews()
	m.choice = 2
	newCommand := m.choose()
	m.Update(oldResult)
	if !m.evaluating || len(m.docs) != 2 {
		t.Fatal("stale result replaced the pending view")
	}
	results := make(chan tea.Msg, 1)
	go func() { results <- newCommand() }()
	tuiPress(m, '/')
	m.Update(tea.PasteMsg{Content: "Beta"})
	_, next := m.Update(<-results)
	tuiRun(t, m, next)
	if m.evaluating || m.mode != "search" || m.input != "Beta" || len(m.docs) != 1 || m.docs[0].Frontmatter["title"] != "Beta" {
		t.Fatal("evaluation lost current search input")
	}
	tuiPress(m, tea.KeyEsc)
	m.filters["status"] = "closed"
	oldCommand = m.filterData()
	oldResult = oldCommand()
	m.filters["status"] = "open"
	newCommand = m.filterData()
	m.Update(oldResult)
	tuiRun(t, m, newCommand)
	if len(m.docs) != 0 {
		t.Fatal("older metadata filters overwrote newer filters")
	}
	oldCommand = m.filterData()
	m.openViews()
	m.choice = 0
	m.choose()
	if result := oldCommand(); result != nil {
		t.Fatal("superseded work was not canceled")
	}
	if m.evaluating || m.view != "" || len(m.docs) != 1 {
		t.Fatal("All documents did not cancel custom evaluation")
	}
}

func TestTUIRefreshSupersedesViewEvaluation(t *testing.T) {
	c := tuiTestCollection(t)
	tuiCreate(t, c, "Alpha", map[string]any{})
	writeTUIApp(t, c.Path, "views: [{title: Everything}]")
	m := newTUIModel(c.Path)
	tuiRun(t, m, m.Init())
	m.openViews()
	m.choice = 1
	oldResult := m.choose()()
	tuiCreate(t, c, "Beta", map[string]any{})
	tuiRun(t, m, m.load())
	m.Update(oldResult)
	if len(m.docs) != 2 {
		t.Fatal("old snapshot overwrote refreshed results")
	}
}
