package cli

import (
	"fmt"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/hmans/means/internal/collection"
)

func tuiClick(m *tuiModel, x, y int) tea.Cmd {
	_, cmd := m.Update(tea.MouseClickMsg{X: x, Y: y, Button: tea.MouseLeft})
	return cmd
}

func tuiClickButton(t *testing.T, m *tuiModel, key string) tea.Cmd {
	t.Helper()
	for _, b := range m.geometry().buttons {
		if b.key == key {
			return tuiClick(m, b.x+b.w/2, b.y)
		}
	}
	t.Fatalf("no button for %q", key)
	return nil
}

func tuiWheel(m *tuiModel, x, y int, down bool) {
	button := tea.MouseWheelUp
	if down {
		button = tea.MouseWheelDown
	}
	m.Update(tea.MouseWheelMsg{X: x, Y: y, Button: button})
}

func TestTUIDividerDrag(t *testing.T) {
	c := tuiTestCollection(t)
	tuiCreate(t, c, "A document", map[string]any{})
	m := newTUIModel(c.Path)
	m.Update(tea.WindowSizeMsg{Width: 140, Height: 26})
	tuiRun(t, m, m.Init())
	for _, grabOffset := range []int{-1, 0, 1} {
		g := m.geometry()
		tuiClick(m, g.list.w+grabOffset, g.body.y+2)
		if !m.draggingDivider {
			t.Fatal("divider did not capture drag")
		}
		m.Update(tea.MouseMotionMsg{X: 70 + grabOffset, Y: 0, Button: tea.MouseLeft})
		if m.geometry().list.w != 70 || m.previewColumns != m.previewWidth() {
			t.Fatal("drag did not resize panes and reflow preview")
		}
		m.Update(tea.MouseReleaseMsg{X: 75 + grabOffset, Y: 0, Button: tea.MouseLeft})
		m.Update(tea.MouseMotionMsg{X: 80, Y: 0, Button: tea.MouseLeft})
		if m.draggingDivider || m.geometry().list.w != 75 || m.selected != 0 || m.previewFocus {
			t.Fatal("release did not stop drag or drag changed selection/focus")
		}
	}
	g := m.geometry()
	tuiClick(m, g.list.w, g.body.y)
	m.Update(tea.MouseMotionMsg{X: -100, Button: tea.MouseLeft})
	if m.geometry().list.w != 32 {
		t.Fatal("list minimum width was not enforced")
	}
	m.Update(tea.MouseMotionMsg{X: 1000, Button: tea.MouseLeft})
	if m.geometry().preview.w != 32 {
		t.Fatal("preview minimum width was not enforced")
	}
	m.Update(tea.WindowSizeMsg{Width: 90, Height: 26})
	if m.draggingDivider || m.geometry().preview.w != 32 {
		t.Fatal("terminal resize did not cancel drag and clamp panes")
	}
	m.Update(tea.WindowSizeMsg{Width: 70, Height: 26})
	if m.geometry().split {
		t.Fatal("narrow terminal should use one pane")
	}
	m.Update(tea.WindowSizeMsg{Width: 140, Height: 26})
	if m.geometry().list.w != 107 {
		t.Fatal("terminal resize lost preferred width")
	}
	for _, mode := range []string{"fields", "values", "diagnostics", "views"} {
		m.mode = mode
		g := m.geometry()
		tuiClick(m, g.list.w, g.body.y)
		if m.draggingDivider {
			t.Fatalf("divider active in %s overlay", mode)
		}
	}
}

func TestTUIDocumentHoverAndEmptyFilterRow(t *testing.T) {
	c := tuiTestCollection(t)
	for i := 0; i < 30; i++ {
		tuiCreate(t, c, fmt.Sprintf("Document %02d", i), map[string]any{})
	}
	m := newTUIModel(c.Path)
	m.Update(tea.WindowSizeMsg{Width: 120, Height: 26})
	tuiRun(t, m, m.Init())
	g := m.geometry()
	if g.status || g.body.y != g.search.y+g.search.h || strings.Contains(m.View().Content, "No data filters") {
		t.Fatal("empty filter row still occupies space")
	}
	m.Update(tea.MouseMotionMsg{X: 4, Y: g.list.y + 2})
	if m.hoveredDocument() != 1 || m.selected != 0 || m.previewFocus {
		t.Fatal("hover should target a row without changing selection or focus")
	}
	tuiWheel(m, 4, g.list.y+2, true)
	if m.hoveredDocument() != 4 {
		t.Fatal("hover did not follow scrolled content")
	}
	m.Update(tea.MouseMotionMsg{X: g.preview.x + 4, Y: g.preview.y + 2})
	if m.hoveredDocument() != -1 {
		t.Fatal("hover remained after leaving list")
	}
	m.filters["status"] = "open"
	if !m.geometry().status || m.geometry().body.y != g.body.y+1 {
		t.Fatal("active filter row was not restored")
	}
	m.mode = "views"
	m.Update(tea.MouseMotionMsg{X: 4, Y: m.geometry().list.y + 2})
	if m.hoveredDocument() != -1 {
		t.Fatal("dropdown should suppress document hover")
	}
	if tuiDocumentRow("abcde", "Title", true, false, 30) != tuiDocumentRow("abcde", "Title", true, true, 30) {
		t.Fatal("hover changed selected row styling")
	}
}

