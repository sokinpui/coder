package core

import (
	"strings"
)

const (
	systemInstructionsHeader  = "# SYSTEM INSTRUCTIONS\n\n"
	providedDocumentsHeader   = "# PROVIDED DOCUMENTS\n\n"
	conversationHistoryHeader = "# CONVERSATION HISTORY\n\n"
	separator                 = "\n\n---\n\n"
)

// BuildPrompt constructs the full prompt with conversation history.
func BuildPrompt(systemInstructions, providedDocuments string, messages []Message) string {
	var sb strings.Builder

	// Predefined system instructions (without header)
	if CoderSystemInstructions != "" {
		sb.WriteString(CoderSystemInstructions)
	}

	// User-defined system instructions (with header)
	if systemInstructions != "" {
		if CoderSystemInstructions != "" {
			sb.WriteString(separator)
		}
		sb.WriteString(systemInstructionsHeader)
		sb.WriteString(systemInstructions)
		sb.WriteString(separator)
	} else if CoderSystemInstructions != "" {
		// If there are only predefined instructions, we still need a separator
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
				sb.WriteString("User:\n")
				sb.WriteString(msg.Content)
				sb.WriteString("\n")
			case ActionMessage:
				sb.WriteString("Action Execute:\n")
				sb.WriteString(msg.Content)
				sb.WriteString("\n")

				// Look ahead for the action result
				if i+1 < len(messages) {
					nextMsg := messages[i+1]
					if nextMsg.Type == ActionResultMessage {
						sb.WriteString("Action Execute Result:\n")
						sb.WriteString(nextMsg.Content)
						sb.WriteString("\n")
						i++ // Skip the result message in the next iteration
					}
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
