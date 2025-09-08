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

	// The headers are omitted if the content is empty.
	systemInstructions := m.systemInstructions
	providedDocuments := m.providedDocuments

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
			switch msg.mType {
			case userMessage:
				sb.WriteString("User:\n")
				sb.WriteString(msg.content)
				sb.WriteString("\n")
			case aiMessage:
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
