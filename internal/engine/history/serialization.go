package history

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/sokinpui/coder/internal/types"
)

const ConversationHistoryHeader = "# CONVERSATION HISTORY\n\n"

var roleToMessageType = map[string]types.MessageType{
	"User:":                    types.UserMessage,
	"AI Assistant:":            types.AIMessage,
	"Tool Call:":               types.ToolCallMessage,
	"Tool Result:":             types.ToolResultMessage,
	"Image:":                   types.ImageMessage,
	"Command Execute:":         types.CommandMessage,
	"Command Execute Result:":  types.CommandResultMessage,
	"Command Execute Error:":   types.CommandErrorResultMessage,
	"Instruction:":             types.InstructionMessage,
	"Source Code:":             types.SourceCodeMessage,
	"Shell Command:":           types.ShellCmdMessage,
	"Shell Command Result:":    types.ShellCmdResultMessage,
	"Context Command:":         types.ContextCmdMessage,
	"Context Command Result:":  types.ContextCmdResultMessage,
	"File Apply Command:":      types.FileApplyCmdMessage,
	"File Apply Result:":       types.FileApplyCmdResultMessage,
	"File Apply Error:":        types.FileApplyCmdErrorMessage,
	"File Apply Undo Command:": types.FileApplyUndoCmdMessage,
	"File Apply Undo Result:":  types.FileApplyUndoCmdResultMessage,
	"File Apply Undo Error:":   types.FileApplyUndoCmdErrorMessage,
}

var imageMarkdownRegex = regexp.MustCompile(`^!\[image\]\((.*)\)$`)

func writeYamlList(b *bytes.Buffer, key string, items []string) {
	if len(items) == 0 {
		return
	}
	fmt.Fprintf(b, "%s:\n", key)
	for _, item := range items {
		fmt.Fprintf(b, "  - %s\n", item)
	}
}

func processMessageContent(msg *types.Message, rawContent string) {
	content := strings.TrimSpace(rawContent)
	if msg.Type == types.ImageMessage {
		matches := imageMarkdownRegex.FindStringSubmatch(content)
		if len(matches) > 1 {
			content = matches[1]
		}
	}
	if msg.Type == types.ToolResultMessage {
		if strings.HasPrefix(content, "[call_id: ") {
			if endIdx := strings.Index(content, "]\n"); endIdx != -1 {
				msg.ToolCallID = strings.TrimPrefix(content[:endIdx], "[call_id: ")
				content = strings.TrimSpace(content[endIdx+2:])
			} else if endIdx := strings.Index(content, "]"); endIdx != -1 && len(content) == endIdx+1 {
				msg.ToolCallID = strings.TrimPrefix(content[:endIdx], "[call_id: ")
				content = ""
			}
		}
	}
	if msg.Type == types.ToolCallMessage {
		var tc types.ToolCall
		if err := json.Unmarshal([]byte(content), &tc); err == nil && tc.Name != "" {
			msg.ToolCalls = []types.ToolCall{tc}
		}
	}
	msg.Content = content
}

func appendParsedMessage(messages *[]types.Message, current *types.Message, rawContent string, isAgent bool) {
	content := strings.TrimSpace(rawContent)
	if current.Type == types.AIMessage && strings.Contains(content, "```tool_call") {
		var textLines []string
		var toolCalls []types.ToolCall
		inBlock := false
		var blockBuf strings.Builder
		for line := range strings.SplitSeq(content, "\n") {
			trimmed := strings.TrimSpace(line)
			if trimmed == "```tool_call" {
				inBlock = true
				blockBuf.Reset()
				continue
			}
			if inBlock {
				if trimmed == "```" {
					inBlock = false
					var tc types.ToolCall
					if err := json.Unmarshal([]byte(strings.TrimSpace(blockBuf.String())), &tc); err == nil {
						toolCalls = append(toolCalls, tc)
					}
					continue
				}
				blockBuf.WriteString(line)
				blockBuf.WriteByte('\n')
				continue
			}
			textLines = append(textLines, line)
		}
		cleanText := strings.TrimSpace(strings.Join(textLines, "\n"))
		if cleanText != "" {
			*messages = append(*messages, types.Message{
				Type:    types.AIMessage,
				Content: cleanText,
			})
		}
		if len(toolCalls) > 0 {
			*messages = append(*messages, types.Message{
				Type:      types.ToolCallMessage,
				ToolCalls: toolCalls,
			})
		}
		return
	}

	processMessageContent(current, rawContent)
	if current.Type != types.InstructionMessage && current.Type != types.SourceCodeMessage {
		if !current.IsDocumentImage() || isAgent {
			*messages = append(*messages, *current)
		}
	}
}

