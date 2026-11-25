package ui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/sahilm/fuzzy"
	"github.com/sokinpui/coder/internal/clipboard"
	"github.com/sokinpui/coder/internal/engine/commands"
)

type CommandPaletteItem struct {
	Name        string
	Display     string
	Description string
	IsArg       bool
}

type CommandPaletteModel struct {
	Input        textinput.Model
	Items        []CommandPaletteItem
	Cursor       int
	Offset       int
	HasNavigated bool
	Width        int
	Height       int
}

func NewCommandPalette() CommandPaletteModel {
	ti := textinput.New()
	ti.Placeholder = "Type a command (e.g. /model, /btw ..., @file, !cmd)..."
	ti.Prompt = "> "
	ti.CharLimit = 0
	ti.PlaceholderStyle = SearchPlaceholderStyle

	return CommandPaletteModel{
		Input: ti,
	}
}

func (cp *CommandPaletteModel) open(m Model) tea.Cmd {
	cp.Input.Reset()
	cp.Input.Focus()
	cp.Cursor = 0
	cp.Offset = 0
	cp.HasNavigated = false
	cp.updateSuggestions(m)
	return textinput.Blink
}

func (cp *CommandPaletteModel) updateSuggestions(m Model) {
	cp.Items = nil
	cp.Cursor = 0
	cp.Offset = 0
	cp.HasNavigated = false

	raw := cp.Input.Value()
	trimmedLeft := strings.TrimLeft(raw, " \t")

	if cmdName, argPrefix, _, isArg := parsePaletteCommandArg(trimmedLeft); isArg {
		suggestions := m.Session.GetCommandSuggestions(cmdName, argPrefix)
		filtered := filterPaletteArgs(suggestions, argPrefix)
		cp.Items = make([]CommandPaletteItem, 0, len(filtered))
		for _, s := range filtered {
			cp.Items = append(cp.Items, CommandPaletteItem{
				Name:    s,
				Display: s,
				IsArg:   true,
			})
		}
		return
	}

	query := strings.TrimPrefix(strings.TrimSpace(trimmedLeft), "/")
	filteredCmds := filterPaletteCommands(m.AvailableCommands, m.CommandDescriptions, query)
	cp.Items = make([]CommandPaletteItem, 0, len(filteredCmds))
	for _, cmd := range filteredCmds {
		cp.Items = append(cp.Items, CommandPaletteItem{
			Name:        cmd,
			Display:     "/" + cmd,
			Description: m.CommandDescriptions[cmd],
			IsArg:       false,
		})
	}
}

func parsePaletteCommandArg(trimmedLeft string) (cmdName, argPrefix, replacePrefix string, ok bool) {
	if trimmedLeft == "" {
		return "", "", "", false
	}
	if strings.HasPrefix(trimmedLeft, "/") || strings.HasPrefix(trimmedLeft, "@") || strings.HasPrefix(trimmedLeft, "!") {
		return commands.ActiveCommandArg(trimmedLeft)
	}
	if !strings.ContainsAny(trimmedLeft, " \t") {
		return "", "", "", false
	}
	return commands.ActiveCommandArg("/" + trimmedLeft)
}

func filterPaletteCommands(available []string, descriptions map[string]string, query string) []string {
	if query == "" {
		res := append([]string(nil), available...)
		sort.Strings(res)
		return res
	}

	lowerQuery := strings.ToLower(query)
	var prefixMatches []string
	var rest []string
	for _, cmd := range available {
		if strings.HasPrefix(strings.ToLower(cmd), lowerQuery) {
			prefixMatches = append(prefixMatches, cmd)
			continue
		}
		rest = append(rest, cmd)
	}
	sort.Strings(prefixMatches)

	if len(rest) == 0 {
		return prefixMatches
	}

	targets := make([]string, len(rest))
	for i, cmd := range rest {
		if desc := descriptions[cmd]; desc != "" {
			targets[i] = cmd + " " + desc
		} else {
			targets[i] = cmd
		}
	}

	matches := fuzzy.Find(query, targets)
	result := make([]string, 0, len(prefixMatches)+len(matches))
	result = append(result, prefixMatches...)
	for _, match := range matches {
		result = append(result, rest[match.Index])
	}
	return result
}

