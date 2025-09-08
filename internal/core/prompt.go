package core

import (
	"strings"
)

const (
	systemInstructionsHeader  = "# SYSTEM INSTRUCTIONS\n\n"
	providedDocumentsHeader   = "# PROVIDED DOCUMENTS\n\n"
	conversationHistoryHeader = "# CONVERSATION HISTORY\n\n"
	separator                 = "---\n\n"
)

// BuildPrompt constructs the full prompt with conversation history.
func BuildPrompt(systemInstructions, providedDocuments string, messages []Message) string {
	var sb strings.Builder

	// The headers are omitted if the content is empty.
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

	if len(messages) > 0 {
		sb.WriteString(conversationHistoryHeader)

		for i := 0; i < len(messages); i++ {
			msg := messages[i]

			switch msg.Type {
			case UserMessage:
				if strings.HasPrefix(msg.Content, "/") {
					sb.WriteString("Command Execute:\n")
					sb.WriteString(msg.Content)
					sb.WriteString("\n")

					// Look ahead for the command result
					if i+1 < len(messages) {
						nextMsg := messages[i+1]
						if nextMsg.Type == CommandResultMessage {
							sb.WriteString("Command Execute Result:\n")
							sb.WriteString(nextMsg.Content)
							sb.WriteString("\n")
							i++ // Skip the result message in the next iteration
						}
					}
				} else {
					sb.WriteString("User:\n")
					sb.WriteString(msg.Content)
					sb.WriteString("\n")
				}
			case AIMessage:
				if msg.Content != "" {
					sb.WriteString("AI Assistant:\n")
					sb.WriteString(msg.Content)
					sb.WriteString("\n")
				}
			}
		}
		sb.WriteString("AI Assistant:\n")
	}

	return sb.String()
}
