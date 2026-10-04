package ui

import (
	"strings"

	"github.com/sokinpui/coder/internal/types"
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
		return strings.Split(rendered, "\n")
	}

	if cache, ok := m.Chat.RenderCache[idx]; ok && cache.Content == msg.Content && cache.Width == viewportWidth {
		return cache.Lines
	}

	rendered := m.renderMessage(msg, viewportWidth)
	if rendered == "" {
		return nil
	}
	lines := strings.Split(rendered, "\n")
	m.Chat.RenderCache[idx] = markdown.CachedRender{
		Lines:   lines,
		Content: msg.Content,
		Width:   viewportWidth,
	}
	return lines
}

func (m Model) renderConversationWithOffsets() (string, map[int]int) {
	messages := m.Session.GetMessages()
	viewportWidth := m.Chat.Viewport.Width

	messageLineOffsets := make(map[int]int, len(messages))
	currentLine := 0
	var allLines []string

	total := len(messages)
	for i, msg := range messages {
		messageLineOffsets[i] = currentLine
		lines := m.getMessageLines(msg, i, total, viewportWidth)

		allLines = append(allLines, lines...)
		currentLine += len(lines)
	}
	if m.State == stateAsking || m.State == stateThinking {
		thinkingLine := m.renderThinkingLine()
		allLines = append(allLines, strings.Split(thinkingLine, "\n")...)
	}
	return strings.Join(allLines, "\n"), messageLineOffsets
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
	content, offsets := m.renderConversationWithOffsets()
	m.Chat.MessageLineOffsets = offsets
	return content
}
