package modes

// NewStrategy creates a new mode strategy.
func NewStrategy(instruction string) ModeStrategy {
	return &CodingMode{instruction: instruction}
}

// NewChatStrategy creates a strategy for pure chat.
func NewChatStrategy() ModeStrategy {
	return &ChatMode{}
}

// GetStrategy returns a strategy based on the provided mode name.
func GetStrategy(mode string, instruction string) ModeStrategy {
	if mode == "chat" {
		return NewChatStrategy()
	}

	return NewStrategy(instruction)
}
