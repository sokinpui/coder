package coagent

import (
	"bufio"
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/sokinpui/coder/internal/config"
	"github.com/sokinpui/coder/internal/types"
)

// -----------------------------------------------------------------------------
// Tool Definitions and Registry
// -----------------------------------------------------------------------------

type ToolDeclaration struct {
	Type        string         `json:"type"`
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	Parameters  map[string]any `json:"parameters,omitempty"`
}

type Tool interface {
	Declaration() ToolDeclaration
	Execute(ctx context.Context, arguments string) (string, error)
}

type Registry struct {
	mu    sync.RWMutex
	tools map[string]Tool
}

func NewRegistry() *Registry {
	return &Registry{
		tools: make(map[string]Tool),
	}
}

func (r *Registry) Register(tool Tool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.tools[tool.Declaration().Name] = tool
}

func (r *Registry) Declarations() []ToolDeclaration {
	r.mu.RLock()
	defer r.mu.RUnlock()
	decls := make([]ToolDeclaration, 0, len(r.tools))
	for _, t := range r.tools {
		decls = append(decls, t.Declaration())
	}
	return decls
}

func (r *Registry) Execute(ctx context.Context, name, arguments string) (string, error) {
	r.mu.RLock()
	tool, exists := r.tools[name]
	r.mu.RUnlock()

	if !exists {
		return "", fmt.Errorf("tool '%s' not found", name)
	}
	return tool.Execute(ctx, arguments)
}

var DefaultRegistry = NewRegistry()

// -----------------------------------------------------------------------------
// Bash Tool Implementation
// -----------------------------------------------------------------------------

const (
	defaultBashTimeout = 60 * time.Second
	maxOutputBytes     = 64 * 1024
)

type BashTool struct{}

func (t *BashTool) Declaration() ToolDeclaration {
	return ToolDeclaration{
		Type:        "function",
		Name:        "bash",
		Description: "Execute a bash shell command on the host system. Returns combined stdout and stderr.",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"command": map[string]any{
					"type":        "string",
					"description": "The bash shell command to execute",
				},
			},
			"required": []string{"command"},
		},
	}
}

func (t *BashTool) Execute(ctx context.Context, arguments string) (string, error) {
	var params struct {
		Command string `json:"command"`
	}

	if err := json.Unmarshal([]byte(arguments), &params); err != nil || params.Command == "" {
		trimmedArgs := strings.TrimSpace(arguments)
		if !strings.HasPrefix(trimmedArgs, "{") && trimmedArgs != "" {
			params.Command = trimmedArgs
		} else if err != nil {
			return "", fmt.Errorf("invalid arguments: %w", err)
		}
	}

	if strings.TrimSpace(params.Command) == "" {
		return "", fmt.Errorf("command is required")
	}

	runCtx := ctx
	var cancel context.CancelFunc
	if _, hasDeadline := ctx.Deadline(); !hasDeadline {
		runCtx, cancel = context.WithTimeout(ctx, defaultBashTimeout)
		defer cancel()
	}

	shell := "bash"
	if _, err := exec.LookPath("bash"); err != nil {
		shell = "sh"
	}

	cmd := exec.CommandContext(runCtx, shell, "-c", params.Command)
	outputBytes, err := cmd.CombinedOutput()

	output := string(outputBytes)
	if len(output) > maxOutputBytes {
		output = output[:maxOutputBytes] + "\n...[output truncated]"
	}

	trimmedOutput := strings.TrimSpace(output)
	if err != nil {
		if trimmedOutput != "" {
			return fmt.Sprintf("%s\nCommand exited with error: %v", trimmedOutput, err), nil
		}
		return fmt.Sprintf("Command exited with error: %v", err), nil
	}

	if trimmedOutput == "" {
		return "(no output)", nil
	}
	return trimmedOutput, nil
}

func init() {
	DefaultRegistry.Register(&BashTool{})
}

// -----------------------------------------------------------------------------
// Agent Loop and Execution Types (Preserved for future 'co' implementation)
// -----------------------------------------------------------------------------

