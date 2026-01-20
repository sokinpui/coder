package modes

// NewStrategy creates a new mode strategy.
func NewStrategy() ModeStrategy {
	return &CodingMode{}
}

// NewChatStrategy creates a strategy for pure chat.
func NewChatStrategy() ModeStrategy {
	return &ChatMode{}
}

// GetStrategy returns a strategy based on the provided mode name.
func GetStrategy(mode string) ModeStrategy {
	if mode == "chat" {
		return NewChatStrategy()
	}

	return NewStrategy()
}
