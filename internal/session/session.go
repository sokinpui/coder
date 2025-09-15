package session

import (
	"coder/internal/config"
	"coder/internal/contextdir"
	"coder/internal/core"
	"coder/internal/generation"
	"coder/internal/history"
	"coder/internal/source"
	"context"
	"fmt"
	"log"
	"strings"
)

// EventType defines the type of event returned by the session.
type EventType int

const (
	// NoOp indicates that no significant action was taken.
	NoOp EventType = iota
	// MessagesUpdated indicates that the message list was updated.
	MessagesUpdated
	// GenerationStarted indicates a new AI generation task has begun.
	GenerationStarted
	// NewSessionStarted indicates the session has been reset.
	NewSessionStarted
)

// Event is returned by session methods to inform the UI about what happened.
type Event struct {
	Type EventType
	Data interface{} // Can be a stream channel for GenerationStarted or an error for ErrorOccurred
}

// Session manages the state of a single conversation.
type Session struct {
	config             *config.Config
	generator          *generation.Generator
	historyManager     *history.Manager
	messages           []core.Message
	systemInstructions string
	relatedDocuments   string
	projectSourceCode  string
	cancelGeneration   context.CancelFunc
}

// New creates a new session.
func New(cfg *config.Config) (*Session, error) {
	gen, err := generation.New(cfg)
	if err != nil {
		return nil, err
	}

	hist, err := history.NewManager()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize history manager: %w", err)
	}

	return &Session{
		config:         cfg,
		generator:      gen,
		historyManager: hist,
		messages:       []core.Message{}, // Start with empty messages
	}, nil
}

// LoadContext loads the initial context for the session.
func (s *Session) LoadContext() error {
	sysInstructions, docs, ctxErr := contextdir.LoadContext()
	if ctxErr != nil {
		return fmt.Errorf("failed to load context directory: %w", ctxErr)
	}

	projSource, srcErr := source.LoadProjectSource(s.config.AppMode)
	if srcErr != nil {
		return fmt.Errorf("failed to load project source: %w", srcErr)
	}

	s.systemInstructions = sysInstructions
	s.relatedDocuments = docs
	s.projectSourceCode = projSource
	return nil
}

// GetMessages returns the current conversation messages.
func (s *Session) GetMessages() []core.Message {
	return s.messages
}

// AddMessage allows adding a message to the history from outside (e.g., UI-specific messages).
func (s *Session) AddMessage(msg core.Message) {
	s.messages = append(s.messages, msg)
}

// ReplaceLastMessage allows updating the last message (e.g., for streaming).
func (s *Session) ReplaceLastMessage(msg core.Message) {
	if len(s.messages) > 0 {
		s.messages[len(s.messages)-1] = msg
	}
}

// DeleteMessages removes messages at the given indices from the session.
func (s *Session) DeleteMessages(indices []int) {
	if len(indices) == 0 {
		return
	}

	toDelete := make(map[int]struct{})
	for _, idx := range indices {
		toDelete[idx] = struct{}{}
	}

	newMessages := make([]core.Message, 0, len(s.messages)-len(indices))
	for i, msg := range s.messages {
		if _, found := toDelete[i]; !found {
			newMessages = append(newMessages, msg)
		}
	}
	s.messages = newMessages
}

// RemoveLastInteraction removes the last user message and AI response,
// typically after a failed or cancelled generation.
func (s *Session) RemoveLastInteraction() {
	if len(s.messages) >= 2 {
		s.messages = s.messages[:len(s.messages)-2]
	}
}

// GetConfig returns the application configuration.
func (s *Session) GetConfig() *config.Config {
	return s.config
}

func (s *Session) getCurrentRole() string {
	switch s.config.AppMode {
	case config.DocumentingMode:
		return core.DocumentingRole
	case config.CodingMode:
		fallthrough
	default:
		return core.CodingRole
	}
}

// GetPromptForTokenCount builds and returns the full prompt string for token counting.
func (s *Session) GetPromptForTokenCount() string {
	role := s.getCurrentRole()
	return core.BuildPrompt(role, s.systemInstructions, s.relatedDocuments, s.projectSourceCode, s.messages)
}

