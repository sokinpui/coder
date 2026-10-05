package server

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/sokinpui/coder/internal/engine/coder"
	"github.com/sokinpui/coder/internal/engine/token"
	"github.com/sokinpui/coder/internal/project"
	"github.com/sokinpui/coder/internal/types"
)

func (s *Server) handleInit(req Request) {
	var params InitParams
	if len(req.Params) > 0 {
		_ = json.Unmarshal(req.Params, &params)
	}

	if params.Model != "" {
		s.cfg.Coder.ModelCode = params.Model
	}

	sess, err := coder.New(s.cfg, params.Mode, params.Instruction, params.ContextFiles)
	if err != nil {
		s.sendError(req.ID, -32603, fmt.Sprintf("Failed to initialize session: %v", err))
		return
	}

	s.session = sess

	_ = sess.LoadContext()
	tokenCount := token.CountTokens(sess.GetPrompt())
	s.sendResult(req.ID, map[string]any{
		"sessionId":        sess.ID,
		"mode":             sess.GetMode(),
		"contextFiles":     sess.GetContextFiles(),
		"contextDocuments": sess.GetContextDocuments(),
		"model":            sess.GetConfig().Coder.ModelCode,
		"tokenCount":       tokenCount,
	})
}

func (s *Server) handlePrompt(req Request) {
	var params SendPromptParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		s.sendError(req.ID, -32602, "Invalid params")
		return
	}

	if err := s.ensureSession(); err != nil {
		s.sendError(req.ID, -32603, err.Error())
		return
	}

	s.sendResult(req.ID, map[string]any{"status": "accepted"})

	repoRoot := project.Root()
	imagesDir := filepath.Join(repoRoot, ".coder", "images")
	for _, imgB64 := range params.Images {
		if commaIdx := strings.Index(imgB64, ","); commaIdx != -1 {
			imgB64 = imgB64[commaIdx+1:]
		}
		data, err := base64.StdEncoding.DecodeString(imgB64)
		if err != nil || len(data) == 0 {
			continue
		}
		_ = os.MkdirAll(imagesDir, 0755)
		filename := fmt.Sprintf("%d.png", time.Now().UnixNano())
		filePath := filepath.Join(imagesDir, filename)
		if err := os.WriteFile(filePath, data, 0644); err != nil {
			continue
		}
		relPath, err := filepath.Rel(repoRoot, filePath)
		if err != nil {
			relPath = filePath
		}
		s.session.AddMessages(types.Message{
			Type:    types.ImageMessage,
			Content: filepath.ToSlash(relPath),
			Data:    data,
		})
	}

	promptContent := params.Content
	if strings.TrimSpace(promptContent) == "" && len(params.Images) > 0 {
		promptContent = "What is in this image?"
	}

	if !s.session.IsTitleGenerated() {
		go func(promptText string) {
			title := s.session.GenerateTitle(context.Background(), promptText)
			_ = s.session.SaveConversation()
			s.sendNotification("session/event", map[string]any{"type": "title", "title": title})
		}(promptContent)
	}

	event := s.session.HandleInput(promptContent)
	if event.Type != types.GenerationStarted {
		s.sendNotification("session/event", map[string]any{
			"type":    event.Type,
			"payload": event.Data,
		})
		tokenCount := token.CountTokens(s.session.GetPrompt())
		s.sendNotification("session/chunk", StreamChunkNotification{Done: true, TokenCount: tokenCount})
		return
	}

	streamChan, ok := event.Data.(chan types.StreamChunk)
	if !ok {
		s.sendNotification("session/chunk", StreamChunkNotification{Done: true})
		return
	}

	s.streamToClient(streamChan)
}