func filterPaletteArgs(suggestions []string, prefix string) []string {
	if len(suggestions) == 0 {
		return nil
	}
	if prefix == "" {
		return suggestions
	}

	lowerPrefix := strings.ToLower(prefix)
	var prefixMatches []string
	var rest []string
	for _, s := range suggestions {
		if strings.HasPrefix(strings.ToLower(s), lowerPrefix) {
			prefixMatches = append(prefixMatches, s)
			continue
		}
		rest = append(rest, s)
	}

	if len(rest) == 0 {
		return prefixMatches
	}

	matches := fuzzy.Find(prefix, rest)
	result := make([]string, 0, len(prefixMatches)+len(matches))
	result = append(result, prefixMatches...)
	for _, match := range matches {
		result = append(result, rest[match.Index])
	}
	return result
}

func (cp *CommandPaletteModel) moveCursor(delta int) {
	total := len(cp.Items)
	if total == 0 {
		return
	}
	cp.Cursor = clamp(cp.Cursor+delta, 0, total-1)
	cp.HasNavigated = true
}

func (cp *CommandPaletteModel) applyTabCompletion(m Model) {
	if len(cp.Items) == 0 || cp.Cursor < 0 || cp.Cursor >= len(cp.Items) {
		return
	}

	item := cp.Items[cp.Cursor]
	if !item.IsArg {
		cp.Input.SetValue("/" + item.Name + " ")
		cp.Input.CursorEnd()
		cp.updateSuggestions(m)
		return
	}

	trimmedLeft := strings.TrimLeft(cp.Input.Value(), " \t")
	probe := trimmedLeft
	if !strings.HasPrefix(probe, "/") && !strings.HasPrefix(probe, "@") && !strings.HasPrefix(probe, "!") {
		probe = "/" + probe
	}

	_, _, replacePrefix, ok := commands.ActiveCommandArg(probe)
	if !ok {
		return
	}

	var completed string
	if strings.HasSuffix(item.Name, "/") {
		dirBase := strings.TrimSuffix(item.Name, "/")
		formatted := commands.FormatArg(dirBase)
		if strings.HasSuffix(formatted, `"`) {
			completed = formatted[:len(formatted)-1] + `/"`
		} else {
			completed = formatted + "/"
		}
	} else {
		completed = commands.FormatArg(item.Name) + " "
	}

	cp.Input.SetValue(replacePrefix + completed)
	cp.Input.CursorEnd()
	cp.updateSuggestions(m)
}

func (cp *CommandPaletteModel) resolveSubmitCommand(m Model) string {
	raw := strings.TrimSpace(cp.Input.Value())
	if raw == "" {
		if len(cp.Items) == 0 || cp.Cursor < 0 || cp.Cursor >= len(cp.Items) {
			return ""
		}
		return "/" + cp.Items[cp.Cursor].Name
	}

	if _, _, _, isArg := parsePaletteCommandArg(raw); isArg {
		cmdLine := normalizePaletteCommandLine(raw)
		if cp.HasNavigated && len(cp.Items) > 0 && cp.Cursor >= 0 && cp.Cursor < len(cp.Items) && cp.Items[cp.Cursor].IsArg {
			if _, _, replacePrefix, ok := commands.ActiveCommandArg(cmdLine); ok {
				argVal := strings.TrimSuffix(cp.Items[cp.Cursor].Name, "/")
				return replacePrefix + commands.FormatArg(argVal)
			}
		}
		return cmdLine
	}

	if len(cp.Items) > 0 && !cp.Items[0].IsArg && cp.Cursor >= 0 && cp.Cursor < len(cp.Items) {
		query := strings.TrimPrefix(raw, "/")
		if cp.HasNavigated || !isExactCommandName(m.AvailableCommands, query) {
			return "/" + cp.Items[cp.Cursor].Name
		}
	}

	return normalizePaletteCommandLine(raw)
}

func normalizePaletteCommandLine(raw string) string {
	if strings.HasPrefix(raw, "/") || strings.HasPrefix(raw, "@") || strings.HasPrefix(raw, "!") {
		return raw
	}
	return "/" + raw
}

func isExactCommandName(available []string, name string) bool {
	for _, cmd := range available {
		if cmd == name {
			return true
		}
	}
	return false
}

type CommandPaletteOverlay struct{}

func (o *CommandPaletteOverlay) IsVisible(main *Model) bool {
	return main.ActiveOverlay == overlayCommandPalette
}

