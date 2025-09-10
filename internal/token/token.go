package token

import (
	"strings"
)

// CountTokens approximates the number of tokens in a string using a simple word count.
// This method is not highly accurate but serves as a basic, dependency-free estimator.
func CountTokens(text string) int {
	return len(strings.Fields(text))
}
