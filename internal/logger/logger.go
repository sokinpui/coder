package logger

import (
	"coder/internal/config"
	"coder/internal/utils"
	"log"
	"os"
	"path/filepath"
)

const (
	coderDirName = ".coder"
	logFileName  = "coder.log"
)

// Init sets up the global logger to write to a file.
func Init() {
	repoRoot, err := utils.FindRepoRoot()
	if err != nil {
		// If we can't find the repo root, we can't create the log file in the desired location.
		// This is a fatal error as logging is essential and other parts of the app will also fail.
		log.Fatalf("failed to find git repository root to initialize logger: %v", err)
	}

	coderPath := filepath.Join(repoRoot, coderDirName)
	if err := os.MkdirAll(coderPath, 0755); err != nil {
		log.Fatalf("failed to create .coder directory for log file: %v", err)
	}

	configPath := filepath.Join(coderPath, "config.yaml")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		if err := os.WriteFile(configPath, []byte(config.ConfigTemplate), 0644); err != nil {
			log.Printf("failed to write template config file: %v", err)
		}
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