func parseStringSlice(value string) []string {
	var s []string
	if err := json.Unmarshal([]byte(value), &s); err != nil {
		return nil
	}
	return s
}

func parseFrontmatter(scanner *bufio.Scanner) (*Metadata, bool) {
	metadata := &Metadata{}
	var currentKey string
	for scanner.Scan() {
		line := scanner.Text()
		if line == "---" {
			return metadata, true
		}

		if after, ok := strings.CutPrefix(line, "  - "); ok {
			val := strings.TrimSpace(after)
			switch currentKey {
			case "contextFiles", "files":
				metadata.ContextFiles = append(metadata.ContextFiles, val)
			case "contextDocuments", "documents":
				metadata.ContextDocuments = append(metadata.ContextDocuments, val)
			case "exclusions":
				metadata.Exclusions = append(metadata.Exclusions, val)
			}
			continue
		}

		kv := strings.SplitN(line, ":", 2)
		if len(kv) != 2 {
			continue
		}

		key, value := strings.TrimSpace(kv[0]), strings.TrimSpace(kv[1])
		currentKey = key
		switch key {
		case "title":
			metadata.Title = value
		case "mode":
			metadata.Mode = value
		case "workingDir":
			metadata.WorkingDir = value
		case "contextFiles", "files":
			if value != "" {
				metadata.ContextFiles = append(metadata.ContextFiles, parseStringSlice(value)...)
			}
		case "contextDocuments", "documents":
			if value != "" {
				metadata.ContextDocuments = append(metadata.ContextDocuments, parseStringSlice(value)...)
			}
		case "exclusions":
			if value != "" {
				metadata.Exclusions = append(metadata.Exclusions, parseStringSlice(value)...)
			}
		case "createdAt", "modifiedAt":
			t, err := time.Parse(time.RFC3339Nano, value)
			if err != nil {
				continue
			}
			if key == "createdAt" {
				metadata.CreatedAt = t
			} else {
				metadata.ModifiedAt = t
			}
		}
	}
	return metadata, false
}

func ParseConversation(content []byte) (*Metadata, []types.Message, error) {
	parts := bytes.SplitN(content, []byte("---\n"), 3)
	if len(parts) < 3 {
		return nil, nil, fmt.Errorf("invalid file format: missing YAML frontmatter")
	}

	metaScanner := bufio.NewScanner(bytes.NewReader(parts[1]))
	metadata, _ := parseFrontmatter(metaScanner)

	bodyBytes := parts[2]
	historyHeader := []byte("# CONVERSATION HISTORY")
	_, after, ok := bytes.Cut(bodyBytes, historyHeader)
	if !ok {
		return metadata, []types.Message{}, nil
	}

	conversationContentBytes := bytes.TrimSpace(after)

	var messages []types.Message
	var currentMessage *types.Message
	var contentBuilder strings.Builder
	isAgent := metadata != nil && normalizeMode(metadata.Mode) == "agent"

	convScanner := bufio.NewScanner(bytes.NewReader(conversationContentBytes))
	for convScanner.Scan() {
		line := convScanner.Text()
		foundRole := false
		for role, msgType := range roleToMessageType {
			if strings.HasPrefix(line, role) {
				if currentMessage != nil {
					appendParsedMessage(&messages, currentMessage, contentBuilder.String(), isAgent)
				}
				contentBuilder.Reset()
				currentMessage = &types.Message{Type: msgType}
				contentBuilder.WriteString(strings.TrimSpace(strings.TrimPrefix(line, role)))
				foundRole = true
				break
			}
		}
		if !foundRole && currentMessage != nil {
			contentBuilder.WriteString("\n")
			contentBuilder.WriteString(line)
		}
	}

	if currentMessage != nil {
		appendParsedMessage(&messages, currentMessage, contentBuilder.String(), isAgent)
	}

	return metadata, messages, nil
}

