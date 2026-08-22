package ui

import (
	"fmt"
	"slices"
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/sokinpui/coder/internal/commands"
	"github.com/sokinpui/coder/internal/types"
	"github.com/sokinpui/coder/internal/utils"
)

type AtomicMsgModel struct {
	Cursor      int
	Anchor      int
	IsSelecting bool
	GGPressed   bool
	Width       int
	Height      int
}

func NewAtomicMsg() AtomicMsgModel {
	return AtomicMsgModel{
		Cursor:      0,
		Anchor:      0,
		IsSelecting: false,
		GGPressed:   false,
	}
}

func (m AtomicMsgModel) renderMsgItem(msg types.Message, index int, isCursor bool, isSelected bool, width int) string {
	badge := fmt.Sprintf("[%02d %s]", index+1, msg.Type.String())
	checkPrefix := ""
	if m.IsSelecting {
		if isSelected {
			checkPrefix = "[✓] "
		} else {
			checkPrefix = "[ ] "
		}
	}
	summary := getOneLineSummary(msg.Content)

	availableWidth := max(10, width-lipgloss.Width(badge)-lipgloss.Width(checkPrefix)-5)
	runes := []rune(summary)
	if len(runes) > availableWidth {
		if availableWidth > 3 {
			summary = string(runes[:availableWidth-3]) + "..."
		} else {
			summary = string(runes[:availableWidth])
		}
	}

	cursorPrefix := "  "
	if isCursor {
		cursorPrefix = "▸ "
	}

	line := fmt.Sprintf("%s%s%s %s", cursorPrefix, checkPrefix, badge, summary)
	if isCursor || isSelected {
		return paletteSelectedItemStyle.Width(width).Render(line)
	}
	return paletteItemStyle.Width(width).Render(line)
}

func (m AtomicMsgModel) View(messages []types.Message, maxHeight int) string {
	selectable := getSelectableIndices(messages)
	if len(selectable) == 0 {
		return paletteContainerStyle.Width(m.Width).Render("No selectable atomic messages")
	}

	var msgLines []string
	currPos := 0
	for pos, idx := range selectable {
		if idx == m.Cursor {
			currPos = pos
			break
		}
	}

	selectedSet := make(map[int]struct{})
	if m.IsSelecting {
		for _, idx := range m.getSelectedIndices(selectable) {
			selectedSet[idx] = struct{}{}
		}
	}

	maxItems := max(5, maxHeight)
	start := 0
	if currPos >= maxItems {
		start = currPos - maxItems + 1
	}
	end := min(start+maxItems, len(selectable))

	itemWidth := max(20, m.Width-4)
	for i := start; i < end; i++ {
		idx := selectable[i]
		msg := messages[idx]
		_, isSelected := selectedSet[idx]
		msgLines = append(msgLines, m.renderMsgItem(msg, idx, idx == m.Cursor, isSelected, itemWidth))
	}

	header := paletteHeaderStyle.Render("── Atomic Messages [Esc/C-c: exit | v: select | o: swap | y/d: copy/del | a/e/r/b] ──")
	body := strings.Join(msgLines, "\n")
	content := lipgloss.JoinVertical(lipgloss.Left, header, body)
	return paletteContainerStyle.Width(m.Width).Render(content)
}

func (m AtomicMsgModel) getSelectedIndices(selectable []int) []int {
	if !m.IsSelecting || len(selectable) == 0 {
		return []int{m.Cursor}
	}
	anchorPos := -1
	cursorPos := -1
	for pos, idx := range selectable {
		if idx == m.Anchor {
			anchorPos = pos
		}
		if idx == m.Cursor {
			cursorPos = pos
		}
	}
	if anchorPos == -1 || cursorPos == -1 {
		return []int{m.Cursor}
	}
	start := min(anchorPos, cursorPos)
	end := max(anchorPos, cursorPos)
	return selectable[start : end+1]
}

type AtomicMsgOverlay struct{}

func (o *AtomicMsgOverlay) IsVisible(main *Model) bool {
	return main.ActiveOverlay == overlayAtomicMsg
}