func (o *CommandPaletteOverlay) View(main *Model) string {
	maxWidth := max(44, main.Width-4)
	maxHeight := max(10, main.Height-4)
	modalWidth := min(90, maxWidth)
	modalHeight := min(26, maxHeight)

	innerWidth := max(20, modalWidth-PaletteContainerStyle.GetHorizontalFrameSize())
	main.CommandPalette.Width = modalWidth
	main.CommandPalette.Height = modalHeight
	main.CommandPalette.Input.Width = max(10, innerWidth-lipgloss.Width(main.CommandPalette.Input.Prompt)-1)

	header := PaletteHeaderStyle.Render("Command Palette [Esc: close | C-j/C-k: move | Tab: complete | Enter: run]")
	inputLine := main.CommandPalette.Input.View()

	var listBuf strings.Builder
	items := main.CommandPalette.Items
	if len(items) == 0 {
		listBuf.WriteString(ToolMutedStyle.Render("  Press Enter to run typed command..."))
	} else {
		maxItems := clamp(modalHeight-6, 4, 16)
		if main.CommandPalette.Cursor < main.CommandPalette.Offset {
			main.CommandPalette.Offset = main.CommandPalette.Cursor
		} else if main.CommandPalette.Cursor >= main.CommandPalette.Offset+maxItems {
			main.CommandPalette.Offset = main.CommandPalette.Cursor - maxItems + 1
		}
		maxOffset := max(0, len(items)-maxItems)
		if main.CommandPalette.Offset > maxOffset {
			main.CommandPalette.Offset = maxOffset
		}

		start := main.CommandPalette.Offset
		end := min(start+maxItems, len(items))

		maxDisplayWidth := 0
		for i := start; i < end; i++ {
			if !items[i].IsArg {
				w := lipgloss.Width(items[i].Display)
				if w > maxDisplayWidth {
					maxDisplayWidth = w
				}
			}
		}

		for i := start; i < end; i++ {
			item := items[i]
			isCursor := i == main.CommandPalette.Cursor
			prefix := "  "
			if isCursor {
				prefix = "▸ "
			}

			var line string
			if item.IsArg || item.Description == "" {
				text := ansi.Truncate(item.Display, max(5, innerWidth-2), "...")
				if isCursor {
					line = PaletteSelectedItemStyle.Render(prefix + text)
				} else {
					line = prefix + PaletteItemStyle.Render(text)
				}
			} else {
				nameStyle := PaletteItemStyle
				descStyle := PaletteDescriptionStyle
				if isCursor {
					nameStyle = PaletteSelectedItemStyle
					descStyle = PaletteSelectedItemStyle
				}
				namePart := nameStyle.Render(item.Display)
				padding := max(3, maxDisplayWidth-lipgloss.Width(item.Display)+3)
				availDesc := max(0, innerWidth-lipgloss.Width(prefix)-lipgloss.Width(item.Display)-padding)
				descText := ansi.Truncate(item.Description, availDesc, "...")
				line = prefix + namePart + strings.Repeat(" ", padding) + descStyle.Render(descText)
			}

			if i > start {
				listBuf.WriteByte('\n')
			}
			listBuf.WriteString(line)
		}
	}

	body := lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		inputLine,
		"",
		listBuf.String(),
	)

	content := PaletteContainerStyle.Width(modalWidth).Render(body)
	return OverlayCenter(content, main.View())
}

func (m Model) openCommandPalette() (Model, tea.Cmd) {
	m.ActiveOverlay = overlayCommandPalette
	m.Chat.ShowPalette = false
	m.Chat.Selection.Active = false
	m.Chat.Selection.Dragging = false
	m.Chat.TextArea.Blur()
	blinkCmd := m.CommandPalette.open(m)
	return m, blinkCmd
}

func (m Model) closeCommandPalette() (Model, tea.Cmd) {
	m.ActiveOverlay = overlayNone
	m.CommandPalette.Input.Reset()
	if m.shouldFocusTextArea() {
		m.Chat.TextArea.Focus()
		return m, textarea.Blink
	}
	return m, nil
}

