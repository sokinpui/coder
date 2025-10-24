package token

import (
	"coder/internal/tokenizer"
	"log"

	"google.golang.org/genai"
)

var tok *tokenizer.LocalTokenizer

func init() {
	var err error
	// The tokenizer is compatible across Gemini models.
	// Using NewLocalTokenizer as inspired by sllmi-go for offline tokenization.
	tok, err = tokenizer.NewLocalTokenizer("gemini-2.5-flash")
	if err != nil {
		log.Fatalf("could not load tokenizer: %v", err)
	}
}

func CountTokens(text string) int {
	ntoks, err := tok.CountTokens(genai.Text(text), nil)
	if err != nil {
		// Fallback to a rough character-based estimate on encoding failure.
		log.Printf("token counting failed, falling back to estimate: %v", err)
		return len(text) / 4
	}
	return int(ntoks.TotalTokens)
}
