package server

import (
	"coder/internal/files"
	"fmt"
	"log"
)

// handleGetSourceTree retrieves and sends the project's file tree.
func (c *Client) handleGetSourceTree() {
	tree, err := files.GetFileTree()
	if err != nil {
		log.Printf("Error getting source tree: %v", err)
		c.send <- ServerToClientMessage{
			Type:    "error",
			Payload: "Failed to get project file tree.",
		}
		return
	}
	c.send <- ServerToClientMessage{
		Type:    "sourceTree",
		Payload: tree,
	}
}

// handleGetFileContent retrieves and sends the content of a specific file.
func (c *Client) handleGetFileContent(path string) {
	content, err := files.GetFileContent(path)
	if err != nil {
		log.Printf("Error getting file content for %s: %v", path, err)
		c.send <- ServerToClientMessage{
			Type:    "error",
			Payload: fmt.Sprintf("Failed to get content for file: %s", path),
		}
		return
	}
	c.send <- ServerToClientMessage{
		Type: "fileContent",
		Payload: map[string]string{
			"path":    path,
			"content": content,
		},
	}
}
