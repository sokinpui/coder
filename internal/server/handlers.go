package server

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/sokinpui/coder/internal/commands"
	"github.com/sokinpui/coder/internal/rpc"
	"github.com/sokinpui/coder/internal/session"
	"github.com/sokinpui/coder/internal/token"
	"github.com/sokinpui/coder/internal/types"
	"github.com/sokinpui/coder/internal/utils"
)

func (s *Server) handleInit(req rpc.Request) {
	var params rpc.InitParams
	if len(req.Params) > 0 {
		_ = json.Unmarshal(req.Params, &params)
	}

	if params.Model != "" {
		s.cfg.Generation.ModelCode = params.Model
	}

	sess, err := session.New(s.cfg, params.Mode, params.Instruction, params.ContextFiles)
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
		"model":            sess.GetConfig().Generation.ModelCode,
		"tokenCount":       tokenCount,
	})
}

func (s *Server) handlePrompt(req rpc.Request) {
	var params rpc.SendPromptParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		s.sendError(req.ID, -32602, "Invalid params")
		return
	}

	if err := s.ensureSession(); err != nil {
		s.sendError(req.ID, -32603, err.Error())
		return
	}

	s.sendResult(req.ID, map[string]any{"status": "accepted"})

	repoRoot := utils.GetProjectRoot()
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
		s.sendNotification("session/chunk", rpc.StreamChunkNotification{Done: true, TokenCount: tokenCount})
		return
	}

	streamChan, ok := event.Data.(chan types.StreamChunk)
	if !ok {
		s.sendNotification("session/chunk", rpc.StreamChunkNotification{Done: true})
		return
	}

	s.streamToClient(streamChan)
}

func (s *Server) streamToClient(streamChan chan types.StreamChunk) {
	for chunk := range streamChan {
		if chunk.Content != "" {
			msgs := s.session.GetMessages()
			if len(msgs) > 0 && msgs[len(msgs)-1].Type == types.AIMessage {
				msgs[len(msgs)-1].Content += chunk.Content
			}
		}
		s.sendNotification("session/chunk", rpc.StreamChunkNotification{
			Content:          chunk.Content,
			ReasoningContent: chunk.ReasoningContent,
			Done:             false,
		})
	}

	if !s.session.IsStreaming() {
		s.sendNotification("session/chunk", rpc.StreamChunkNotification{Done: true, Error: "Generation cancelled"})
		return
	}

	s.session.SetStreaming(false)
	_ = s.session.SaveConversation()
	tokenCount := token.CountTokens(s.session.GetPrompt())
	s.sendNotification("session/chunk", rpc.StreamChunkNotification{
		Done:       true,
		TokenCount: tokenCount,
	})
}

func (s *Server) handleMessageDelete(req rpc.Request) {
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

func (s *Server) handleRegenerate(req rpc.Request) {
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
		s.sendNotification("session/chunk", rpc.StreamChunkNotification{Done: true, TokenCount: tokenCount})
		return
	}
	streamChan, ok := event.Data.(chan types.StreamChunk)
	if !ok {
		s.sendNotification("session/chunk", rpc.StreamChunkNotification{Done: true})
		return
	}
	s.streamToClient(streamChan)
}

