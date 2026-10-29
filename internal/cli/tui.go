package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/colorprofile"
	"github.com/charmbracelet/x/ansi"
	"github.com/charmbracelet/x/term"
	"github.com/hmans/means/internal/collection"
	"github.com/spf13/cobra"
)

func newTUICommand(opts *options) *cobra.Command {
	return &cobra.Command{
		Use: "tui", Short: "Browse documents and filter data in an interactive terminal",
		Long: `Open a read-only document browser with a Markdown preview. Requires terminal
input and output. Up/down or j/k select documents; Tab switches to preview scrolling.
Click toolbar buttons, documents, or filter choices. Click a pane to focus it;
scroll the mouse wheel over a pane to scroll its content. Drag the divider between
panes to resize them for this session. Narrow terminals use
the View button or Tab to switch panes. Mouse support requires a compatible terminal.
/ enters a live case-insensitive literal title/body search; Enter keeps it,
Esc restores the previous search. Typing filters the loaded documents immediately.
f opens a data field/value picker. Select a field, then a value; filters combine
with AND. Type to narrow fields or values immediately; use arrows to navigate,
Enter or click to choose, Backspace to edit, Ctrl+U to clear, and Esc to cancel.
Letters including j, k, and q are search text in the picker; Ctrl+C still quits.
Fields and enum/const choices come from the schema's direct properties,
including fields with no documents yet. Schema titles and descriptions guide the
picker. Stored fields and values remain available, including schema-invalid data.
v opens collection views from MEANS.md app.views (Expr filters and columns).
Views combine with search/data filters; All documents leaves the selected view.
The first value option removes that field's filter. c clears all filters and search.
r reloads the collection; d shows diagnostics; q or Ctrl+C quits. Esc closes a view.
All documents are loaded in filename order, without a limit. Array fields offer
individual items and match arrays containing the selected item (exact JSON equality).
Choose one item per field; filters on different fields combine with AND. Other data
filters use list/search semantics, including partial object matches.
Missing fields do not match null. Malformed files and duplicate IDs are reported
in diagnostics and excluded. Nested collections, hidden paths, and symlinks are
excluded. Filesystem changes trigger a background refresh after a short debounce.
Refresh checks immediately. Active input, filters, and selection are preserved.
No document writes, embeddings, or search index are needed. --json is supported
for help and errors only; use list/search --json for structured document results.`,
		Example: "  means tui\n  means tui --path ./issues",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if opts.json {
				return fmt.Errorf("tui is interactive and does not support --json; use list or search --json")
			}
			for _, stream := range []any{cmd.InOrStdin(), cmd.OutOrStdout()} {
				f, ok := stream.(interface{ Fd() uintptr })
				if !ok || !term.IsTerminal(f.Fd()) {
					return fmt.Errorf("tui requires terminal input and output; use list or search for pipes")
				}
			}
			c, err := opts.loadCollection()
			if err != nil {
				return err
			}
			m := newTUIModel(c.Path)
			defer func() {
				if m.cancelEvaluation != nil {
					m.cancelEvaluation()
				}
			}()
			watcher, err := newTUIWatcher(c.Path)
			if err != nil {
				return fmt.Errorf("watch collection: %w", err)
			}
			defer watcher.close()
			m.watch = watcher.messages
			programOpts := []tea.ProgramOption{tea.WithInput(cmd.InOrStdin()), tea.WithOutput(cmd.OutOrStdout()), tea.WithContext(cmd.Context())}
			if os.Getenv("NO_COLOR") != "" {
				programOpts = append(programOpts, tea.WithColorProfile(colorprofile.Ascii))
			}
			_, err = tea.NewProgram(m, programOpts...).Run()
			return err
		},
	}
}

type tuiLoaded struct {
	app        *tuiApp
	background bool
	fields     map[string]tuiField
	all        *collection.SearchResult
	err        error
}

type tuiField struct {
	title, description string
	values             []any
	items              []any
	array              bool
}

