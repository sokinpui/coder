package token

import (
	"math"
	"strings"
	"unicode"

	"github.com/sokinpui/coder/internal/types"
)

const (
	englishCharTokens = 0.3
	chineseCharTokens = 0.6
	imageTokens       = 1200
)

func CountTokens(messages []types.Message) int {
	if len(messages) == 0 {
		return 0
	}

	total := 0
	for _, msg := range messages {
		total += countSingleMessage(msg)
	}
	return total
}

func countSingleMessage(msg types.Message) int {
	if !msg.CanSendToAI() {
		return 0
	}
	if msg.Type == types.ImageMessage {
		return imageTokens
	}
	content := messageTokenContent(msg)
	if content == "" {
		return 0
	}
	return estimateTextTokens(content)
}

func estimateTextTokens(text string) int {
	if text == "" {
		return 0
	}

	text = strings.ReplaceAll(text, "        ", " ")
	text = strings.ReplaceAll(text, "    ", " ")

	var chineseChars, otherChars int
	for _, r := range text {
		if unicode.Is(unicode.Han, r) {
			chineseChars++
			continue
		}
		otherChars++
	}

	total := float64(chineseChars)*chineseCharTokens + float64(otherChars)*englishCharTokens
	return int(math.Ceil(total))
}

func messageTokenContent(msg types.Message) string {
	if len(msg.ToolCalls) == 0 {
		return msg.Content
	}
	var sb strings.Builder
	if msg.Content != "" {
		sb.WriteString(msg.Content)
	}
	for _, tc := range msg.ToolCalls {
		if sb.Len() > 0 {
			sb.WriteString("\n")
		}
		sb.WriteString(tc.Name)
		if tc.Arguments != "" {
			sb.WriteString(" ")
			sb.WriteString(tc.Arguments)
		}
	}
	return sb.String()
}
