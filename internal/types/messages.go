package types

import "encoding/json"
import "fmt"
import "slices"
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
	ToolCallMessage
	ToolResultMessage
	CompactedSummaryMessage
	PDFMessage
	InterruptedMessage
)

type PDFMetadata struct {
	Filename  string   `json:"filename"`
	Path      string   `json:"path"`
	Pages     string   `json:"pages,omitempty"`
	CacheDir  string   `json:"cache_dir"`
	PageCount int      `json:"page_count"`
	Files     []string `json:"files"`
	Mode      string   `json:"mode,omitempty"`
	Size      int64    `json:"size,omitempty"`
}

func ParsePDFMetadata(content string) (PDFMetadata, bool) {
	var meta PDFMetadata
	if err := json.Unmarshal([]byte(content), &meta); err != nil {
		return PDFMetadata{}, false
	}
	return meta, true
}

type PDFPage struct {
	Index int
	Data  []byte
}

type PDFResolved struct {
	Filename string
	RawData  []byte
	Pages    []PDFPage
}

type PDFResolver func(msg Message, mode string) PDFResolved

var pdfResolver PDFResolver

func RegisterPDFResolver(resolver PDFResolver) {
	pdfResolver = resolver
}

type Message struct {
	GroupID    string
	Type       MessageType
	Content    string
	Data       []byte
	ToolCalls  []ToolCall
	ToolCallID string
}

type ChatRole string

const (
	RoleSystem    ChatRole = "system"
	RoleUser      ChatRole = "user"
	RoleAssistant ChatRole = "assistant"
	RoleTool      ChatRole = "tool"
)

type ToolCall struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type ToolCallDelta struct {
	Index          int    `json:"index"`
	ID             string `json:"id,omitempty"`
	Name           string `json:"name,omitempty"`
	ArgumentsDelta string `json:"arguments_delta"`
}

type ToolDeclaration struct {
	Type        string         `json:"type"`
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	Parameters  map[string]any `json:"parameters,omitempty"`
}

type ChatMessage struct {
	Role       ChatRole
	Content    string
	Data       []byte
	MimeType   string
	Filename   string
	ToolCalls  []ToolCall
	ToolCallID string
}

type StreamChunk struct {
	Content          string
	ReasoningContent string
	ToolCall         *ToolCall
	ToolCallDelta    *ToolCallDelta
	Error            error
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
	case ToolCallMessage:
		return "Tool Call"
	case ToolResultMessage:
		return "Tool Result"
	case CompactedSummaryMessage:
		return "Compacted Summary"
	case PDFMessage:
		return "PDF Document"
	case InterruptedMessage:
		return "Interrupted"
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

func ExpandGroupIndices(messages []Message, indices []int) []int {
	if len(indices) == 0 || len(messages) == 0 {
		return nil
	}

	selected := make(map[int]struct{}, len(indices))
	targetGroups := make(map[string]struct{})

	for _, idx := range indices {
		if idx < 0 || idx >= len(messages) {
			continue
		}
		selected[idx] = struct{}{}
		if grp := messages[idx].GroupID; grp != "" {
			targetGroups[grp] = struct{}{}
		}
	}

	if len(targetGroups) > 0 {
		for i, msg := range messages {
			if msg.GroupID != "" {
				if _, ok := targetGroups[msg.GroupID]; ok {
					selected[i] = struct{}{}
				}
			}
		}
	}

	res := make([]int, 0, len(selected))
	for idx := range selected {
		res = append(res, idx)
	}
	slices.Sort(res)
	return res
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
		FileApplyUndoCmdMessage, FileApplyUndoCmdResultMessage, FileApplyUndoCmdErrorMessage,
		ToolCallMessage, ToolResultMessage,
		CompactedSummaryMessage, PDFMessage, InterruptedMessage:
		return true
	default:
		return false
	}
}

// IsRegeneratable returns true if the message can serve as a starting point for regeneration.
func (t MessageType) IsRegeneratable() bool {
	switch t {
	case UserMessage, ImageMessage, PDFMessage, ShellCmdMessage, ShellCmdResultMessage:
		return true
	default:
		return false
	}
}

func (t MessageType) ChatRole() (ChatRole, bool) {
	switch t {
	case InstructionMessage, DirectoryMessage:
		return RoleSystem, true
	case AIMessage, ToolCallMessage:
		return RoleAssistant, true
	case ToolResultMessage:
		return RoleTool, true
	case UserMessage, ImageMessage, PDFMessage, SourceCodeMessage,
		ShellCmdMessage, ShellCmdResultMessage,
		ContextCmdMessage, ContextCmdResultMessage,
		FileApplyCmdMessage, FileApplyCmdResultMessage, FileApplyCmdErrorMessage,
		FileApplyUndoCmdMessage, FileApplyUndoCmdResultMessage, FileApplyUndoCmdErrorMessage,
		CompactedSummaryMessage, InterruptedMessage:
		return RoleUser, true
	default:
		return "", false
	}
}

