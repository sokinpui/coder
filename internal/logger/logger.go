package logger

import (
	"log"
	"os"
)

const logFileName = "coder.log"

// Init sets up the global logger to write to a file.
func Init() {
	// If the file doesn't exist, create it, or append to the file
	file, err := os.OpenFile(logFileName, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		log.Fatalf("failed to open log file %s: %v", logFileName, err)
	}

	log.SetOutput(file)
	log.Println("Logger initialized. Logging to", logFileName)
}
