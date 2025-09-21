package token

import (
	"log"

	vgenai "cloud.google.com/go/vertexai/genai"
	"cloud.google.com/go/vertexai/genai/tokenizer"
)

var tok *tokenizer.Tokenizer

func init() {
	var err error
	// The tokenizer is compatible across Gemini models.
	tok, err = tokenizer.New("gemini-1.5-flash")
	if err != nil {
		log.Fatalf("could not load tokenizer: %v", err)
	}
}

func CountTokens(text string) int {
	ntoks, err := tok.CountTokens(vgenai.Text(text))
	if err != nil {
		// Fallback to a rough character-based estimate on encoding failure.
		log.Printf("token counting failed, falling back to estimate: %v", err)
		return len(text) / 4
	}
	return int(ntoks.TotalTokens)
}
