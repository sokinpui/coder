package core

import (
	"strings"
	"fmt"
)

const (
	systemInstructionsHeader  = "# SYSTEM INSTRUCTIONS\n\n"
	relatedDocumentsHeader    = "# RELATED DOCUMENTS\n\n"
	projectSourceCodeHeader   = "# PROJECT SOURCE CODE\n\n"
	conversationHistoryHeader = "# CONVERSATION HISTORY\n\n"
	separator                 = "\n\n---\n\n"
)

// BuildPrompt constructs the full prompt with conversation history.
func BuildPrompt(role, systemInstructions, relatedDocuments, projectSourceCode string, messages []Message) string {
	var sb strings.Builder

	hasPredefinedContent := false
	if role != "" {
		sb.WriteString(role)
		hasPredefinedContent = true
	}

	if CoderInstructions != "" {
		sb.WriteString(CoderInstructions)
		hasPredefinedContent = true
	}

	// User-defined system instructions (with header)
	if systemInstructions != "" {
		if hasPredefinedContent {
			sb.WriteString(separator)
		}
		sb.WriteString(systemInstructionsHeader)
		sb.WriteString(systemInstructions)
		sb.WriteString(separator)
	} else if hasPredefinedContent {
		// If there are only predefined instructions, we still need a separator
		sb.WriteString(separator)
	}

	if relatedDocuments != "" {
		sb.WriteString(relatedDocumentsHeader)
		sb.WriteString(relatedDocuments)
		sb.WriteString(separator)
	}

	if projectSourceCode != "" {
		sb.WriteString(projectSourceCodeHeader)
		sb.WriteString(projectSourceCode)
		sb.WriteString(separator)
	}

	if len(messages) > 0 {
		sb.WriteString(conversationHistoryHeader)

		for i := 0; i < len(messages); i++ {
			msg := messages[i]

			switch msg.Type {
			case UserMessage:
				sb.WriteString("User:\n")
				sb.WriteString(msg.Content)
				sb.WriteString("\n")
			case ImageMessage:
				sb.WriteString(msg.Content)
				sb.WriteString("\n")
			case AIMessage:
				if msg.Content == "" {
					continue
				}
				sb.WriteString("AI Assistant:\n")
				sb.WriteString(msg.Content)
				sb.WriteString("\n")
			}
		}
		sb.WriteString("AI Assistant:\n")
	}

	return sb.String()
}

// BuildHistorySnippet constructs a string representation of a list of messages for copying.
func BuildHistorySnippet(messages []Message) string {
	var sb strings.Builder

	for i := 0; i < len(messages); i++ {
		msg := messages[i]

		switch msg.Type {
		case UserMessage:
			sb.WriteString("User:\n")
			sb.WriteString(msg.Content)
		case ImageMessage:
			sb.WriteString("Image:\n")
			sb.WriteString(fmt.Sprintf("![image](%s)", msg.Content))
		case AIMessage:
			if msg.Content == "" {
				continue
			}
			sb.WriteString("AI Assistant:\n")
			sb.WriteString(msg.Content)
		case CommandMessage:
			sb.WriteString("Command Execute:\n")
			sb.WriteString(msg.Content)
		case CommandResultMessage:
			sb.WriteString("Command Execute Result:\n")
			sb.WriteString(msg.Content)
		case CommandErrorResultMessage:
			sb.WriteString("Command Execute Error:\n")
			sb.WriteString(msg.Content)
		default:
			// Skip system messages like InitMessage, DirectoryMessage
			continue
		}
		sb.WriteString("\n\n")
	}

	return strings.TrimSpace(sb.String())
}
