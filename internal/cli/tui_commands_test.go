package cli

import (
	"fmt"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/hmans/means/internal/collection"
)

func TestTUIViewCommandMatchesPicker(t *testing.T) {
	c := tuiTestCollection(t)
	for i := 0; i < 30; i++ {
		tuiCreate(t, c, fmt.Sprintf("Document %02d", i), map[string]any{})
	}
	writeTUIApp(t, c.Path, "views:\n  - title: Open work\n  - title: Broken\n    filter: 'data.status =='\n")
	models := []*tuiModel{newTUIModel(c.Path), newTUIModel(c.Path)}
	for _, m := range models {
		m.Update(tea.WindowSizeMsg{Width: 120, Height: 20})
		tuiRun(t, m, m.Init())
		m.selected, m.listTop, m.offset, m.previewFocus = 12, 4, 2, true
		m.renderPreview()
	}
	picker, command := models[0], models[1]
	picker.openViews()
	for i, choice := range picker.choices {
		if choice.value == "Open work" {
			picker.choice = i
		}
	}
	tuiRun(t, picker, picker.choose())
	command.openCommandBar()
	command.Update(tea.PasteMsg{Content: `views "Open work"`})
	tuiRun(t, command, tuiPress(command, tea.KeyEnter))
	if command.previewFocus || command.mode != picker.mode || command.view != picker.view || command.selected != picker.selected || command.listTop != picker.listTop || command.offset != picker.offset {
		t.Fatalf("command state differs: focus %v, selected %d/%d, top %d/%d, offset %d/%d", command.previewFocus, command.selected, picker.selected, command.listTop, picker.listTop, command.offset, picker.offset)
	}
	command.openCommandBar()
	command.Update(tea.PasteMsg{Content: `views Broken`})
	tuiPress(command, tea.KeyEnter)
	if !strings.Contains(command.commandBar.message, "invalid; see Diagnostics") || command.view != "Open work" {
		t.Fatal("invalid view was not distinguished from unknown view")
	}
	command.Update(tea.WindowSizeMsg{Width: 30, Height: 8})
	if !strings.Contains(ansi.Strip(command.View().Content), "Error:") {
		t.Fatal("small terminal hides error")
	}
}

func TestTUICommandBackspace(t *testing.T) {
	m := newTUIModel("unused")
	m.openCommandBar()
	m.Update(tea.PasteMsg{Content: "q"})
	tuiPress(m, tea.KeyBackspace)
	if !m.commandBar.active || m.commandBar.input.Value() != "" {
		t.Fatal("deleting the last character should keep command mode open")
	}
	tuiPress(m, tea.KeyBackspace)
	if m.commandBar.active || m.commandBar.input.Focused() {
		t.Fatal("backspacing the prompt should leave command mode")
	}
}

func TestTUIHomeAndClearCommands(t *testing.T) {
	c := tuiTestCollection(t)
	writeTUIApp(t, c.Path, "views:\n  - title: Open work\n")
	m := newTUIModel(c.Path)
	m.Update(tea.WindowSizeMsg{Width: 120, Height: 20})
	tuiRun(t, m, m.Init())
	run := func(command string) {
		t.Helper()
		m.openCommandBar()
		m.Update(tea.PasteMsg{Content: command})
		tuiRun(t, m, tuiPress(m, tea.KeyEnter))
	}
	m.view, m.query = "Open work", "query"
	m.filters = map[string]any{"status": "open"}
	run("home")
	if m.commandBar.failed || m.view != "" || m.query != "query" || m.filters["status"] != "open" {
		t.Fatal("home must leave the view and preserve search and filters")
	}
	m.view = "Open work"
	m.openCommandBar()
	tuiPress(m, tea.KeyEsc)
	if m.view != "Open work" {
		t.Fatal("Escape in command mode should only dismiss the input")
	}
	tuiRun(t, m, tuiPress(m, tea.KeyEsc))
	if m.view != "" || m.query != "query" || m.filters["status"] != "open" {
		t.Fatal("Escape while browsing must go home and preserve search and filters")
	}
	m.view = "Open work"
	run("clear")
	if m.commandBar.failed || m.view != "Open work" || m.query != "" || len(m.filters) != 0 {
		t.Fatal("clear must clear search and filters and preserve the view")
	}
	run("refresh")
	if !m.commandBar.failed {
		t.Fatal("refresh command is still available")
	}
}

