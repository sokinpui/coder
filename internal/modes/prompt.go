package modes

import (
	"strings"

	"github.com/sokinpui/coder/internal/types"
)

const (
	ProjectSourceCodeHeader   = "# PROJECT SOURCE CODE\n\n"
	ConversationHistoryHeader = "# CONVERSATION HISTORY\n\n"
	Separator                 = "\n\n---\n\n"
)

type PromptSection struct {
	Header    string
	Content   string
	Separator string
}

type PromptSectionArray struct {
	Sections []PromptSection
}

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

func ProjectSourceCodeSection(content string) PromptSection {
	return PromptSection{
		Header:    ProjectSourceCodeHeader,
		Content:   content,
		Separator: Separator,
	}
}

func ConversationHistorySection(messages []types.Message) PromptSection {
	if len(messages) == 0 {
		return PromptSection{}
	}

	content := BuildChatString(messages)

	return PromptSection{
		Header:  ConversationHistoryHeader,
		Content: content,
	}
}

func BuildChatString(messages []types.Message) string {
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
