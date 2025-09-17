package git

import (
	"os/exec"
	"strings"
)

// LogEntry represents a single commit in the git log.
type LogEntry struct {
	Hash         string `json:"hash"`
	AuthorName   string `json:"authorName"`
	RelativeDate string `json:"relativeDate"`
	Subject      string `json:"subject"`
	Body         string `json:"body"`
}

// GetLog retrieves a formatted git log from the repository.
func GetLog() ([]LogEntry, error) {
	// Using unit separator (0x1f) for fields and record separator (0x1e) for commits.
	const fieldSeparator = "\x1f"
	const recordSeparator = "\x1e"
	const format = "%H" + fieldSeparator + "%an" + fieldSeparator + "%ar" + fieldSeparator + "%B"

	cmd := exec.Command("git", "log", "--pretty=format:"+format+recordSeparator)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, err
	}

	var entries []LogEntry
	logs := strings.Split(string(output), recordSeparator)

	for _, log := range logs {
		if log == "" {
			continue
		}
		parts := strings.SplitN(log, fieldSeparator, 4)
		if len(parts) != 4 {
			continue // Skip malformed lines
		}

		fullMessage := parts[3]
		messageLines := strings.SplitN(fullMessage, "\n", 2)
		subject := messageLines[0]
		body := ""
		if len(messageLines) > 1 {
			body = strings.TrimSpace(messageLines[1])
		}

		entry := LogEntry{
			Hash:         parts[0],
			AuthorName:   parts[1],
			RelativeDate: parts[2],
			Subject:      subject,
			Body:         body,
		}
		entries = append(entries, entry)
	}
	return entries, nil
}
