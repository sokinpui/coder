package core

import (
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/lipgloss"
	"github.com/sokinpui/coder/internal/types"
	"github.com/sokinpui/coder/internal/ui/core/markdown"
)

type Viewport struct {
	viewport.Model
	Renderer           *markdown.Renderer
	RenderCache        map[int]markdown.CachedRender
	MessageLineOffsets map[int]int
	ToolsExpanded      bool
	ToolExpandedCache  map[int][]string
	callIDToName       map[string]string
}

func NewViewport(width, height int) Viewport {
	vp := viewport.New(width, height)
	renderer, _ := markdown.NewRenderer(width)
	return Viewport{
		Model:              vp,
		Renderer:           renderer,
		RenderCache:        make(map[int]markdown.CachedRender),
		MessageLineOffsets: make(map[int]int),
		ToolExpandedCache:  make(map[int][]string),
		callIDToName:       make(map[string]string),
	}
}

func (v *Viewport) ClearCache() {
	v.RenderCache = make(map[int]markdown.CachedRender)
	v.ToolExpandedCache = make(map[int][]string)
	v.callIDToName = make(map[string]string)
}

func (v *Viewport) Resize(width, height int) {
	v.Width = width
	v.Height = height
	if v.Renderer == nil || v.Renderer.Width() != width {
		renderer, err := markdown.NewRenderer(width)
		if err == nil {
			v.Renderer = renderer
		}
	}
	v.ClearCache()
}

func (v *Viewport) WarmupCache(messages []types.Message, isStreaming bool) {
	total := len(messages)
	viewportWidth := v.Width
	var items []markdown.RenderItem

	for i, msg := range messages {
		if isStreaming && i == total-1 && msg.Type == types.AIMessage {
			continue
		}
		if msg.Content == "" {
			continue
		}
		cache, ok := v.RenderCache[i]
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
		v.RenderCache[res.Index] = markdown.CachedRender{
			Lines:   res.Lines,
			Content: res.Content,
			Width:   viewportWidth,
		}
	}
}

func (v *Viewport) GetMessageLines(msg types.Message, idx, total int, isStreaming bool) []string {
	viewportWidth := v.Width
	isLiveAI := isStreaming && idx == total-1 && msg.Type == types.AIMessage

	if len(msg.ToolCalls) > 0 {
		for _, tc := range msg.ToolCalls {
			if tc.ID != "" && tc.Name != "" {
				v.callIDToName[tc.ID] = tc.Name
			}
		}
	}

	if msg.Type == types.ToolCallMessage || msg.Type == types.ToolResultMessage {
		if v.ToolsExpanded {
			if lines, ok := v.ToolExpandedCache[idx]; ok {
				return lines
			}
			rendered := v.renderToolMessage(msg, true)
			if rendered == "" {
				v.ToolExpandedCache[idx] = nil
				return nil
			}
			lines := strings.Split(rendered, "\n")
			v.ToolExpandedCache[idx] = lines
			return lines
		}

		cache, isCached := v.RenderCache[idx]
		if isCached && cache.Content == msg.Content && cache.Width == viewportWidth {
			return cache.Lines
		}

		rendered := v.renderToolMessage(msg, false)
		if rendered == "" {
			return nil
		}

		lines := strings.Split(rendered, "\n")
		v.RenderCache[idx] = markdown.CachedRender{
			Lines:   lines,
			Content: msg.Content,
			Width:   viewportWidth,
		}
		return lines
	}

	if msg.Type == types.AIMessage && len(msg.ToolCalls) > 0 && v.ToolsExpanded {
		if lines, ok := v.ToolExpandedCache[idx]; ok {
			return lines
		}
		renderedCalls := RenderToolCallMessage(msg, true, viewportWidth)
		renderedAI := RenderMessage(types.Message{Type: types.AIMessage, Content: msg.Content}, viewportWidth, v.Renderer)
		combined := strings.TrimSpace(renderedCalls + "\n" + renderedAI)
		lines := strings.Split(combined, "\n")
		v.ToolExpandedCache[idx] = lines
		return lines
	}

	cache, isCached := v.RenderCache[idx]

	if isLiveAI {
		if msg.Content == "" || !isCached {
			return nil
		}
		return cache.Lines
	}

	if isCached && cache.Content == msg.Content && cache.Width == viewportWidth {
		return cache.Lines
	}

	rendered := RenderMessage(msg, viewportWidth, v.Renderer)
	if rendered == "" && msg.Type != types.AIMessage {
		return nil
	}

	lines := strings.Split(rendered, "\n")
	v.RenderCache[idx] = markdown.CachedRender{
		Lines:   lines,
		Content: msg.Content,
		Width:   viewportWidth,
	}
	return lines
}

func (v *Viewport) renderToolMessage(msg types.Message, expanded bool) string {
	if msg.Type == types.ToolCallMessage {
		return RenderToolCallMessage(msg, expanded, v.Width)
	}
	toolName := v.callIDToName[msg.ToolCallID]
	return RenderToolResultMessage(msg, toolName, expanded, v.Width)
}

func (v *Viewport) PrecomputeToolExpanded(idx int, msg types.Message) {
	if msg.Type != types.ToolCallMessage && msg.Type != types.ToolResultMessage {
		return
	}
	v.syncToolCallIDs([]types.Message{msg})
	if _, ok := v.ToolExpandedCache[idx]; ok {
		return
	}
	rendered := v.renderToolMessage(msg, true)
	if rendered == "" {
		v.ToolExpandedCache[idx] = nil
		return
	}
	v.ToolExpandedCache[idx] = strings.Split(rendered, "\n")
}

func (v *Viewport) RenderMessages(messages []types.Message, isStreaming bool, trailingLine string) string {
	v.syncToolCallIDs(messages)
	v.WarmupCache(messages, isStreaming)

	offsets := make(map[int]int, len(messages))
	currentLine := 0
	var allLines []string

	total := len(messages)
	for i, msg := range messages {
		offsets[i] = currentLine
		lines := v.GetMessageLines(msg, i, total, isStreaming)
		allLines = append(allLines, lines...)
		currentLine += len(lines)
	}

	if trailingLine != "" {
		allLines = append(allLines, strings.Split(trailingLine, "\n")...)
	}

	v.MessageLineOffsets = offsets
	return strings.Join(allLines, "\n")
}

func (v *Viewport) syncToolCallIDs(messages []types.Message) {
	for _, msg := range messages {
		if msg.Type == types.ToolCallMessage || msg.Type == types.AIMessage {
			for _, tc := range msg.ToolCalls {
				if tc.ID != "" && tc.Name != "" {
					v.callIDToName[tc.ID] = tc.Name
				}
			}
		}
	}
}

func (v *Viewport) UpdateContent(messages []types.Message, isStreaming bool, trailingLine string) {
	content := v.RenderMessages(messages, isStreaming, trailingLine)
	v.SetContent(content)
}

func (v *Viewport) SyncToMessage(msgIdx int) {
	if line, ok := v.MessageLineOffsets[msgIdx]; ok {
		targetY := max(line-(v.Height/3), 0)
		v.SetYOffset(targetY)
	}
}

func RenderThinkingSpinner(text, spinnerView string) string {
	thinkingText := ThinkingTextStyle.Render(text)
	fullMessage := lipgloss.JoinHorizontal(lipgloss.Bottom, thinkingText, spinnerView)
	return lipgloss.NewStyle().Padding(0, 2).Render(fullMessage)
}
