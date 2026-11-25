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

func processBoxMiddleLines(items []boxRunItem, metas []LineMeta, msgIdx int) {
	if len(items) == 0 {
		return
	}
	minIndent := -1
	for _, it := range items {
		content := ansi.Cut(it.plain, it.sc, it.ec)
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
			IsContinuation:  false,
		}
	}
}

func buildLineMetasForMessage(msg types.Message, msgIdx int, renderedLines []string, viewportWidth int, isMarkdownRendered bool) []LineMeta {
	if len(renderedLines) == 0 {
		return nil
	}

	metas := make([]LineMeta, len(renderedLines))

	switch msg.Type {
	case types.UserMessage, types.ImageMessage, types.PDFMessage, types.CommandMessage, types.ShellCmdMessage,
		types.ContextCmdMessage, types.FileApplyCmdMessage, types.FileApplyUndoCmdMessage,
		types.CommandErrorResultMessage, types.FileApplyCmdErrorMessage, types.FileApplyUndoCmdErrorMessage,
		types.CompactedSummaryMessage:
		colStart := 2
		var boxRun []boxRunItem
		for i, line := range renderedLines {
			plain := ansi.Strip(line)
			if i == 0 || i == len(renderedLines)-1 || isBoxTopBorder(plain) || isBoxBottomBorder(plain) {
				processBoxMiddleLines(boxRun, metas, msgIdx)
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
			processBoxMiddleLines(boxRun, metas, msgIdx)
			boxRun = nil
			contentTrimmed := strings.TrimRight(strings.TrimSuffix(plain, "│"), " ")
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
		processBoxMiddleLines(boxRun, metas, msgIdx)

		switch msg.Type {
		case types.ImageMessage, types.PDFMessage:
		case types.ShellCmdMessage:
			annotateWrappedLinesFromSource(metas, "Shell: "+msg.Content)
		case types.CompactedSummaryMessage:
			cardWidth := max(30, viewportWidth-4)
			contentWidth := max(20, cardWidth-PreviewBoxStyle.GetHorizontalFrameSize())
			annotateMarkdownWrappedLines(metas, renderedLines, msg.Content, contentWidth, true)
		default:
			annotateWrappedLinesFromSource(metas, msg.Content)
		}

	case types.CommandResultMessage, types.ShellCmdResultMessage, types.ContextCmdResultMessage,
		types.FileApplyCmdResultMessage, types.FileApplyUndoCmdResultMessage, types.InterruptedMessage:
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
		annotateWrappedLinesFromSource(metas, msg.Content)

	default:
		var boxRun []boxRunItem
		for i, line := range renderedLines {
			plain := ansi.Strip(line)
			if isBoxTopBorder(plain) || isBoxBottomBorder(plain) {
				processBoxMiddleLines(boxRun, metas, msgIdx)
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
			processBoxMiddleLines(boxRun, metas, msgIdx)
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
		processBoxMiddleLines(boxRun, metas, msgIdx)

		switch msg.Type {
		case types.InitMessage, types.DirectoryMessage:
			annotateWrappedLinesFromSource(metas, msg.Content)
		case types.AIMessage:
			if isMarkdownRendered {
				annotateMarkdownWrappedLines(metas, renderedLines, msg.Content, viewportWidth, false)
			}
		}
	}

	return metas
}

func annotateWrappedLinesFromSource(metas []LineMeta, src string) {
	if len(metas) == 0 || src == "" {
		return
	}
	src = strings.ReplaceAll(src, "\r\n", "\n")
	pos := 0
	seenContent := false

	for i := range metas {
		if metas[i].IsDecoration {
			continue
		}
		trimmed := strings.TrimSpace(metas[i].Text)

		if !seenContent {
			seenContent = true
			if trimmed != "" {
				if idx := strings.Index(src[pos:], trimmed); idx != -1 {
					pos += idx + len(trimmed)
				}
			}
			continue
		}

		sawNewline := false
		sawSpace := false
		scanPos := pos
		for scanPos < len(src) {
			b := src[scanPos]
			if b == '\n' {
				sawNewline = true
				scanPos++
				if trimmed == "" {
					break
				}
				continue
			}
			if b == ' ' || b == '\t' {
				sawSpace = true
				scanPos++
				continue
			}
			break
		}

		if trimmed == "" {
			if sawNewline {
				pos = scanPos
			}
			metas[i].IsContinuation = false
			continue
		}

		if idx := strings.Index(src[scanPos:], trimmed); idx != -1 {
			if !sawNewline {
				metas[i].IsContinuation = true
				metas[i].WrapJoinSpace = sawSpace || idx > 0
			}
			pos = scanPos + idx + len(trimmed)
		}
	}
}

func annotateMarkdownWrappedLines(metas []LineMeta, renderedLines []string, rawContent string, wrapWidth int, boxed bool) {
	if len(metas) <= 1 || wrapWidth <= 0 {
		return
	}

	maxInnerWidth := max(10, wrapWidth-4)
	docMargin := 2
	if boxed {
		docMargin = 0
	}

	for i := 1; i < len(metas); i++ {
		prev := metas[i-1]
		curr := metas[i]
		if prev.IsDecoration || curr.IsDecoration {
			continue
		}

		prevTrimmed := strings.TrimSpace(prev.Text)
		currTrimmed := strings.TrimSpace(curr.Text)
		if prevTrimmed == "" || currTrimmed == "" {
			continue
		}

		if isMarkdownCodeBlockPair(renderedLines, prevTrimmed, currTrimmed, i) {
			continue
		}
		if isMarkdownBlockStart(currTrimmed) || isMarkdownHorizontalRule(prevTrimmed) || isMarkdownHeadingLine(prevTrimmed) {
			continue
		}

		firstWord := currTrimmed
		if sp := strings.IndexAny(currTrimmed, " \t"); sp != -1 {
			firstWord = currTrimmed[:sp]
		}
		firstWordWidth := ansi.StringWidth(firstWord)

		usedWidth := prev.ContentColEnd - docMargin
		if boxed {
			usedWidth = prev.ContentColEnd - prev.ContentColStart
		}
		if usedWidth+1+firstWordWidth < maxInnerWidth-2 {
			continue
		}

		metas[i].IsContinuation = true
		metas[i].WrapJoinSpace = shouldJoinWithSpace(prevTrimmed, firstWord, rawContent)
	}
}

func isMarkdownCodeBlockPair(renderedLines []string, prevTrimmed, currTrimmed string, idx int) bool {
	if idx <= 0 || idx >= len(renderedLines) {
		return false
	}
	prevRaw := renderedLines[idx-1]
	currRaw := renderedLines[idx]
	if hasIndentedBackgroundSGR(prevRaw) && hasIndentedBackgroundSGR(currRaw) {
		return true
	}
	if isMarkdownBlockStart(prevTrimmed) {
		return false
	}
	return hasIndentedBackgroundSGR(currRaw) && hasIndentedBackgroundSGR(prevRaw)
}

func hasIndentedBackgroundSGR(rawLine string) bool {
	i := 0
	for i < len(rawLine) && rawLine[i] == ' ' {
		i++
	}
	if i < 4 || i >= len(rawLine) || rawLine[i] != '\x1b' {
		return false
	}
	for i < len(rawLine) && rawLine[i] == '\x1b' {
		if i+1 >= len(rawLine) || rawLine[i+1] != '[' {
			break
		}
		end := i + 2
		for end < len(rawLine) && !(rawLine[end] >= 0x40 && rawLine[end] <= 0x7E) {
			end++
		}
		if end >= len(rawLine) {
			break
		}
		seq := rawLine[i+2 : end]
		if rawLine[end] == 'm' && (strings.HasPrefix(seq, "48;") || strings.Contains(seq, ";48;")) {
			return true
		}
		i = end + 1
	}
	return false
}

func isMarkdownBlockStart(trimmed string) bool {
	if trimmed == "" {
		return false
	}
	for _, prefix := range []string{
		"• ", "- ", "* ", "+ ", "[ ] ", "[✓] ", "[x] ", "[X] ",
		"# ", "## ", "### ", "#### ", "##### ", "###### ",
		"│", "┌", "├", "└", "╭", "╰", "┃", "║", "╔", "╚", "┏", "┗",
		"🠶 ", "⚡ ", "↳ ",
	} {
		if strings.HasPrefix(trimmed, prefix) {
			return true
		}
	}
	if isMarkdownHorizontalRule(trimmed) {
		return true
	}
	i := 0
	for i < len(trimmed) && trimmed[i] >= '0' && trimmed[i] <= '9' {
		i++
	}
	return i > 0 && i+1 < len(trimmed) && trimmed[i] == '.' && trimmed[i+1] == ' '
}

func isMarkdownHorizontalRule(trimmed string) bool {
	count := 0
	for _, r := range trimmed {
		if r != '-' && r != '─' && r != '━' {
			return false
		}
		count++
	}
	return count >= 3
}

func isMarkdownHeadingLine(trimmed string) bool {
	for _, prefix := range []string{"# ", "## ", "### ", "#### ", "##### ", "###### "} {
		if strings.HasPrefix(trimmed, prefix) {
			return true
		}
	}
	return false
}

func shouldJoinWithSpace(prevTrimmed, firstWord, rawContent string) bool {
	if prevTrimmed == "" || firstWord == "" {
		return true
	}
	lastWord := prevTrimmed
	if sp := strings.LastIndexAny(prevTrimmed, " \t"); sp != -1 {
		lastWord = prevTrimmed[sp+1:]
	}
	if lastWord == "" || rawContent == "" {
		return true
	}
	if !strings.Contains(rawContent, lastWord+" "+firstWord) &&
		!strings.Contains(rawContent, lastWord+"\n"+firstWord) &&
		strings.Contains(rawContent, lastWord+firstWord) {
		return false
	}
	return true
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

		isAIRendered := false
		if msg.Type == types.AIMessage && msg.Content != "" {
			if cache, ok := m.Chat.RenderCache[i]; ok && cache.Width == viewportWidth {
				isAIRendered = true
			}
		}
		metas := buildLineMetasForMessage(msg, i, lines, viewportWidth, isAIRendered)

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
				processBoxMiddleLines(boxRun, allMetas, -1)
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
			processBoxMiddleLines(boxRun, allMetas, -1)
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
		processBoxMiddleLines(boxRun, allMetas, -1)
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
	if msg.Type == types.ToolCallMessage {
		return m.renderToolCalls(msg.ToolCalls, viewportWidth)
	}
	if msg.Type == types.ToolResultMessage {
		return m.getOrRenderToolResult(msg, viewportWidth)
	}
	return RenderMessage(msg, viewportWidth, nil, m.KittyImage)
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