// Read presentation hints from direct top-level properties. Collection.Load has
// already validated the schema; UseNumber retains exact filter values.
func tuiSchemaFields(schema json.RawMessage) map[string]tuiField {
	var root any
	decoder := json.NewDecoder(bytes.NewReader(schema))
	decoder.UseNumber()
	if err := decoder.Decode(&root); err != nil {
		return nil
	}
	object, _ := root.(map[string]any)
	properties, _ := object["properties"].(map[string]any)
	fields := make(map[string]tuiField, len(properties))
	for name, raw := range properties {
		property, _ := raw.(map[string]any)
		title, _ := property["title"].(string)
		description, _ := property["description"].(string)
		values, _ := property["enum"].([]any)
		if value, ok := property["const"]; ok {
			values = []any{value}
		}
		items, _ := property["items"].(map[string]any)
		itemValues, _ := items["enum"].([]any)
		if value, ok := items["const"]; ok {
			itemValues = []any{value}
		}
		array := property["type"] == "array"
		if types, ok := property["type"].([]any); ok {
			for _, kind := range types {
				if kind == "array" {
					array = true
				}
			}
		}
		fields[name] = tuiField{title: title, description: description, values: values, items: itemValues, array: array}
	}
	return fields
}

type tuiChoice struct {
	label    string
	value    any
	remove   bool
	contains bool
	count    *int // Documents containing a string item; nil for other choices.
}

type tuiModel struct {
	evaluation         uint64
	evaluating         bool
	cancelEvaluation   context.CancelFunc
	app                *tuiApp
	view               string
	viewErrors         []collection.Diagnostic
	path               string
	width, height      int
	listWidth          int // Preferred width in columns; zero uses the default.
	draggingDivider    bool
	dividerGrabOffset  int
	mouseX, mouseY     int
	mousePresent       bool
	all, docs          []collection.SearchHit
	base               []collection.SearchHit // Loaded documents matching the data filters.
	diagnostics        []collection.Diagnostic
	filters            map[string]any
	contains           map[string]bool
	fields             map[string]tuiField
	query, input       string
	selected, offset   int
	listTop, choiceTop int
	preview            []string
	previewID          string
	previewColumns     int
	mode               string // documents, search, fields, values, diagnostics
	previewFocus       bool
	choices            []tuiChoice
	allChoices         []tuiChoice
	pickerQuery        string
	choice             int
	field              string
	loading            bool
	refreshing         bool
	watch              <-chan tea.Msg
	refreshPending     bool
	err                error
}

func newTUIModel(path string) *tuiModel {
	return &tuiModel{path: path, width: 80, height: 24, filters: map[string]any{}, contains: map[string]bool{}, mode: "documents", loading: true}
}

func (m *tuiModel) Init() tea.Cmd {
	if m.watch == nil {
		return m.load()
	}
	return tea.Batch(m.load(), m.waitForChange())
}

type tuiChanged struct{}

func (m *tuiModel) waitForChange() tea.Cmd {
	if m.watch == nil {
		return nil
	}
	ch := m.watch
	return func() tea.Msg { return <-ch }
}

// Refresh runs outside the event loop. Filters use this single loaded snapshot.
func (m *tuiModel) load() tea.Cmd {
	m.loading = true
	return m.readSnapshot(false)
}

func (m *tuiModel) readSnapshot(background bool) tea.Cmd {
	path := m.path
	previousApp := m.app // Compiled apps are immutable and safe to reuse.
	return func() tea.Msg {
		c, err := collection.Load(path)
		if err != nil {
			return tuiLoaded{background: background, err: err}
		}
		all, err := c.Search(collection.SearchOptions{WithBody: true})
		if err != nil {
			return tuiLoaded{background: background, err: err}
		}
		app := compileTUIApp(c.App, previousApp)
		all.Diagnostics = append(all.Diagnostics, app.errors...)
		return tuiLoaded{background: background, fields: tuiSchemaFields(c.Schema), all: all, app: app}
	}
}

