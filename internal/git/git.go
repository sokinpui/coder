package git

import (
	"fmt"
	"os/exec"
	"strings"
)

// GraphLogEntry represents a commit node for graph visualization.
type GraphLogEntry struct {
	Hash         string   `json:"hash"`
	ParentHashes []string `json:"parentHashes"`
	AuthorName   string   `json:"authorName"`
	RelativeDate string   `json:"relativeDate"`
	Subject      string   `json:"subject"`
	Refs         []string `json:"refs"`
}

// GetGraphLog retrieves a structured git log for graph visualization.
func GetGraphLog() ([]GraphLogEntry, error) {
	const fieldSeparator = "\x1f"
	const recordSeparator = "\x1e"
	const format = "%H" + fieldSeparator + "%P" + fieldSeparator + "%an" + fieldSeparator + "%ar" + fieldSeparator + "%s" + fieldSeparator + "%D"
	cmd := exec.Command("git", "log", "--all", "--pretty=format:"+format+recordSeparator)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, err
	}
	var entries []GraphLogEntry
	logs := strings.Split(string(output), recordSeparator)

	for _, log := range logs {
		log = strings.TrimSpace(log)
		if log == "" {
			continue
		}
		parts := strings.SplitN(log, fieldSeparator, 6)
		if len(parts) != 6 {
			continue // Skip malformed lines
		}

		parentHashes := strings.Fields(parts[1])

		refs := []string{}
		if parts[5] != "" {
			refStr := strings.Trim(parts[5], " ()")
			rawRefs := strings.Split(refStr, ", ")
			for _, r := range rawRefs {
				if r != "" {
					refs = append(refs, r)
				}
			}
		}
		entry := GraphLogEntry{
			Hash:         parts[0],
			ParentHashes: parentHashes,
			AuthorName:   parts[2],
			RelativeDate: parts[3],
			Subject:      parts[4],
			Refs:         refs,
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
