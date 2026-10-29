package cli

import (
	"fmt"
	"os"
	"sort"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

// Rendering and mouse hit testing use the same terminal-cell geometry.
type tuiRect struct{ x, y, w, h int }

func (r tuiRect) contains(x, y int) bool {
	return x >= r.x && x < r.x+r.w && y >= r.y && y < r.y+r.h
}

func (r tuiRect) inside(x, y int) bool {
	return x > r.x && x < r.x+r.w-1 && y > r.y && y < r.y+r.h-1
}

type tuiButton struct {
	tuiRect
	label, hint, key string
	toolbar          bool
}

type tuiGeometry struct {
	search, body, list, preview tuiRect
	buttons                     []tuiButton
	split, small                bool
}

func (m *tuiModel) geometry() tuiGeometry {
	g := tuiGeometry{split: m.width >= 90, small: m.width < 32 || m.height < 12}
	x, y := 1, 0
	const buttonHeight = 1
	buttons := []struct{ label, hint, key string }{
		{"Search", "/", "/"}, {"Filter", "f", "f"},
	}
	if m.app != nil && len(m.app.Views) > 0 {
		buttons = append(buttons, struct{ label, hint, key string }{"Views ▾", "v", "v"})
	}
	for _, item := range buttons {
		b := tuiControl(item.label, item.hint, item.key, m.width >= 60)
		width := b.w
		if x > 1 && x+width > m.width {
			x, y = 1, y+buttonHeight
		}
		b.x, b.y, b.toolbar = x, y, true
		g.buttons = append(g.buttons, b)
		x += width + 1
	}
	g.search = tuiRect{0, y + buttonHeight, m.width, 3}
	g.body = tuiRect{0, g.search.y + 4, m.width, max(3, m.height-g.search.y-5)}
	g.list, g.preview = g.body, g.body
	if g.split {
		g.list.w = (m.width - 1) / 2
		if m.listWidth > 0 {
			g.list.w = max(32, min(m.listWidth, m.width-33))
		}
		g.preview.x = g.list.w + 1
		g.preview.w = m.width - g.preview.x
	}
	footer := []tuiButton{
		tuiControl("View", "Tab", "tab", m.width >= 64),
		tuiControl("Diagnostics", "d", "d", m.width >= 64),
		tuiControl("Back", "Esc", "esc", m.width >= 64),
		tuiControl("Quit", "q", "q", m.width >= 64),
	}
	if m.width < 64 {
		footer[1] = tuiControl("Diag", "d", "d", false)
	}
	x = m.width
	for i := len(footer) - 1; i >= 0; i-- {
		x -= footer[i].w
		footer[i].x, footer[i].y = x, m.height-1
		g.buttons = append(g.buttons, footer[i])
		x--
	}
	if len(m.filters) > 0 || m.query != "" || (m.mode == "search" && m.input != "") {
		b := tuiControl("Clear", "c", "c", m.width >= 60)
		b.x, b.y = m.width-b.w, g.body.y-1
		g.buttons = append(g.buttons, b)
	}
	return g
}

func tuiControl(label, hint, key string, showHint bool) tuiButton {
	if !showHint {
		hint = ""
	}
	width := ansi.StringWidth(label) + 2
	if hint != "" {
		width += ansi.StringWidth(hint) + 2
	}
	return tuiButton{tuiRect: tuiRect{w: width, h: 1}, label: label, hint: hint, key: key}
}

func tuiKeepVisible(selected, top, height, total int) int {
	if selected < top {
		top = selected
	}
	if selected >= top+height {
		top = selected - height + 1
	}
	return max(0, min(max(0, total-height), top))
}

func (m *tuiModel) mouse(mouse tea.Mouse, wheel bool) tea.Cmd {
	g := m.geometry()
	if g.small {
		return nil
	}
	if m.mode == "views" && !m.loading {
		menu := m.viewMenu()
		if menu.contains(mouse.X, mouse.Y) {
			if wheel {
				delta := 0
				if mouse.Button == tea.MouseWheelUp {
					delta = -3
				}
				if mouse.Button == tea.MouseWheelDown {
					delta = 3
				}
				m.choiceTop = max(0, min(max(0, len(m.choices)-m.pickerHeight()), m.choiceTop+delta))
			} else if mouse.Button == tea.MouseLeft && menu.inside(mouse.X, mouse.Y) && mouse.Y >= menu.y+2 {
				index := m.choiceTop + mouse.Y - menu.y - 2
				if index < len(m.choices) {
					m.choice = index
					return m.choose()
				}
			}
			return nil
		}
		if !wheel && mouse.Button == tea.MouseLeft {
			m.mode = "documents"
		}
		return nil // Dismiss without clicking through to the underlying document.
	}
	if !wheel {
		if mouse.Button != tea.MouseLeft {
			return nil
		}
		for _, button := range g.buttons {
			if !button.contains(mouse.X, mouse.Y) {
				continue
			}
			if button.key == "q" {
				return tea.Quit
			}
			if m.loading {
				return nil
			}
			if button.key == "esc" {
				_, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEsc})
				return cmd
			}
			if m.mode == "search" {
				if button.key == "/" {
					return nil
				}
				m.query, m.mode = m.input, "documents"
			}
			return m.action(button.key)
		}
	}
	if m.loading {
		return nil
	}
	if !wheel && g.split && (m.mode == "documents" || m.mode == "search") &&
		mouse.X >= g.list.w-1 && mouse.X <= g.preview.x &&
		mouse.Y >= g.body.y && mouse.Y < g.body.y+g.body.h {
		m.draggingDivider = true
		m.dividerGrabOffset = mouse.X - g.list.w
		return nil
	}
	if !wheel && g.search.contains(mouse.X, mouse.Y) {
		if m.picking() && m.mode != "views" {
			return nil
		}
		if m.mode != "search" {
			return m.action("/")
		}
		return nil
	}
	if !g.body.inside(mouse.X, mouse.Y) {
		return nil
	}
	delta := 0
	if wheel {
		switch mouse.Button {
		case tea.MouseWheelUp:
			delta = -3
		case tea.MouseWheelDown:
			delta = 3
		default:
			return nil
		}
	}
	if m.picking() {
		if wheel {
			m.choiceTop = max(0, min(max(0, len(m.choices)-m.pageHeight()), m.choiceTop+delta))
			return nil
		}
		index := m.choiceTop + mouse.Y - g.body.y - 1
		if index < len(m.choices) {
			m.choice = index
			return m.choose()
		}
		return nil
	}
	if m.mode == "diagnostics" {
		if wheel {
			m.move(delta)
		}
		return nil
	}
	list := (!g.split && !m.previewFocus) || (g.split && g.list.inside(mouse.X, mouse.Y))
	preview := (!g.split && m.previewFocus) || (g.split && g.preview.inside(mouse.X, mouse.Y))
	if !list && !preview {
		return nil
	}
	if wheel {
		if list {
			m.listTop = max(0, min(max(0, len(m.docs)-m.pageHeight()), m.listTop+delta))
		} else {
			m.offset = max(0, min(max(0, len(m.preview)-m.pageHeight()), m.offset+delta))
		}
		return nil
	}
	if m.mode == "search" {
		m.query = m.input
	}
	m.mode, m.previewFocus = "documents", preview
	if list {
		index := m.listTop + mouse.Y - g.list.y - 1
		if index < len(m.docs) {
			m.selected, m.offset = index, 0
			m.renderPreview()
		}
	}
	return nil
}

