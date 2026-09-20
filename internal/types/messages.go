package types

import "strings"

type MessageType int

const (
	UserMessage MessageType = iota
	AIMessage
	CommandMessage
	CommandResultMessage
	CommandErrorResultMessage
	InitMessage
	DirectoryMessage
	ImageMessage
	InstructionMessage
	SourceCodeMessage
	ShellCmdMessage
	ShellCmdResultMessage
	ContextCmdMessage
	ContextCmdResultMessage
	FileApplyCmdMessage
	FileApplyCmdResultMessage
	FileApplyCmdErrorMessage
	FileApplyUndoCmdMessage
	FileApplyUndoCmdResultMessage
	FileApplyUndoCmdErrorMessage
)

type Message struct {
	Type    MessageType
	Content string // For text content, or file path for images (for prompt)
	Data    []byte // For raw image data
}

type ChatRole string

const (
	RoleSystem    ChatRole = "system"
	RoleUser      ChatRole = "user"
	RoleAssistant ChatRole = "assistant"
)

type ChatMessage struct {
	Role    ChatRole
	Content string
	Data    []byte
}

type StreamChunk struct {
	Content          string
	ReasoningContent string
}

func (t MessageType) String() string {
	switch t {
	case UserMessage:
		return "User"
	case AIMessage:
		return "AI"
	case CommandMessage:
		return "Command"
	case CommandResultMessage:
		return "Command Result"
	case CommandErrorResultMessage:
		return "Command Error"
	case InitMessage:
		return "System"
	case DirectoryMessage:
		return "Directory"
	case ImageMessage:
		return "Image"
	case InstructionMessage:
		return "Instruction"
	case SourceCodeMessage:
		return "Source Code"
	case ShellCmdMessage:
		return "Shell Command"
	case ShellCmdResultMessage:
		return "Shell Command Result"
	case ContextCmdMessage:
		return "Context Command"
	case ContextCmdResultMessage:
		return "Context Command Result"
	case FileApplyCmdMessage:
		return "File Apply Command"
	case FileApplyCmdResultMessage:
		return "File Apply Result"
	case FileApplyCmdErrorMessage:
		return "File Apply Error"
	case FileApplyUndoCmdMessage:
		return "File Apply Undo Command"
	case FileApplyUndoCmdResultMessage:
		return "File Apply Undo Result"
	case FileApplyUndoCmdErrorMessage:
		return "File Apply Undo Error"
	default:
		return "Unknown"
	}
}

// IsEditable returns true if the message content can be edited by the user.
func (t MessageType) IsEditable() bool {
	switch t {
	case UserMessage:
		return true
	default:
		return false
	}
}

// IsSelectable returns true if the message can be focused/selected in Atomic Messages overlay.
func (t MessageType) IsSelectable() bool {
	switch t {
	case InitMessage, DirectoryMessage, InstructionMessage, SourceCodeMessage:
		return false
	default:
		return true
	}
}

// IsHistory returns true if the message should be saved to the session history file.
func (t MessageType) IsHistory() bool {
	switch t {
	case UserMessage, AIMessage, CommandMessage, CommandResultMessage, CommandErrorResultMessage, ImageMessage,
		InstructionMessage, SourceCodeMessage, ShellCmdMessage, ShellCmdResultMessage,
		ContextCmdMessage, ContextCmdResultMessage,
		FileApplyCmdMessage, FileApplyCmdResultMessage, FileApplyCmdErrorMessage,
		FileApplyUndoCmdMessage, FileApplyUndoCmdResultMessage, FileApplyUndoCmdErrorMessage:
		return true
	default:
		return false
	}
}

// IsRegeneratable returns true if the message can serve as a starting point for regeneration.
func (t MessageType) IsRegeneratable() bool {
	switch t {
	case UserMessage, ImageMessage, ShellCmdMessage, ShellCmdResultMessage:
		return true
	default:
		return false
	}
}

func (t MessageType) ChatRole() (ChatRole, bool) {
	switch t {
	case InstructionMessage, DirectoryMessage:
		return RoleSystem, true
	case AIMessage:
		return RoleAssistant, true
	case UserMessage, ImageMessage, SourceCodeMessage,
		ShellCmdMessage, ShellCmdResultMessage,
		ContextCmdMessage, ContextCmdResultMessage,
		FileApplyCmdMessage, FileApplyCmdResultMessage, FileApplyCmdErrorMessage,
		FileApplyUndoCmdMessage, FileApplyUndoCmdResultMessage, FileApplyUndoCmdErrorMessage:
		return RoleUser, true
	default:
		return "", false
	}
}

func AssemblePrompt(messages []Message, defaultInstruction string) (string, []ChatMessage) {
	var instructions []string
	var chatMessages []ChatMessage

	for _, msg := range messages {
		if !msg.CanSendToAI() {
			continue
		}
		role, ok := msg.Type.ChatRole()
		if !ok {
			continue
		}

		if role == RoleSystem {
			if trimmed := strings.TrimSpace(msg.Content); trimmed != "" {
				instructions = append(instructions, trimmed)
			}
			continue
		}

		if msg.Type == ImageMessage && len(msg.Data) == 0 {
			continue
		}

		chatMessages = append(chatMessages, ChatMessage{
			Role:    role,
			Content: msg.Content,
			Data:    msg.Data,
		})
	}

	systemInstruction := strings.Join(instructions, "\n\n")
	if systemInstruction == "" {
		systemInstruction = defaultInstruction
	}
	return systemInstruction, chatMessages
}

func (m Message) CanSendToAI() bool {
	switch m.Type {
	case InstructionMessage, DirectoryMessage, SourceCodeMessage,
		UserMessage, AIMessage, ImageMessage,
		ShellCmdMessage, ShellCmdResultMessage,
		ContextCmdMessage, ContextCmdResultMessage,
		FileApplyCmdMessage, FileApplyCmdResultMessage, FileApplyCmdErrorMessage,
		FileApplyUndoCmdMessage, FileApplyUndoCmdResultMessage, FileApplyUndoCmdErrorMessage:
		return true
	default:
		return false
	}
}

// IsAI returns true if the message was generated by the AI.
func (t MessageType) IsAI() bool {
	switch t {
	case AIMessage:
		return true
	default:
		return false
	}
}