func (m *tuiModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tuiEvaluated:
		if msg.generation != m.evaluation {
			return m, nil
		}
		m.evaluating = false
		id, offset, top := "", m.offset, m.listTop
		if len(m.docs) > 0 {
			id = m.docs[m.selected].ID
		}
		m.base, m.viewErrors = msg.docs, msg.errors
		query := m.query
		if m.mode == "search" {
			query = m.input
		}
		m.filterText(query)
		if m.mode == "diagnostics" {
			m.offset = min(offset, max(0, len(m.diagnosticLines())-m.pageHeight()))
		} else if len(m.docs) > 0 && m.docs[m.selected].ID == id {
			m.offset = min(offset, max(0, len(m.preview)-m.pageHeight()))
		}
		if len(m.docs) > 0 && m.docs[m.selected].ID == id {
			m.listTop = min(top, max(0, len(m.docs)-m.pageHeight()))
		}
		return m, nil
	case tuiChanged:
		if m.loading || m.refreshing {
			m.refreshPending = true
			return m, m.waitForChange()
		}
		m.refreshing = true
		return m, tea.Batch(m.readSnapshot(true), m.waitForChange())
	case tuiWatchError:
		m.err = fmt.Errorf("filesystem watch: %w; r refreshes manually", msg.err)
		return m, m.waitForChange()
	case tea.WindowSizeMsg:
		m.draggingDivider = false
		m.width, m.height = max(1, msg.Width), max(1, msg.Height)
		m.renderPreview()
		m.listTop = tuiKeepVisible(m.selected, m.listTop, m.pageHeight(), len(m.docs))
		m.choiceTop = tuiKeepVisible(m.choice, m.choiceTop, m.pickerHeight(), len(m.choices))
		if m.mode == "diagnostics" {
			m.offset = min(m.offset, max(0, len(m.diagnosticLines())-m.pageHeight()))
		}
	case tuiLoaded:
		var next tea.Cmd
		m.refreshing = false
		m.loading, m.err = false, msg.err
		if m.refreshPending {
			m.refreshPending, m.refreshing = false, true
			next = m.readSnapshot(true)
		}
		if msg.err != nil {
			return m, next
		}
		if msg.background && m.app != nil && msg.app != nil && m.app.source == msg.app.source && reflect.DeepEqual(m.all, msg.all.Documents) && reflect.DeepEqual(m.diagnostics, msg.all.Diagnostics) && reflect.DeepEqual(m.fields, msg.fields) {
			return m, next
		}
		id, offset, listTop := "", m.offset, m.listTop
		if len(m.docs) > 0 {
			id = m.docs[m.selected].ID
		}
		var chosen *tuiChoice
		choiceTop := m.choiceTop
		if m.picking() && len(m.choices) > 0 {
			value := m.choices[m.choice]
			chosen = &value
		}
		m.all, m.diagnostics = msg.all.Documents, msg.all.Diagnostics
		m.fields = msg.fields
		m.app = msg.app
		if view := m.currentView(); view == nil || view.compileErr != nil {
			m.view = ""
		}
		m.previewID = "" // Refresh may change content without changing its ID.
		evaluate := m.filterData()
		if m.mode == "diagnostics" {
			m.offset = min(offset, max(0, len(m.diagnosticLines())-m.pageHeight()))
		} else if len(m.docs) > 0 && m.docs[m.selected].ID == id {
			m.offset = min(offset, max(0, len(m.preview)-m.pageHeight()))
		}
		if len(m.docs) > 0 && m.docs[m.selected].ID == id {
			m.listTop = min(listTop, max(0, len(m.docs)-m.pageHeight()))
		}
		pickerQuery := m.pickerQuery
		if m.mode == "fields" {
			m.openFields()
			m.filterChoices(pickerQuery)
		} else if m.mode == "values" {
			m.openValues()
			m.filterChoices(pickerQuery)
		} else if m.mode == "views" {
			m.openViews()
			m.filterChoices(pickerQuery)
		}
		if m.picking() && chosen != nil {
			for i, choice := range m.choices {
				if choice.remove == chosen.remove && choice.contains == chosen.contains && tuiEqual(choice.value, chosen.value) {
					m.choice = i
					m.choiceTop = min(choiceTop, max(0, len(m.choices)-m.pickerHeight()))
					break
				}
			}
		}
		return m, tea.Batch(next, evaluate)
	case tea.PasteMsg:
		if m.mode == "search" && !m.loading {
			m.input += terminalCell(msg.Content)
			m.filterText(m.input)
		} else if m.picking() && !m.loading {
			m.filterChoices(m.pickerQuery + terminalCell(msg.Content))
		}
	case tea.MouseClickMsg:
		m.mouseX, m.mouseY, m.mousePresent = msg.X, msg.Y, true
		m.draggingDivider = false
		return m, m.mouse(msg.Mouse(), false)
	case tea.MouseMotionMsg:
		m.mouseX, m.mouseY, m.mousePresent = msg.X, msg.Y, true
		if m.draggingDivider && msg.Button == tea.MouseLeft {
			m.resizeDivider(msg.X)
		}
	case tea.MouseReleaseMsg:
		if m.draggingDivider {
			m.resizeDivider(msg.X)
			m.draggingDivider = false
		}
	case tea.MouseWheelMsg:
		m.mouseX, m.mouseY, m.mousePresent = msg.X, msg.Y, true
		return m, m.mouse(msg.Mouse(), true)
	case tea.KeyPressMsg:
		m.draggingDivider = false
		key := msg.String()
		if key == "ctrl+c" || (key == "q" && m.mode != "search" && !m.picking()) {
			return m, tea.Quit
		}
		if m.loading {
			return m, nil
		}
		if m.mode == "search" {
			switch key {
			case "esc":
				m.mode = "documents"
				m.filterText(m.query)
				return m, nil
			case "enter":
				m.query, m.mode = m.input, "documents"
				return m, nil
			case "backspace":
				runes := []rune(m.input)
				if len(runes) > 0 {
					m.input = string(runes[:len(runes)-1])
				}
			case "ctrl+u":
				m.input = ""
			default:
				m.input += msg.Text
			}
			m.filterText(m.input)
			return m, nil
		}
		if key == "esc" {
			if m.mode == "views" {
				m.mode = "documents"
				return m, nil
			}
			m.mode, m.previewFocus, m.offset = "documents", false, 0
			return m, nil
		}
		if m.picking() {
			switch key {
			case "up":
				m.choice = max(0, m.choice-1)
			case "down":
				m.choice = min(max(0, len(m.choices)-1), m.choice+1)
			case "pgup":
				m.choice = max(0, m.choice-m.pickerHeight())
			case "pgdown":
				m.choice = min(max(0, len(m.choices)-1), m.choice+m.pickerHeight())
			case "enter":
				return m, m.choose()
			case "backspace":
				runes := []rune(m.pickerQuery)
				if len(runes) > 0 {
					m.filterChoices(string(runes[:len(runes)-1]))
				}
			case "ctrl+u":
				m.filterChoices("")
			default:
				if msg.Text != "" {
					m.filterChoices(m.pickerQuery + msg.Text)
				}
			}
			m.choiceTop = tuiKeepVisible(m.choice, m.choiceTop, m.pickerHeight(), len(m.choices))
			return m, nil
		}
		return m, m.action(key)
	}
	return m, nil
}