func TestTUICommandErrorKeepsSelectionVisible(t *testing.T) {
	c := tuiTestCollection(t)
	for i := 0; i < 30; i++ {
		tuiCreate(t, c, fmt.Sprintf("Document %02d", i), map[string]any{})
	}
	m := newTUIModel(c.Path)
	m.Update(tea.WindowSizeMsg{Width: 80, Height: 20})
	tuiRun(t, m, m.Init())
	m.selected = m.pageHeight() - 1
	m.openCommandBar()
	m.Update(tea.PasteMsg{Content: "unknown"})
	tuiPress(m, tea.KeyEnter)
	if m.selected < m.listTop || m.selected >= m.listTop+m.pageHeight() {
		t.Fatal("error hid selected row")
	}
}

func TestTUICommandsWhileLoading(t *testing.T) {
	for _, input := range []string{"help", "help views", "views --help", "q", "quit", "views"} {
		m := newTUIModel("unused")
		m.openCommandBar()
		m.Update(tea.PasteMsg{Content: input})
		cmd := tuiPress(m, tea.KeyEnter)
		if input == "views" {
			if !m.commandBar.failed || !strings.Contains(m.commandBar.message, "still loading") {
				t.Fatal("collection command allowed while loading")
			}
		} else if m.commandBar.failed {
			t.Fatalf("%s blocked: %s", input, m.commandBar.message)
		}
		if input == "q" || input == "quit" {
			if cmd == nil {
				t.Fatal("quit blocked while loading")
			}
		}
	}
}

func TestTUICommandCompletionEdgeCases(t *testing.T) {
	for _, tc := range []struct{ input, title string }{
		{`  views C`, "Closed issues"},
		{`views "Say \"h`, `Say "hello"`},
		{`views -`, "-Archived"},
		{`views 'C`, "Closed issues"},
	} {
		t.Run(tc.input, func(t *testing.T) {
			m := newTUIModel("unused")
			m.loading = false
			m.app = compileTUIApp(collection.AppConfig{Views: []collection.AppView{{Title: tc.title}}}, nil)
			m.openCommandBar()
			m.Update(tea.PasteMsg{Content: tc.input})
			tuiPress(m, tea.KeyTab)
			tuiPress(m, tea.KeyEnter)
			if m.commandBar.failed || m.view != tc.title {
				t.Fatalf("view=%q error=%q", m.view, m.commandBar.message)
			}
		})
	}
	m := newTUIModel("unused")
	m.openCommandBar()
	m.Update(tea.PasteMsg{Content: "qu"})
	tuiPress(m, tea.KeyLeft)
	tuiPress(m, tea.KeyTab)
	if m.commandBar.input.Value() != "qu" {
		t.Fatal("completion overwrote input while editing mid-line")
	}
	m.openCommandBar()
	m.Update(tea.PasteMsg{Content: "q"})
	tuiPress(m, tea.KeyDown)
	tuiPress(m, tea.KeyTab)
	if m.commandBar.input.Value() != "quit" {
		t.Fatal("Tab did not accept the displayed suggestion")
	}
}

func TestTUICommandPaneBounds(t *testing.T) {
	m := newTUIModel("unused")
	m.loading = false
	m.openCommandBar()
	for _, size := range [][2]int{{120, 26}, {60, 12}, {32, 12}, {20, 8}, {1, 1}} {
		m.Update(tea.WindowSizeMsg{Width: size[0], Height: size[1]})
		for _, failed := range []bool{false, true} {
			m.commandBar.failed = failed
			m.commandBar.message = strings.Repeat("long error message ", 20)
			lines := strings.Split(m.View().Content, "\n")
			if len(lines) != size[1] {
				t.Fatal("command pane exceeded terminal height")
			}
			for _, line := range lines {
				if ansi.StringWidth(line) != size[0] {
					t.Fatal("command pane exceeded terminal width")
				}
			}
		}
	}
}

func TestTUICommandColorsPreserveInputLayout(t *testing.T) {
	m := newTUIModel("unused")
	m.openCommandBar()
	for _, width := range []int{8, 20, 80} {
		for _, value := range []string{`views "日本語 long view name"`, "  views open", "qu", ""} {
			m.commandBar.input.SetWidth(width)
			m.commandBar.input.SetValue(value)
			for position := 0; position <= len([]rune(value)); position++ {
				m.commandBar.input.SetCursor(position)
				if got, want := ansi.Strip(m.commandBar.view()), ansi.Strip(m.commandBar.input.View()); got != want {
					t.Fatalf("width %d cursor %d: %q != %q", width, position, got, want)
				}
			}
		}
	}
}

