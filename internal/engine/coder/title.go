package coder

import (
	"context"
	coderprompt "github.com/sokinpui/coder/internal/engine/coder/prompt"
	"strings"
)

func (s *Session) GetTitle() string {
	return s.title
}

func (s *Session) IsTitleGenerated() bool {
	return s.titleGenerated
}

func (s *Session) GenerateTitle(ctx context.Context, userPrompt string) string {
	s.titleGenerated = true // Set this first to prevent concurrent calls.
	s.Runtime.Config = s.config.Coder.ModelConfig()
	s.Runtime.Generator.TitleModel = s.config.Title.ModelCode

	prompt := strings.Replace(coderprompt.TitleGenerationPrompt, "{{PROMPT}}", userPrompt, 1)

	title, err := s.Runtime.GenerateTitle(ctx, prompt)
	if err != nil {
		words := strings.Fields(userPrompt)
		numWords := min(len(words), 5)
		fallbackTitle := strings.Join(words[:numWords], " ")
		if len(words) > numWords {
			fallbackTitle += "..."
		}
		s.title = fallbackTitle
		return s.title
	}

	s.title = strings.Trim(title, "\"") // Models sometimes add quotes
	return s.title
}

func (s *Session) SetTitle(title string) {
	if strings.TrimSpace(title) == "" {
		return
	}
	s.title = title
	s.titleGenerated = true // Mark as manually set/generated
}
