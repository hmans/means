package cli

import "github.com/charmbracelet/x/ansi"

func (m *tuiModel) renderViewMenu(frame []string) {
	menu := m.viewMenu()
	search := "Find: " + terminalCell(m.pickerQuery) + "█"
	lines := []string{tuiMuted.Render(search)}
	for i := m.choiceTop; i < min(len(m.choices), m.choiceTop+m.pickerHeight()); i++ {
		label := m.choices[i].label
		if m.choices[i].value == m.view {
			label += " ✓"
		}
		lines = append(lines, tuiSelectedRow(label, i == m.choice, menu.w-4))
	}
	if len(m.choices) == 0 {
		lines = append(lines, tuiMuted.Render("No matching views"))
	}
	rows := tuiPane(menu, "Views", lines, true, tuiPosition(m.choiceTop, m.pickerHeight(), len(m.choices)))
	for i, row := range rows {
		line := tuiFit(frame[menu.y+i], m.width)
		frame[menu.y+i] = ansi.Cut(line, 0, menu.x) + row + ansi.Cut(line, menu.x+menu.w, m.width)
	}
}

func (m *tuiModel) viewMenu() tuiRect {
	x, y := 1, 1
	for _, button := range m.geometry().buttons {
		if button.key == "v" {
			x, y = button.x, button.y+1
			break
		}
	}
	width := 26
	for _, choice := range m.allChoices {
		width = max(width, ansi.StringWidth(terminalCell(choice.label))+8)
	}
	width = min(width, 48, m.width)
	x = max(0, min(x, m.width-width))
	height := min(max(1, len(m.choices)), 8) + 3
	height = min(height, max(4, m.height-y-1))
	return tuiRect{x, y, width, height}
}

func (m *tuiModel) pickerHeight() int {
	if m.mode == "views" {
		return max(1, m.viewMenu().h-3)
	}
	return m.pageHeight()
}
