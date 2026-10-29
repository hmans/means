package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/hmans/means/internal/collection"
)

func tuiTestCollection(t *testing.T) *collection.Collection {
	t.Helper()
	c, err := collection.Init(t.TempDir(), []byte(`true`))
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func tuiCreate(t *testing.T, c *collection.Collection, title string, data map[string]any) *collection.Document {
	t.Helper()
	doc, err := c.Create(map[string]any{"title": title, "data": data}, "## Preview heading\n\nA **bold** body.\n"+strings.Repeat("More body text.\n\n", 30))
	if err != nil {
		t.Fatal(err)
	}
	return doc
}

func tuiRun(t *testing.T, m *tuiModel, cmd tea.Cmd) {
	t.Helper()
	if cmd == nil {
		return
	}
	msg := cmd()
	if batch, ok := msg.(tea.BatchMsg); ok {
		for _, child := range batch {
			tuiRun(t, m, child)
		}
		return
	}
	_, next := m.Update(msg)
	if m.err != nil {
		t.Fatal(m.err)
	}
	tuiRun(t, m, next)
}

func tuiPress(m *tuiModel, code rune) tea.Cmd {
	key := tea.KeyPressMsg{Code: code}
	if code >= 32 && code <= 126 {
		key.Text = string(code)
	}
	_, cmd := m.Update(key)
	return cmd
}

func TestTUISnapshotFiltersAndPreviewCache(t *testing.T) {
	c := tuiTestCollection(t)
	doc := tuiCreate(t, c, "Alpha", map[string]any{"status": "open", "tags": []any{"cli"}})
	m := newTUIModel(c.Path)
	tuiRun(t, m, m.Init())
	cached := &m.preview[0]
	m.filterText("Alpha")
	if &m.preview[0] != cached {
		t.Fatal("search rerendered an unchanged preview")
	}
	m.offset = len(m.preview) - m.pageHeight()
	m.Update(tea.WindowSizeMsg{Width: m.width, Height: m.height + 10})
	if &m.preview[0] != cached || m.offset > max(0, len(m.preview)-m.pageHeight()) {
		t.Fatal("height-only resize must reuse the preview and clamp scrolling")
	}
	m.Update(tea.WindowSizeMsg{Width: m.width + 1, Height: m.height})
	if &m.preview[0] == cached {
		t.Fatal("width resize did not rerender the preview")
	}
	newBody := "Changed body after refresh."
	if _, err := c.Update(doc.ID, doc.Hash, collection.Patch{Body: &newBody}); err != nil {
		t.Fatal(err)
	}
	tuiCreate(t, c, "Beta", map[string]any{"status": "open", "tags": []any{"cli"}})
	m.field = "status"
	m.openValues()
	m.choice = 1
	if cmd := m.choose(); cmd != nil || m.loading || len(m.docs) != 1 {
		t.Fatal("choosing a filter must use the loaded snapshot synchronously")
	}
	if strings.Contains(ansi.Strip(strings.Join(m.preview, "\n")), newBody) {
		t.Fatal("filtering read external content changes")
	}
	m.field = "tags"
	m.openValues()
	if m.choices[1].count == nil || *m.choices[1].count != 1 {
		t.Fatal("counts did not use the same snapshot as results")
	}
	m.choice = 1
	m.choose()
	if len(m.docs) != 1 {
		t.Fatal("array membership lost snapshot consistency")
	}
	if cmd := m.action("c"); cmd != nil || len(m.docs) != 1 {
		t.Fatal("clearing filters read external changes")
	}
	tuiRun(t, m, m.load())
	if len(m.docs) != 2 || !strings.Contains(ansi.Strip(strings.Join(m.preview, "\n")), newBody) {
		t.Fatal("refresh did not load new documents and invalidate cached content")
	}
	m.filterText("no matching document")
	if len(m.preview) != 0 {
		t.Fatal("empty results retained a cached preview")
	}
	m.filterText("Alpha")
	if !strings.Contains(ansi.Strip(strings.Join(m.preview, "\n")), newBody) {
		t.Fatal("restored selection did not render its preview")
	}
}

func TestTUIBrowseRefreshAndBoundaries(t *testing.T) {
	c := tuiTestCollection(t)
	for i := 0; i < 25; i++ {
		tuiCreate(t, c, fmt.Sprintf("Document %02d", i), map[string]any{})
	}
	if err := os.WriteFile(filepath.Join(c.Path, "broken.md"), []byte("broken"), 0600); err != nil {
		t.Fatal(err)
	}
	nested, err := collection.Init(filepath.Join(c.Path, "nested"), []byte(`true`))
	if err != nil {
		t.Fatal(err)
	}
	tuiCreate(t, nested, "Nested document", map[string]any{})
	m := newTUIModel(c.Path)
	tuiRun(t, m, m.Init())
	if len(m.docs) != 25 || len(m.diagnostics) != 1 {
		t.Fatalf("docs=%d diagnostics=%v", len(m.docs), m.diagnostics)
	}
	m.move(10)
	id := m.docs[m.selected].ID
	preview := ansi.Strip(strings.Join(m.preview, "\n"))
	if !strings.Contains(preview, "Preview heading") || strings.Contains(preview, "**bold**") {
		t.Fatal(preview)
	}
	tuiCreate(t, c, "New document", map[string]any{})
	tuiRun(t, m, tuiPress(m, 'r'))
	if len(m.docs) != 26 || m.docs[m.selected].ID != id {
		t.Fatal("refresh lost selection or new document")
	}
	// Invalid configuration must leave the last usable results available.
	if err := os.WriteFile(filepath.Join(c.Path, "MEANS.md"), []byte("broken"), 0600); err != nil {
		t.Fatal(err)
	}
	cmd := tuiPress(m, 'r')
	m.Update(cmd())
	if m.err == nil || m.loading || len(m.docs) != 26 || !strings.Contains(m.View().Content, "Error") {
		t.Fatal("refresh failure was hidden")
	}
}

func TestTUIFiltersUseCollectionSemantics(t *testing.T) {
	c := tuiTestCollection(t)
	a := tuiCreate(t, c, "Open", map[string]any{"status": "open", "number": json.Number("9007199254740993"), "nullable": nil, "tags": []any{"a", "b"}, "object": map[string]any{"a": true, "b": false}})
	tuiCreate(t, c, "Closed", map[string]any{"status": "closed", "number": json.Number("9007199254740992"), "tags": []any{"a"}})
	m := newTUIModel(c.Path)
	tuiRun(t, m, m.Init())
	// Drive the actual field/value picker, including a JSON null value.
	tuiPress(m, 'f')
	for i, choice := range m.choices {
		if choice.value == "nullable" {
			m.choice = i
		}
	}
	tuiPress(m, tea.KeyEnter)
	if len(m.choices) != 2 || m.choices[1].label != "null" {
		t.Fatalf("choices: %+v", m.choices)
	}
	m.choice = 1
	tuiRun(t, m, tuiPress(m, tea.KeyEnter))
	if len(m.docs) != 1 || m.docs[0].ID != a.ID {
		t.Fatal("null matched a missing field")
	}
	m.openFields()
	if len(m.choices) != 5 {
		t.Fatalf("picker lost fields from other documents: %+v", m.choices)
	}
	m.field = "nullable"
	m.openValues()
	tuiRun(t, m, tuiPress(m, tea.KeyEnter))
	if len(m.filters) != 0 || len(m.docs) != 2 {
		t.Fatal("remove filter failed")
	}
	for _, filter := range []map[string]any{
		{"number": json.Number("9007199254740993")},
		{"status": "open", "tags": []any{"a", "b"}},
		{"object": map[string]any{"a": true}},
	} {
		m.filters = filter
		tuiRun(t, m, m.load())
		if len(m.docs) != 1 || m.docs[0].ID != a.ID {
			t.Fatalf("wrong match for %v: %+v", filter, m.docs)
		}
	}
	m.filters = map[string]any{"status": "open", "tags": []any{"a"}}
	tuiRun(t, m, m.load())
	if len(m.docs) != 0 {
		t.Fatal("array filter matched only part of array")
	}
	tuiRun(t, m, tuiPress(m, 'c'))
	if len(m.docs) != 2 {
		t.Fatal("clear did not restore documents")
	}
}

func TestTUISearchNavigationAndSizing(t *testing.T) {
	c := tuiTestCollection(t)
	tuiCreate(t, c, "Alpha 日本語", map[string]any{})
	tuiCreate(t, c, "Beta", map[string]any{})
	m := newTUIModel(c.Path)
	tuiRun(t, m, m.Init())
	tuiPress(m, '/')
	m.Update(tea.PasteMsg{Content: "ALPHA 日本語"})
	if len(m.docs) != 1 {
		t.Fatal("paste did not filter immediately")
	}
	if cmd := tuiPress(m, tea.KeyEnter); cmd != nil {
		t.Fatal("accepting search triggered a scan")
	}
	if len(m.docs) != 1 {
		t.Fatal("literal Unicode search failed")
	}
	tuiPress(m, '/')
	tuiPress(m, 'q')
	if len(m.docs) != 0 {
		t.Fatal("typing did not filter immediately")
	}
	tuiPress(m, tea.KeyEsc)
	if m.query != "ALPHA 日本語" || len(m.docs) != 1 {
		t.Fatal("cancel changed search")
	}
	tuiPress(m, tea.KeyTab)
	m.move(10)
	if m.offset == 0 {
		t.Fatal("preview did not scroll")
	}
	for _, size := range [][2]int{{120, 30}, {80, 24}, {32, 10}, {1, 1}} {
		m.Update(tea.WindowSizeMsg{Width: size[0], Height: size[1]})
		view := m.View()
		if !view.AltScreen {
			t.Fatal("not using alternate screen")
		}
		lines := strings.Split(view.Content, "\n")
		if len(lines) > size[1] {
			t.Fatalf("view exceeds height: %d", len(lines))
		}
		for _, line := range lines {
			if ansi.StringWidth(line) > size[0] {
				t.Fatalf("line exceeds width %d: %q", size[0], line)
			}
		}
	}
	m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m.diagnostics = []collection.Diagnostic{{Path: "bad.md", Error: strings.Repeat("long error ", 300)}}
	tuiPress(m, 'd')
	m.move(20)
	if m.offset != 20 {
		t.Fatal("wrapped diagnostics do not scroll")
	}
}

func TestTUILiveSearchEditingWithinDataFilters(t *testing.T) {
	c := tuiTestCollection(t)
	a := tuiCreate(t, c, "Alpha", map[string]any{"status": "open"})
	tuiCreate(t, c, "Beta", map[string]any{"status": "open"})
	tuiCreate(t, c, "Alpha closed", map[string]any{"status": "closed"})
	m := newTUIModel(c.Path)
	m.filters["status"] = "open"
	tuiRun(t, m, m.Init())
	// Typing must use the snapshot, even when the on-disk config becomes invalid.
	if err := os.WriteFile(filepath.Join(c.Path, "MEANS.md"), []byte("invalid"), 0600); err != nil {
		t.Fatal(err)
	}
	tuiPress(m, '/')
	for _, key := range "Al" {
		if cmd := tuiPress(m, key); cmd != nil {
			t.Fatal("typing started asynchronous work")
		}
	}
	if len(m.docs) != 1 || m.docs[0].ID != a.ID || m.loading {
		t.Fatal("live search ignored data filters")
	}
	tuiPress(m, 'z')
	if len(m.docs) != 0 {
		t.Fatal("expected no matches")
	}
	tuiPress(m, tea.KeyBackspace)
	if len(m.docs) != 1 {
		t.Fatal("backspace did not restore match")
	}
	m.Update(tea.KeyPressMsg{Code: 'u', Mod: tea.ModCtrl})
	if m.input != "" || len(m.docs) != 2 {
		t.Fatal("clear input did not restore filtered snapshot")
	}
	m.Update(tea.PasteMsg{Content: "**BOLD**"})
	if len(m.docs) != 2 {
		t.Fatal("body search was not literal and case-insensitive")
	}
	tuiPress(m, tea.KeyEnter)
	if m.query != "**BOLD**" || m.mode != "documents" {
		t.Fatal("enter did not retain search")
	}
}

func TestTUIEmptyAndTerminalErrors(t *testing.T) {
	c := tuiTestCollection(t)
	m := newTUIModel(c.Path)
	tuiRun(t, m, m.Init())
	t.Setenv("NO_COLOR", "1")
	if strings.Contains(m.View().Content, "\x1b") {
		t.Fatal("NO_COLOR retained terminal styling")
	}
	if !strings.Contains(m.View().Content, "No matching documents") {
		t.Fatal(m.View().Content)
	}
	tuiPress(m, 'f')
	tuiPress(m, tea.KeyEnter)
	if !strings.Contains(m.View().Content, "No data fields") {
		t.Fatal(m.View().Content)
	}
	for _, args := range [][]string{{"tui", "--path", c.Path}, {"tui", "--path", c.Path, "--json"}} {
		var out, errs bytes.Buffer
		if err := Execute(args, strings.NewReader("q"), &out, &errs, "test"); err == nil {
			t.Fatal("accepted nonterminal/JSON session")
		}
		if out.Len() != 0 {
			t.Fatal("interactive output leaked")
		}
		if args[len(args)-1] == "--json" && !json.Valid(errs.Bytes()) {
			t.Fatal(errs.String())
		}
	}
}

func TestTUISchemaChoicesAndRefresh(t *testing.T) {
	c, err := collection.Init(t.TempDir(), []byte(`{"type":"object","properties":{
		"status":{"title":"Workflow status","description":"Choose the work stage.","enum":["todo","done"]},
		"number":{"enum":[9007199254740993]},"nullable":{"const":null},"free":true,
		"tags":{"type":"array","items":{"enum":["a","b"]}}
	}}`))
	if err != nil {
		t.Fatal(err)
	}
	m := newTUIModel(c.Path)
	tuiRun(t, m, m.Init())
	m.openFields()
	if len(m.choices) != 5 {
		t.Fatalf("empty collection fields: %+v", m.choices)
	}
	for i, choice := range m.choices {
		if choice.value == "status" {
			m.choice = i
		}
	}
	if !strings.Contains(m.View().Content, "Workflow status (status)") || !strings.Contains(m.View().Content, "Choose the work stage.") {
		t.Fatal(m.View().Content)
	}
	tuiPress(m, tea.KeyEnter)
	if len(m.choices) != 3 || m.choices[1].value != "todo" || m.choices[2].value != "done" {
		t.Fatalf("enum order: %+v", m.choices)
	}
	m.choice = 1
	tuiRun(t, m, tuiPress(m, tea.KeyEnter))
	if m.filters["status"] != "todo" || len(m.docs) != 0 {
		t.Fatal("unused enum value was not filterable")
	}
	m.field = "number"
	m.openValues()
	if m.choices[1].value != json.Number("9007199254740993") {
		t.Fatalf("lost number precision: %+v", m.choices)
	}
	m.field = "nullable"
	m.openValues()
	if len(m.choices) != 2 || m.choices[1].label != "null" {
		t.Fatal("null const missing")
	}
	m.field = "tags"
	m.openValues()
	if len(m.choices) != 3 || m.choices[1].value != "a" || !m.choices[1].contains {
		t.Fatal("array item enum choices missing")
	}
	// Direct edits can leave values outside the current schema. Keep those visible.
	if err := os.WriteFile(filepath.Join(c.Path, "legacy.md"), []byte("---\nid: old01\ntitle: Legacy\ndata:\n  status: retired\n  extra: yes\n---\n"), 0600); err != nil {
		t.Fatal(err)
	}
	tuiRun(t, m, m.load())
	m.field = "status"
	m.openValues()
	if len(m.choices) != 4 || m.choices[3].label != `retired (stored)` {
		t.Fatalf("legacy choices: %+v", m.choices)
	}
	m.choice = 3
	tuiRun(t, m, tuiPress(m, tea.KeyEnter))
	if len(m.docs) != 1 || m.docs[0].ID != "old01" {
		t.Fatal("legacy value not filterable")
	}
	m.openFields()
	if len(m.choices) != 6 {
		t.Fatal("stored field not included")
	}
	if err := os.WriteFile(filepath.Join(c.Path, "MEANS.md"), []byte("---\nschema:\n  properties:\n    status:\n      enum: [retired, archived]\n---\n"), 0600); err != nil {
		t.Fatal(err)
	}
	tuiRun(t, m, m.load())
	m.field = "status"
	m.openValues()
	if len(m.choices) != 3 || m.choices[1].value != "retired" || m.choices[2].value != "archived" {
		t.Fatalf("schema refresh/deduplication failed: %+v", m.choices)
	}
}

func TestTUIPickerTypeAhead(t *testing.T) {
	c, err := collection.Init(t.TempDir(), []byte(`{"properties":{
		"status":{"title":"Workflow status","enum":["todo","done","qjk","日本語"]},
		"nullable":{"enum":[null]},"number":{"enum":[9007199254740993]}
	}}`))
	if err != nil {
		t.Fatal(err)
	}
	tuiCreate(t, c, "One", map[string]any{"status": "done", "nullable": nil, "number": json.Number("9007199254740993")})
	m := newTUIModel(c.Path)
	tuiRun(t, m, m.Init())
	tuiPress(m, 'f')
	m.Update(tea.PasteMsg{Content: "WORKFLOW"})
	if len(m.choices) != 1 || m.choices[0].value != "status" {
		t.Fatal("field title type-ahead failed")
	}
	if !strings.Contains(m.View().Content, "Find a field") {
		t.Fatal("missing picker input")
	}
	tuiClick(m, 3, m.geometry().search.y+1)
	if m.mode != "fields" {
		t.Fatal("picker input click changed mode")
	}
	tuiPress(m, tea.KeyEnter)
	if m.pickerQuery != "" || m.mode != "values" || len(m.choices) != 5 {
		t.Fatal("value step did not reset type-ahead")
	}
	for _, key := range "qjk" {
		if cmd := tuiPress(m, key); cmd != nil {
			t.Fatal("typing invoked a command")
		}
	}
	if len(m.choices) != 1 || m.choices[0].value != "qjk" {
		t.Fatal("shortcut letters were not search text")
	}
	tuiRun(t, m, tuiPress(m, tea.KeyEnter))
	if m.filters["status"] != "qjk" {
		t.Fatal("first matching value removed the filter instead of applying it")
	}
	m.field = "status"
	m.openValues()
	m.Update(tea.PasteMsg{Content: "日本語"})
	tuiPress(m, 'x')
	if len(m.choices) != 0 || !strings.Contains(m.View().Content, "No matching choices") {
		t.Fatal("no-match state missing")
	}
	if cmd := tuiPress(m, tea.KeyEnter); cmd != nil {
		t.Fatal("empty picker applied a choice")
	}
	tuiPress(m, tea.KeyBackspace)
	if len(m.choices) != 1 {
		t.Fatal("Unicode backspace failed")
	}
	m.Update(tea.KeyPressMsg{Code: 'u', Mod: tea.ModCtrl})
	if len(m.choices) != 5 || m.pickerQuery != "" {
		t.Fatal("Ctrl+U did not restore choices")
	}
	m.filterChoices("DONE")
	tuiRun(t, m, tuiClick(m, 3, m.geometry().body.y+1))
	if m.filters["status"] != "done" || len(m.docs) != 1 {
		t.Fatal("filtered mouse choice applied wrong value")
	}
	m.field = "nullable"
	m.openValues()
	m.filterChoices("null")
	tuiRun(t, m, tuiPress(m, tea.KeyEnter))
	if value, ok := m.filters["nullable"]; !ok || value != nil {
		t.Fatal("null confused with remove filter")
	}
	m.field = "number"
	m.openValues()
	m.filterChoices("9007199254740993")
	tuiRun(t, m, tuiPress(m, tea.KeyEnter))
	if m.filters["number"] != json.Number("9007199254740993") {
		t.Fatal("type-ahead lost numeric precision")
	}
	m.field = "status"
	m.openValues()
	m.filterChoices("remove")
	tuiRun(t, m, tuiPress(m, tea.KeyEnter))
	if _, ok := m.filters["status"]; ok {
		t.Fatal("remove choice did not remove filter")
	}
	m.openFields()
	m.filterChoices("num")
	tuiPollNow(t, m)
	if m.pickerQuery != "num" || len(m.choices) != 1 {
		t.Fatal("refresh dropped picker query")
	}
	tuiPress(m, tea.KeyEsc)
	if m.mode != "documents" || m.query != "" {
		t.Fatal("cancel changed document search")
	}
}

func TestTUIArrayItemFilters(t *testing.T) {
	c, err := collection.Init(t.TempDir(), []byte(`{"properties":{"tags":{"items":{"enum":["cli","docs","future"]}}}}`))
	if err != nil {
		t.Fatal(err)
	}
	a := tuiCreate(t, c, "Alpha", map[string]any{"status": "open", "tags": []any{"cli", "docs", "cli"}})
	tuiCreate(t, c, "Beta", map[string]any{"status": "closed", "tags": []any{"docs", "cli"}})
	tuiCreate(t, c, "Docs", map[string]any{"status": "open", "tags": []any{"docs"}})
	tuiCreate(t, c, "Empty", map[string]any{"tags": []any{}})
	tuiCreate(t, c, "Missing", map[string]any{})
	tuiCreate(t, c, "Scalar", map[string]any{"tags": "cli"})
	m := newTUIModel(c.Path)
	tuiRun(t, m, m.Init())
	m.field = "tags"
	m.openValues()
	if len(m.choices) != 5 {
		t.Fatalf("expected deduplicated items, unused enum, and distinct scalar: %+v", m.choices)
	}
	if m.choices[1].label != "cli" || !m.choices[1].contains {
		t.Fatal("item is not presented as a tag")
	}
	for _, choice := range m.choices {
		if choice.contains {
			want := map[string]int{"cli": 2, "docs": 3, "future": 0}[choice.value.(string)]
			if choice.count == nil || *choice.count != want {
				t.Fatalf("document count for %v: %v, want %d", choice.value, choice.count, want)
			}
		} else if choice.count != nil {
			t.Fatal("non-item choice has a count")
		}
	}
	if !strings.Contains(ansi.Strip(m.View().Content), "cli (2)") {
		t.Fatal("item count not rendered")
	}
	m.filterChoices("(2)")
	if len(m.choices) != 0 {
		t.Fatal("type-ahead matched presentation-only count")
	}
	m.filterChoices("cli")
	if len(m.choices) != 2 {
		t.Fatal("membership and scalar equality were conflated")
	}
	tuiRun(t, m, tuiPress(m, tea.KeyEnter))
	if len(m.docs) != 2 || !m.contains["tags"] {
		t.Fatal("contains did not match across array combinations")
	}
	if !strings.Contains(ansi.Strip(m.View().Content), "tags contains cli") {
		t.Fatal("membership operator is not visible")
	}
	m.filters["status"] = "open"
	tuiRun(t, m, m.load())
	if len(m.docs) != 1 || m.docs[0].ID != a.ID {
		t.Fatal("contains did not combine with exact filters")
	}
	tuiPress(m, '/')
	m.Update(tea.PasteMsg{Content: "Alpha"})
	if len(m.docs) != 1 {
		t.Fatal("live search lost contains matches")
	}
	tuiPress(m, tea.KeyEnter)
	m.openValues()
	if m.choices[1].count == nil || *m.choices[1].count != 2 {
		t.Fatal("counts changed with active document filters")
	}
	m.filterChoices("future")
	tuiRun(t, m, tuiPress(m, tea.KeyEnter))
	if len(m.docs) != 0 {
		t.Fatal("unused item enum should yield no matches")
	}
	m.openValues()
	m.filterChoices("remove")
	tuiRun(t, m, tuiPress(m, tea.KeyEnter))
	if _, ok := m.contains["tags"]; ok {
		t.Fatal("removing filter left membership active")
	}
	tuiRun(t, m, tuiPress(m, 'c'))
	if len(m.docs) != 6 || len(m.contains) != 0 {
		t.Fatal("clear did not restore all documents")
	}
	m.openValues()
	m.filterChoices("cli")
	tuiCreate(t, c, "New CLI", map[string]any{"tags": []any{"cli", "cli"}})
	tuiPollNow(t, m)
	if m.choices[0].count == nil || *m.choices[0].count != 3 {
		t.Fatal("refresh did not update document count")
	}
}

func TestTUIArrayItemTypes(t *testing.T) {
	c := tuiTestCollection(t)
	items := []any{json.Number("9007199254740993"), json.Number("1"), json.Number("1.0"), "1", nil, true, []any{"nested"}, map[string]any{"key": "value"}}
	tuiCreate(t, c, "Mixed", map[string]any{"items": items})
	m := newTUIModel(c.Path)
	tuiRun(t, m, m.Init())
	m.field = "items"
	m.openValues()
	if len(m.choices) != 8 {
		t.Fatalf("numeric deduplication failed: %+v", m.choices)
	}
	for _, expected := range []any{json.Number("9007199254740993"), json.Number("1.00"), "1", nil, true, []any{"nested"}, map[string]any{"key": "value"}} {
		if !tuiContainsMatches(map[string]any{"items": items}, map[string]any{"items": expected}) {
			t.Fatalf("missing typed item: %#v", expected)
		}
	}
	for _, wrong := range []any{json.Number("9007199254740992"), "true", "null", "nested", map[string]any{}} {
		if tuiContainsMatches(map[string]any{"items": items}, map[string]any{"items": wrong}) {
			t.Fatalf("wrong item matched: %#v", wrong)
		}
	}
	if tuiContainsMatches(map[string]any{"items": []any{"1"}}, map[string]any{"items": json.Number("1")}) {
		t.Fatal("string matched number")
	}
	if tuiContainsMatches(map[string]any{}, map[string]any{"items": nil}) {
		t.Fatal("missing field matched null item")
	}
}