func (s *Server) handleBranch(req rpc.Request) {
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
	newSess, err := s.session.Branch(idx)
	if err != nil {
		s.sendError(req.ID, -32603, fmt.Sprintf("Failed to branch session: %v", err))
		return
	}

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

func (s *Server) handleMessageEdit(req rpc.Request) {
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

func (s *Server) handleTokens(req rpc.Request) {
	if err := s.ensureSession(); err != nil {
		s.sendError(req.ID, -32603, err.Error())
		return
	}
	s.sendResult(req.ID, map[string]any{"tokenCount": token.CountTokens(s.session.GetPrompt())})
}

func (s *Server) handlePDFAdd(req rpc.Request) {
	var params rpc.AddPDFParams
	if err := json.Unmarshal(req.Params, &params); err != nil || params.Path == "" {
		s.sendError(req.ID, -32602, "Path is required")
		return
	}
	if err := s.ensureSession(); err != nil {
		s.sendError(req.ID, -32603, err.Error())
		return
	}

	docEntry := filepath.ToSlash(params.Path)
	s.session.SetContextDocuments(commands.AppendUnique(s.session.GetContextDocuments(), []string{docEntry}))
	if err := s.session.LoadContext(); err != nil {
		s.sendError(req.ID, -32603, fmt.Sprintf("Failed to load PDF document: %v", err))
		return
	}
	_ = s.session.SaveConversation()
	s.sendResult(req.ID, map[string]any{
		"success":    true,
		"pagesAdded": s.session.GetDocumentPageCount(docEntry),
		"tokenCount": token.CountTokens(s.session.GetPrompt()),
	})
}

func (s *Server) handleCancel(req rpc.Request) {
	if s.session != nil {
		s.session.CancelGeneration()
	}
	s.sendResult(req.ID, map[string]any{"cancelled": true})
}

func (s *Server) handleContextAdd(req rpc.Request) {
	var params rpc.ContextModifyParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		s.sendError(req.ID, -32602, "Invalid params")
		return
	}
	if err := s.ensureSession(); err != nil {
		s.sendError(req.ID, -32603, err.Error())
		return
	}

	res, _, _ := commands.ProcessCommand("/file "+joinArgs(params.Paths), s.session)
	s.sendResult(req.ID, map[string]any{
		"message":      res.Payload,
		"contextFiles": s.session.GetContextFiles(),
	})
}

func (s *Server) handleContextExclude(req rpc.Request) {
	var params rpc.ContextModifyParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		s.sendError(req.ID, -32602, "Invalid params")
		return
	}
	if err := s.ensureSession(); err != nil {
		s.sendError(req.ID, -32603, err.Error())
		return
	}

	res, _, _ := commands.ProcessCommand("/exclude "+joinArgs(params.Paths), s.session)
	s.sendResult(req.ID, map[string]any{
		"message":      res.Payload,
		"contextFiles": s.session.GetContextFiles(),
	})
}

func (s *Server) handleContextGet(req rpc.Request) {
	if err := s.ensureSession(); err != nil {
		s.sendError(req.ID, -32603, err.Error())
		return
	}

	promptMsgs := s.session.GetPrompt()
	tokenCount := token.CountTokens(promptMsgs)

	s.hydrateSessionImages()
	s.sendResult(req.ID, map[string]any{
		"mode":             s.session.GetMode(),
		"title":            s.session.GetTitle(),
		"contextFiles":     s.session.GetContextFiles(),
		"contextDocuments": s.session.GetContextDocuments(),
		"tokenCount":       tokenCount,
		"messages":         s.session.GetMessages(),
	})
}

func (s *Server) handleModelSet(req rpc.Request) {
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
		"model": s.session.GetConfig().Generation.ModelCode,
	})
}

