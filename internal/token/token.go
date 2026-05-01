package token

import (
	"fmt"
	"log"
	"strings"

	"github.com/sokinpui/coder/internal/types"
	"google.golang.org/genai"
)

var tok *LocalTokenizer

func Init() error {
	var err error
	tok, err = NewLocalTokenizer("gemini-2.5-flash")
	if err != nil {
		return fmt.Errorf("could not load tokenizer: %w", err)
	}
	return nil
}

func CountTokens(messages []types.Message) int {
	var combinedText strings.Builder
	for _, msg := range messages {
		combinedText.WriteString(msg.Content + "\n")
	}

	if tok == nil {
		return len(combinedText.String()) / 4
	}
	ntoks, err := tok.CountTokens(genai.Text(combinedText.String()), nil)
	if err != nil {
		// Fallback to a rough character-based estimate on encoding failure.
		log.Printf("token counting failed, falling back to estimate: %v", err)
		return len(combinedText.String()) / 4
	}
	return int(ntoks.TotalTokens)
}
