package ui

import (
	"strings"
)

const (
	systemInstructionsHeader  = "# SYSTEM INSTRUCTIONS\n\n"
	providedDocumentsHeader   = "# PROVIDED DOCUMENTS\n\n"
	conversationHistoryHeader = "# CONVERSATION HISTORY\n\n"
	separator                 = "---\n\n"
)

// buildPrompt constructs the full prompt with conversation history.
func (m Model) buildPrompt() string {
	var sb strings.Builder

	// For now, system instructions and documents are empty.
	// The headers are omitted if the content is empty.
	systemInstructions := ""
	providedDocuments := ""

	if systemInstructions != "" {
		sb.WriteString(systemInstructionsHeader)
		sb.WriteString(systemInstructions)
		sb.WriteString(separator)
	}

	if providedDocuments != "" {
		sb.WriteString(providedDocumentsHeader)
		sb.WriteString(providedDocuments)
		sb.WriteString(separator)
	}

	if len(m.messages) > 0 {
		sb.WriteString(conversationHistoryHeader)

		for _, msg := range m.messages {
			if msg.isUser {
				sb.WriteString("User:\n")
				sb.WriteString(msg.content)
				sb.WriteString("\n")
			} else {
				// Don't include empty AI messages (placeholders) in the prompt
				if msg.content != "" {
					sb.WriteString("AI Assistant:\n")
					sb.WriteString(msg.content)
					sb.WriteString("\n")
				}
			}
		}
		sb.WriteString("AI Assistant:\n")
	}

	return sb.String()
}