// Toolbar buttons and keyboard shortcuts share the same actions.
func (m *tuiModel) action(key string) tea.Cmd {
	switch key {
	case "q":
		return tea.Quit
	case "r":
		if m.refreshing || m.loading {
			return nil
		}
		return m.load()
	case "/":
		m.mode, m.input = "search", m.query
		m.previewFocus = false
	case "f":
		m.openFields()
	case "v":
		m.openViews()
	case "c":
		m.filters, m.query, m.mode = map[string]any{}, "", "documents"
		m.contains = map[string]bool{}
		return m.filterData()
	case "d":
		if m.mode == "diagnostics" {
			m.mode = "documents"
		} else {
			m.mode = "diagnostics"
		}
		m.offset = 0
	case "tab", "enter":
		m.previewFocus = !m.previewFocus
		m.mode = "documents"
	case "up", "k":
		m.move(-1)
	case "down", "j":
		m.move(1)
	case "pgup":
		m.move(-m.pageHeight())
	case "pgdown":
		m.move(m.pageHeight())
	case "home":
		m.move(-int(^uint(0) >> 2))
	case "end":
		m.move(int(^uint(0) >> 2))
	}
	return nil
}

func (m *tuiModel) choose() tea.Cmd {
	if len(m.choices) == 0 {
		return nil
	}
	if m.mode == "views" {
		chosen := m.choices[m.choice].value.(string)
		if m.app != nil {
			for _, view := range m.app.Views {
				if view.Title == chosen && view.compileErr != nil {
					return nil
				}
			}
		}
		m.view = chosen
		m.mode, m.previewFocus = "documents", false
		return m.filterData()
	}
	if m.mode == "fields" {
		m.field = m.choices[m.choice].value.(string)
		m.openValues()
		return nil
	}
	if m.choices[m.choice].remove {
		delete(m.filters, m.field)
		delete(m.contains, m.field)
	} else {
		m.filters[m.field] = m.choices[m.choice].value
		m.contains[m.field] = m.choices[m.choice].contains
	}
	m.mode, m.previewFocus = "documents", false
	return m.filterData()
}

