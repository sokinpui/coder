package ui

import (
	"strings"

	"github.com/sokinpui/coder/internal/types"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/sokinpui/coder/internal/ui/markdown"
)

func (m Model) isLiveAIMessage(idx, total int, msg types.Message) bool {
	return (m.Chat.IsStreaming || m.Chat.IsAIRendering) && idx == total-1 && msg.Type == types.AIMessage
}

func (m Model) getAIMessageLines(msg types.Message, idx, total, viewportWidth int) []string {
	var lines []string
	if len(msg.ToolCalls) > 0 {
		tcPart := RenderToolCallMessage(msg, m.ToolsExpanded, viewportWidth)
		if tcPart != "" {
			lines = append(lines, strings.Split(tcPart, "\n")...)
		}
	}

	if msg.Content == "" {
		return lines
	}

	cache, isCached := m.Chat.RenderCache[idx]
	if isCached && cache.Content == msg.Content && cache.Width == viewportWidth {
		return append(lines, cache.Lines...)
	}

	if m.isLiveAIMessage(idx, total, msg) {
		return lines
	}

	return append(lines, strings.Split(msg.Content, "\n")...)
}

func (m Model) getMessageLines(msg types.Message, idx, total, viewportWidth int) []string {
	if msg.IsDocumentImage() {
		return nil
	}

	if msg.Type == types.AIMessage {
		return m.getAIMessageLines(msg, idx, total, viewportWidth)
	}

	if msg.Type == types.ToolCallMessage || msg.Type == types.ToolResultMessage {
		rendered := m.renderMessage(msg, viewportWidth)
		if rendered == "" {
			return nil
		}
		normalized := strings.ReplaceAll(rendered, "\r\n", "\n")
		return strings.Split(normalized, "\n")
	}

	if cache, ok := m.Chat.RenderCache[idx]; ok && cache.Content == msg.Content && cache.Width == viewportWidth {
		return cache.Lines
	}

	rendered := m.renderMessage(msg, viewportWidth)
	if rendered == "" {
		return nil
	}
	normalized := strings.ReplaceAll(rendered, "\r\n", "\n")
	lines := strings.Split(normalized, "\n")
	m.Chat.RenderCache[idx] = markdown.CachedRender{
		Lines:   lines,
		Content: msg.Content,
		Width:   viewportWidth,
	}
	return lines
}

func buildLineMetasForMessage(msg types.Message, msgIdx int, renderedLines []string) []LineMeta {
	if len(renderedLines) == 0 {
		return nil
	}

	metas := make([]LineMeta, len(renderedLines))

	switch msg.Type {
	case types.UserMessage, types.ImageMessage, types.CommandMessage, types.ShellCmdMessage,
		types.ContextCmdMessage, types.FileApplyCmdMessage, types.FileApplyUndoCmdMessage:
		colStart := 2
		prefix := ""
		if msg.Type == types.ShellCmdMessage {
			prefix = "Shell: "
		} else if msg.Type == types.ImageMessage {
			prefix = "Image: "
		}

		for i, line := range renderedLines {
			plain := ansi.Strip(line)
			lineWidth := ansi.StringWidth(line)
			if i == 0 || i == len(renderedLines)-1 {
				metas[i] = LineMeta{
					MsgIndex:     msgIdx,
					IsDecoration: true,
				}
				continue
			}

			contentEnd := max(colStart, lineWidth-1)
			rawText := plain
			if len(rawText) > colStart {
				rawText = rawText[colStart:]
			}
			if len(rawText) > 0 && strings.HasSuffix(rawText, "│") {
				rawText = strings.TrimSuffix(rawText, "│")
			}
			rawText = strings.TrimRight(rawText, " ")
			if i == 1 && prefix != "" && strings.HasPrefix(rawText, prefix) {
				rawText = strings.TrimPrefix(rawText, prefix)
			}

			metas[i] = LineMeta{
				MsgIndex:        msgIdx,
				ContentColStart: colStart,
				ContentColEnd:   contentEnd,
				Text:            rawText,
				IsContinuation:  i > 1,
			}
		}

	case types.CommandResultMessage, types.ShellCmdResultMessage, types.ContextCmdResultMessage,
		types.FileApplyCmdResultMessage, types.FileApplyUndoCmdResultMessage,
		types.CommandErrorResultMessage, types.FileApplyCmdErrorMessage, types.FileApplyUndoCmdErrorMessage:
		colStart := 2
		for i, line := range renderedLines {
			plain := ansi.Strip(line)
			lineWidth := ansi.StringWidth(line)
			rawText := plain
			if len(rawText) > colStart {
				rawText = rawText[colStart:]
			}
			rawText = strings.TrimRight(rawText, " ")
			metas[i] = LineMeta{
				MsgIndex:        msgIdx,
				ContentColStart: colStart,
				ContentColEnd:   lineWidth,
				Text:            rawText,
				IsContinuation:  false,
			}
		}

	default:
		for i, line := range renderedLines {
			plain := ansi.Strip(line)
			lineWidth := ansi.StringWidth(line)
			metas[i] = LineMeta{
				MsgIndex:        msgIdx,
				ContentColStart: 0,
				ContentColEnd:   lineWidth,
				Text:            strings.TrimRight(plain, " \r\n"),
				IsContinuation:  false,
			}
		}
	}

	return metas
}

