package cli

import (
	"fmt"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/hmans/means/collection"
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
	m := newTUIModel(c.Path())
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

func TestTUIPreviewSplitAndIndependentScrolling(t *testing.T) {
	c := tuiTestCollection(t)
	long := tuiCreate(t, c, "Long data", map[string]any{"items": strings.Split(strings.Repeat("value\n", 40), "\n")})
	tuiCreate(t, c, "Short data", map[string]any{})
	m := newTUIModel(c.Path())
	m.Update(tea.WindowSizeMsg{Width: 140, Height: 36})
	tuiRun(t, m, m.Init())
	if m.docs[m.selected].ID != long.ID {
		tuiPress(m, tea.KeyDown)
	}
	step := 1 - 2*m.selected
	g := m.geometry()
	if g.data.h != 10 || g.content.h != 21 {
		t.Fatalf("default split is not one third/two thirds: %d/%d", g.data.h, g.content.h)
	}
	m.move(step)
	if m.geometry().data != g.data || m.geometry().content != g.content {
		t.Fatal("document selection changed the panel heights")
	}
	m.move(-step)
	tuiWheel(m, g.data.x+3, g.data.y+2, true)
	if m.dataOffset != 3 || m.offset != 0 || m.previewFocus {
		t.Fatal("data wheel changed content scroll or keyboard focus")
	}
	tuiWheel(m, g.content.x+3, g.content.y+2, true)
	if m.dataOffset != 3 || m.offset != 3 || m.previewFocus {
		t.Fatal("content wheel changed data scroll or keyboard focus")
	}
	tuiClick(m, g.data.x+3, g.data.y+2)
	tuiPress(m, tea.KeyDown)
	if !m.previewFocus || !m.dataFocus || m.dataOffset != 4 || m.offset != 3 {
		t.Fatal("keyboard did not follow data focus")
	}
	tuiPress(m, tea.KeyPgDown)
	if m.dataOffset != 4+g.data.h-2 || m.offset != 3 {
		t.Fatal("page down did not use the data panel height")
	}
	tuiPress(m, tea.KeyTab)
	tuiPress(m, tea.KeyDown)
	if m.dataFocus || m.offset != 4 {
		t.Fatal("Tab did not focus content")
	}
	m.Update(tea.KeyPressMsg{Code: tea.KeyTab, Mod: tea.ModShift})
	if !m.previewFocus || !m.dataFocus {
		t.Fatal("Shift+Tab did not return to data")
	}
	tuiPress(m, tea.KeyEnd)
	dataEnd := m.dataOffset
	if !strings.Contains(m.View().Content, "Content") || dataEnd == 0 || m.offset != 4 {
		t.Fatal("data scrolling moved or hid content")
	}
	tuiCreate(t, c, "New document", map[string]any{})
	tuiPollNow(t, m)
	if m.dataOffset != dataEnd || m.offset != 4 {
		t.Fatal("refresh did not preserve independent scroll positions")
	}
	tuiPress(m, tea.KeyHome)
	if m.dataOffset != 0 || m.offset != 4 {
		t.Fatal("Home scrolled the wrong panel")
	}
}

func TestTUIPreviewDividerDrag(t *testing.T) {
	c := tuiTestCollection(t)
	tuiCreate(t, c, "Document", map[string]any{"items": strings.Split(strings.Repeat("value\n", 40), "\n")})
	m := newTUIModel(c.Path())
	m.Update(tea.WindowSizeMsg{Width: 140, Height: 36})
	tuiRun(t, m, m.Init())
	for _, grabOffset := range []int{-1, 0} {
		g := m.geometry()
		x := g.preview.x + 3
		tuiClick(m, x, g.data.y+g.data.h+grabOffset)
		if !m.draggingPreview || m.draggingDivider {
			t.Fatal("preview separator did not capture drag")
		}
		cached := &m.preview[0]
		m.Update(tea.MouseMotionMsg{X: -10, Y: g.preview.y + 15 + grabOffset, Button: tea.MouseLeft})
		if m.geometry().data.h != 15 || &m.preview[0] != cached {
			t.Fatal("drag did not resize panels or unnecessarily rerendered Markdown")
		}
		m.Update(tea.MouseReleaseMsg{X: x, Y: g.preview.y + 12 + grabOffset, Button: tea.MouseLeft})
		m.Update(tea.MouseMotionMsg{X: x, Y: 0, Button: tea.MouseLeft})
		if m.draggingPreview || m.geometry().data.h != 12 || m.previewFocus || m.selected != 0 {
			t.Fatal("release did not finish drag or changed focus/selection")
		}
	}
	fraction := m.dataFraction
	m.Update(tea.WindowSizeMsg{Width: 70, Height: 51})
	if g := m.geometry(); g.data.h != 18 || m.dataFraction != fraction {
		t.Fatal("resize did not retain the chosen proportion")
	}
	// The separator also works when a narrow terminal shows the preview.
	tuiPress(m, tea.KeyTab)
	m.dataOffset, m.offset = m.previewDataMaxOffset(), m.previewMaxOffset()
	g := m.geometry()
	tuiClick(m, 3, g.data.y+g.data.h)
	m.Update(tea.MouseMotionMsg{X: 3, Y: -100, Button: tea.MouseLeft})
	if m.geometry().data.h != 3 || m.offset > m.previewMaxOffset() {
		t.Fatal("data panel minimum height was not enforced")
	}
	m.Update(tea.MouseMotionMsg{X: 3, Y: 1000, Button: tea.MouseLeft})
	if m.geometry().content.h != 3 || m.dataOffset > m.previewDataMaxOffset() {
		t.Fatal("content panel minimum height was not enforced")
	}
	m.Update(tea.WindowSizeMsg{Width: 70, Height: 12})
	if m.draggingPreview {
		t.Fatal("terminal resize did not cancel preview drag")
	}
	for _, mode := range []string{"fields", "values", "diagnostics", "views"} {
		m.mode = mode
		g := m.geometry()
		tuiClick(m, 3, g.data.y+g.data.h)
		if m.draggingPreview {
			t.Fatalf("preview separator active in %s overlay", mode)
		}
	}
}

func TestTUIDocumentHoverAndEmptyFilterRow(t *testing.T) {
	c := tuiTestCollection(t)
	for i := 0; i < 30; i++ {
		tuiCreate(t, c, fmt.Sprintf("Document %02d", i), map[string]any{})
	}
	m := newTUIModel(c.Path())
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
	m := newTUIModel(c.Path())
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
	tuiWheel(m, g.content.x+3, g.content.y+2, true)
	if m.offset != 3 || m.previewFocus {
		t.Fatal("wheel should scroll hovered pane without changing focus")
	}
	tuiClick(m, g.content.x+3, g.content.y+2)
	if !m.previewFocus {
		t.Fatal("click did not focus preview")
	}
	tuiPress(m, tea.KeyDown)
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
	tuiWheel(m, 3, m.geometry().content.y+1, true)
	if m.offset == 0 {
		t.Fatal("narrow preview did not scroll")
	}
}

func TestTUIMouseToolbarSearchAndFilters(t *testing.T) {
	c := tuiTestCollection(t)
	tuiCreate(t, c, "Alpha", map[string]any{"status": "open"})
	tuiCreate(t, c, "Beta", map[string]any{"status": "closed"})
	m := newTUIModel(c.Path())
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
	tuiRun(t, m, tuiClickButton(t, m, "x"))
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
	m := newTUIModel(c.Path())
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
			if b.toolbar && b.key != "/" && b.key != "f" && b.key != "x" {
				t.Fatalf("unexpected toolbar action: %s", b.key)
			}
			if b.key == "x" {
				clear = true
				if !b.toolbar || b.y != 0 {
					t.Fatal("Clear is not in the top toolbar")
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
	if g := m.geometry(); g.status || g.body.y != g.search.y+g.search.h {
		t.Fatal("search alone reserves an empty status row")
	}
	m.query, m.input, m.mode = "example", "", "documents"
	check(true)
	if m.geometry().status {
		t.Fatal("applied search reserves an empty status row")
	}
	m.query = ""
	m.input, m.mode = "", "documents"
	m.filters["status"] = "open"
	check(true)
}
