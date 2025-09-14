package token

import (
	"log"

	"github.com/tiktoken-go/tokenizer"
)

var codec tokenizer.Codec

func init() {
	var err error
	// Using cl100k_base as a general-purpose encoder for good approximation.
	codec, err = tokenizer.Get(tokenizer.Cl100kBase)
	if err != nil {
		log.Fatalf("could not load tokenizer: %v", err)
	}
}

// CountTokens provides a more accurate token count using the tiktoken library.
func CountTokens(text string) int {
	ids, _, err := codec.Encode(text)
	if err != nil {
		// Fallback to a rough character-based estimate on encoding failure.
		return len(text) / 4
	}
	return len(ids)
}
