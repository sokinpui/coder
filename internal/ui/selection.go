package ui

import "coder/internal/types"

// filter out system messages and Directory messages into blocks
func groupMessages(messages []types.Message) []messageBlock {
	var blocks []messageBlock
	if len(messages) == 0 {
		return blocks
	}

	i := 0
	for i < len(messages) {
		msg := messages[i]
		block := messageBlock{startIdx: i, endIdx: i}

		// NOTE: Group consecutive messages of the same type into a single block
		// NOTE: deceparted
		// switch msg.Type {
		// // by design message after command/tool is always result/error, and they alwasy come in pairs
		// case core.CommandMessage:
		// 	if i+1 < len(messages) {
		// 		block.endIdx = i + 1
		// 	}
		//
		// }
		//

		// Skip system messages from being selectable blocks
		if msg.Type != types.InitMessage && msg.Type != types.DirectoryMessage {
			blocks = append(blocks, block)
		}

		i = block.endIdx + 1
	}

	return blocks
}