func (s *Server) streamToClient(streamChan chan types.StreamChunk) {
	for chunk := range streamChan {
		if chunk.Error != nil {
			s.sendNotification("session/chunk", StreamChunkNotification{
				Error: chunk.Error.Error(),
				Done:  true,
			})
			return
		}
		if chunk.Content != "" {
			msgs := s.session.GetMessages()
			if len(msgs) > 0 && msgs[len(msgs)-1].Type == types.AIMessage {
				msgs[len(msgs)-1].Content += chunk.Content
			}
		}
		s.sendNotification("session/chunk", StreamChunkNotification{
			Content:          chunk.Content,
			ReasoningContent: chunk.ReasoningContent,
			Done:             false,
		})
	}

	msgs := s.session.GetMessages()
	if len(msgs) > 0 && msgs[len(msgs)-1].Type == types.AIMessage && msgs[len(msgs)-1].Content == "" {
		s.session.DeleteMessages([]int{len(msgs) - 1})
	}

	if !s.session.IsStreaming() {
		s.sendNotification("session/chunk", StreamChunkNotification{Done: true, Error: "Generation cancelled"})
		return
	}

	s.session.SetStreaming(false)
	_ = s.session.SaveConversation()
	tokenCount := token.CountTokens(s.session.GetPrompt())
	s.sendNotification("session/chunk", StreamChunkNotification{
		Done:       true,
		TokenCount: tokenCount,
	})
}

func (s *Server) handleMessageDelete(req Request) {
	var params struct {
		Index   *int  `json:"index"`
		Indices []int `json:"indices"`
	}
	if err := json.Unmarshal(req.Params, &params); err != nil {
		s.sendError(req.ID, -32602, "Invalid params")
		return
	}

	if err := s.ensureSession(); err != nil {
		s.sendError(req.ID, -32603, err.Error())
		return
	}

	indices := params.Indices
	if params.Index != nil {
		indices = append(indices, *params.Index)
	}
	if len(indices) == 0 {
		s.sendError(req.ID, -32602, "Index is required")
		return
	}

	s.session.DeleteMessages(indices)
	_ = s.session.SaveConversation()
	s.hydrateSessionImages()
	tokenCount := token.CountTokens(s.session.GetPrompt())
	s.sendResult(req.ID, map[string]any{
		"deleted":    true,
		"messages":   s.session.GetMessages(),
		"tokenCount": tokenCount,
	})
}

func (s *Server) handleRegenerate(req Request) {
	var params struct {
		Index *int `json:"index"`
	}
	if err := json.Unmarshal(req.Params, &params); err != nil || params.Index == nil {
		s.sendError(req.ID, -32602, "Index is required")
		return
	}

	if err := s.ensureSession(); err != nil {
		s.sendError(req.ID, -32603, err.Error())
		return
	}

	idx := *params.Index
	msgs := s.session.GetMessages()
	if idx < 0 || idx >= len(msgs) {
		s.sendError(req.ID, -32602, "Index out of range")
		return
	}

	if msgs[idx].Type == types.AIMessage {
		found := false
		for i := idx - 1; i >= 0; i-- {
			if msgs[i].Type.IsRegeneratable() {
				idx = i
				found = true
				break
			}
		}
		if !found {
			s.sendError(req.ID, -32602, "No regeneratable prompt found before this message")
			return
		}
	}

	s.sendResult(req.ID, map[string]any{"status": "accepted"})

	event := s.session.RegenerateFrom(idx)
	if event.Type != types.GenerationStarted {
		s.sendNotification("session/event", map[string]any{"type": event.Type, "payload": event.Data})
		tokenCount := token.CountTokens(s.session.GetPrompt())
		s.sendNotification("session/chunk", StreamChunkNotification{Done: true, TokenCount: tokenCount})
		return
	}
	streamChan, ok := event.Data.(chan types.StreamChunk)
	if !ok {
		s.sendNotification("session/chunk", StreamChunkNotification{Done: true})
		return
	}
	s.streamToClient(streamChan)
}

