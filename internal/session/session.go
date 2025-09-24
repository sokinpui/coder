package session

import (
	"coder/internal/config"
	"coder/internal/contextdir"
	"coder/internal/core"
	"coder/internal/generation"
	"coder/internal/history"
	"coder/internal/source"
	"coder/internal/utils"
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
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
	// VisualModeStarted indicates the UI should enter visual mode.
	VisualModeStarted
	// GenerateModeStarted indicates the UI should enter visual generate mode.
	GenerateModeStarted
	// EditModeStarted indicates the UI should enter visual edit mode.
	EditModeStarted
	// BranchModeStarted indicates the UI should enter visual branch mode.
	BranchModeStarted
	// HistoryModeStarted indicates the UI should enter history browsing mode.
	HistoryModeStarted
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
	title              string
	titleGenerated     bool
	historyFilename    string
	createdAt          time.Time
}

// New creates a new session.
func New(cfg *config.Config) (*Session, error) {
	return NewWithMessages(cfg, nil)
}

// NewWithMessages creates a new session with a pre-existing set of messages.
func NewWithMessages(cfg *config.Config, initialMessages []core.Message) (*Session, error) {
	gen, err := generation.New(cfg)
	if err != nil {
		return nil, err
	}

	hist, err := history.NewManager()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize history manager: %w", err)
	}

	// Make a defensive copy of the slice to avoid external modifications.
	messages := make([]core.Message, len(initialMessages))
	copy(messages, initialMessages)

	return &Session{
		config:         cfg,
		generator:      gen,
		historyManager: hist,
		messages:       messages,
		title:          "New Chat",
		titleGenerated: false,
		createdAt:      time.Now(),
		historyFilename:    "",
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

	repoRoot, err := utils.FindRepoRoot()
	if err != nil {
		log.Printf("Error finding repo root for deleting image: %v", err)
		// We can still proceed to delete the message from history, but log the error.
		repoRoot = ""
	}

	toDelete := make(map[int]struct{})
	for _, idx := range indices {
		if idx < 0 || idx >= len(s.messages) {
			continue
		}
		toDelete[idx] = struct{}{}

		msg := s.messages[idx]
		if msg.Type == core.ImageMessage && repoRoot != "" {
			imagePath := filepath.Join(repoRoot, msg.Content)
			// Security check to prevent path traversal
			if !strings.HasPrefix(imagePath, filepath.Join(repoRoot, ".coder", "images")) {
				log.Printf("Skipping deletion of potential path traversal: %s", msg.Content)
				continue
			}

			err := os.Remove(imagePath)
			if err != nil && !os.IsNotExist(err) {
				log.Printf("Failed to delete image file %s: %v", imagePath, err)
			}
		}
	}

	newMessages := make([]core.Message, 0, len(s.messages)-len(indices))
	for i, msg := range s.messages {
		if _, found := toDelete[i]; !found {
			newMessages = append(newMessages, msg)
		}
	}
	s.messages = newMessages
}

// EditMessage updates the content of a user message at a given index.
// It only allows editing of UserMessage types.
func (s *Session) EditMessage(index int, newContent string) error {
	if index < 0 || index >= len(s.messages) {
		return fmt.Errorf("index out of bounds: %d", index)
	}
	if s.messages[index].Type != core.UserMessage {
		return fmt.Errorf("can only edit user messages, but got type %v at index %d", s.messages[index].Type, index)
	}

	s.messages[index].Content = newContent
	return nil
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
	case config.AutoMode:
		return core.AutoRole
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
	if s.historyFilename == "" {
		s.historyFilename = fmt.Sprintf("%d.md", s.createdAt.Unix())
	}

	role := s.getCurrentRole()
	data := &history.ConversationData{
		Filename:          s.historyFilename,
		Title:             s.title,
		CreatedAt:         s.createdAt,
		Messages:          s.messages,
		Role:              role,
		SystemInstructions: s.systemInstructions,
		RelatedDocuments:  s.relatedDocuments,
		ProjectSourceCode: s.projectSourceCode,
	}
	return s.historyManager.SaveConversation(data)
}

// GetTitle returns the conversation title.
func (s *Session) GetTitle() string {
	return s.title
}

// IsTitleGenerated checks if a title has been generated for the session.
func (s *Session) IsTitleGenerated() bool {
	return s.titleGenerated
}

// GenerateTitle generates and sets a title for the conversation based on the first user prompt.
func (s *Session) GenerateTitle(ctx context.Context, userPrompt string) string {
	s.titleGenerated = true // Set this first to prevent concurrent calls.

	prompt := strings.Replace(core.TitleGenerationPrompt, "{{PROMPT}}", userPrompt, 1)

	title, err := s.generator.GenerateTitle(ctx, prompt)
	if err != nil {
		log.Printf("Error generating title, falling back to first few words: %v", err)
		words := strings.Fields(userPrompt)
		numWords := 5
		if len(words) < numWords {
			numWords = len(words)
		}
		fallbackTitle := strings.Join(words[:numWords], " ")
		if len(words) > numWords {
			fallbackTitle += "..."
		}
		s.title = fallbackTitle
		return s.title
	}

	s.title = strings.Trim(title, "\"") // Models sometimes add quotes
	log.Printf("Generated title: %s", s.title)
	return s.title
}

// SetTitle manually sets the conversation title.
func (s *Session) SetTitle(title string) {
	if strings.TrimSpace(title) == "" {
		return
	}
	s.title = title
	s.titleGenerated = true // Mark as manually set/generated
}