type ToolCallInfo struct {
	CallID    string `json:"call_id"`
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type ToolResultInfo struct {
	CallID string `json:"call_id"`
	Name   string `json:"name"`
	Output string `json:"output"`
}

type AgentStreamChunk struct {
	Content          string
	ReasoningContent string
	ToolCall         *ToolCallInfo
	ToolResult       *ToolResultInfo
}

type responseStreamEvent struct {
	Type      string `json:"type"`
	Delta     string `json:"delta,omitempty"`
	Arguments string `json:"arguments,omitempty"`
	Item      *struct {
		Type      string `json:"type"`
		ID        string `json:"id"`
		CallID    string `json:"call_id"`
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	} `json:"item,omitempty"`
	Response *struct {
		Output []struct {
			Type      string `json:"type"`
			ID        string `json:"id"`
			CallID    string `json:"call_id"`
			Name      string `json:"name"`
			Arguments string `json:"arguments"`
		} `json:"output"`
	} `json:"response,omitempty"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

type openAIImageURL struct {
	URL string `json:"url"`
}

type openAIContentPart struct {
	Type     string          `json:"type"`
	Text     string          `json:"text,omitempty"`
	ImageURL *openAIImageURL `json:"image_url,omitempty"`
}

type AgentRuntime struct {
	Config            config.Generation
	BaseURL           string
	APIKey            string
	MaxToolIterations int
	Registry          *Registry
}

func NewAgentRuntime(cfg *config.Config, registry *Registry) *AgentRuntime {
	if registry == nil {
		registry = DefaultRegistry
	}
	return &AgentRuntime{
		Config:            cfg.Generation,
		BaseURL:           cfg.Server.URL,
		APIKey:            cfg.Server.APIKey,
		MaxToolIterations: 20,
		Registry:          registry,
	}
}

func (ar *AgentRuntime) RunLoop(ctx context.Context, messages []types.Message, streamChan chan<- AgentStreamChunk) {
	defer close(streamChan)

	currentMessages := append([]types.Message(nil), messages...)
	maxIterations := ar.MaxToolIterations
	if maxIterations <= 0 {
		maxIterations = 20
	}

	for range maxIterations {
		if ctx.Err() != nil {
			return
		}

		toolCalls, turnText, hasError := ar.executeTurn(ctx, currentMessages, streamChan, true)
		if turnText != "" {
			currentMessages = append(currentMessages, types.Message{Type: types.AIMessage, Content: turnText})
		}
		if hasError || ctx.Err() != nil || len(toolCalls) == 0 {
			return
		}

		for _, tc := range toolCalls {
			if ctx.Err() != nil {
				return
			}

			streamChan <- AgentStreamChunk{ToolCall: &tc}
			output, err := ar.Registry.Execute(ctx, tc.Name, tc.Arguments)
			if err != nil {
				output = fmt.Sprintf("Error: %v", err)
			}

			resInfo := ToolResultInfo{
				CallID: tc.CallID,
				Name:   tc.Name,
				Output: output,
			}
			streamChan <- AgentStreamChunk{ToolResult: &resInfo}

			// In future co implementation, tool call message will append to internal agent message history
		}
	}

	if ctx.Err() != nil {
		return
	}

	_, _, _ = ar.executeTurn(ctx, currentMessages, streamChan, false)
}

func (ar *AgentRuntime) executeTurn(ctx context.Context, messages []types.Message, streamChan chan<- AgentStreamChunk, enableTools bool) ([]ToolCallInfo, string, bool) {
	var instructionsBuilder strings.Builder
	var inputItems []any

	for _, msg := range messages {
		if !msg.CanSendToAI() {
			continue
		}

		switch msg.Type {
		case types.InstructionMessage, types.DirectoryMessage:
			if instructionsBuilder.Len() > 0 {
				instructionsBuilder.WriteString("\n\n")
			}
			instructionsBuilder.WriteString(msg.Content)

		case types.UserMessage, types.ShellCmdMessage, types.ShellCmdResultMessage,
			types.ContextCmdMessage, types.ContextCmdResultMessage,
			types.FileApplyCmdMessage, types.FileApplyCmdResultMessage, types.FileApplyCmdErrorMessage,
			types.FileApplyUndoCmdMessage, types.FileApplyUndoCmdResultMessage, types.FileApplyUndoCmdErrorMessage:
			inputItems = append(inputItems, map[string]any{
				"type":    "message",
				"role":    "user",
				"content": msg.Content,
			})

		case types.AIMessage:
			if msg.Content != "" {
				inputItems = append(inputItems, map[string]any{
					"type":    "message",
					"role":    "assistant",
					"content": msg.Content,
				})
			}

		case types.ImageMessage:
			if msg.Data == nil {
				continue
			}
			b64 := base64.StdEncoding.EncodeToString(msg.Data)
			mimeType := "image/png"
			if len(msg.Data) > 4 && bytes.Equal(msg.Data[:4], []byte{0xFF, 0xD8, 0xFF, 0xE0}) {
				mimeType = "image/jpeg"
			}
			inputItems = append(inputItems, map[string]any{
				"type": "message",
				"role": "user",
				"content": []openAIContentPart{
					{
						Type: "input_image",
						ImageURL: &openAIImageURL{
							URL: fmt.Sprintf("data:%s;base64,%s", mimeType, b64),
						},
					},
				},
			})
		}
	}

	if !enableTools {
		inputItems = append(inputItems, map[string]any{
			"type":    "message",
			"role":    "user",
			"content": "Tool execution iteration limit reached. Do not call any tools. Provide your final response to the user based on the tool results so far.",
		})
	}

	body := map[string]any{
		"model":        ar.Config.ModelCode,
		"stream":       true,
		"store":        false,
		"instructions": instructionsBuilder.String(),
		"input":        inputItems,
	}

	toolDecls := ar.Registry.Declarations()
	if len(toolDecls) > 0 {
		body["tools"] = toolDecls
		if enableTools {
			body["tool_choice"] = "auto"
		} else {
			body["tool_choice"] = "none"
		}
	}

	if ar.Config.ReasoningEffort != "" {
		body["reasoning"] = map[string]any{
			"effort": ar.Config.ReasoningEffort,
		}
	}

	jsonBody, _ := json.Marshal(body)
	url := strings.TrimSuffix(ar.BaseURL, "/") + "/responses"
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		streamChan <- AgentStreamChunk{Content: fmt.Sprintf("Error: Failed to create request: %v", err)}
		return nil, "", true
	}
	httpReq.Header.Set("Content-Type", "application/json")

	if ar.APIKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+ar.APIKey)
	}

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		if ctx.Err() == context.Canceled {
			return nil, "", false
		}
		streamChan <- AgentStreamChunk{Content: fmt.Sprintf("Error: Failed to connect to server: %v", err)}
		return nil, "", true
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		errBytes, _ := io.ReadAll(resp.Body)
		streamChan <- AgentStreamChunk{Content: fmt.Sprintf("Error: Server returned status %d: %s", resp.StatusCode, string(errBytes))}
		return nil, "", true
	}

	var toolCalls []ToolCallInfo
	var currentToolCall *ToolCallInfo
	var textBuilder strings.Builder

	reader := bufio.NewReader(resp.Body)
	for {
		if ctx.Err() != nil {
			return nil, textBuilder.String(), false
		}

		line, readErr := reader.ReadString('\n')
		if len(line) > 0 {
			line = strings.TrimRight(line, "\r\n")
			if after, ok := strings.CutPrefix(line, "data:"); ok {
				raw := after
				if strings.HasPrefix(raw, " ") {
					raw = raw[1:]
				}
				trimmed := strings.TrimSpace(raw)
				if trimmed == "[DONE]" {
					break
				}

				var ev responseStreamEvent
				if err := json.Unmarshal([]byte(trimmed), &ev); err == nil {
					switch ev.Type {
					case "response.output_text.delta":
						if ev.Delta != "" {
							textBuilder.WriteString(ev.Delta)
							select {
							case <-ctx.Done():
								return nil, textBuilder.String(), false
							case streamChan <- AgentStreamChunk{Content: ev.Delta}:
							}
						}
					case "response.reasoning_summary_text.delta", "response.reasoning_text.delta":
						if ev.Delta != "" {
							select {
							case <-ctx.Done():
								return nil, textBuilder.String(), false
							case streamChan <- AgentStreamChunk{ReasoningContent: ev.Delta}:
							}
						}
					case "response.output_item.added":
						if ev.Item != nil && ev.Item.Type == "function_call" {
							currentToolCall = &ToolCallInfo{
								CallID:    ev.Item.CallID,
								Name:      ev.Item.Name,
								Arguments: ev.Item.Arguments,
							}
							if currentToolCall.CallID == "" {
								currentToolCall.CallID = ev.Item.ID
							}
						}
					case "response.function_call_arguments.delta":
						if currentToolCall != nil {
							currentToolCall.Arguments += ev.Delta
						}
					case "response.function_call_arguments.done":
						if currentToolCall != nil && ev.Arguments != "" {
							currentToolCall.Arguments = ev.Arguments
						}
					case "response.output_item.done":
						if ev.Item != nil && ev.Item.Type == "function_call" {
							cid := ev.Item.CallID
							if cid == "" {
								cid = ev.Item.ID
							}
							args := ev.Item.Arguments
							if args == "" && currentToolCall != nil {
								args = currentToolCall.Arguments
							}
							toolCalls = append(toolCalls, ToolCallInfo{
								CallID:    cid,
								Name:      ev.Item.Name,
								Arguments: args,
							})
							currentToolCall = nil
						}
					case "response.completed":
						return toolCalls, textBuilder.String(), false
					case "response.incomplete":
						return toolCalls, textBuilder.String(), false
					case "error", "response.failed":
						errMsg := trimmed
						if ev.Error != nil && ev.Error.Message != "" {
							errMsg = ev.Error.Message
						}
						streamChan <- AgentStreamChunk{Content: fmt.Sprintf("Error: %s", errMsg)}
						return nil, textBuilder.String(), true
					}
				}
			}
		}

		if readErr != nil {
			if readErr != io.EOF && ctx.Err() == nil {
				streamChan <- AgentStreamChunk{Content: fmt.Sprintf("Error: Stream interrupted: %v", readErr)}
				return nil, textBuilder.String(), true
			}
			break
		}
	}

	if len(toolCalls) == 0 && currentToolCall != nil {
		toolCalls = append(toolCalls, *currentToolCall)
	}

	return toolCalls, textBuilder.String(), false
}
