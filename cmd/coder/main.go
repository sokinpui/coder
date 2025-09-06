package main

import (
	"coder/internal/logger"
	"coder/internal/ui"
)

func main() {
	logger.Init()
	ui.Start()
}
