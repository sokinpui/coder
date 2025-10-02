package modes

import (
	"coder/internal/core"
	"strings"
)

const (
	systemInstructionsHeader  = "# SYSTEM INSTRUCTIONS\n\n"
	relatedDocumentsHeader    = "# RELATED DOCUMENTS\n\n"
	projectSourceCodeHeader   = "# PROJECT SOURCE CODE\n\n"
	conversationHistoryHeader = "# CONVERSATION HISTORY\n\n"
	separator                 = "\n\n---\n\n"
)

// BuildPrompt constructs the full prompt from its components.
// This is a generic builder used by mode strategies and other parts of the application.
func BuildPrompt(role, instructions, systemInstructions, relatedDocuments, projectSourceCode string, messages []core.Message) string {
	var sb strings.Builder

	hasPredefinedContent := false
	if role != "" {
		sb.WriteString(role)
		hasPredefinedContent = true
	}

	if instructions != "" {
		sb.WriteString(instructions)
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
			case core.UserMessage:
				sb.WriteString("User:\n")
				sb.WriteString(msg.Content)
				sb.WriteString("\n")
			case core.ImageMessage:
				sb.WriteString(msg.Content)
				sb.WriteString("\n")
			case core.AIMessage:
				if msg.Content == "" {
					continue
				}
				sb.WriteString("AI Assistant:\n")
				sb.WriteString(msg.Content)
				sb.WriteString("\n")
			case core.ToolCallMessage:
				sb.WriteString("Tool Call:\n")
				sb.WriteString(msg.Content)
				sb.WriteString("\n")
			case core.ToolResultMessage:
				sb.WriteString("Tool Result:\n")
				sb.WriteString(msg.Content)
				sb.WriteString("\n")
			}
		}
		sb.WriteString("AI Assistant:\n")
	}

	return sb.String()
}