func (m Model) handleKeyPressCommandPalette(msg tea.KeyMsg) (tea.Model, tea.Cmd, bool) {
	km := m.Keymap()
	keyStr := msg.String()

	if km.Palette != "" && keyStr == km.Palette {
		newModel, cmd := m.closeCommandPalette()
		return newModel, cmd, true
	}

	switch msg.Type {
	case tea.KeyEsc, tea.KeyCtrlC:
		newModel, cmd := m.closeCommandPalette()
		return newModel, cmd, true

	case tea.KeyUp, tea.KeyCtrlK:
		m.CommandPalette.moveCursor(-1)
		return m, nil, true

	case tea.KeyDown, tea.KeyCtrlJ, tea.KeyCtrlN:
		m.CommandPalette.moveCursor(1)
		return m, nil, true

	case tea.KeyTab:
		m.CommandPalette.applyTabCompletion(m)
		return m, nil, true

	case tea.KeyEnter:
		return m.executeCommandPaletteSelection()
	}

	switch keyStr {
	case km.Paste:
		return m, commandPalettePasteCmd(), true
	}

	prevVal := m.CommandPalette.Input.Value()
	var cmd tea.Cmd
	m.CommandPalette.Input, cmd = m.CommandPalette.Input.Update(msg)
	if cleaned := CleanSGRLeakedText(m.CommandPalette.Input.Value()); cleaned != m.CommandPalette.Input.Value() {
		m.CommandPalette.Input.SetValue(cleaned)
	}
	if m.CommandPalette.Input.Value() != prevVal {
		m.CommandPalette.updateSuggestions(m)
	}
	return m, cmd, true
}

func (m Model) executeCommandPaletteSelection() (tea.Model, tea.Cmd, bool) {
	cmdLine := m.CommandPalette.resolveSubmitCommand(m)
	m.ActiveOverlay = overlayNone
	m.CommandPalette.Input.Reset()

	if strings.TrimSpace(cmdLine) == "" {
		if m.shouldFocusTextArea() {
			m.Chat.TextArea.Focus()
			return m, textarea.Blink, true
		}
		return m, nil, true
	}

	if m.State != stateIdle {
		cmdName, args, isCmd := commands.ParseCommand(cmdLine)
		if isCmd && isImmediateStreamingCommand(cmdName, args) {
			nextModel, cmd := m.submitInput(cmdLine, false)
			if updated, ok := nextModel.(Model); ok && updated.ActiveOverlay == overlayNone && updated.shouldFocusTextArea() && !updated.Chat.TextArea.Focused() {
				updated.Chat.TextArea.Focus()
				return updated, tea.Batch(cmd, textarea.Blink), true
			}
			return nextModel, cmd, true
		}

		if m.canInputWhileStreaming() {
			m.enqueuePrompt(cmdLine)
			m.StatusBarMessage = fmt.Sprintf("Queued: %s", TruncateSingleLine(cmdLine, 50))
			if m.shouldFocusTextArea() {
				m.Chat.TextArea.Focus()
				return m, tea.Batch(clearStatusBarCmd(), textarea.Blink), true
			}
			return m, clearStatusBarCmd(), true
		}

		m.StatusBarMessage = fmt.Sprintf("Cannot run %s while generation is in progress.", TruncateSingleLine(cmdLine, 40))
		return m, clearStatusBarCmd(), true
	}

	nextModel, cmd := m.submitInput(cmdLine, false)
	cmdName, _, isCmd := commands.ParseCommand(cmdLine)
	if updated, ok := nextModel.(Model); ok {
		if updated.ActiveOverlay == overlayNone && updated.shouldFocusTextArea() && !updated.Chat.TextArea.Focused() && (!isCmd || (cmdName != "shell" && cmdName != "!")) {
			updated.Chat.TextArea.Focus()
			return updated, tea.Batch(cmd, textarea.Blink), true
		}
		return updated, cmd, true
	}
	return nextModel, cmd, true
}

func isImmediateStreamingCommand(cmdName, args string) bool {
	switch cmdName {
	case "btw", "queue", "msg", "cards", "gen", "edit", "branch", "history", "active", "skills", "help", "list", "image", "new", "quit":
		return true
	case "config":
		return strings.TrimSpace(args) != "reload"
	default:
		return false
	}
}

func commandPalettePasteCmd() tea.Cmd {
	return func() tea.Msg {
		text, err := clipboard.PasteText()
		return commandPalettePasteResultMsg{content: text, err: err}
	}
}
