package compactor

import (
	"fmt"

	"github.com/sokinpui/coder/internal/types"
)

const (
	minConversationalMessages = 4
	minTailMessages           = 2
)

func FindSplitPoint(messages []types.Message) (int, int, error) {
	startIndex := findConversationalStart(messages)
	if startIndex < 0 || startIndex >= len(messages) {
		return -1, -1, fmt.Errorf("no conversational messages available to compact")
	}

	convMsgs := messages[startIndex:]
	if len(convMsgs) < minConversationalMessages {
		return -1, -1, fmt.Errorf("conversation too short to compact (need at least %d turns, got %d)", minConversationalMessages, len(convMsgs))
	}

	initialCut := len(convMsgs) - minTailMessages - 1
	balancedCut := balanceToolCallBoundaries(convMsgs, initialCut)
	if balancedCut < 0 {
		return -1, -1, fmt.Errorf("unable to find a safe turn boundary for compaction")
	}

	return startIndex, balancedCut, nil
}

func findConversationalStart(messages []types.Message) int {
	for i, msg := range messages {
		switch msg.Type {
		case types.InitMessage, types.DirectoryMessage, types.InstructionMessage, types.SourceCodeMessage:
			continue
		default:
			return i
		}
	}
	return -1
}

func balanceToolCallBoundaries(convMsgs []types.Message, cut int) int {
	maxAllowedCut := len(convMsgs) - 2
	if cut > maxAllowedCut {
		cut = maxAllowedCut
	}

	for cut >= 0 {
		if convMsgs[cut].Type == types.ToolCallMessage {
			cut--
			continue
		}

		unanswered := countUnansweredToolCalls(convMsgs[:cut+1])
		if len(unanswered) == 0 {
			return cut
		}

		forwardCandidate := tryResolveForward(convMsgs, cut, unanswered, maxAllowedCut)
		if forwardCandidate >= 0 {
			return forwardCandidate
		}

		earliestIndex := findEarliestToolCallIndex(convMsgs[:cut+1], unanswered)
		cut = earliestIndex - 1
	}

	return -1
}

func countUnansweredToolCalls(span []types.Message) map[string]struct{} {
	unanswered := make(map[string]struct{})
	for _, msg := range span {
		if msg.Type == types.ToolCallMessage {
			for _, tc := range msg.ToolCalls {
				if tc.ID != "" {
					unanswered[tc.ID] = struct{}{}
				}
			}
		}
		if msg.Type == types.ToolResultMessage && msg.ToolCallID != "" {
			delete(unanswered, msg.ToolCallID)
		}
	}
	return unanswered
}

func tryResolveForward(convMsgs []types.Message, start int, unanswered map[string]struct{}, maxAllowed int) int {
	remaining := make(map[string]struct{}, len(unanswered))
	for k := range unanswered {
		remaining[k] = struct{}{}
	}

	for idx := start + 1; idx <= maxAllowed; idx++ {
		msg := convMsgs[idx]
		if msg.Type == types.ToolCallMessage {
			for _, tc := range msg.ToolCalls {
				if tc.ID != "" {
					remaining[tc.ID] = struct{}{}
				}
			}
		}
		if msg.Type == types.ToolResultMessage && msg.ToolCallID != "" {
			delete(remaining, msg.ToolCallID)
		}
		if len(remaining) == 0 && msg.Type != types.ToolCallMessage {
			return idx
		}
	}

	return -1
}

func findEarliestToolCallIndex(span []types.Message, callIDs map[string]struct{}) int {
	for i, msg := range span {
		if msg.Type != types.ToolCallMessage {
			continue
		}
		for _, tc := range msg.ToolCalls {
			if _, exists := callIDs[tc.ID]; exists {
				return i
			}
		}
	}
	return len(span) - 1
}
