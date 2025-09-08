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

		for i := 0; i < len(m.messages); i++ {
			msg := m.messages[i]

			switch msg.mType {
			case userMessage:
				if strings.HasPrefix(msg.content, "/") {
					sb.WriteString("Command Execute:\n")
					sb.WriteString(msg.content)
					sb.WriteString("\n")

					// Look ahead for the command result
					if i+1 < len(m.messages) {
						nextMsg := m.messages[i+1]
						if nextMsg.mType == commandResultMessage {
							sb.WriteString("Command Execute Result:\n")
							sb.WriteString(nextMsg.content)
							sb.WriteString("\n")
							i++ // Skip the result message in the next iteration
						}
					}
				} else {
					sb.WriteString("User:\n")
					sb.WriteString(msg.content)
					sb.WriteString("\n")
				}
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