func TestTUICommands(t *testing.T) {
	c := tuiTestCollection(t)
	tuiCreate(t, c, "Document", map[string]any{})
	writeTUIApp(t, c.Path, "views:\n  - title: Closed issues\n  - title: Current work\n")
	m := newTUIModel(c.Path)
	m.Update(tea.WindowSizeMsg{Width: 120, Height: 26})
	tuiRun(t, m, m.Init())
	if cmd := tuiPress(m, 'q'); cmd != nil {
		t.Fatal("bare q must not quit")
	}
	tuiPress(m, ':')
	if !m.commandBar.active {
		t.Fatal("colon did not open command bar")
	}
	m.Update(tea.PasteMsg{Content: "qu"})
	tuiPollNow(t, m)
	if !m.commandBar.active || m.commandBar.input.Value() != "qu" {
		t.Fatal("refresh replaced command input")
	}
	if m.commandBar.input.CurrentSuggestion() != "quit" {
		t.Fatal("missing quit preview")
	}
	tuiPress(m, tea.KeyTab)
	if m.commandBar.input.Value() != "quit" {
		t.Fatal("tab did not complete command")
	}
	tuiPress(m, tea.KeyEsc)
	if m.commandBar.active {
		t.Fatal("escape did not dismiss")
	}
	tuiPress(m, ':')
	m.Update(tea.PasteMsg{Content: "views C"})
	if !strings.Contains(m.commandBar.input.CurrentSuggestion(), "Closed issues") {
		t.Fatal("missing argument preview")
	}
	tuiPress(m, tea.KeyTab)
	if m.commandBar.input.Value() != `views "Closed issues"` {
		t.Fatal(m.commandBar.input.Value())
	}
	tuiPress(m, tea.KeyTab)
	if m.commandBar.input.Value() != `views "Current work"` {
		t.Fatal("tab did not cycle")
	}
	m.Update(tea.KeyPressMsg{Code: tea.KeyTab, Mod: tea.ModShift})
	if m.commandBar.input.Value() != `views "Closed issues"` {
		t.Fatal("shift-tab did not cycle backwards")
	}
	tuiRun(t, m, tuiPress(m, tea.KeyEnter))
	if m.view != "Closed issues" || m.commandBar.active {
		t.Fatal("view command failed")
	}
	tuiPress(m, ':')
	m.Update(tea.PasteMsg{Content: "bogus"})
	tuiPress(m, tea.KeyEnter)
	if !m.commandBar.active || !strings.Contains(m.commandBar.message, "unknown command") {
		t.Fatal("unknown command error not shown")
	}
	if !m.commandBar.failed || m.geometry().search.h < 4 || !strings.Contains(ansi.Strip(m.View().Content), "Error: unknown command") {
		t.Fatal("error is not shown in the command pane")
	}
	tuiPress(m, tea.KeyLeft)
	if !m.commandBar.failed {
		t.Fatal("cursor movement dismissed the error")
	}
	tuiPress(m, tea.KeyBackspace)
	if m.commandBar.failed || m.geometry().search.h != 3 {
		t.Fatal("editing did not clear the error")
	}
	tuiPress(m, tea.KeyEsc)
	tuiPress(m, ':')
	m.Update(tea.PasteMsg{Content: `views "unfinished`})
	tuiPress(m, tea.KeyEnter)
	if !m.commandBar.active || m.commandBar.message == "" {
		t.Fatal("invalid quoting was not reported")
	}
	tuiPress(m, tea.KeyEsc)
	tuiPress(m, ':')
	m.Update(tea.PasteMsg{Content: "help views"})
	tuiPress(m, tea.KeyEnter)
	if !m.commandBar.active || !strings.Contains(m.commandBar.message, "views [name]") {
		t.Fatal("command help not displayed")
	}
	tuiPress(m, tea.KeyEsc)
	tuiPress(m, ':')
	m.Update(tea.PasteMsg{Content: "q"})
	cmd := tuiPress(m, tea.KeyEnter)
	if cmd == nil {
		t.Fatal("quit alias did not run")
	}
	if _, ok := cmd().(tea.QuitMsg); !ok {
		t.Fatal("quit alias returned wrong message")
	}
}