func (m *tuiModel) resizeDivider(x int) {
	g := m.geometry()
	if !g.split || g.small {
		m.draggingDivider = false
		return
	}
	m.listWidth = max(32, min(x-m.dividerGrabOffset, m.width-33))
	m.renderPreview()
}

var (
	tuiAccent      = lipgloss.NewStyle().Foreground(lipgloss.Color("37"))
	tuiTitle       = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("231"))
	tuiMuted       = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	tuiSelected    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("231")).Background(lipgloss.Color("24"))
	tuiButtonStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("252")).Background(lipgloss.Color("238"))
)

func tuiFit(text string, width int) string {
	if width <= 0 {
		return ""
	}
	text = ansi.Truncate(text, width, "…")
	return text + strings.Repeat(" ", max(0, width-ansi.StringWidth(text)))
}

func tuiSelectedRow(text string, selected bool, width int) string {
	prefix := "  "
	if selected {
		prefix = "> "
	}
	text = tuiFit(prefix+terminalCell(text), width)
	if selected {
		return tuiSelected.Render(text)
	}
	return text
}

func tuiDocumentRow(id, title string, selected bool, width int) string {
	prefix := "  "
	idStyle := tuiAccent
	titleStyle := tuiTitle
	rowStyle := lipgloss.NewStyle()
	if selected {
		prefix = "> "
		background := lipgloss.Color("236")
		rowStyle = rowStyle.Bold(true).Background(background)
		idStyle = idStyle.Background(background)
		titleStyle = titleStyle.Background(background)
	}
	return rowStyle.Render(tuiFit(prefix+idStyle.Render(terminalCell(id))+"  "+titleStyle.Render(terminalCell(title)), width))
}

