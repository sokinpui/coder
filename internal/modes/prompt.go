package modes

import (
	"strings"

	"github.com/sokinpui/coder/internal/types"
)

const (
	projectSourceCodeHeader   = "# PROJECT SOURCE CODE\n\n"
	conversationHistoryHeader = "# CONVERSATION HISTORY\n\n"
	Separator                 = "\n\n---\n\n"
)

// PromptSection represents a distinct part of a larger prompt.
type PromptSection struct {
	Header    string
	Content   string
	Separator string
}

// PromptSectionArray holds a slice of PromptSection structs.
type PromptSectionArray struct {
	Sections []PromptSection
}

// BuildPrompt constructs the full prompt from a series of sections.
func BuildPrompt(promptSections PromptSectionArray) string {
	var sb strings.Builder
	hasContent := false
	for _, section := range promptSections.Sections {
		if section.Content == "" {
			continue
		}

		if hasContent {
			sb.WriteString(section.Separator)
		}

		if section.Header != "" {
			sb.WriteString(section.Header)
		}
		sb.WriteString(section.Content)
		hasContent = true
	}
	return sb.String()
}

// RoleSection creates a prompt section for the role and initial instructions.
func RoleSection(role, instructions string) PromptSection {
	var content strings.Builder
	if role != "" {
		content.WriteString(role)
	}
	if instructions != "" {
		content.WriteString(instructions)
	}
	return PromptSection{Content: content.String(), Separator: Separator}
}

// ProjectSourceCodeSection creates a prompt section for project source code.
func ProjectSourceCodeSection(content string) PromptSection {
	return PromptSection{
		Header:    projectSourceCodeHeader,
		Content:   content,
		Separator: Separator,
	}
}

// ConversationHistorySection creates a prompt section for the conversation history.
func ConversationHistorySection(messages []types.Message) PromptSection {
	if len(messages) == 0 {
		return PromptSection{Separator: Separator}
	}

	content := BuildHistoryString(messages)

	return PromptSection{
		Header:  conversationHistoryHeader,
		Content: content,
	}
}

// BuildHistoryString constructs the conversation history part of the prompt.
func BuildHistoryString(messages []types.Message) string {
	var sb strings.Builder
	for _, msg := range messages {
		switch msg.Type {
		case types.UserMessage:
			sb.WriteString("User:\n")
			sb.WriteString(msg.Content)
			sb.WriteString("\n")
		case types.ImageMessage:
			sb.WriteString(msg.Content)
			sb.WriteString("\n")
		case types.AIMessage:
			if msg.Content == "" {
				continue
			}
			sb.WriteString("AI Assistant:\n")
			sb.WriteString(msg.Content)
			sb.WriteString("\n")
		}
	}
	sb.WriteString("AI Assistant:\n")
	return sb.String()
}