func ParseFileMetadata(filePath string) (*Metadata, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)

	if !scanner.Scan() || scanner.Text() != "---" {
		return nil, fmt.Errorf("invalid file format: missing YAML frontmatter start")
	}

	metadata, closed := parseFrontmatter(scanner)

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	if !closed {
		return nil, fmt.Errorf("invalid file format: YAML frontmatter not closed")
	}

	return metadata, nil
}

func BuildHistorySnippet(messages []types.Message) string {
	var sb strings.Builder

	for i := range messages {
		msg := messages[i]

		if !msg.Type.IsHistory() {
			continue
		}

		switch msg.Type {
		case types.InstructionMessage:
			sb.WriteString("Instruction:\n")
			sb.WriteString(msg.Content)
		case types.SourceCodeMessage:
			sb.WriteString("Source Code:\n")
			sb.WriteString(msg.Content)
		case types.UserMessage:
			sb.WriteString("User:\n")
			sb.WriteString(msg.Content)
		case types.ImageMessage:
			sb.WriteString("Image:\n")
			fmt.Fprintf(&sb, "![image](%s)", msg.Content)
		case types.AIMessage:
			if msg.Content == "" && len(msg.ToolCalls) == 0 {
				continue
			}
			sb.WriteString("AI Assistant:\n")
			if msg.Content != "" {
				sb.WriteString(msg.Content)
			}
			for _, tc := range msg.ToolCalls {
				if msg.Content != "" {
					sb.WriteString("\n")
				}
				data, _ := json.Marshal(tc)
				sb.WriteString("```tool_call\n")
				sb.Write(data)
				sb.WriteString("\n```")
			}
		case types.ToolCallMessage:
			sb.WriteString("Tool Call:\n")
			if len(msg.ToolCalls) > 0 {
				for i, tc := range msg.ToolCalls {
					if i > 0 {
						sb.WriteString("\n")
					}
					data, _ := json.Marshal(tc)
					sb.Write(data)
				}
			} else {
				sb.WriteString(msg.Content)
			}
		case types.ToolResultMessage:
			sb.WriteString("Tool Result:\n")
			if msg.ToolCallID != "" {
				fmt.Fprintf(&sb, "[call_id: %s]\n", msg.ToolCallID)
			}
			sb.WriteString(msg.Content)
		case types.CommandMessage:
			sb.WriteString("Command Execute:\n")
			sb.WriteString(msg.Content)
		case types.CommandResultMessage:
			sb.WriteString("Command Execute Result:\n")
			sb.WriteString(msg.Content)
		case types.CommandErrorResultMessage:
			sb.WriteString("Command Execute Error:\n")
			sb.WriteString(msg.Content)
		case types.ShellCmdMessage:
			sb.WriteString("Shell Command:\n")
			sb.WriteString(msg.Content)
		case types.ShellCmdResultMessage:
			sb.WriteString("Shell Command Result:\n")
			sb.WriteString(msg.Content)
		case types.ContextCmdMessage:
			sb.WriteString("Context Command:\n")
			sb.WriteString(msg.Content)
		case types.ContextCmdResultMessage:
			sb.WriteString("Context Command Result:\n")
			sb.WriteString(msg.Content)
		case types.FileApplyCmdMessage:
			sb.WriteString("File Apply Command:\n")
			sb.WriteString(msg.Content)
		case types.FileApplyCmdResultMessage:
			sb.WriteString("File Apply Result:\n")
			sb.WriteString(msg.Content)
		case types.FileApplyCmdErrorMessage:
			sb.WriteString("File Apply Error:\n")
			sb.WriteString(msg.Content)
		case types.FileApplyUndoCmdMessage:
			sb.WriteString("File Apply Undo Command:\n")
			sb.WriteString(msg.Content)
		case types.FileApplyUndoCmdResultMessage:
			sb.WriteString("File Apply Undo Result:\n")
			sb.WriteString(msg.Content)
		case types.FileApplyUndoCmdErrorMessage:
			sb.WriteString("File Apply Undo Error:\n")
			sb.WriteString(msg.Content)
		}
		sb.WriteString("\n\n")
	}

	return strings.TrimSpace(sb.String())
}