func AssemblePrompt(messages []Message, defaultInstruction string, pdfMode ...string) (string, []ChatMessage) {
	var instructions []string
	var chatMessages []ChatMessage
	var deferredMedia []ChatMessage
	pendingToolCalls := make(map[string]struct{})

	flushDeferredMedia := func() {
		if len(deferredMedia) > 0 {
			chatMessages = append(chatMessages, deferredMedia...)
			deferredMedia = nil
		}
		clear(pendingToolCalls)
	}

	appendMedia := func(media ...ChatMessage) {
		if len(pendingToolCalls) > 0 {
			deferredMedia = append(deferredMedia, media...)
			return
		}
		chatMessages = append(chatMessages, media...)
	}

	hasInstructionMessage := false
	for _, msg := range messages {
		role, ok := msg.Type.ChatRole()
		if !ok {
			continue
		}

		if msg.Type == CompactedSummaryMessage {
			flushDeferredMedia()
			chatMessages = append(chatMessages,
				ChatMessage{
					Role:    RoleUser,
					Content: "[Conversation Checkpoint]\n" + msg.Content,
				},
				ChatMessage{
					Role:    RoleAssistant,
					Content: "Understood. I have loaded the conversation checkpoint and will continue based on this context.",
				},
			)
			continue
		}

		if msg.Type == PDFMessage {
			docName := msg.Content
			if meta, ok := ParsePDFMetadata(msg.Content); ok && meta.Filename != "" {
				docName = meta.Filename
			}
			mode := ""
			if len(pdfMode) > 0 {
				mode = pdfMode[0]
			}
			if pdfResolver != nil {
				resolved := pdfResolver(msg, mode)
				if resolved.Filename != "" {
					docName = resolved.Filename
				}
				if len(resolved.RawData) > 0 {
					appendMedia(ChatMessage{
						Role:     RoleUser,
						Data:     resolved.RawData,
						MimeType: "application/pdf",
						Filename: docName,
					})
					continue
				}
				if len(resolved.Pages) > 0 {
					pages := make([]ChatMessage, 0, len(resolved.Pages))
					for _, page := range resolved.Pages {
						pages = append(pages, ChatMessage{
							Role:    RoleUser,
							Content: fmt.Sprintf("%s page %d:", docName, page.Index),
							Data:    page.Data,
						})
					}
					appendMedia(pages...)
					continue
				}
			}
			appendMedia(ChatMessage{
				Role:    RoleUser,
				Content: "[Attached Document: " + docName + "]",
			})
			continue
		}

		if role == RoleSystem {
			if msg.Type == InstructionMessage {
				hasInstructionMessage = true
			}
			if trimmed := strings.TrimSpace(msg.Content); trimmed != "" {
				instructions = append(instructions, trimmed)
			}
			continue
		}

		if msg.Type == ImageMessage {
			if len(msg.Data) == 0 {
				continue
			}
			appendMedia(ChatMessage{
				Role: RoleUser,
				Data: msg.Data,
			})
			continue
		}

		if role == RoleTool {
			delete(pendingToolCalls, msg.ToolCallID)
			chatMessages = append(chatMessages, ChatMessage{
				Role:       role,
				Content:    msg.Content,
				Data:       msg.Data,
				ToolCalls:  msg.ToolCalls,
				ToolCallID: msg.ToolCallID,
			})
			if len(pendingToolCalls) == 0 {
				flushDeferredMedia()
			}
			continue
		}

		if role == RoleAssistant && len(chatMessages) > 0 && chatMessages[len(chatMessages)-1].Role == RoleAssistant {
			last := &chatMessages[len(chatMessages)-1]
			if msg.Content != "" {
				if last.Content != "" {
					last.Content += "\n\n" + msg.Content
				} else {
					last.Content = msg.Content
				}
			}
			if len(msg.ToolCalls) > 0 {
				last.ToolCalls = append(last.ToolCalls, msg.ToolCalls...)
				for _, tc := range msg.ToolCalls {
					if tc.ID != "" {
						pendingToolCalls[tc.ID] = struct{}{}
					}
				}
			}
			continue
		}

		flushDeferredMedia()

		if role == RoleAssistant {
			for _, tc := range msg.ToolCalls {
				if tc.ID != "" {
					pendingToolCalls[tc.ID] = struct{}{}
				}
			}
		}

		chatMessages = append(chatMessages, ChatMessage{
			Role:       role,
			Content:    msg.Content,
			Data:       msg.Data,
			ToolCalls:  msg.ToolCalls,
			ToolCallID: msg.ToolCallID,
		})
	}

	flushDeferredMedia()

	if !hasInstructionMessage && strings.TrimSpace(defaultInstruction) != "" {
		instructions = append([]string{strings.TrimSpace(defaultInstruction)}, instructions...)
	}

	systemInstruction := strings.Join(instructions, "\n\n")
	return systemInstruction, chatMessages
}
