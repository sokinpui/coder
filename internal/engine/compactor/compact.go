package compactor

import (
	"context"
	"fmt"
	"strings"

	"github.com/sokinpui/coder/internal/config"
	"github.com/sokinpui/coder/internal/engine/generation"
	"github.com/sokinpui/coder/internal/types"
)

const maxToolResultChars = 4000
const maxCommandResultChars = 2000

func Execute(ctx context.Context, gen *generation.Generator, modelCfg *config.ModelConfig, span []types.Message) (string, error) {
	if len(span) == 0 {
		return "", fmt.Errorf("no messages to compact")
	}

	transcript := buildTranscript(span)
	chatMsgs := []types.ChatMessage{
		{
			Role: types.RoleUser,
			Content: fmt.Sprintf("Here is the conversation history to condense:\n\n%s\n"+
				"Please generate the structured compaction checkpoint according to instructions.", transcript),
		},
	}

	streamChan := make(chan types.StreamChunk, 100)
	go gen.GenerateTask(ctx, CompactionPrompt, chatMsgs, nil, streamChan, modelCfg)

	var sb strings.Builder
	for chunk := range streamChan {
		if chunk.Error != nil {
			return "", chunk.Error
		}
		sb.WriteString(chunk.Content)
	}

	summary := strings.TrimSpace(sb.String())
	if summary == "" {
		return "", fmt.Errorf("compaction model returned an empty summary")
	}

	return summary, nil
}

func buildTranscript(span []types.Message) string {
	var sb strings.Builder

	for _, msg := range span {
		switch msg.Type {
		case types.UserMessage:
			sb.WriteString("User:\n")
			sb.WriteString(msg.Content)
			sb.WriteString("\n\n")

		case types.ImageMessage:
			sb.WriteString(fmt.Sprintf("User:\nimage:%s\n\n", msg.Content))

		case types.AIMessage:
			if msg.Content != "" {
				sb.WriteString("Assistant:\n")
				sb.WriteString(msg.Content)
				sb.WriteString("\n\n")
			}

		case types.ToolCallMessage:
			sb.WriteString("Assistant (Tool Calls):\n")
			for _, tc := range msg.ToolCalls {
				sb.WriteString(fmt.Sprintf("- Call %s(%s)\n", tc.Name, tc.Arguments))
			}
			sb.WriteString("\n")

		case types.ToolResultMessage:
			sb.WriteString(fmt.Sprintf("Tool Result (%s):\n", msg.ToolCallID))
			content := msg.Content
			if len(content) > maxToolResultChars {
				content = content[:maxToolResultChars] + "\n...[truncated for compaction]..."
			}
			sb.WriteString(content)
			sb.WriteString("\n\n")

		case types.CompactedSummaryMessage:
			sb.WriteString("Prior Checkpoint:\n")
			sb.WriteString(msg.Content)
			sb.WriteString("\n\n")

		case types.CommandMessage, types.ShellCmdMessage:
			sb.WriteString(fmt.Sprintf("Command:\n%s\n\n", msg.Content))

		case types.CommandResultMessage, types.ShellCmdResultMessage:
			sb.WriteString("Command Result:\n")
			content := msg.Content
			if len(content) > maxCommandResultChars {
				content = content[:maxCommandResultChars] + "\n...[truncated]..."
			}
			sb.WriteString(content)
			sb.WriteString("\n\n")
		}
	}

	return sb.String()
}
