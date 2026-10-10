package ui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/sokinpui/coder/internal/types"
	"github.com/sokinpui/coder/internal/ui/markdown"
)

func (m Model) handleRenderMessage(msg tea.Msg) (tea.Model, tea.Cmd, bool) {
	switch msg := msg.(type) {
	case aiRenderedMsg:
		if m.Session == nil || m.Session.GetID() != msg.sessID {
			return m, nil, true
		}
		m.Chat.IsAIRendering = false
		messages := m.Session.GetMessages()
		if msg.msgIdx < 0 || msg.msgIdx >= len(messages) {
			return m, nil, true
		}
		if messages[msg.msgIdx].Type != types.AIMessage {
			return m, nil, true
		}

		m.Chat.RenderCache[msg.msgIdx] = markdown.CachedRender{
			Lines:   msg.lines,
			Content: msg.content,
			Width:   msg.width,
		}

		m.Chat.Viewport.SetContent(m.renderConversation())
		if m.Chat.AutoScroll {
			m.Chat.Viewport.GotoBottom()
		}
		if m.Chat.PendingAIRender || messages[msg.msgIdx].Content != msg.content || msg.width != m.Chat.Viewport.Width {
			return m.renderLastAIMessage(msg.sessID)
		}
		return m, nil, true

	case markdownBatchRenderedMsg:
		if m.Session == nil || m.Session.GetID() != msg.sessID || msg.width != m.Chat.Viewport.Width {
			return m, nil, true
		}

		for _, res := range msg.results {
			m.Chat.RenderCache[res.Index] = markdown.CachedRender{
				Lines:   res.Lines,
				Content: res.Content,
				Width:   res.Width,
			}
		}

		m.Chat.Viewport.SetContent(m.renderConversation())
		if m.Chat.AutoScroll {
			m.Chat.Viewport.GotoBottom()
		}
		return m, nil, true

	case toolsBatchRenderedMsg:
		if m.Session == nil || m.Session.GetID() != msg.sessID || msg.width != m.Chat.Viewport.Width {
			return m, nil, true
		}

		for _, res := range msg.results {
			m.Chat.ToolRenderCache[res.key] = res.value
		}

		m.Chat.Viewport.SetContent(m.renderConversation())
		if m.Chat.AutoScroll {
			m.Chat.Viewport.GotoBottom()
		}
		return m, nil, true
	}

	return m, nil, false
}

func (m Model) renderLastAIMessage(sessID string) (tea.Model, tea.Cmd, bool) {
	messages := m.Session.GetMessages()
	lastIdx := len(messages) - 1
	if lastIdx < 0 || messages[lastIdx].Type != types.AIMessage {
		return m, nil, true
	}

	m.Chat.PendingAIRender = false
	m.Chat.IsAIRendering = true
	viewportWidth := max(10, m.Chat.Viewport.Width)
	return m, renderAIMessageCmd(sessID, lastIdx, messages[lastIdx].Content, viewportWidth), true
}

func (m Model) finalizeAIMessageRender(sessID string) (Model, tea.Cmd) {
	if m.Chat.IsAIRendering {
		m.Chat.PendingAIRender = true
		return m, nil
	}

	messages := m.Session.GetMessages()
	lastIdx := len(messages) - 1
	if lastIdx < 0 || messages[lastIdx].Type != types.AIMessage {
		m.Chat.Viewport.SetContent(m.renderConversation())
		if m.Chat.AutoScroll {
			m.Chat.Viewport.GotoBottom()
		}
		return m, nil
	}

	cache, ok := m.Chat.RenderCache[lastIdx]
	isStale := !ok || cache.Content != messages[lastIdx].Content || cache.Width != m.Chat.Viewport.Width
	if isStale {
		m.Chat.IsAIRendering = true
		m.Chat.PendingAIRender = false
		viewportWidth := max(10, m.Chat.Viewport.Width)
		return m, renderAIMessageCmd(sessID, lastIdx, messages[lastIdx].Content, viewportWidth)
	}

	m.Chat.Viewport.SetContent(m.renderConversation())
	if m.Chat.AutoScroll {
		m.Chat.Viewport.GotoBottom()
	}
	return m, nil
}
