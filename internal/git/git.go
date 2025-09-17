package git

import (
	"fmt"
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
	const format = "%H" + fieldSeparator + "%an" + fieldSeparator + "%ar" + fieldSeparator + "%s" + fieldSeparator + "%b"

	cmd := exec.Command("git", "log", "--pretty=format:"+format+recordSeparator)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, err
	}

	var entries []LogEntry
	logs := strings.Split(string(output), recordSeparator)

	for _, log := range logs {
		log = strings.TrimSpace(log)
		if log == "" {
			continue
		}
		parts := strings.SplitN(log, fieldSeparator, 5)
		if len(parts) != 5 {
			continue // Skip malformed lines
		}

		subject := parts[3]
		body := strings.TrimSpace(parts[4])

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

// GetCommitDiff retrieves the diff for a specific commit hash.
func GetCommitDiff(hash string) (string, error) {
	// Security: Basic validation to prevent command injection.
	if !isSafeHash(hash) {
		return "", fmt.Errorf("invalid commit hash: %s", hash)
	}

	cmd := exec.Command("git", "show", hash)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git show failed for %s: %w\n%s", hash, err, string(output))
	}
	return string(output), nil
}

// isSafeHash checks if a string looks like a valid git hash (hexadecimal).
// This is a simple check to prevent arbitrary command execution.
func isSafeHash(hash string) bool {
	if len(hash) == 0 || len(hash) > 40 {
		return false
	}
	for _, r := range hash {
		if !((r >= '0' && r <= '9') || (r >= 'a' && r <= 'f') || (r >= 'A' && r <= 'F')) {
			return false
		}
	}
	return true
}