// GetHistoryFilename returns the filename for the current conversation in history.
// It returns an empty string if the session hasn't been saved yet.
func (s *Session) GetHistoryFilename() string {
	return s.historyFilename
}

// GetHistoryManager returns the session's history manager.
func (s *Session) GetHistoryManager() *history.Manager {
	return s.historyManager
}

// LoadConversation loads a conversation from a history file, replacing the current session state.
func (s *Session) LoadConversation(filename string) error {
	if len(s.messages) > 0 {
		if err := s.SaveConversation(); err != nil {
			// Log the error but continue, as loading a new session is more important.
			log.Printf("Error saving current conversation before loading another: %v", err)
		}
	}

	metadata, messages, err := s.historyManager.LoadConversation(filename)
	if err != nil {
		return fmt.Errorf("failed to load conversation %s: %w", filename, err)
	}

	s.messages = messages
	s.title = metadata.Title
	s.titleGenerated = true // A loaded conversation always has a title.
	s.createdAt = metadata.CreatedAt
	s.historyFilename = filename

	if err := s.reloadProjectSource(); err != nil {
		log.Printf("Error reloading project source after loading conversation: %v", err)
	}
	return nil
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
	s.title = "New Chat"
	s.titleGenerated = false
	s.createdAt = time.Now()
	s.historyFilename = ""
}

// Branch saves the current session and creates a new one containing messages
// up to the specified index.
func (s *Session) Branch(endMessageIndex int) (*Session, error) {
	if err := s.SaveConversation(); err != nil {
		return nil, fmt.Errorf("failed to save current session before branching: %w", err)
	}

	if endMessageIndex < 0 || endMessageIndex >= len(s.messages) {
		return nil, fmt.Errorf("invalid index for branching: %d", endMessageIndex)
	}

	messagesToKeep := s.messages[:endMessageIndex+1]

	// NewWithMessages makes a defensive copy, so this is safe.
	newSess, err := NewWithMessages(s.config, messagesToKeep)
	if err != nil {
		return nil, err
	}

	// The new session needs the context from the old one.
	newSess.systemInstructions = s.systemInstructions
	newSess.relatedDocuments = s.relatedDocuments
	newSess.projectSourceCode = s.projectSourceCode

	return newSess, nil
}

// RegenerateFrom truncates the message history to the specified user message
// and starts a new generation.
func (s *Session) RegenerateFrom(userMessageIndex int) Event {
	if userMessageIndex < 0 || userMessageIndex >= len(s.messages) || s.messages[userMessageIndex].Type != core.UserMessage {
		s.messages = append(s.messages, core.Message{
			Type:    core.CommandErrorResultMessage,
			Content: "Invalid index for regeneration.",
		})
		return Event{Type: MessagesUpdated}
	}

	s.messages = s.messages[:userMessageIndex+1]
	return s.startGeneration()
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

	// Collect image paths from recent messages that precede the current user prompt.
	var imgPaths []string
	// Iterate backwards from the message before the last one (which is the user prompt).
	for i := len(s.messages) - 2; i >= 0; i-- {
		msg := s.messages[i]
		if msg.Type == core.ImageMessage {
			imgPaths = append(imgPaths, msg.Content)
		} else if msg.Type == core.UserMessage || msg.Type == core.AIMessage {
			// Stop when we hit the previous conversation turn.
			break
		}
	}
	// Reverse the slice to maintain the original order of images.
	for i, j := 0, len(imgPaths)-1; i < j; i, j = i+1, j-1 {
		imgPaths[i], imgPaths[j] = imgPaths[j], imgPaths[i]
	}

	// Convert relative image paths to absolute paths for the generation server.
	if len(imgPaths) > 0 {
		repoRoot, err := utils.FindRepoRoot()
		if err != nil {
			log.Printf("Error finding repo root for image paths: %v", err)
			s.messages = append(s.messages, core.Message{
				Type:    core.CommandErrorResultMessage,
				Content: fmt.Sprintf("Failed to resolve image paths:\n%v", err),
			})
			return Event{Type: MessagesUpdated}
		}
		for i, p := range imgPaths {
			imgPaths[i] = filepath.Join(repoRoot, p)
		}
	}

	streamChan := make(chan string)
	ctx, cancel := context.WithCancel(context.Background())
	s.cancelGeneration = cancel
	go s.generator.GenerateTask(ctx, prompt, imgPaths, streamChan)

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

	cmdOutput, _, cmdSuccess := core.ProcessCommand(input, s.messages, s.config, s)
	// ProcessCommand returns isCmd=true for any string with ':', so we don't need to check it.

	if cmdSuccess {
		switch cmdOutput.Type {
		case core.CommandResultNewSession:
			s.newSession()
			return Event{Type: NewSessionStarted}
		case core.CommandResultVisualMode:
			return Event{Type: VisualModeStarted}
		case core.CommandResultGenerateMode:
			return Event{Type: GenerateModeStarted}
		case core.CommandResultEditMode:
			return Event{Type: EditModeStarted}
		case core.CommandResultBranchMode:
			return Event{Type: BranchModeStarted}
		case core.CommandResultHistoryMode:
			return Event{Type: HistoryModeStarted}
		}
	}

	s.generator.Config = s.config.Generation
	s.messages = append(s.messages, core.Message{Type: core.CommandMessage, Content: input})
	if cmdSuccess {
		s.messages = append(s.messages, core.Message{Type: core.CommandResultMessage, Content: cmdOutput.Payload})
	} else {
		s.messages = append(s.messages, core.Message{Type: core.CommandErrorResultMessage, Content: cmdOutput.Payload})
	}
	return Event{Type: MessagesUpdated}
}
