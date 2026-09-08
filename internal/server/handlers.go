package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/sokinpui/coder/internal/commands"
	"github.com/sokinpui/coder/internal/rpc"
	"github.com/sokinpui/coder/internal/session"
	"github.com/sokinpui/coder/internal/token"
	"github.com/sokinpui/coder/internal/types"
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

	s.mu.Lock()
	s.session = sess
	s.mu.Unlock()

	_ = s.session.LoadContext()
	s.sendResult(req.ID, map[string]any{
		"sessionId":    s.session.ID,
		"mode":         s.session.GetMode(),
		"contextFiles": s.session.GetContextFiles(),
		"model":        s.cfg.Generation.ModelCode,
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

	s.mu.Lock()
	ctx, cancel := context.WithCancel(context.Background())
	s.cancelFunc = cancel
	s.session.SetCancelGeneration(cancel)
	s.mu.Unlock()

	s.sendResult(req.ID, map[string]any{"status": "accepted"})

	event := s.session.HandleInput(params.Content)
	if event.Type != types.GenerationStarted {
		s.sendNotification("session/event", map[string]any{
			"type":    event.Type,
			"payload": event.Data,
		})
		s.sendNotification("session/chunk", rpc.StreamChunkNotification{Done: true})
		return
	}

	streamChan, ok := event.Data.(chan types.StreamChunk)
	if !ok {
		s.sendNotification("session/chunk", rpc.StreamChunkNotification{Done: true})
		return
	}

	for chunk := range streamChan {
		s.sendNotification("session/chunk", rpc.StreamChunkNotification{
			Content:          chunk.Content,
			ReasoningContent: chunk.ReasoningContent,
			Done:             false,
		})
	}

	if ctx.Err() == context.Canceled {
		s.sendNotification("session/chunk", rpc.StreamChunkNotification{Done: true, Error: "Generation cancelled"})
		return
	}

	s.sendNotification("session/chunk", rpc.StreamChunkNotification{Done: true})
	_ = s.session.SaveConversation()
}

func (s *Server) handleCancel(req rpc.Request) {
	s.mu.Lock()
	if s.cancelFunc != nil {
		s.cancelFunc()
		s.cancelFunc = nil
	}
	s.mu.Unlock()
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

	s.sendResult(req.ID, map[string]any{
		"mode":         s.session.GetMode(),
		"title":        s.session.GetTitle(),
		"contextFiles": s.session.GetContextFiles(),
		"tokenCount":   tokenCount,
		"messages":     s.session.GetMessages(),
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
	if s.cfg.Server.APIKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+s.cfg.Server.APIKey)
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

	s.sendResult(req.ID, map[string]any{
		"loaded":       true,
		"title":        s.session.GetTitle(),
		"contextFiles": s.session.GetContextFiles(),
		"messages":     s.session.GetMessages(),
	})
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
	s.cfg = s.session.GetConfig()
	s.sendResult(req.ID, map[string]any{"reloaded": true, "config": s.cfg})
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