// GetInitialPromptForTokenCount returns the prompt with only the context.
func (s *Session) GetInitialPromptForTokenCount() string {
	role := s.getCurrentRole()
	return core.BuildPrompt(role, s.systemInstructions, s.relatedDocuments, s.projectSourceCode, nil)
}

// SaveConversation saves the current conversation to history.
func (s *Session) SaveConversation() error {
	role := s.getCurrentRole()
	return s.historyManager.SaveConversation(s.messages, role, s.systemInstructions, s.relatedDocuments, s.projectSourceCode)
}

// CancelGeneration cancels any ongoing AI generation.
func (s *Session) CancelGeneration() {
	if s.cancelGeneration != nil {
		s.cancelGeneration()
	}
}

func (s *Session) newSession() {
	if err := s.SaveConversation(); err != nil {
		log.Printf("Error saving conversation for /new command: %v", err)
	}
	s.messages = []core.Message{} // Clear messages
}

// reloadProjectSource reloads the project source code from disk.
func (s *Session) reloadProjectSource() error {
	projSource, err := source.LoadProjectSource(s.config.AppMode)
	if err != nil {
		return fmt.Errorf("failed to reload project source: %w", err)
	}
	s.projectSourceCode = projSource
	return nil
}

func (s *Session) startGeneration() Event {
	if err := s.reloadProjectSource(); err != nil {
		log.Printf("Error reloading project source for generation: %v", err)
		s.messages = append(s.messages, core.Message{
			Type:    core.CommandErrorResultMessage,
			Content: fmt.Sprintf("Failed to reload project source before generation:\n%v", err),
		})
		return Event{Type: MessagesUpdated}
	}

	prompt := s.GetPromptForTokenCount()

	streamChan := make(chan string)
	ctx, cancel := context.WithCancel(context.Background())
	s.cancelGeneration = cancel

	go s.generator.GenerateTask(ctx, prompt, streamChan)

	s.messages = append(s.messages, core.Message{Type: core.AIMessage, Content: ""}) // Placeholder for AI

	return Event{
		Type: GenerationStarted,
		Data: streamChan,
	}
}

// HandleInput processes user input (prompts, commands, actions).
func (s *Session) HandleInput(input string) Event {
	if strings.TrimSpace(input) == "" {
		return Event{Type: NoOp}
	}

	if !strings.HasPrefix(input, ":") {
		// This is a new user prompt.
		s.messages = append(s.messages, core.Message{Type: core.UserMessage, Content: input})
		return s.startGeneration()
	}

	actionResult, isAction, actionSuccess := core.ProcessAction(input)
	if isAction {
		s.messages = append(s.messages, core.Message{Type: core.ActionMessage, Content: input})
		if actionSuccess {
			s.messages = append(s.messages, core.Message{Type: core.ActionResultMessage, Content: actionResult})
		} else {
			s.messages = append(s.messages, core.Message{Type: core.ActionErrorResultMessage, Content: actionResult})
		}
		return Event{Type: MessagesUpdated}
	}

	cmdResult, _, cmdSuccess := core.ProcessCommand(input, s.messages, s.config)
	// ProcessCommand returns isCmd=true for any string with ':', so we don't need to check it.

	if cmdSuccess && cmdResult == core.NewSessionResult {
		s.newSession()
		return Event{Type: NewSessionStarted}
	}

	if cmdSuccess && cmdResult == core.RegenerateResult {
		lastUserMsgIndex := -1
		for i := len(s.messages) - 1; i >= 0; i-- {
			if s.messages[i].Type == core.UserMessage {
				lastUserMsgIndex = i
				break
			}
		}

		if lastUserMsgIndex == -1 {
			s.messages = append(s.messages, core.Message{Type: core.CommandErrorResultMessage, Content: "No previous user prompt to regenerate from."})
			return Event{Type: MessagesUpdated}
		}

		s.messages = s.messages[:lastUserMsgIndex+1]
		return s.startGeneration()
	}

	s.generator.Config = s.config.Generation
	s.messages = append(s.messages, core.Message{Type: core.CommandMessage, Content: input})
	if cmdSuccess {
		s.messages = append(s.messages, core.Message{Type: core.CommandResultMessage, Content: cmdResult})
	} else {
		s.messages = append(s.messages, core.Message{Type: core.CommandErrorResultMessage, Content: cmdResult})
	}
	return Event{Type: MessagesUpdated}
}
