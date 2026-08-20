package token

import (
	"sync"

	"github.com/sokinpui/coder/internal/types"
	"github.com/tiktoken-go/tokenizer"
)

var (
	enc     tokenizer.Codec
	encOnce sync.Once
)

func getEncoder() tokenizer.Codec {
	encOnce.Do(func() {
		var err error
		enc, err = tokenizer.Get(tokenizer.Cl100kBase)
		if err != nil {
			enc, _ = tokenizer.Get(tokenizer.O200kBase)
		}
	})
	return enc
}

func CountTokens(messages []types.Message) int {
	encoder := getEncoder()
	total := 0

	for _, msg := range messages {
		if !msg.CanSendToAI() {
			continue
		}
		if msg.Type == types.ImageMessage {
			total += 1500
			continue
		}

		if encoder != nil {
			ids, _, err := encoder.Encode(msg.Content)
			if err == nil {
				total += len(ids)
				continue
			}
		}

		// Fallback heuristic if encoder fails
		total += estimateTokensFallback(msg.Content)
	}

	return total
}

func estimateTokensFallback(text string) int {
	// Simple fallback: ~4 characters per token average
	return len(text) / 4
}