func tuiChoiceRow(choice tuiChoice, selected bool, width int) string {
	if choice.count == nil {
		return tuiSelectedRow(choice.label, selected, width)
	}
	suffix := fmt.Sprintf(" (%d)", *choice.count)
	label := ansi.Truncate(terminalCell(choice.label), max(0, width-2-ansi.StringWidth(suffix)), "…")
	if selected {
		return tuiSelectedRow(label+suffix, true, width)
	}
	return tuiFit("  "+label+tuiMuted.Render(suffix), width)
}

func tuiPane(r tuiRect, title string, lines []string, focused bool, position string) []string {
	border := tuiMuted
	if focused {
		border = tuiAccent
	}
	label := ansi.Truncate(" "+terminalCell(title)+" ", max(0, r.w-2), "…")
	rows := []string{border.Render("╭" + label + strings.Repeat("─", max(0, r.w-2-ansi.StringWidth(label))) + "╮")}
	for i := 0; i < r.h-2; i++ {
		line := ""
		if i < len(lines) {
			line = lines[i]
		}
		rows = append(rows, border.Render("│")+" "+tuiFit(line, r.w-4)+" "+border.Render("│"))
	}
	if position != "" {
		position = ansi.Truncate(" "+position+" ", max(0, r.w-2), "…")
	}
	rows = append(rows, border.Render("╰"+strings.Repeat("─", max(0, r.w-2-ansi.StringWidth(position)))+position+"╯"))
	return rows
}

func tuiPosition(top, height, total int) string {
	if total == 0 {
		return "0 items"
	}
	return fmt.Sprintf("%d–%d / %d", top+1, min(top+height, total), total)
}