func (m Model) renderConversationWithOffsets() (string, map[int]int, []string, []LineMeta) {
	messages := m.Session.GetMessages()
	viewportWidth := m.Chat.Viewport.Width

	messageLineOffsets := make(map[int]int, len(messages))
	currentLine := 0
	var allLines []string
	var allMetas []LineMeta

	total := len(messages)
	for i, msg := range messages {
		messageLineOffsets[i] = currentLine
		lines := m.getMessageLines(msg, i, total, viewportWidth)
		if len(lines) == 0 {
			continue
		}

		metas := buildLineMetasForMessage(msg, i, lines)

		allLines = append(allLines, lines...)
		allMetas = append(allMetas, metas...)
		currentLine += len(lines)
	}
	if m.State == stateAsking || m.State == stateThinking {
		thinkingLine := m.renderThinkingLine()
		tLines := strings.Split(thinkingLine, "\n")
		allLines = append(allLines, tLines...)
		for _, tl := range tLines {
			allMetas = append(allMetas, LineMeta{
				MsgIndex:        -1,
				ContentColStart: 0,
				ContentColEnd:   lipgloss.Width(tl),
				Text:            ansi.Strip(tl),
				IsDecoration:    true,
			})
		}
	}

	if len(allMetas) < len(allLines) {
		for i := len(allMetas); i < len(allLines); i++ {
			allMetas = append(allMetas, LineMeta{
				MsgIndex:      -1,
				ContentColEnd: ansi.StringWidth(allLines[i]),
				Text:          ansi.Strip(allLines[i]),
			})
		}
	}
	return strings.Join(allLines, "\n"), messageLineOffsets, allLines, allMetas
}

func (m Model) renderThinkingLine() string {
	text := "Thinking "
	switch m.State {
	case stateAsking:
		text = "Asking "
	case stateThinking:
		if m.StatusText != "" {
			text = m.StatusText + " "
		}
	}
	return RenderThinkingSpinner(text, m.Chat.Spinner.View())
}

func (m Model) renderMessage(msg types.Message, viewportWidth int) string {
	if msg.IsDocumentImage() {
		return ""
	}
	if msg.Type == types.ToolCallMessage {
		return RenderToolCallMessage(msg, m.ToolsExpanded, viewportWidth)
	}
	if msg.Type == types.ToolResultMessage {
		toolName := m.getToolNameForCallID(msg.ToolCallID)
		return RenderToolResultMessage(msg, toolName, m.ToolsExpanded, viewportWidth)
	}
	return RenderMessage(msg, viewportWidth, nil)
}

func (m Model) getToolNameForCallID(callID string) string {
	if callID == "" {
		return ""
	}
	for _, msg := range m.Session.GetMessages() {
		for _, tc := range msg.ToolCalls {
			if tc.ID == callID {
				return tc.Name
			}
		}
	}
	return ""
}

func (m *Model) renderConversation() string {
	content, offsets, lines, metas := m.renderConversationWithOffsets()
	m.Chat.MessageLineOffsets = offsets
	m.Chat.RenderedLines = lines
	m.Chat.LineMetas = metas
	if m.Chat.Selection.Active && !m.Chat.Selection.IsEmpty() {
		return m.applySelectionToLines(lines)
	}
	return content
}
