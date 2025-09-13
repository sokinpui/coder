package core

import (
	"strings"
)

const (
	systemInstructionsHeader  = "# SYSTEM INSTRUCTIONS\n\n"
	relatedDocumentsHeader    = "# RELATED DOCUMENTS\n\n"
	projectSourceCodeHeader   = "# PROJECT SOURCE CODE\n\n"
	conversationHistoryHeader = "# CONVERSATION HISTORY\n\n"
	separator                 = "\n\n---\n\n"
)

// BuildPrompt constructs the full prompt with conversation history.
func BuildPrompt(systemInstructions, relatedDocuments, projectSourceCode string, messages []Message) string {
	var sb strings.Builder

	hasPredefinedContent := false
	if CoderRole != "" {
		sb.WriteString(CoderRole)
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
			case ActionMessage:
				// Look ahead for the action result
				if i+1 < len(messages) {
					nextMsg := messages[i+1]
					if nextMsg.Type == ActionResultMessage {
						// only save action that was executed successfully
						sb.WriteString("Action Execute:\n")
						sb.WriteString(msg.Content)
						sb.WriteString("\n")

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
