package token

import (
	"github.com/sokinpui/coder/internal/tokenizer"
	"fmt"
	"log"

	"google.golang.org/genai"
)

var tok *tokenizer.LocalTokenizer

func Init() error {
	var err error
	// The tokenizer is compatible across Gemini models.
	// Using NewLocalTokenizer as inspired by sllmi-go for offline tokenization.
	tok, err = tokenizer.NewLocalTokenizer("gemini-2.5-flash")
	if err != nil {
		return fmt.Errorf("could not load tokenizer: %w", err)
	}
	return nil
}

func CountTokens(text string) int {
	if tok == nil {
		return len(text) / 4
	}
	ntoks, err := tok.CountTokens(genai.Text(text), nil)
	if err != nil {
		// Fallback to a rough character-based estimate on encoding failure.
		log.Printf("token counting failed, falling back to estimate: %v", err)
		return len(text) / 4
	}
	return int(ntoks.TotalTokens)
}