func (m *tuiModel) View() tea.View {
	g := m.geometry()
	frame := make([]string, m.height)
	if g.small {
		frame[0] = "Resize to at least 32 × 12. q quits."
	} else {
		query := m.query
		searchTitle := "Search"
		if m.mode == "search" {
			query = m.input
		}
		if m.picking() && m.mode != "views" {
			query = m.pickerQuery
			searchTitle = "Find a field"
			if m.mode == "values" {
				searchTitle = "Find a value"
			}
		}
		search := terminalCell(query)
		if m.mode == "search" || (m.picking() && m.mode != "views") {
			search += "█"
			// Keep the insertion point visible for long searches.
			width := max(1, g.search.w-4)
			search = ansi.Cut(search, max(0, ansi.StringWidth(search)-width), ansi.StringWidth(search))
		} else if search == "" {
			search = tuiMuted.Render("Search titles and bodies…  / or click to type")
		}
		// The search field is one bordered row; its bottom border has no counter.
		rows := tuiPane(g.search, searchTitle, []string{search}, m.mode == "search" || (m.picking() && m.mode != "views"), "")
		copy(frame[g.search.y:], rows)
		keys := make([]string, 0, len(m.filters))
		for key := range m.filters {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		filters := make([]string, 0, len(keys))
		chip := lipgloss.NewStyle().Background(lipgloss.Color("236"))
		for _, key := range keys {
			filters = append(filters,
				chip.Foreground(lipgloss.Color("37")).Render(" "+terminalCell(key))+
					chip.Foreground(lipgloss.Color("245")).Render(" "+m.filterOperator(key)+" ")+
					chip.Foreground(lipgloss.Color("99")).Bold(true).Render(terminalCell(tuiValueLabel(m.filters[key]))+" "))
		}
		status := tuiMuted.Render("No data filters")
		if len(filters) > 0 {
			status = tuiMuted.Render("Filters ") + strings.Join(filters, " ")
		}
		if m.view != "" {
			status = tuiAccent.Render(terminalCell(m.view)) + "  " + status
		}
		if len(m.viewErrors) > 0 {
			status += tuiMuted.Render(fmt.Sprintf(" · %d view errors (d)", len(m.viewErrors)))
		}
		if m.loading {
			status = tuiMuted.Render("Loading…")
		}
		if m.evaluating {
			status += tuiMuted.Render(" · Updating view…")
		}
		if m.err != nil {
			status = tuiMuted.Render("Error (previous results; r retries): " + terminalCell(m.err.Error()))
		}
		frame[g.body.y-1] = " " + status
		help := " Wheel scroll · ↑↓ move"
		var body []string
		switch m.mode {
		case "fields", "values":
			title := "Filter / choose a field"
			field := m.field
			if m.mode == "fields" && len(m.choices) > 0 {
				field = m.choices[m.choice].value.(string)
			}
			if m.mode == "values" {
				title = "Filter / " + m.fieldLabel(m.field)
			}
			if description := m.fields[field].description; description != "" {
				frame[g.body.y-1] = tuiMuted.Render(" " + terminalCell(description))
			}
			var lines []string
			for i := m.choiceTop; i < min(len(m.choices), m.choiceTop+m.pageHeight()); i++ {
				lines = append(lines, tuiChoiceRow(m.choices[i], i == m.choice, g.body.w-4))
			}
			if len(lines) == 0 {
				lines = []string{"No data fields in this collection."}
				if m.pickerQuery != "" {
					lines = []string{"No matching choices. Backspace or Ctrl+U to clear."}
				}
			}
			body = tuiPane(g.body, title, lines, true, tuiPosition(m.choiceTop, m.pageHeight(), len(m.choices)))
			help = " Type to find · ↑↓ · Enter chooses"
		case "diagnostics":
			lines := m.diagnosticLines()
			total := len(lines)
			lines = lines[min(m.offset, len(lines)):]
			if len(lines) == 0 {
				lines = []string{"No diagnostics. Your documents are ready to browse."}
			}
			body = tuiPane(g.body, fmt.Sprintf("Diagnostics / %d", len(m.diagnostics)+len(m.viewErrors)), lines, true, tuiPosition(m.offset, m.pageHeight(), total))
			help = " Wheel or ↑↓ to scroll"
		default:
			var lines []string
			view := m.currentView()
			for i := m.listTop; i < min(len(m.docs), m.listTop+m.pageHeight()); i++ {
				doc := m.docs[i]
				if view != nil {
					lines = append(lines, tuiAppRow(doc, view.Columns, i == m.selected, g.list.w-4))
				} else {
					lines = append(lines, tuiDocumentRow(doc.ID, doc.Frontmatter["title"].(string), i == m.selected, g.list.w-4))
				}
			}
			if len(lines) == 0 {
				lines = []string{"No matching documents.", "Clear the search or data filters."}
			}
			listTitle := "Documents"
			if view != nil {
				listTitle = strings.Join(view.Columns, " / ")
			}
			list := tuiPane(g.list, listTitle, lines, !m.previewFocus && m.mode != "search", tuiPosition(m.listTop, m.pageHeight(), len(m.docs)))
			preview := m.preview[min(m.offset, len(m.preview)):]
			if len(m.docs) == 0 {
				preview = []string{"Select a document to read it here."}
			}
			right := tuiPane(g.preview, "Preview", preview, m.previewFocus && m.mode != "search", tuiPosition(m.offset, m.pageHeight(), len(m.preview)))
			if g.split {
				for i := range list {
					body = append(body, list[i]+" "+right[i])
				}
			} else if m.previewFocus {
				body = right
			} else {
				body = list
			}
			if m.mode == "search" {
				help = " Live search · Enter keeps"
			}
		}
		copy(frame[g.body.y:], body)
		footerStart := m.width
		statusEnd := m.width
		for _, b := range g.buttons {
			if b.y == m.height-1 {
				footerStart = min(footerStart, b.x)
			}
			if b.key == "c" {
				statusEnd = b.x - 1
			}
		}
		frame[m.height-1] = tuiMuted.Render(tuiFit(help, max(0, footerStart-1)))
		frame[g.body.y-1] = tuiFit(frame[g.body.y-1], max(0, statusEnd))
		for _, b := range g.buttons {
			style := lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
			if b.toolbar {
				style = tuiButtonStyle
			}
			active := (b.key == "/" && m.mode == "search") || (b.key == "f" && (m.mode == "fields" || m.mode == "values")) || (b.key == "d" && m.mode == "diagnostics") || (b.key == "v" && (m.mode == "views" || m.view != ""))
			if active {
				style = style.Foreground(lipgloss.Color("231")).Background(lipgloss.Color("24"))
			}
			hintStyle := style.Foreground(lipgloss.Color("245"))
			if active {
				hintStyle = style.Foreground(lipgloss.Color("153"))
			}
			label := style.Render(" " + b.label)
			if b.hint != "" {
				label += hintStyle.Render("  " + b.hint)
			}
			label += style.Render(" ")
			line := tuiFit(frame[b.y], m.width)
			frame[b.y] = ansi.Cut(line, 0, b.x) + label + ansi.Cut(line, b.x+b.w, m.width)
		}
		if m.mode == "views" {
			m.renderViewMenu(frame)
		}
	}
	for i, line := range frame {
		frame[i] = tuiFit(line, m.width)
	}
	content := strings.Join(frame, "\n")
	if os.Getenv("NO_COLOR") != "" {
		content = ansi.Strip(content)
	}
	v := tea.NewView(content)
	v.AltScreen = true
	v.MouseMode = tea.MouseModeCellMotion
	return v
}