func (s *Server) handleSessionRename(req rpc.Request) {
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

func (s *Server) handleItfApply(req rpc.Request) {
	var params rpc.ApplyItfParams
	if len(req.Params) > 0 {
		_ = json.Unmarshal(req.Params, &params)
	}

	if err := s.ensureSession(); err != nil {
		s.sendError(req.ID, -32603, err.Error())
		return
	}

	if params.Content == "" {
		cmdOut, _, _ := commands.ProcessCommand("/itf "+params.Args, s.session)
		s.sendResult(req.ID, map[string]any{"summary": cmdOut.Payload})
		return
	}

	res := commands.ExecuteItf(params.Content, params.Args)
	s.sendResult(req.ID, map[string]any{
		"success":       res.Success,
		"summary":       res.Summary,
		"affectedFiles": res.AffectedFiles,
		"raw":           res.Raw,
	})
}

func (s *Server) handleItfUndo(req rpc.Request) {
	if err := s.ensureSession(); err != nil {
		s.sendError(req.ID, -32603, err.Error())
		return
	}

	cmdOut, _, success := commands.ProcessCommand("/undo", s.session)
	s.sendResult(req.ID, map[string]any{
		"success": success,
		"summary": cmdOut.Payload,
	})
}

func (s *Server) handleModelsList(req rpc.Request) {
	if len(s.cfg.AvailableModels) > 0 {
		s.sendResult(req.ID, map[string]any{
			"models":  s.cfg.AvailableModels,
			"current": s.cfg.Generation.ModelCode,
		})
		return
	}

	endpoint := strings.TrimSuffix(s.cfg.Server.URL, "/") + "/models"
	httpReq, err := http.NewRequestWithContext(context.Background(), "GET", endpoint, nil)
	if err != nil {
		s.sendError(req.ID, -32603, err.Error())
		return
	}
	apiKey := s.cfg.Server.APIKey
	if apiKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+apiKey)
	}

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil || resp.StatusCode != http.StatusOK {
		s.sendResult(req.ID, map[string]any{
			"models":  []string{s.cfg.Generation.ModelCode},
			"current": s.cfg.Generation.ModelCode,
		})
		return
	}
	defer resp.Body.Close()

	type openAIModel struct {
		ID string `json:"id"`
	}
	type openAIModelList struct {
		Data []openAIModel `json:"data"`
	}

	var result openAIModelList
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		s.sendResult(req.ID, map[string]any{
			"models":  []string{s.cfg.Generation.ModelCode},
			"current": s.cfg.Generation.ModelCode,
		})
		return
	}

	modelIDs := make([]string, len(result.Data))
	for i, m := range result.Data {
		modelIDs[i] = m.ID
	}

	s.cfg.AvailableModels = modelIDs

	s.sendResult(req.ID, map[string]any{
		"models":  modelIDs,
		"current": s.cfg.Generation.ModelCode,
	})
}

func (s *Server) handleHistoryList(req rpc.Request) {
	if err := s.ensureSession(); err != nil {
		s.sendError(req.ID, -32603, err.Error())
		return
	}

	items, err := s.session.GetHistoryManager().ListConversations()
	if err != nil {
		s.sendError(req.ID, -32603, err.Error())
		return
	}
	s.sendResult(req.ID, items)
}

func (s *Server) handleHistoryLoad(req rpc.Request) {
	var params struct {
		Filename string `json:"filename"`
	}
	if err := json.Unmarshal(req.Params, &params); err != nil || params.Filename == "" {
		s.sendError(req.ID, -32602, "Filename is required")
		return
	}
	if err := s.ensureSession(); err != nil {
		s.sendError(req.ID, -32603, err.Error())
		return
	}

	if err := s.session.LoadConversation(params.Filename); err != nil {
		s.sendError(req.ID, -32603, err.Error())
		return
	}

	s.hydrateSessionImages()
	tokenCount := token.CountTokens(s.session.GetPrompt())
	s.sendResult(req.ID, map[string]any{
		"loaded":           true,
		"title":            s.session.GetTitle(),
		"contextFiles":     s.session.GetContextFiles(),
		"contextDocuments": s.session.GetContextDocuments(),
		"messages":         s.session.GetMessages(),
		"tokenCount":       tokenCount,
	})
}

func (s *Server) hydrateSessionImages() {
	if s.session == nil {
		return
	}
	repoRoot := utils.GetProjectRoot()
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

func (s *Server) handleConfigReload(req rpc.Request) {
	if err := s.ensureSession(); err != nil {
		s.sendError(req.ID, -32603, err.Error())
		return
	}

	if err := s.session.ReloadConfig(); err != nil {
		s.sendError(req.ID, -32603, err.Error())
		return
	}
	newCfg := s.session.GetConfig()
	s.cfg = newCfg
	s.sendResult(req.ID, map[string]any{"reloaded": true, "config": newCfg})
}

func joinArgs(args []string) string {
	var res string
	for _, a := range args {
		if res != "" {
			res += " "
		}
		res += a
	}
	return res
}
