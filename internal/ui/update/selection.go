package update

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
		// by design message after command/tool is always result/error, and they alwasy come in pairs
		case core.CommandMessage, core.ToolCallMessage:
			if i+1 < len(messages) {
				block.endIdx = i + 1
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