func TestTUIMouseSelectionScrollAndFocus(t *testing.T) {
	c := tuiTestCollection(t)
	for i := 0; i < 30; i++ {
		tuiCreate(t, c, fmt.Sprintf("Document %02d 日本語", i), map[string]any{})
	}
	m := newTUIModel(c.Path)
	m.Update(tea.WindowSizeMsg{Width: 120, Height: 26})
	tuiRun(t, m, m.Init())
	g := m.geometry()
	tuiClick(m, g.list.x+3, g.list.y+3)
	if m.selected != 2 || m.previewFocus {
		t.Fatal("click selected wrong document")
	}
	id := m.docs[m.selected].ID
	if !strings.Contains(ansi.Strip(strings.Join(m.preview, "\n")), id) {
		t.Fatal("preview did not follow selection")
	}
	tuiWheel(m, g.list.x+3, g.list.y+3, true)
	if m.listTop != 3 || m.selected != 2 {
		t.Fatal("wheel should scroll without changing selection")
	}
	tuiClick(m, g.list.x+3, g.list.y+1)
	if m.selected != 3 {
		t.Fatal("click after scroll used stale row index")
	}
	tuiWheel(m, g.preview.x+3, g.preview.y+3, true)
	if m.offset != 3 || m.previewFocus {
		t.Fatal("wheel should scroll hovered pane without changing focus")
	}
	tuiClick(m, g.preview.x+3, g.preview.y+3)
	if !m.previewFocus {
		t.Fatal("click did not focus preview")
	}
	tuiPress(m, 'j')
	if m.offset != 4 {
		t.Fatal("keyboard did not follow mouse focus")
	}
	// Separator and frame clicks must not select a row.
	tuiClick(m, g.list.w, g.list.y+3)
	tuiClick(m, g.list.x, g.list.y+1)
	if m.selected != 3 || !m.previewFocus {
		t.Fatal("border click changed focus or selection")
	}
	m.Update(tea.WindowSizeMsg{Width: 65, Height: 20})
	g = m.geometry()
	if g.split {
		t.Fatal("expected one-pane layout")
	}
	tuiClickButton(t, m, "tab")
	tuiClick(m, g.list.x+3, g.list.y+2)
	if m.previewFocus || m.selected != m.listTop+1 {
		t.Fatal("narrow layout hit testing failed")
	}
	tuiClickButton(t, m, "tab")
	tuiWheel(m, 3, m.geometry().preview.y+2, true)
	if m.offset == 0 {
		t.Fatal("narrow preview did not scroll")
	}
}

func TestTUIMouseToolbarSearchAndFilters(t *testing.T) {
	c := tuiTestCollection(t)
	tuiCreate(t, c, "Alpha", map[string]any{"status": "open"})
	tuiCreate(t, c, "Beta", map[string]any{"status": "closed"})
	m := newTUIModel(c.Path)
	m.Update(tea.WindowSizeMsg{Width: 100, Height: 24})
	tuiRun(t, m, m.Init())
	g := m.geometry()
	tuiClick(m, 5, g.search.y+1)
	m.Update(tea.PasteMsg{Content: "Alpha"})
	if m.mode != "search" || len(m.docs) != 1 {
		t.Fatal("click-to-search failed")
	}
	tuiClickButton(t, m, "esc")
	if m.query != "" || len(m.docs) != 2 {
		t.Fatal("cancel button did not restore search")
	}
	tuiClickButton(t, m, "/")
	m.Update(tea.PasteMsg{Content: "Alpha"})
	tuiClick(m, 4, m.geometry().list.y+1)
	if m.mode != "documents" || m.query != "Alpha" {
		t.Fatal("document click did not commit live search")
	}
	tuiRun(t, m, tuiClickButton(t, m, "c"))
	tuiClickButton(t, m, "f")
	tuiClick(m, 3, m.geometry().body.y+1)
	if m.mode != "values" || m.field != "status" {
		t.Fatal("field click failed")
	}
	tuiRun(t, m, tuiClick(m, 3, m.geometry().body.y+2))
	if m.filters["status"] != "closed" || len(m.docs) != 1 {
		t.Fatal("value click failed")
	}
	tuiClickButton(t, m, "f")
	tuiClick(m, 3, m.geometry().body.y+1)
	tuiRun(t, m, tuiClick(m, 3, m.geometry().body.y+1))
	if len(m.filters) != 0 || len(m.docs) != 2 {
		t.Fatal("mouse did not remove filter")
	}
	tuiClickButton(t, m, "f")
	tuiCreate(t, c, "New field", map[string]any{"priority": "high"})
	tuiPollNow(t, m)
	if m.mode != "fields" || len(m.choices) != 2 {
		t.Fatal("refresh left stale choices in the open picker")
	}
	tuiClickButton(t, m, "d")
	if m.mode != "diagnostics" {
		t.Fatal("diagnostics button failed")
	}
	tuiClickButton(t, m, "esc")
	if m.mode != "documents" {
		t.Fatal("back button failed")
	}
	tuiRun(t, m, tuiPress(m, 'r'))
	if cmd := tuiClickButton(t, m, "q"); cmd == nil {
		t.Fatal("quit button failed")
	} else if _, ok := cmd().(tea.QuitMsg); !ok {
		t.Fatal("quit did not send quit message")
	}
}

