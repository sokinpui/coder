package session

import (
	"coder/internal/core"
	"context"
	"log"
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

func (s *Session) SetTitle(title string) {
	if strings.TrimSpace(title) == "" {
		return
	}
	s.title = title
	s.titleGenerated = true // Mark as manually set/generated
}
