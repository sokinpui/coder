package logger

import (
	"github.com/sokinpui/coder/internal/utils"
	"log"
	"os"
	"path/filepath"
)

const (
	coderDirName = ".coder"
	logFileName  = "coder.log"
)

func Init() {
	repoRoot := utils.GetProjectRoot()
	coderPath := filepath.Join(repoRoot, coderDirName)
	if err := os.MkdirAll(coderPath, 0755); err != nil {
		log.Fatalf("failed to create .coder directory for log file: %v", err)
	}

	logFilePath := filepath.Join(coderPath, logFileName)
	// If the file doesn't exist, create it, or append to the file
	file, err := os.OpenFile(logFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		log.Fatalf("failed to open log file %s: %v", logFilePath, err)
	}

	log.SetOutput(file)
	log.Println("Logger initialized. Logging to", logFilePath)
}
