package modes

// NewStrategy creates a new mode strategy.
func NewStrategy() ModeStrategy {
	return &CodingMode{}
}

// NewChatStrategy creates a strategy for pure chat.
func NewChatStrategy() ModeStrategy {
	return &ChatMode{}
}