func (o *AtomicMsgOverlay) View(main *Model) string {
	main.AtomicMsg.Height = main.Height

	modalWidth := min(80, max(40, main.Width-4))
	main.AtomicMsg.Width = modalWidth
	content := main.AtomicMsg.View(main.Session.GetMessages(), main.Height-6)

	return OverlayCenter(content, main.View())
}

func (m Model) openAtomicMsgMode() (Model, tea.Cmd) {
	messages := m.Session.GetMessages()
	selectable := getSelectableIndices(messages)
	if len(selectable) == 0 {
		return m, nil
	}

	m.ActiveOverlay = overlayAtomicMsg
	m.AtomicMsg.IsSelecting = false
	m.AtomicMsg.GGPressed = false
	m.AtomicMsg.Cursor = selectable[len(selectable)-1]
	m.AtomicMsg.Anchor = m.AtomicMsg.Cursor
	m.Chat.TextArea.Blur()
	m = m.updateLayout()
	m = m.syncViewportToMessage(m.AtomicMsg.Cursor)
	return m, nil
}

func (m Model) handleKeyPressAtomicMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd, bool) {
	selectable := getSelectableIndices(m.Session.GetMessages())
	if len(selectable) == 0 {
		m.ActiveOverlay = overlayNone
		m.AtomicMsg.IsSelecting = false
		m.AtomicMsg.GGPressed = false
		if m.State == stateIdle {
			m.Chat.TextArea.Focus()
		}
		return m, textarea.Blink, true
	}

	prevGGPressed := m.AtomicMsg.GGPressed
	m.AtomicMsg.GGPressed = false

	currIdx := m.AtomicMsg.Cursor
	keyStr := msg.String()

	switch keyStr {
	case "g":
		if prevGGPressed {
			m.AtomicMsg.Cursor = selectable[0]
			m = m.syncViewportToMessage(m.AtomicMsg.Cursor)
			return m, nil, true
		}
		m.AtomicMsg.GGPressed = true
		return m, nil, true

	case "G":
		m.AtomicMsg.Cursor = selectable[len(selectable)-1]
		m = m.syncViewportToMessage(m.AtomicMsg.Cursor)
		return m, nil, true

	case "j", "down", "ctrl+n", "ctrl+j":
		m.AtomicMsg.Cursor = nextSelectableIndex(m.Session.GetMessages(), currIdx, 1)
		m = m.syncViewportToMessage(m.AtomicMsg.Cursor)
		return m, nil, true

	case "k", "up", "ctrl+k":
		m.AtomicMsg.Cursor = nextSelectableIndex(m.Session.GetMessages(), currIdx, -1)
		m = m.syncViewportToMessage(m.AtomicMsg.Cursor)
		return m, nil, true

	case "ctrl+d":
		m.Chat.Viewport.HalfPageDown()
		return m, nil, true

	case "ctrl+u":
		m.Chat.Viewport.HalfPageUp()
		return m, nil, true

	case "v":
		if m.AtomicMsg.IsSelecting {
			m.AtomicMsg.IsSelecting = false
		} else {
			m.AtomicMsg.IsSelecting = true
			m.AtomicMsg.Anchor = currIdx
		}
		return m, nil, true

	case "o", "O":
		if m.AtomicMsg.IsSelecting {
			m.AtomicMsg.Cursor, m.AtomicMsg.Anchor = m.AtomicMsg.Anchor, m.AtomicMsg.Cursor
			m = m.syncViewportToMessage(m.AtomicMsg.Cursor)
		}
		return m, nil, true

	case "esc", "ctrl+c":
		m.ActiveOverlay = overlayNone
		m.AtomicMsg.IsSelecting = false
		if m.State == stateIdle {
			m.Chat.TextArea.Focus()
		}
		return m, textarea.Blink, true

	case "a":
		m.AtomicMsg.IsSelecting = false
		messages := m.Session.GetMessages()
		targetMsg := messages[currIdx]
		var aiResponseToApply string

		if targetMsg.Type == types.AIMessage && targetMsg.Content != "" {
			aiResponseToApply = targetMsg.Content
		} else {
			for i := currIdx; i >= 0; i-- {
				if messages[i].Type == types.AIMessage && messages[i].Content != "" {
					aiResponseToApply = messages[i].Content
					break
				}
			}
		}

		if aiResponseToApply == "" {
			m.StatusBarMessage = "No AI response found to apply."
			m.ActiveOverlay = overlayNone
			if m.State == stateIdle {
				m.Chat.TextArea.Focus()
			}
			return m, tea.Batch(clearStatusBarCmd(), textarea.Blink), true
		}

		res := commands.ExecuteItf(aiResponseToApply, "")
		m.Session.SetLastModifiedFiles(res.AffectedFiles)
		m.Session.AddMessages(types.Message{Type: types.CommandMessage, Content: "/itf"})

		if res.Success {
			m.Session.AddMessages(types.Message{Type: types.CommandResultMessage, Content: res.Summary})
		} else {
			m.Session.AddMessages(types.Message{Type: types.CommandErrorResultMessage, Content: res.Summary})
		}

		m.ActiveOverlay = overlayNone
		if m.State == stateIdle {
			m.Chat.TextArea.Focus()
		}
		m.Chat.Viewport.SetContent(m.renderConversation())
		m.Chat.Viewport.GotoBottom()
		return m, textarea.Blink, true

	case "y":
		messages := m.Session.GetMessages()
		var targetIndices []int
		if m.AtomicMsg.IsSelecting {
			targetIndices = m.AtomicMsg.getSelectedIndices(selectable)
		} else {
			targetIndices = []int{currIdx}
		}

		if len(targetIndices) == 1 && messages[targetIndices[0]].Type == types.ImageMessage {
			msg := messages[targetIndices[0]]
			err := utils.CopyImage(msg.Content, msg.Data)
			if err != nil {
				m.StatusBarMessage = fmt.Sprintf("Failed to copy image: %v", err)
			} else {
				m.StatusBarMessage = "Image copied to clipboard."
			}
			m.ActiveOverlay = overlayNone
			m.AtomicMsg.IsSelecting = false
			if m.State == stateIdle {
				m.Chat.TextArea.Focus()
			}
			return m, tea.Batch(clearStatusBarCmd(), textarea.Blink), true
		}

		var contents []string
		for _, idx := range targetIndices {
			contents = append(contents, messages[idx].Content)
		}
		combined := strings.Join(contents, "\n\n")
		cfg := m.Session.GetConfig()
		_ = utils.Copy(combined, cfg.Clipboard.CopyCmd)
		if len(targetIndices) > 1 {
			m.StatusBarMessage = fmt.Sprintf("%d messages copied to clipboard.", len(targetIndices))
		} else {
			m.StatusBarMessage = "Message copied to clipboard."
		}
		m.ActiveOverlay = overlayNone
		m.AtomicMsg.IsSelecting = false
		if m.State == stateIdle {
			m.Chat.TextArea.Focus()
		}
		return m, tea.Batch(clearStatusBarCmd(), textarea.Blink), true

	case "e":
		m.AtomicMsg.IsSelecting = false
		messages := m.Session.GetMessages()
		targetMsg := messages[currIdx]
		if !targetMsg.Type.IsEditable() {
			m.StatusBarMessage = "Only user messages can be edited."
			m.ActiveOverlay = overlayNone
			if m.State == stateIdle {
				m.Chat.TextArea.Focus()
			}
			return m, tea.Batch(clearStatusBarCmd(), textarea.Blink), true
		}

		m.ActiveOverlay = overlayNone
		m.Chat.EditingMessageIndex = currIdx
		return m, editInEditorCmd(targetMsg.Content), true

	case "r":
		m.AtomicMsg.IsSelecting = false
		messages := m.Session.GetMessages()
		targetMsg := messages[currIdx]
		if !targetMsg.Type.IsRegeneratable() {
			m.StatusBarMessage = "Selected message cannot be regenerated."
			m.ActiveOverlay = overlayNone
			if m.State == stateIdle {
				m.Chat.TextArea.Focus()
			}
			return m, tea.Batch(clearStatusBarCmd(), textarea.Blink), true
		}

		if m.Chat.IsStreaming {
			m.Session.CancelGeneration()
			m.Chat.IsStreaming = false
			m.Chat.StreamSub = nil
		}

		m.ActiveOverlay = overlayNone
		m.Chat.TextArea.Focus()
		event := m.Session.RegenerateFrom(currIdx)
		model, cmd := m.startGeneration(event)
		return model, cmd, true

	case "d":
		var targetIndices []int
		if m.AtomicMsg.IsSelecting {
			targetIndices = m.AtomicMsg.getSelectedIndices(selectable)
		} else {
			targetIndices = []int{currIdx}
		}

		if m.Chat.IsStreaming {
			if slices.Contains(targetIndices, len(m.Session.GetMessages())-1) {
				m.Session.CancelGeneration()
				m.Chat.IsStreaming = false
				m.Chat.StreamSub = nil
			}
		}

		m.Session.DeleteMessages(targetIndices)
		m.ClearCache()
		m.UpdateTokenCount()
		if len(targetIndices) > 1 {
			m.StatusBarMessage = fmt.Sprintf("Deleted %d messages.", len(targetIndices))
		} else {
			m.StatusBarMessage = "Deleted message."
		}
		m.ActiveOverlay = overlayNone
		m.AtomicMsg.IsSelecting = false
		if m.State == stateIdle {
			m.Chat.TextArea.Focus()
		}
		m.Chat.Viewport.SetContent(m.renderConversation())
		return m, tea.Batch(clearStatusBarCmd(), textarea.Blink), true

	case "b":
		m.AtomicMsg.IsSelecting = false
		if m.Chat.IsStreaming {
			m.Session.CancelGeneration()
			m.Chat.IsStreaming = false
			m.Chat.StreamSub = nil
		}

		newSess, err := m.Session.Branch(currIdx)
		if err != nil {
			m.StatusBarMessage = fmt.Sprintf("Error branching: %v", err)
			m.ActiveOverlay = overlayNone
			if m.State == stateIdle {
				m.Chat.TextArea.Focus()
			}
			return m, tea.Batch(clearStatusBarCmd(), textarea.Blink), true
		}

		m.ActiveOverlay = overlayNone
		m.Session = newSess
		m.addActiveSession(newSess)
		m.StatusBarMessage = "Branched to a new session."
		m.Chat.LastInteractionFailed = false
		m.Chat.TextArea.Reset()
		m.Chat.TextArea.SetHeight(1)
		m.Chat.TextArea.Focus()
		m.Chat.Viewport.SetContent(m.renderConversation())
		m.Chat.Viewport.GotoBottom()
		m.UpdateTokenCount()
		return m, tea.Batch(clearStatusBarCmd(), textarea.Blink), true
	}

	return m, nil, true
}

