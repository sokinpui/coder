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
}

// GetLog retrieves a formatted git log from the repository.
func GetLog() ([]LogEntry, error) {
	// Using a custom format that's easy to parse.
	// %H: commit hash
	// %an: author name
	// %ar: author date, relative
	// %s: subject
	const format = "%H|%an|%ar|%s"
	cmd := exec.Command("git", "log", "--pretty=format:"+format)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, err
	}

	var entries []LogEntry
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")

	for _, line := range lines {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "|", 4)
		if len(parts) != 4 {
			continue // Skip malformed lines
		}
		entry := LogEntry{
			Hash:         parts[0],
			AuthorName:   parts[1],
			RelativeDate: parts[2],
			Subject:      parts[3],
		}
		entries = append(entries, entry)
	}
	return entries, nil
}
