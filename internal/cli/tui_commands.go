package cli

import (
	"fmt"
	"io"
	"sort"
	"strings"
	"unicode"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/google/shlex"
	"github.com/spf13/cobra"
)

type tuiCommandBar struct {
	active  bool
	input   textinput.Model
	message string
	failed  bool
	matches []string
	cycle   int
}

func (m *tuiModel) openCommandBar() tea.Cmd {
	input := textinput.New()
	input.Prompt = ":"
	input.Placeholder = "Command… Tab completes · Enter runs · Esc cancels"
	input.ShowSuggestions = true
	input.SetVirtualCursor(true)
	styles := input.Styles()
	styles.Focused.Text = lipgloss.NewStyle().Foreground(lipgloss.Color("231"))
	styles.Focused.Prompt = tuiMuted
	styles.Focused.Suggestion = tuiMuted
	input.SetStyles(styles)
	m.commandBar = tuiCommandBar{active: true, input: input, cycle: -1}
	m.draggingDivider = false
	m.completeCommandBar()
	return m.commandBar.input.Focus()
}

// Render two styled copies and combine only the visible command-name span.
// Bubbles keeps ownership of editing, horizontal scrolling, and the cursor.
func (b *tuiCommandBar) view() string {
	plain := b.input.View()
	value := b.input.Value()
	start := len(value) - len(strings.TrimLeftFunc(value, unicode.IsSpace))
	end := strings.IndexFunc(value[start:], unicode.IsSpace)
	if end < 0 {
		end = len(value) - start
	}
	end += start
	if start == end {
		return plain
	}
	copy := b.input
	copy.SetVirtualCursor(false)
	cursor := copy.Cursor()
	if cursor == nil {
		return plain
	}
	position := string([]rune(value)[:b.input.Position()])
	promptWidth := ansi.StringWidth(b.input.Prompt)
	offset := promptWidth + ansi.StringWidth(position) - cursor.X
	left := max(promptWidth, promptWidth+ansi.StringWidth(value[:start])-offset)
	right := max(left, promptWidth+ansi.StringWidth(value[:end])-offset)
	styled := b.input
	styles := styled.Styles()
	styles.Focused.Text = lipgloss.NewStyle().Foreground(lipgloss.Color("179"))
	styled.SetStyles(styles)
	return ansi.Cut(plain, 0, left) + ansi.Cut(styled.View(), left, right) + ansi.Cut(plain, right, ansi.StringWidth(plain))
}

