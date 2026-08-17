package token

import (
	"math"
	"unicode"

	"github.com/sokinpui/coder/internal/types"
)

func CountTokens(messages []types.Message) int {
	var total float64
	for _, msg := range messages {
		if msg.Type == types.ImageMessage {
			total += 1500
			continue
		}
		for _, r := range msg.Content {
			if unicode.Is(unicode.Han, r) {
				total += 0.66
				continue
			}
			total += 0.36
		}
		total += 0.36
	}

	return int(math.Round(total))
}
