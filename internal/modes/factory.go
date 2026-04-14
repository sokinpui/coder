package modes

func NewStrategy(instruction string) ModeStrategy {
	return &CodingMode{instruction: instruction}
}

func NewChatStrategy() ModeStrategy {
	return &ChatMode{}
}

func GetStrategy(mode string, instruction string) ModeStrategy {
	if mode == "chat" {
		return NewChatStrategy()
	}

	return NewStrategy(instruction)
}