type tuiEvaluated struct {
	generation uint64
	docs       []collection.SearchHit
	errors     []collection.Diagnostic
}

func (m *tuiModel) filterData() tea.Cmd {
	m.evaluation++
	if m.cancelEvaluation != nil {
		m.cancelEvaluation()
		m.cancelEvaluation = nil
	}
	m.evaluating = false
	m.viewErrors = nil
	view, now := m.currentView(), time.Now()
	filters, contains := map[string]any{}, map[string]any{}
	for key, value := range m.filters {
		if m.contains[key] {
			contains[key] = value
		} else {
			filters[key] = value
		}
	}
	if view != nil {
		ctx, cancel := context.WithCancel(context.Background())
		m.cancelEvaluation, m.evaluating = cancel, true
		generation, docs := m.evaluation, m.all
		return func() tea.Msg {
			defer cancel()
			result := tuiEvaluated{generation: generation}
			for _, doc := range docs {
				if ctx.Err() != nil {
					return nil
				}
				matches, err := view.matches(doc, now)
				if err != nil {
					result.errors = append(result.errors, collection.Diagnostic{Path: tuiAppLocation, Error: fmt.Sprintf("view %q, document %s: %v", view.Title, doc.ID, err)})
					continue
				}
				data := doc.Frontmatter["data"].(map[string]any)
				if matches && collection.MatchesFilter(data, filters) && tuiContainsMatches(data, contains) {
					result.docs = append(result.docs, doc)
				}
			}
			return result
		}
	}
	m.base = nil
	for _, doc := range m.all {
		data := doc.Frontmatter["data"].(map[string]any)
		if collection.MatchesFilter(data, filters) && tuiContainsMatches(data, contains) {
			m.base = append(m.base, doc)
		}
	}
	query := m.query
	if m.mode == "search" {
		query = m.input
	}
	m.filterText(query)
	return nil
}

// Search the loaded snapshot so typing never waits for a scan or loses keys to
// an in-flight request. Use the same Unicode case folding as literal search.
func (m *tuiModel) filterText(query string) {
	id := ""
	if len(m.docs) > 0 {
		id = m.docs[m.selected].ID
	}
	pattern := regexp.MustCompile("(?i)" + regexp.QuoteMeta(query))
	m.docs = nil
	for _, doc := range m.base {
		if pattern.MatchString(doc.Frontmatter["title"].(string)) || (doc.Body != nil && pattern.MatchString(*doc.Body)) {
			m.docs = append(m.docs, doc)
		}
	}
	m.selected, m.offset = 0, 0
	for i, doc := range m.docs {
		if doc.ID == id {
			m.selected = i
			break
		}
	}
	m.listTop = tuiKeepVisible(m.selected, m.listTop, m.pageHeight(), len(m.docs))
	m.renderPreview()
}

