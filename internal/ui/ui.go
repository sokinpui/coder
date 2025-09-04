package ui

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
)

// Start begins the read-eval-print loop (REPL) for the coder application.
func Start() {
	reader := bufio.NewReader(os.Stdin)
	fmt.Println("Enter your code (type 'EOF' on a new line to submit):")

	for {
		fmt.Print("> ")
		var input strings.Builder

		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				if err == io.EOF {
					fmt.Println()
					return
				}
				fmt.Println("Error reading input:", err)
				return
			}

			// Use "EOF" on a new line as the delimiter
			if strings.TrimSpace(line) == "EOF" {
				break
			}

			input.WriteString(line)
		}

		userInput := input.String()
		charCount := len(userInput)

		// Placeholder for the AI response
		fmt.Printf("output: You input %d char\n\n", charCount)
	}
}
