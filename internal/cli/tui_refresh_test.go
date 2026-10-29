package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/hmans/means/internal/collection"
)

func tuiPollNow(t *testing.T, m *tuiModel) {
	t.Helper()
	_, cmd := m.Update(tuiChanged{})
	if cmd == nil || !m.refreshing || m.loading {
		t.Fatal("poll did not start a nonblocking refresh")
	}
	_, next := m.Update(cmd())
	if m.refreshing {
		t.Fatal("refresh scheduled an unnecessary check")
	}
	tuiRun(t, m, next)
}

func TestTUIAutomaticRefresh(t *testing.T) {
	c := tuiTestCollection(t)
	doc := tuiCreate(t, c, "Alpha", map[string]any{"tags": []any{"cli"}})
	m := newTUIModel(c.Path)
	m.Update(m.Init()())
	m.offset = 5
	cached := &m.preview[0]
	tuiPollNow(t, m)
	if &m.preview[0] != cached || m.offset != 5 {
		t.Fatal("unchanged snapshot disturbed the preview")
	}
	other := tuiCreate(t, c, "Beta", map[string]any{"tags": []any{"cli"}})
	tuiPollNow(t, m)
	if len(m.docs) != 2 || m.docs[m.selected].ID != doc.ID || m.offset != 5 {
		t.Fatal("new file lost selection or scroll position")
	}
	_, cmd := m.Update(tuiChanged{})
	if m.action("r") != nil {
		t.Fatal("manual refresh overlapped a background check")
	}
	tuiPress(m, '/')
	m.Update(tea.PasteMsg{Content: "Alpha"})
	m.Update(cmd())
	if m.input != "Alpha" || m.mode != "search" || len(m.docs) != 1 {
		t.Fatal("background check blocked or discarded live input")
	}
	newBody := "Externally updated body."
	updated, err := c.Update(doc.ID, doc.Hash, collection.Patch{Body: &newBody})
	if err != nil {
		t.Fatal(err)
	}
	tuiPollNow(t, m)
	if m.mode != "search" || m.input != "Alpha" || len(m.docs) != 1 || !strings.Contains(ansi.Strip(strings.Join(m.preview, "\n")), newBody) {
		t.Fatal("changed file lost live search or retained stale preview")
	}
	tuiPress(m, tea.KeyEsc)
	if len(m.docs) != 2 {
		t.Fatal("cancel did not restore committed search")
	}
	m.field = "tags"
	m.openValues()
	m.filterChoices("cli")
	if _, err := c.Delete(other.ID, other.Hash); err != nil {
		t.Fatal(err)
	}
	tuiPollNow(t, m)
	if m.mode != "values" || m.pickerQuery != "cli" || len(m.choices) != 1 || *m.choices[0].count != 1 {
		t.Fatal("deletion did not update picker counts while retaining input")
	}
	m.choose()
	if _, err := c.Delete(updated.ID, updated.Hash); err != nil {
		t.Fatal(err)
	}
	tuiPollNow(t, m)
	if len(m.docs) != 0 || len(m.preview) != 0 || len(m.filters) != 1 {
		t.Fatal("selected file deletion did not clear preview and preserve filters")
	}
}

func TestTUIAutomaticRefreshRecoveryAndBoundaries(t *testing.T) {
	c := tuiTestCollection(t)
	tuiCreate(t, c, "Root", map[string]any{})
	m := newTUIModel(c.Path)
	tuiRun(t, m, m.Init())
	config := filepath.Join(c.Path, "MEANS.md")
	original, err := os.ReadFile(config)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(config, []byte("broken"), 0600); err != nil {
		t.Fatal(err)
	}
	tuiPollNow(t, m)
	if m.err == nil || len(m.docs) != 1 || m.loading {
		t.Fatal("invalid configuration discarded last usable snapshot")
	}
	if err := os.WriteFile(config, original, 0600); err != nil {
		t.Fatal(err)
	}
	nested, err := collection.Init(filepath.Join(c.Path, "nested"), []byte(`true`))
	if err != nil {
		t.Fatal(err)
	}
	tuiCreate(t, nested, "Nested", map[string]any{})
	tuiPollNow(t, m)
	if m.err != nil || len(m.docs) != 1 {
		t.Fatal("recovery failed or crossed a nested collection boundary")
	}
	if err := os.Remove(filepath.Join(nested.Path, "MEANS.md")); err != nil {
		t.Fatal(err)
	}
	tuiPollNow(t, m)
	if len(m.docs) != 2 {
		t.Fatal("removing a nested boundary did not refresh ownership")
	}
	if err := os.WriteFile(config, []byte("---\nschema:\n  properties:\n    status:\n      enum: [open, closed]\n---\n"), 0600); err != nil {
		t.Fatal(err)
	}
	tuiPollNow(t, m)
	if len(m.fields["status"].values) != 2 {
		t.Fatal("schema choices did not refresh")
	}
}
