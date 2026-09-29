package coderui

import (
	"strings"

	"github.com/sokinpui/coder/internal/types"
	"github.com/sokinpui/coder/internal/ui/core"
	"github.com/sokinpui/coder/internal/ui/core/markdown"
)

func (m Model) isLiveAIMessage(idx, total int, msg types.Message) bool {
	return (m.Chat.IsStreaming || m.Chat.IsAIRendering) && idx == total-1 && msg.Type == types.AIMessage
}

func (m Model) getMessageLines(msg types.Message, idx, total, viewportWidth int) []string {
	cache, isCached := m.Chat.RenderCache[idx]

	switch {
	case m.isLiveAIMessage(idx, total, msg):
		if msg.Content == "" || !isCached {
			return nil
		}
		return cache.Lines

	case isCached && cache.Content == msg.Content && cache.Width == viewportWidth:
		return cache.Lines

	default:
		rendered := m.renderMessage(msg, viewportWidth)
		if rendered == "" && msg.Type != types.AIMessage {
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
}

func (m Model) renderConversationWithOffsets() (string, map[int]int) {
	messages := m.Session.GetMessages()
	viewportWidth := m.Chat.Viewport.Width
	m.warmupRenderCache(messages, viewportWidth)

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

func (m Model) warmupRenderCache(messages []types.Message, viewportWidth int) {
	total := len(messages)
	var items []markdown.RenderItem
	for i, msg := range messages {
		if m.isLiveAIMessage(i, total, msg) || msg.Content == "" {
			continue
		}
		cache, ok := m.Chat.RenderCache[i]
		if ok && cache.Content == msg.Content && cache.Width == viewportWidth {
			continue
		}
		if msg.Type == types.AIMessage {
			items = append(items, markdown.RenderItem{Index: i, Content: msg.Content})
		}
	}

	if len(items) <= 1 {
		return
	}

	results := markdown.BatchRender(items, viewportWidth)
	for _, res := range results {
		m.Chat.RenderCache[res.Index] = markdown.CachedRender{
			Lines:   res.Lines,
			Content: res.Content,
			Width:   viewportWidth,
		}
	}
}

func (m Model) renderMessage(msg types.Message, viewportWidth int) string {
	return core.RenderMessage(msg, viewportWidth, m.GlamourRenderer)
}

func (m Model) renderThinkingLine() string {
	text := "Thinking "
	switch m.State {
	case stateAsking:
		text = "Asking "
	}
	return core.RenderThinkingSpinner(text, m.Chat.Spinner.View())
}

func (m *Model) renderConversation() string {
	content, offsets := m.renderConversationWithOffsets()
	m.Chat.MessageLineOffsets = offsets
	return content
}
