package ui

import (
	"fmt"
	"runtime"
	"strings"
	"sync"

	"github.com/sokinpui/coder/internal/types"

	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
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
		return cache.lines

	case isCached && cache.content == msg.Content && cache.width == viewportWidth:
		return cache.lines

	default:
		rendered := m.renderMessage(msg, viewportWidth)
		if rendered == "" && msg.Type != types.AIMessage {
			return nil
		}
		lines := strings.Split(rendered, "\n")
		m.Chat.RenderCache[idx] = cachedRender{
			lines:   lines,
			content: msg.Content,
			width:   viewportWidth,
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

	if m.State == stateAsking || m.State == stateThinking || m.State == stateExecutingTool {
		thinkingLine := m.renderThinkingLine()
		allLines = append(allLines, strings.Split(thinkingLine, "\n")...)
	}
	return strings.Join(allLines, "\n"), messageLineOffsets
}

type renderJob struct {
	index int
	msg   types.Message
}

type renderResult struct {
	index   int
	content string
	lines   []string
}

func (m Model) warmupRenderCache(messages []types.Message, viewportWidth int) {
	total := len(messages)
	var uncached []renderJob
	for i, msg := range messages {
		if m.isLiveAIMessage(i, total, msg) || msg.Content == "" {
			continue
		}
		cache, ok := m.Chat.RenderCache[i]
		if ok && cache.content == msg.Content && cache.width == viewportWidth {
			continue
		}
		uncached = append(uncached, renderJob{index: i, msg: msg})
	}

	if len(uncached) <= 1 {
		return
	}

	workerCount := max(min(len(uncached), runtime.NumCPU()), 1)

	jobs := make(chan renderJob, len(uncached))
	for _, j := range uncached {
		jobs <- j
	}
	close(jobs)

	results := make(chan renderResult, len(uncached))
	var wg sync.WaitGroup
	theme := m.Session.GetConfig().UI.MarkdownTheme

	for range workerCount {
		wg.Go(func() {
			renderer, _ := glamour.NewTermRenderer(
				glamour.WithStandardStyle(theme),
				glamour.WithWordWrap(viewportWidth),
			)
			for job := range jobs {
				rendered := renderMessageWithRenderer(job.msg, viewportWidth, renderer)
				if rendered != "" || job.msg.Type == types.AIMessage {
					results <- renderResult{
						index:   job.index,
						content: job.msg.Content,
						lines:   strings.Split(rendered, "\n"),
					}
				}
			}
		})
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	for res := range results {
		m.Chat.RenderCache[res.index] = cachedRender{
			lines:   res.lines,
			content: res.content,
			width:   viewportWidth,
		}
	}
}

func (m Model) renderMessage(msg types.Message, viewportWidth int) string {
	return renderMessageWithRenderer(msg, viewportWidth, m.GlamourRenderer)
}

func renderMessageWithRenderer(msg types.Message, viewportWidth int, renderer *glamour.TermRenderer) string {
	content := msg.Content
	switch msg.Type {
	case types.InitMessage:
		return initMessageStyle.Width(viewportWidth - initMessageStyle.GetHorizontalFrameSize()).Render(content)
	case types.DirectoryMessage:
		return directoryWelcomeStyle.Width(viewportWidth - directoryWelcomeStyle.GetHorizontalFrameSize()).Render(content)
	case types.UserMessage:
		return userInputStyle.Width(viewportWidth - userInputStyle.GetHorizontalFrameSize()).Render(content)
	case types.CommandMessage, types.ShellCmdMessage, types.ContextCmdMessage,
		types.FileApplyCmdMessage, types.FileApplyUndoCmdMessage:
		prefix := ""
		if msg.Type == types.ShellCmdMessage {
			prefix = "Shell: "
		}
		return commandInputStyle.Width(viewportWidth - commandInputStyle.GetHorizontalFrameSize()).Render(prefix + content)
	case types.ImageMessage:
		return imageMessageStyle.Width(viewportWidth - imageMessageStyle.GetHorizontalFrameSize()).Render("Image: " + content)
	case types.AIMessage:
		if content == "" {
			return ""
		}
		if renderer == nil {
			return content
		}
		renderedAI, err := renderer.Render(content)
		if err != nil {
			return content
		}
		return renderedAI
	case types.CommandResultMessage, types.ShellCmdResultMessage, types.ContextCmdResultMessage,
		types.FileApplyCmdResultMessage, types.FileApplyUndoCmdResultMessage:
		return commandResultStyle.Width(viewportWidth - commandResultStyle.GetHorizontalFrameSize()).Render(content)
	case types.CommandErrorResultMessage,
		types.FileApplyCmdErrorMessage, types.FileApplyUndoCmdErrorMessage:
		return commandErrorStyle.Width(viewportWidth - commandErrorStyle.GetHorizontalFrameSize()).Render(content)
	case types.ToolCallMessage:
		header := "Tool Call"
		if msg.ToolName != "" {
			header = fmt.Sprintf("Tool Call: %s", msg.ToolName)
		}
		display := header
		if strings.TrimSpace(content) != "" {
			display = fmt.Sprintf("%s\n%s", header, content)
		}
		return toolCallStyle.Width(viewportWidth - toolCallStyle.GetHorizontalFrameSize()).Render(display)
	case types.ToolCallResultMessage:
		header := "Tool Result"
		if msg.ToolName != "" {
			header = fmt.Sprintf("Tool Result [%s]", msg.ToolName)
		}
		display := fmt.Sprintf("%s:\n%s", header, content)
		return toolCallResultStyle.Width(viewportWidth - toolCallResultStyle.GetHorizontalFrameSize()).Render(display)
	default:
		return ""
	}
}

func (m Model) renderThinkingLine() string {
	text := "Thinking "
	switch m.State {
	case stateAsking:
		text = "Asking "
	case stateExecutingTool:
		if m.Chat.ActiveToolName != "" {
			text = fmt.Sprintf("Executing %s ", m.Chat.ActiveToolName)
		} else {
			text = "Executing tool "
		}
	}
	thinkingText := thinkingTextStyle.Render(text)
	fullMessage := lipgloss.JoinHorizontal(lipgloss.Bottom, thinkingText, m.Chat.Spinner.View())
	return lipgloss.NewStyle().Padding(0, 2).Render(fullMessage)
}

func (m *Model) renderConversation() string {
	content, offsets := m.renderConversationWithOffsets()
	m.Chat.MessageLineOffsets = offsets
	return content
}