// Each invocation gets fresh Cobra state; command handlers return Bubble Tea
// commands instead of writing to the terminal or starting another event loop.
func (m *tuiModel) commandTree(result *tea.Cmd) *cobra.Command {
	root := &cobra.Command{Use: "means", SilenceErrors: true, SilenceUsage: true}
	root.SetOut(io.Discard)
	root.SetErr(io.Discard)
	root.CompletionOptions.DisableDefaultCmd = true
	root.PersistentPreRunE = func(cmd *cobra.Command, _ []string) error {
		if m.loading && cmd.Annotations["requiresCollection"] == "true" {
			return fmt.Errorf("collection is still loading")
		}
		return nil
	}
	add := func(name, description, key string, aliases ...string) {
		command := &cobra.Command{Use: name, Short: description, Aliases: aliases, Args: cobra.NoArgs,
			Run: func(_ *cobra.Command, _ []string) { *result = m.action(key) }}
		if key != "q" {
			command.Annotations = map[string]string{"requiresCollection": "true"}
		}
		root.AddCommand(command)
	}
	add("quit", "Quit means", "q", "q")
	add("clear", "Clear search and data filters", "x")
	root.AddCommand(&cobra.Command{
		Use: "home", Short: "Leave the current view, keeping search and data filters", Args: cobra.NoArgs,
		Annotations: map[string]string{"requiresCollection": "true"},
		RunE: func(_ *cobra.Command, _ []string) error {
			var err error
			*result, err = m.selectView("")
			return err
		},
	})
	add("filters", "Open the filter picker", "f", "filter")
	add("diagnostics", "Show diagnostics", "d")
	views := &cobra.Command{Use: "views [name]", Short: "Choose a view", Aliases: []string{"view"}, Args: cobra.MaximumNArgs(1)}
	views.Annotations = map[string]string{"requiresCollection": "true"}
	views.ValidArgsFunction = func(_ *cobra.Command, args []string, prefix string) ([]string, cobra.ShellCompDirective) {
		if len(args) != 0 {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		names := []string{"All documents"}
		if m.app != nil {
			for _, v := range m.app.Views {
				if v.compileErr == nil {
					names = append(names, v.Title)
				}
			}
		}
		var matches []string
		for _, name := range names {
			if strings.HasPrefix(strings.ToLower(name), strings.ToLower(prefix)) {
				matches = append(matches, name)
			}
		}
		return matches, cobra.ShellCompDirectiveNoFileComp
	}
	views.RunE = func(_ *cobra.Command, args []string) error {
		if len(args) == 0 {
			m.openViews()
			return nil
		}
		var err error
		*result, err = m.selectView(args[0])
		return err
	}
	root.AddCommand(views)
	root.SetHelpFunc(func(command *cobra.Command, _ []string) {
		var help []string
		for _, c := range root.Commands() {
			if c.Name() != "help" {
				help = append(help, c.Name())
			}
		}
		m.commandBar.active = true
		m.commandBar.failed = false
		m.commandBar.message = "Commands: " + strings.Join(help, ", ") + ". Use help <command>."
		if command != root {
			m.commandBar.message = command.Use + " — " + command.Short
		}
	})
	return root
}

func (m *tuiModel) completeCommandBar() {
	b := &m.commandBar
	b.matches, b.cycle = nil, -1
	var ignored tea.Cmd
	root := m.commandTree(&ignored)
	line := b.input.Value()
	if b.input.Position() != len([]rune(line)) {
		b.input.SetSuggestions(nil)
		return
	}
	trimmed := strings.TrimLeftFunc(line, unicode.IsSpace)
	leading := line[:len(line)-len(trimmed)]
	name, rest, hasArgs := trimmed, "", false
	if split := strings.IndexFunc(trimmed, unicode.IsSpace); split >= 0 {
		name, rest, hasArgs = trimmed[:split], strings.TrimLeftFunc(trimmed[split:], unicode.IsSpace), true
	}
	if !hasArgs {
		for _, c := range root.Commands() {
			for _, candidate := range append([]string{c.Name()}, c.Aliases...) {
				if strings.HasPrefix(candidate, name) {
					b.matches = append(b.matches, leading+candidate)
				}
			}
		}
		if strings.HasPrefix("help", name) {
			b.matches = append(b.matches, leading+"help")
		}
	} else if cmd, _, err := root.Find([]string{name}); err == nil && cmd.ValidArgsFunction != nil {
		// Use the same tokenizer for execution and completion. For an unfinished
		// quoted argument, try closing the quote only in the completion copy.
		args, err := shlex.Split(rest)
		for _, closing := range []string{"\"", "'"} {
			if err == nil {
				break
			}
			args, err = shlex.Split(rest + closing)
		}
		if len(args) > 0 && args[0] == "--" {
			args = args[1:]
		}
		if err != nil || len(args) > 1 {
			b.input.SetSuggestions(nil)
			return
		}
		prefix := ""
		if len(args) == 1 {
			prefix = args[0]
		}
		candidates, _ := cmd.ValidArgsFunction(cmd, nil, prefix)
		for _, candidate := range candidates {
			separator := " "
			if strings.HasPrefix(candidate, "-") {
				separator = " -- "
			}
			quoted := `"` + strings.NewReplacer(`\`, `\\`, `"`, `\"`).Replace(candidate) + `"`
			b.matches = append(b.matches, leading+name+separator+quoted)
		}
	}
	sort.Strings(b.matches)
	suggestions := append([]string(nil), b.matches...)
	if hasArgs && !strings.HasPrefix(strings.TrimSpace(rest), "\"") && !strings.HasPrefix(strings.TrimSpace(rest), "'") {
		for i, match := range suggestions {
			if args, err := shlex.Split(match); err == nil && len(args) == 2 {
				suggestions[i] = leading + name + " " + args[1]
			}
		}
	}
	b.input.SetSuggestions(suggestions)
}

func (m *tuiModel) updateCommandBar(msg tea.Msg) tea.Cmd {
	previousHeight := m.pageHeight()
	defer func() {
		if m.pageHeight() < previousHeight {
			m.listTop = tuiKeepVisible(m.selected, m.listTop, m.pageHeight(), len(m.docs))
		}
	}()
	b := &m.commandBar
	if key, ok := msg.(tea.KeyPressMsg); ok {
		switch key.String() {
		case "ctrl+c":
			return tea.Quit
		case "esc":
			b.active = false
			b.input.Blur()
			return nil
		case "backspace":
			if b.input.Value() == "" {
				b.active = false
				b.input.Blur()
				return nil
			}
		case "tab", "shift+tab":
			if len(b.matches) == 0 || b.input.Position() != len([]rune(b.input.Value())) {
				return nil
			}
			step := 1
			if b.cycle < 0 && key.String() == "tab" {
				b.cycle = b.input.CurrentSuggestionIndex() - 1
			}
			if key.String() == "shift+tab" {
				step = -1
				if b.cycle < 0 {
					b.cycle = 0
				}
			}
			b.cycle = (b.cycle + step + len(b.matches)) % len(b.matches)
			b.input.SetValue(b.matches[b.cycle])
			b.input.CursorEnd()
			b.message, b.failed = "", false
			return nil
		case "enter":
			args, err := shlex.Split(b.input.Value())
			if err != nil {
				b.message, b.failed = err.Error(), true
				return nil
			}
			if len(args) == 0 {
				b.active = false
				return nil
			}
			var result tea.Cmd
			root := m.commandTree(&result)
			root.SetArgs(args)
			b.active, b.message = false, ""
			b.failed = false
			if err := root.Execute(); err != nil {
				b.active, b.message = true, err.Error()
				b.failed = true
			}
			return result
		}
	}
	return m.updateCommandInput(msg)
}

func (m *tuiModel) updateCommandInput(msg tea.Msg) tea.Cmd {
	b := &m.commandBar
	var cmd tea.Cmd
	previous := b.input.Value()
	position := b.input.Position()
	b.input, cmd = b.input.Update(msg)
	if b.input.Value() != previous {
		b.message, b.failed = "", false
	}
	if b.input.Value() != previous || b.input.Position() != position {
		m.completeCommandBar()
	}
	return cmd
}
