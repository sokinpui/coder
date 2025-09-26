package ui

import "coder/internal/core"

// groupMessages analyzes the message history and groups them into selectable blocks.
// User/AI messages are single blocks. Command/Action messages are grouped with their results.
func groupMessages(messages []core.Message) []messageBlock {
	var blocks []messageBlock
	if len(messages) == 0 {
		return blocks
	}

	i := 0
	for i < len(messages) {
		msg := messages[i]
		block := messageBlock{startIdx: i, endIdx: i}

		switch msg.Type {
		case core.CommandMessage:
			if i+1 < len(messages) {
				nextMsgType := messages[i+1].Type
				if nextMsgType == core.CommandResultMessage || nextMsgType == core.CommandErrorResultMessage {
					block.endIdx = i + 1
				}
			}
		}

		// Skip system messages from being selectable blocks
		if msg.Type != core.InitMessage && msg.Type != core.DirectoryMessage {
			blocks = append(blocks, block)
		}

		i = block.endIdx + 1
	}

	return blocks
}