func (m Model) syncViewportToMessage(msgIdx int) Model {
	if line, ok := m.Chat.MessageLineOffsets[msgIdx]; ok {
		viewportHeight := m.Chat.Viewport.Height
		targetY := max(line-(viewportHeight/3), 0)
		m.Chat.Viewport.SetYOffset(targetY)
	}
	return m
}

func getSelectableIndices(messages []types.Message) []int {
	var indices []int
	for i, msg := range messages {
		if msg.Type.IsSelectable() {
			indices = append(indices, i)
		}
	}
	return indices
}

func nextSelectableIndex(messages []types.Message, current int, direction int) int {
	selectable := getSelectableIndices(messages)
	if len(selectable) == 0 {
		return 0
	}

	currPos := -1
	for pos, idx := range selectable {
		if idx == current {
			currPos = pos
			break
		}
	}

	if currPos == -1 {
		if direction >= 0 {
			return selectable[0]
		}
		return selectable[len(selectable)-1]
	}

	newPos := currPos + direction
	if newPos < 0 {
		newPos = 0
	} else if newPos >= len(selectable) {
		newPos = len(selectable) - 1
	}
	return selectable[newPos]
}

func getOneLineSummary(content string) string {
	for line := range strings.SplitSeq(content, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" {
			return trimmed
		}
	}
	return "(empty)"
}
