package server

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/sokinpui/coder/internal/engine/token"
)

func (s *Server) handleTokens(req Request) {
	if err := s.ensureSession(); err != nil {
		s.sendError(req.ID, -32603, err.Error())
		return
	}
	s.sendResult(req.ID, map[string]any{"tokenCount": token.CountTokens(s.session.GetPrompt())})
}

func (s *Server) handleModelsList(req Request) {
	if len(s.cfg.AvailableModels) > 0 {
		s.sendResult(req.ID, map[string]any{
			"models":  s.cfg.AvailableModels,
			"current": s.cfg.Coder.ModelCode,
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
			"models":  []string{s.cfg.Coder.ModelCode},
			"current": s.cfg.Coder.ModelCode,
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
			"models":  []string{s.cfg.Coder.ModelCode},
			"current": s.cfg.Coder.ModelCode,
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
		"current": s.cfg.Coder.ModelCode,
	})
}

func (s *Server) handleHistoryList(req Request) {
	if err := s.ensureSession(); err != nil {
		s.sendError(req.ID, -32603, err.Error())
		return
	}

	items, err := s.session.GetHistoryManager().ListConversationsByMode("coder")
	if err != nil {
		s.sendError(req.ID, -32603, err.Error())
		return
	}
	s.sendResult(req.ID, items)
}

func (s *Server) handleHistoryLoad(req Request) {
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

func (s *Server) handleConfigReload(req Request) {
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
