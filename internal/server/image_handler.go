package server

import (
	"coder/internal/core"
	"coder/internal/utils"
	"encoding/base64"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// handleImageUpload processes an uploaded image, saves it to disk, and adds it to the session.
func (c *Client) handleImageUpload(dataURL string) {
	repoRoot, err := utils.FindRepoRoot()
	if err != nil {
		log.Printf("Error finding repo root for image upload: %v", err)
		c.send <- ServerToClientMessage{Type: "error", Payload: "Could not find repository root to save image."}
		return
	}

	imagesDir := filepath.Join(repoRoot, ".coder", "images")
	if err := os.MkdirAll(imagesDir, 0755); err != nil {
		log.Printf("Error creating images directory: %v", err)
		c.send <- ServerToClientMessage{Type: "error", Payload: "Could not create directory to store images."}
		return
	}

	// dataURL format: data:image/png;base64,iVBORw0KGgo...
	parts := strings.SplitN(dataURL, ",", 2)
	if len(parts) != 2 {
		log.Printf("Invalid data URL format")
		c.send <- ServerToClientMessage{Type: "error", Payload: "Invalid image data format."}
		return
	}

	meta, encodedData := parts[0], parts[1]

	var extension string
	if strings.Contains(meta, "image/png") {
		extension = ".png"
	} else if strings.Contains(meta, "image/jpeg") {
		extension = ".jpeg"
	} else if strings.Contains(meta, "image/gif") {
		extension = ".gif"
	} else if strings.Contains(meta, "image/webp") {
		extension = ".webp"
	} else {
		log.Printf("Unsupported image type: %s", meta)
		c.send <- ServerToClientMessage{Type: "error", Payload: fmt.Sprintf("Unsupported image type: %s", meta)}
		return
	}

	imageData, err := base64.StdEncoding.DecodeString(encodedData)
	if err != nil {
		log.Printf("Error decoding base64 image data: %v", err)
		c.send <- ServerToClientMessage{Type: "error", Payload: "Failed to decode image data."}
		return
	}

	filename := fmt.Sprintf("%d%s", time.Now().UnixNano(), extension)
	filePath := filepath.Join(imagesDir, filename)

	if err := os.WriteFile(filePath, imageData, 0644); err != nil {
		log.Printf("Error saving image file: %v", err)
		c.send <- ServerToClientMessage{Type: "error", Payload: "Failed to save image to disk."}
		return
	}

	imgMsg := core.Message{
		Type:    core.ImageMessage,
		Content: filepath.ToSlash(filepath.Join(".coder", "images", filename)), // Store path for model context
		DataURL: dataURL}                                                       // Store base64 for UI rendering
	c.session.AddMessage(imgMsg)

	c.send <- ServerToClientMessage{Type: "messageUpdate", Payload: MessagePayload{Type: messageTypeToString(imgMsg.Type), Content: imgMsg.Content, DataURL: imgMsg.DataURL}}
}