func (m *tuiModel) pageHeight() int { return max(1, m.geometry().body.h-2) }

func (m *tuiModel) move(delta int) {
	if m.mode == "diagnostics" {
		m.offset = max(0, min(max(0, len(m.diagnosticLines())-m.pageHeight()), m.offset+delta))
	} else if m.previewFocus {
		m.offset = max(0, min(max(0, len(m.preview)-m.pageHeight()), m.offset+delta))
	} else {
		m.selected = max(0, min(max(0, len(m.docs)-1), m.selected+delta))
		m.listTop = tuiKeepVisible(m.selected, m.listTop, m.pageHeight(), len(m.docs))
		m.offset = 0
		m.renderPreview()
	}
}

func (m *tuiModel) openFields() {
	fields := map[string]bool{}
	for key := range m.fields {
		fields[key] = true
	}
	for _, doc := range m.all {
		for key := range doc.Frontmatter["data"].(map[string]any) {
			fields[key] = true
		}
	}
	for key := range m.filters {
		fields[key] = true
	}
	keys := make([]string, 0, len(fields))
	for key := range fields {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	m.choices = nil
	for _, key := range keys {
		label := m.fieldLabel(key)
		if value, ok := m.filters[key]; ok {
			label += " " + m.filterOperator(key) + " " + tuiValueLabel(value)
		}
		m.choices = append(m.choices, tuiChoice{label: label, value: key})
	}
	m.mode, m.choice, m.choiceTop = "fields", 0, 0
	m.allChoices = m.choices
	m.filterChoices("")
}

func (m *tuiModel) openValues() {
	m.choices = []tuiChoice{{label: "(Any value — remove filter)", remove: true}}
	field := m.fields[m.field]
	var hints, stored []tuiChoice
	counts := map[string]int{}
	array := field.array || len(field.items) > 0
	// Flatten the field's array exactly one level. Nested arrays/objects remain
	// typed items, not recursively flattened scalars.
	add := func(target *[]tuiChoice, value any) {
		if items, ok := value.([]any); ok {
			array = true
			for _, item := range items {
				*target = append(*target, tuiChoice{value: item, contains: true})
			}
		} else {
			*target = append(*target, tuiChoice{value: value})
		}
	}
	for _, value := range field.values {
		add(&hints, value)
	}
	for _, value := range field.items {
		hints = append(hints, tuiChoice{value: value, contains: true})
	}
	for _, doc := range m.all {
		if value, ok := doc.Frontmatter["data"].(map[string]any)[m.field]; ok {
			add(&stored, value)
			items, _ := value.([]any)
			seen := map[string]bool{}
			for _, item := range items {
				if text, ok := item.(string); ok && !seen[text] {
					counts[text]++
					seen[text] = true
				}
			}
		}
	}
	sort.SliceStable(stored, func(i, j int) bool {
		return tuiJSON(stored[i].value) < tuiJSON(stored[j].value)
	})
	appendChoice := func(choice tuiChoice, observed bool) {
		for _, existing := range m.choices[1:] {
			if existing.contains == choice.contains && tuiEqual(existing.value, choice.value) {
				return
			}
		}
		choice.label = tuiValueLabel(choice.value)
		if text, ok := choice.value.(string); ok && choice.contains {
			count := counts[text]
			choice.count = &count
		}
		if array && !choice.contains {
			choice.label += " (value)"
		}
		if observed && len(hints) > 0 {
			choice.label += " (stored)"
		}
		m.choices = append(m.choices, choice)
	}
	for _, choice := range hints {
		appendChoice(choice, false)
	}
	for _, choice := range stored {
		appendChoice(choice, true)
	}
	m.mode, m.choice, m.choiceTop = "values", 0, 0
	m.allChoices = m.choices
	m.filterChoices("")
}

func (m *tuiModel) filterOperator(field string) string {
	if m.contains[field] {
		return "contains"
	}
	return "="
}

func tuiContainsMatches(data, filters map[string]any) bool {
	for field, expected := range filters {
		items, ok := data[field].([]any)
		if !ok {
			return false
		}
		found := false
		for _, item := range items {
			if tuiEqual(item, expected) {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

// Exact JSON equality for array membership (objects must also match in full).
// Compare numbers by value without converting through float64.
func tuiEqual(a, b any) bool {
	switch a := a.(type) {
	case json.Number:
		b, ok := b.(json.Number)
		if !ok {
			return false
		}
		x, xOK := new(big.Rat).SetString(string(a))
		y, yOK := new(big.Rat).SetString(string(b))
		return xOK && yOK && x.Cmp(y) == 0
	case []any:
		b, ok := b.([]any)
		if !ok || len(a) != len(b) {
			return false
		}
		for i := range a {
			if !tuiEqual(a[i], b[i]) {
				return false
			}
		}
		return true
	case map[string]any:
		b, ok := b.(map[string]any)
		if !ok || len(a) != len(b) {
			return false
		}
		for key, value := range a {
			other, exists := b[key]
			if !exists || !tuiEqual(value, other) {
				return false
			}
		}
		return true
	default:
		return reflect.DeepEqual(a, b)
	}
}

func (m *tuiModel) picking() bool {
	return m.mode == "fields" || m.mode == "values" || m.mode == "views"
}

func (m *tuiModel) filterChoices(query string) {
	m.pickerQuery, m.choice, m.choiceTop = query, 0, 0
	pattern := regexp.MustCompile("(?i)" + regexp.QuoteMeta(query))
	m.choices = nil
	for _, choice := range m.allChoices {
		if pattern.MatchString(choice.label) {
			m.choices = append(m.choices, choice)
		}
	}
}

func (m *tuiModel) fieldLabel(key string) string {
	if title := m.fields[key].title; strings.TrimSpace(title) != "" && title != key {
		return title + " (" + key + ")"
	}
	return key
}

func tuiJSON(value any) string {
	data, _ := json.Marshal(value) // Collection data is already JSON-compatible.
	return string(data)
}

// Ordinary strings read as labels. Retain JSON notation for empty strings,
// significant whitespace, and strings that could be mistaken for typed values.
func tuiValueLabel(value any) string {
	if text, ok := value.(string); ok && text != "" && terminalCell(text) == text && !json.Valid([]byte(text)) {
		return text
	}
	return tuiJSON(value)
}

func (m *tuiModel) diagnosticLines() []string {
	var lines []string
	for _, diagnostic := range m.diagnostics {
		text := terminalCell(diagnostic.Path + ": " + diagnostic.Error)
		lines = append(lines, strings.Split(ansi.Hardwrap(text, max(1, m.width-4), true), "\n")...)
	}
	for _, diagnostic := range m.viewErrors {
		text := terminalCell(diagnostic.Path + ": " + diagnostic.Error)
		lines = append(lines, strings.Split(ansi.Hardwrap(text, max(1, m.width-4), true), "\n")...)
	}
	return lines
}

func (m *tuiModel) previewWidth() int {
	return max(1, m.geometry().preview.w-4)
}

func (m *tuiModel) renderPreview() {
	defer func() {
		if m.mode != "diagnostics" {
			m.offset = min(m.offset, max(0, len(m.preview)-m.pageHeight()))
		}
	}()
	if len(m.docs) == 0 {
		m.preview, m.previewID = nil, ""
		return
	}
	hit := m.docs[m.selected]
	width := m.previewWidth()
	if m.previewID == hit.ID && m.previewColumns == width {
		return
	}
	m.previewID, m.previewColumns = hit.ID, width
	body := ""
	if hit.Body != nil {
		body = *hit.Body
	}
	text, err := renderDocumentLayout(width, &collection.Document{ID: hit.ID, Frontmatter: hit.Frontmatter, Body: body}, true)
	if err != nil {
		text = "Preview error: " + terminalCell(err.Error())
	}
	metadata, _ := json.MarshalIndent(hit.Frontmatter["data"], "", "  ")
	text += "\n" + renderPreviewData(width, metadata)
	m.preview = strings.Split(text, "\n")
}