func TestTUIMousePickerAndDiagnosticScrolling(t *testing.T) {
	m := newTUIModel("unused")
	m.loading = false
	m.width, m.height = 100, 20
	m.fields = map[string]tuiField{}
	for i := 0; i < 40; i++ {
		m.fields[fmt.Sprintf("field%02d", i)] = tuiField{values: []any{"one", "two"}}
	}
	m.openFields()
	tuiWheel(m, 4, m.geometry().body.y+2, true)
	if m.choiceTop != 3 {
		t.Fatal("picker wheel did not scroll")
	}
	tuiClick(m, 4, m.geometry().body.y+1)
	if m.field != "field03" {
		t.Fatal("scrolled picker click used wrong index")
	}
	m.mode = "diagnostics"
	m.diagnostics = []collection.Diagnostic{{Path: "bad.md", Error: strings.Repeat("long error ", 200)}}
	tuiWheel(m, 4, m.geometry().body.y+2, true)
	if m.offset != 3 {
		t.Fatal("diagnostic wheel did not scroll")
	}
	tuiWheel(m, 4, m.geometry().body.y+2, false)
	if m.offset != 0 {
		t.Fatal("diagnostic upward scroll failed")
	}
	m.loading = true
	tuiClickButton(t, m, "f")
	if m.mode != "diagnostics" {
		t.Fatal("mouse changed state while loading")
	}
}

func TestTUIFramedLayoutBounds(t *testing.T) {
	c := tuiTestCollection(t)
	tuiCreate(t, c, "Wide 日本語 title", map[string]any{"status": "open"})
	m := newTUIModel(c.Path)
	tuiRun(t, m, m.Init())
	m.filters["status"] = "open"
	for _, size := range [][2]int{{140, 35}, {100, 24}, {90, 24}, {80, 24}, {60, 16}, {32, 12}, {20, 8}, {1, 1}} {
		m.Update(tea.WindowSizeMsg{Width: size[0], Height: size[1]})
		for _, mode := range []string{"documents", "search", "fields", "values", "diagnostics"} {
			m.mode = mode
			v := m.View()
			if v.MouseMode != tea.MouseModeAllMotion {
				t.Fatal("mouse reporting is disabled")
			}
			lines := strings.Split(v.Content, "\n")
			if len(lines) != size[1] {
				t.Fatalf("%v %s: height %d", size, mode, len(lines))
			}
			for _, line := range lines {
				if ansi.StringWidth(line) != size[0] {
					t.Fatalf("%v %s: width %d", size, mode, ansi.StringWidth(line))
				}
			}
			g := m.geometry()
			if !g.small {
				if g.body.y+g.body.h != m.height-1 {
					t.Fatal("body overlaps footer")
				}
				for _, b := range g.buttons {
					if b.x < 0 || b.x+b.w > m.width || b.y < 0 || b.y >= m.height {
						t.Fatal("control exceeds screen bounds")
					}
					if b.toolbar && b.y >= g.search.y {
						t.Fatal("toolbar overlaps search")
					}
					if !strings.Contains(ansi.Strip(lines[b.y+b.h/2]), b.label) {
						t.Fatalf("button label differs from hit area: %q", b.label)
					}
				}
			}
		}
	}
}

func TestTUIContextualControls(t *testing.T) {
	m := newTUIModel("unused")
	m.loading = false
	m.width, m.height = 120, 30
	check := func(wantClear bool) {
		t.Helper()
		g := m.geometry()
		clear := false
		for _, b := range g.buttons {
			if b.toolbar && b.key != "/" && b.key != "f" && b.key != "r" {
				t.Fatalf("unexpected toolbar action: %s", b.key)
			}
			if b.key == "c" {
				clear = true
				if b.y != g.body.y-1 {
					t.Fatal("Clear is not beside filters")
				}
			}
			if b.key == "esc" || b.key == "q" || b.key == "tab" || b.key == "d" {
				if b.y != m.height-1 {
					t.Fatal("secondary control is not in footer")
				}
			}
		}
		if clear != wantClear {
			t.Fatalf("clear visible=%v, want %v", clear, wantClear)
		}
	}
	check(false)
	m.mode, m.input = "search", "example"
	check(true)
	m.input, m.mode = "", "documents"
	m.filters["status"] = "open"
	check(true)
}
