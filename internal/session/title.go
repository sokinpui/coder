package session

import (
	"coder/internal/core"
	"context"
	"log"
	"strings"
)

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
