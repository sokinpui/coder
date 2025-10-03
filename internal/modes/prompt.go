package modes

import (
	"strings"

	"coder/internal/core"
)

const (
	systemInstructionsHeader  = "# SYSTEM INSTRUCTIONS\n\n"
	relatedDocumentsHeader    = "# RELATED DOCUMENTS\n\n"
	projectSourceCodeHeader   = "# PROJECT SOURCE CODE\n\n"
	externalToolsHeader       = "# EXTERNAL TOOLS\n\n"
	directoryInformationHeader = "# DIRECTORY INFORMATION\n\n"
	conversationHistoryHeader = "# CONVERSATION HISTORY\n\n"
	separator                 = "\n\n---\n\n"
)

// PromptSection represents a distinct part of a larger prompt.
type PromptSection struct {
	Header  string
	Content string
}

// BuildPrompt constructs the full prompt from a series of sections.
func BuildPrompt(sections ...PromptSection) string {
	var sb strings.Builder
	hasContent := false
	for _, section := range sections {
		if section.Content == "" {
			continue
		}

		if hasContent {
			sb.WriteString(separator)
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
	return PromptSection{Content: content.String()}
}

// SystemInstructionsSection creates a prompt section for user-defined system instructions.
func SystemInstructionsSection(content string) PromptSection {
	return PromptSection{
		Header:  systemInstructionsHeader,
		Content: content,
	}
}

// RelatedDocumentsSection creates a prompt section for related documents.
func RelatedDocumentsSection(content string) PromptSection {
	return PromptSection{
		Header:  relatedDocumentsHeader,
		Content: content,
	}
}

// ProjectSourceCodeSection creates a prompt section for project source code.
func ProjectSourceCodeSection(content string) PromptSection {
	return PromptSection{
		Header:  projectSourceCodeHeader,
		Content: content,
	}
}

// ExternalToolsSection creates a prompt section for external tool documentation.
func ExternalToolsSection(content string) PromptSection {
	return PromptSection{
		Header:  externalToolsHeader,
		Content: content,
	}
}

// DirectoryInformationSection creates a prompt section for directory information.
func DirectoryInformationSection(content string) PromptSection {
	return PromptSection{
		Header:  directoryInformationHeader,
		Content: content,
	}
}

// ConversationHistorySection creates a prompt section for the conversation history.
func ConversationHistorySection(messages []core.Message) PromptSection {
	if len(messages) == 0 {
		return PromptSection{}
	}

	content := buildHistoryString(messages)

	return PromptSection{
		Header:  conversationHistoryHeader,
		Content: content,
	}
}

func buildHistoryString(messages []core.Message) string {
	var sb strings.Builder
	for _, msg := range messages {
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
	return sb.String()
}
