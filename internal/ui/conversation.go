package ui

import (
	"runtime"
	"strings"
	"sync"

	"github.com/sokinpui/coder/internal/types"

	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
)

func (m Model) renderConversationWithOffsets() (string, map[int]int) {
	messages := m.Session.GetMessages()
	viewportWidth := m.Chat.Viewport.Width
	m.warmupRenderCache(messages, viewportWidth)

	messageLineOffsets := make(map[int]int)
	currentLine := 0
	var allLines []string

	for i, msg := range messages {
		messageLineOffsets[i] = currentLine
		var lines []string

		cache, ok := m.Chat.RenderCache[i]
		if ok && cache.content == msg.Content && cache.width == viewportWidth {
			lines = cache.lines
		} else {
			renderedMsg := m.renderMessage(msg, viewportWidth)

			if renderedMsg != "" || msg.Type == types.AIMessage {
				lines = strings.Split(renderedMsg, "\n")
				m.Chat.RenderCache[i] = cachedRender{
					lines:   lines,
					content: msg.Content,
					width:   viewportWidth,
				}
			}
		}

		allLines = append(allLines, lines...)
		currentLine += len(lines)
	}

	if m.State == stateAsking || m.State == stateThinking {
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
	var uncached []renderJob
	for i, msg := range messages {
		cache, ok := m.Chat.RenderCache[i]
		if !ok || cache.content != msg.Content || cache.width != viewportWidth {
			uncached = append(uncached, renderJob{index: i, msg: msg})
		}
	}

	if len(uncached) <= 1 {
		return
	}

	workerCount := min(len(uncached), runtime.NumCPU())
	if workerCount < 1 {
		workerCount = 1
	}

	jobs := make(chan renderJob, len(uncached))
	for _, j := range uncached {
		jobs <- j
	}
	close(jobs)

	results := make(chan renderResult, len(uncached))
	var wg sync.WaitGroup
	theme := m.Session.GetConfig().UI.MarkdownTheme

	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
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
		}()
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
	case types.CommandMessage, types.ShellCmdMessage:
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
	case types.CommandResultMessage, types.ShellCmdResultMessage:
		return commandResultStyle.Width(viewportWidth - commandResultStyle.GetHorizontalFrameSize()).Render(content)
	case types.CommandErrorResultMessage:
		return commandErrorStyle.Width(viewportWidth - commandErrorStyle.GetHorizontalFrameSize()).Render(content)
	default:
		return ""
	}
}

func (m Model) renderThinkingLine() string {
	text := "Thinking "
	if m.State == stateAsking {
		text = "Asking "
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
