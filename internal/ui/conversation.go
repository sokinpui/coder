package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/sokinpui/coder/internal/types"
	"github.com/sokinpui/coder/internal/ui/markdown"
)

func (m Model) getAIMessageLines(msg types.Message, idx, total, viewportWidth int) []string {
	var lines []string
	if len(msg.ToolCalls) > 0 {
		tcPart := m.renderToolCalls(msg.ToolCalls, viewportWidth)
		if tcPart != "" {
			lines = append(lines, strings.Split(tcPart, "\n")...)
		}
	}

	if msg.Content == "" {
		return lines
	}

	cache, isCached := m.Chat.RenderCache[idx]
	if isCached && cache.Width == viewportWidth {
		return append(lines, cache.Lines...)
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

	if msg.Type == types.ToolCallMessage {
		if len(msg.ToolCalls) > 0 {
			rendered := m.renderToolCalls(msg.ToolCalls, viewportWidth)
			if rendered == "" {
				return nil
			}
			normalized := strings.ReplaceAll(rendered, "\r\n", "\n")
			return strings.Split(normalized, "\n")
		}
		if msg.Content == "" {
			return nil
		}
		normalized := strings.ReplaceAll(ToolCallStyle.Render("⚡ "+msg.Content), "\r\n", "\n")
		return strings.Split(normalized, "\n")
	}

	if msg.Type == types.ToolResultMessage {
		rendered := m.getOrRenderToolResult(msg, viewportWidth)
		if rendered == "" {
			return nil
		}
		normalized := strings.ReplaceAll(rendered, "\r\n", "\n")
		return strings.Split(normalized, "\n")
	}

	if msg.Type == types.CompactedSummaryMessage {
		rendered := RenderCompactedSummaryMessage(msg, viewportWidth)
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

func isBoxTopBorder(plain string) bool {
	trimmed := strings.TrimSpace(plain)
	if len(trimmed) < 2 {
		return false
	}
	isCorner := (strings.HasPrefix(trimmed, "╭") && strings.HasSuffix(trimmed, "╮")) ||
		(strings.HasPrefix(trimmed, "┌") && strings.HasSuffix(trimmed, "┐")) ||
		(strings.HasPrefix(trimmed, "┏") && strings.HasSuffix(trimmed, "┓")) ||
		(strings.HasPrefix(trimmed, "╔") && strings.HasSuffix(trimmed, "╗"))
	return isCorner && strings.ContainsAny(trimmed, "─━═")
}

func isBoxBottomBorder(plain string) bool {
	trimmed := strings.TrimSpace(plain)
	if len(trimmed) < 2 {
		return false
	}
	isCorner := (strings.HasPrefix(trimmed, "╰") && strings.HasSuffix(trimmed, "╯")) ||
		(strings.HasPrefix(trimmed, "└") && strings.HasSuffix(trimmed, "┘")) ||
		(strings.HasPrefix(trimmed, "┗") && strings.HasSuffix(trimmed, "┛")) ||
		(strings.HasPrefix(trimmed, "╚") && strings.HasSuffix(trimmed, "╝"))
	return isCorner && strings.ContainsAny(trimmed, "─━═")
}

func isVerticalBorderRune(r rune) bool {
	return r == '│' || r == '┃' || r == '║'
}

func parseBoxMiddleLine(plain string) (int, int, bool) {
	trimmedRight := strings.TrimRight(plain, " \r\n")
	trimmedLeft := strings.TrimLeft(plain, " ")
	if len(trimmedLeft) == 0 || len(trimmedRight) == 0 {
		return 0, 0, false
	}
	runesLeft := []rune(trimmedLeft)
	runesRight := []rune(trimmedRight)
	if !isVerticalBorderRune(runesLeft[0]) || !isVerticalBorderRune(runesRight[len(runesRight)-1]) {
		return 0, 0, false
	}
	leadingCols := ansi.StringWidth(plain[:len(plain)-len(trimmedLeft)])
	colStart := leadingCols + ansi.StringWidth(string(runesLeft[0]))
	if len(runesLeft) > 1 && runesLeft[1] == ' ' {
		colStart++
	}
	colEnd := max(colStart, ansi.StringWidth(strings.TrimRight(string(runesRight[:len(runesRight)-1]), " ")))
	return colStart, colEnd, true
}

type boxRunItem struct {
	index int
	sc    int
	ec    int
	plain string
}

func processBoxMiddleLines(items []boxRunItem, metas []LineMeta, msgIdx int, isContinuation bool) {
	if len(items) == 0 {
		return
	}
	minIndent := -1
	for _, it := range items {
		content := it.plain[it.sc:it.ec]
		trimmed := strings.TrimRight(content, " ")
		if len(trimmed) == 0 {
			continue
		}
		leading := len(trimmed) - len(strings.TrimLeft(trimmed, " "))
		if minIndent == -1 || leading < minIndent {
			minIndent = leading
		}
	}
	if minIndent < 0 {
		minIndent = 0
	}
	for _, it := range items {
		sc := it.sc + minIndent
		rawText := ansi.Cut(it.plain, sc, it.ec)
		metas[it.index] = LineMeta{
			MsgIndex:        msgIdx,
			ContentColStart: sc,
			ContentColEnd:   it.ec,
			Text:            rawText,
			IsContinuation:  isContinuation,
		}
	}
}

func buildLineMetasForMessage(msg types.Message, msgIdx int, renderedLines []string) []LineMeta {
	if len(renderedLines) == 0 {
		return nil
	}

	metas := make([]LineMeta, len(renderedLines))

	switch msg.Type {
	case types.UserMessage, types.ImageMessage, types.CommandMessage, types.ShellCmdMessage,
		types.ContextCmdMessage, types.FileApplyCmdMessage, types.FileApplyUndoCmdMessage,
		types.CommandErrorResultMessage, types.FileApplyCmdErrorMessage, types.FileApplyUndoCmdErrorMessage,
		types.CompactedSummaryMessage:
		colStart := 2
		var boxRun []boxRunItem
		for i, line := range renderedLines {
			plain := ansi.Strip(line)
			if i == 0 || i == len(renderedLines)-1 || isBoxTopBorder(plain) || isBoxBottomBorder(plain) {
				processBoxMiddleLines(boxRun, metas, msgIdx, true)
				boxRun = nil
				metas[i] = LineMeta{
					MsgIndex:     msgIdx,
					IsDecoration: true,
				}
				continue
			}
			if sc, ec, ok := parseBoxMiddleLine(plain); ok {
				boxRun = append(boxRun, boxRunItem{index: i, sc: sc, ec: ec, plain: plain})
				continue
			}
			processBoxMiddleLines(boxRun, metas, msgIdx, true)
			boxRun = nil
			contentTrimmed := strings.TrimRight(strings.TrimSuffix(plain, "│"), " ")
			contentEnd := max(colStart, ansi.StringWidth(contentTrimmed))
			rawText := ansi.Cut(plain, colStart, contentEnd)

			metas[i] = LineMeta{
				MsgIndex:        msgIdx,
				ContentColStart: colStart,
				ContentColEnd:   contentEnd,
				Text:            rawText,
				IsContinuation:  i > 1,
			}
		}
		processBoxMiddleLines(boxRun, metas, msgIdx, true)

	case types.CommandResultMessage, types.ShellCmdResultMessage, types.ContextCmdResultMessage,
		types.FileApplyCmdResultMessage, types.FileApplyUndoCmdResultMessage:
		colStart := 2
		for i, line := range renderedLines {
			plain := ansi.Strip(line)
			contentTrimmed := strings.TrimRight(plain, " ")
			contentEnd := max(colStart, ansi.StringWidth(contentTrimmed))
			rawText := ansi.Cut(plain, colStart, contentEnd)
			metas[i] = LineMeta{
				MsgIndex:        msgIdx,
				ContentColStart: colStart,
				ContentColEnd:   contentEnd,
				Text:            rawText,
				IsContinuation:  false,
			}
		}

	default:
		var boxRun []boxRunItem
		for i, line := range renderedLines {
			plain := ansi.Strip(line)
			if isBoxTopBorder(plain) || isBoxBottomBorder(plain) {
				processBoxMiddleLines(boxRun, metas, msgIdx, false)
				boxRun = nil
				metas[i] = LineMeta{
					MsgIndex:     msgIdx,
					IsDecoration: true,
				}
				continue
			}
			if sc, ec, ok := parseBoxMiddleLine(plain); ok {
				boxRun = append(boxRun, boxRunItem{index: i, sc: sc, ec: ec, plain: plain})
				continue
			}
			processBoxMiddleLines(boxRun, metas, msgIdx, false)
			boxRun = nil
			contentTrimmed := strings.TrimRight(plain, " \r\n")
			if len(contentTrimmed) == 0 {
				metas[i] = LineMeta{
					MsgIndex:        msgIdx,
					ContentColStart: 0,
					ContentColEnd:   0,
					Text:            "",
					IsContinuation:  false,
				}
				continue
			}

			leadingSpaces := len(plain) - len(strings.TrimLeft(plain, " "))
			colStart := min(leadingSpaces, 4)

			contentEnd := ansi.StringWidth(contentTrimmed)
			rawText := ansi.Cut(plain, colStart, contentEnd)
			metas[i] = LineMeta{
				MsgIndex:        msgIdx,
				ContentColStart: colStart,
				ContentColEnd:   contentEnd,
				Text:            rawText,
				IsContinuation:  false,
			}
		}
		processBoxMiddleLines(boxRun, metas, msgIdx, false)
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

		for j, l := range lines {
			if strings.Contains(l, "\t") {
				lines[j] = markdown.ExpandTabs(l, 4)
			}
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
		startIdx := len(allMetas)
		var boxRun []boxRunItem
		for i := len(allMetas); i < len(allLines); i++ {
			plain := ansi.Strip(allLines[i])
			if isBoxTopBorder(plain) || isBoxBottomBorder(plain) {
				processBoxMiddleLines(boxRun, allMetas, -1, false)
				boxRun = nil
				allMetas = append(allMetas, LineMeta{
					MsgIndex:     -1,
					IsDecoration: true,
				})
				continue
			}
			if sc, ec, ok := parseBoxMiddleLine(plain); ok {
				allMetas = append(allMetas, LineMeta{})
				boxRun = append(boxRun, boxRunItem{index: i, sc: sc, ec: ec, plain: plain})
				continue
			}
			processBoxMiddleLines(boxRun, allMetas, -1, false)
			boxRun = nil
			contentTrimmed := strings.TrimRight(plain, " \r\n")
			colStart := 0
			if len(contentTrimmed) > 0 {
				leading := len(plain) - len(strings.TrimLeft(plain, " "))
				colStart = min(leading, 4)
			}
			contentEnd := ansi.StringWidth(contentTrimmed)
			allMetas = append(allMetas, LineMeta{
				MsgIndex:        -1,
				ContentColStart: colStart,
				ContentColEnd:   contentEnd,
				Text:            ansi.Cut(plain, colStart, contentEnd),
			})
		}
		processBoxMiddleLines(boxRun, allMetas, -1, false)
		_ = startIdx
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
		return m.renderToolCalls(msg.ToolCalls, viewportWidth)
	}
	if msg.Type == types.ToolResultMessage {
		return m.getOrRenderToolResult(msg, viewportWidth)
	}
	return RenderMessage(msg, viewportWidth, nil)
}

func (m Model) renderToolCalls(toolCalls []types.ToolCall, width int) string {
	if len(toolCalls) == 0 {
		return ""
	}
	lines := make([]string, 0, len(toolCalls))
	for _, tc := range toolCalls {
		lines = append(lines, m.getOrRenderToolCall(tc, width))
	}
	return strings.Join(lines, "\n")
}

func (m Model) getOrRenderToolCall(tc types.ToolCall, width int) string {
	key := ToolCacheKey{
		ID:    ToolCallKeyID(tc),
		Mode:  m.ToolMode,
		Width: width,
	}
	if cached, ok := m.Chat.ToolRenderCache[key]; ok {
		return cached
	}
	rendered := RenderToolCall(tc, m.ToolMode, width)
	m.Chat.ToolRenderCache[key] = rendered
	return rendered
}

func (m Model) getOrRenderToolResult(msg types.Message, width int) string {
	toolName := m.getToolNameForCallID(msg.ToolCallID)
	key := ToolCacheKey{
		ID:    ToolResultKeyID(msg.ToolCallID, toolName, msg.Content),
		Mode:  m.ToolMode,
		Width: width,
	}
	if cached, ok := m.Chat.ToolRenderCache[key]; ok {
		return cached
	}
	rendered := RenderToolResult(msg.Content, msg.ToolCallID, toolName, m.ToolMode, width)
	m.Chat.ToolRenderCache[key] = rendered
	return rendered
}

func (m Model) getToolNameForCallID(callID string) string {
	return GetToolNameForCallID(m.Session.GetMessages(), callID)
}

func GetToolNameForCallID(messages []types.Message, callID string) string {
	if callID == "" {
		return ""
	}
	for _, msg := range messages {
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
