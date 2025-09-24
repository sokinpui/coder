package server

import (
	"coder/internal/git"
	"fmt"
	"log"
)

// handleGetGitGraphLog retrieves and sends the git graph log.
func (c *Client) handleGetGitGraphLog() {
	logEntries, err := git.GetGraphLog()
	if err != nil {
		log.Printf("Error getting git log: %v", err)
		c.send <- ServerToClientMessage{
			Type:    "error",
			Payload: "Failed to get git graph log.",
		}
		return
	}
	c.send <- ServerToClientMessage{
		Type:    "gitGraphLog",
		Payload: logEntries,
	}
}

// handleGetCommitDiff retrieves and sends the diff for a specific commit hash.
func (c *Client) handleGetCommitDiff(hash string) {
	diff, err := git.GetCommitDiff(hash)
	if err != nil {
		log.Printf("Error getting commit diff for %s: %v", hash, err)
		c.send <- ServerToClientMessage{
			Type:    "error",
			Payload: fmt.Sprintf("Failed to get diff for commit: %s", hash),
		}
		return
	}
	c.send <- ServerToClientMessage{
		Type: "commitDiff",
		Payload: map[string]string{
			"hash": hash,
			"diff": diff,
		},
	}
}