func (s *Server) handleBranch(req Request) {
	var params struct {
		Index *int `json:"index"`
	}
	if err := json.Unmarshal(req.Params, &params); err != nil || params.Index == nil {
		s.sendError(req.ID, -32602, "Index is required")
		return
	}

	if err := s.ensureSession(); err != nil {
		s.sendError(req.ID, -32603, err.Error())
		return
	}

	idx := *params.Index
	msgs := s.session.GetMessages()
	if idx < 0 || idx >= len(msgs) {
		s.sendError(req.ID, -32602, "Index out of range")
		return
	}

	_ = s.session.SaveConversation()
	newSessEngine, err := s.session.Branch(idx)
	if err != nil {
		s.sendError(req.ID, -32603, fmt.Sprintf("Failed to branch session: %v", err))
		return
	}

	newSess := newSessEngine.(*coder.Session)
	s.session = newSess
	_ = s.session.SaveConversation()
	s.hydrateSessionImages()

	tokenCount := token.CountTokens(s.session.GetPrompt())
	s.sendResult(req.ID, map[string]any{
		"branched":         true,
		"sessionId":        newSess.ID,
		"title":            newSess.GetTitle(),
		"contextFiles":     newSess.GetContextFiles(),
		"contextDocuments": newSess.GetContextDocuments(),
		"messages":         newSess.GetMessages(),
		"tokenCount":       tokenCount,
	})
}

func (s *Server) handleMessageEdit(req Request) {
	var params struct {
		Index   *int   `json:"index"`
		Content string `json:"content"`
	}
	if err := json.Unmarshal(req.Params, &params); err != nil || params.Index == nil {
		s.sendError(req.ID, -32602, "Index is required")
		return
	}

	if err := s.ensureSession(); err != nil {
		s.sendError(req.ID, -32603, err.Error())
		return
	}

	idx := *params.Index
	if err := s.session.EditMessage(idx, params.Content); err != nil {
		s.sendError(req.ID, -32603, err.Error())
		return
	}

	_ = s.session.SaveConversation()
	tokenCount := token.CountTokens(s.session.GetPrompt())
	s.sendResult(req.ID, map[string]any{
		"edited":     true,
		"tokenCount": tokenCount,
	})
}

func (s *Server) handleModelSet(req Request) {
	var params struct {
		Model string `json:"model"`
	}
	if err := json.Unmarshal(req.Params, &params); err != nil || params.Model == "" {
		s.sendError(req.ID, -32602, "Model parameter is required")
		return
	}
	if err := s.ensureSession(); err != nil {
		s.sendError(req.ID, -32603, err.Error())
		return
	}
	s.session.SetModel(params.Model)
	s.sendResult(req.ID, map[string]any{
		"model": s.session.GetConfig().Coder.ModelCode,
	})
}

func (s *Server) handleSessionRename(req Request) {
	var params struct {
		Title string `json:"title"`
	}
	if err := json.Unmarshal(req.Params, &params); err != nil || params.Title == "" {
		s.sendError(req.ID, -32602, "Title parameter is required")
		return
	}
	if err := s.ensureSession(); err != nil {
		s.sendError(req.ID, -32603, err.Error())
		return
	}
	s.session.SetTitle(params.Title)
	s.sendResult(req.ID, map[string]any{
		"title": s.session.GetTitle(),
	})
}

func (s *Server) handleCancel(req Request) {
	if s.session != nil {
		s.session.CancelGeneration()
	}
	s.sendResult(req.ID, map[string]any{"cancelled": true})
}

func (s *Server) hydrateSessionImages() {
	if s.session == nil {
		return
	}
	repoRoot := project.Root()
	msgs := s.session.GetMessages()
	for i := range msgs {
		if msgs[i].Type != types.ImageMessage || len(msgs[i].Data) > 0 || msgs[i].Content == "" {
			continue
		}
		absPath := filepath.Join(repoRoot, msgs[i].Content)
		if data, err := os.ReadFile(absPath); err == nil {
			msgs[i].Data = data
		}
	}
}
