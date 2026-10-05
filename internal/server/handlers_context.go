package server

import (
	"encoding/json"
	"fmt"
	"path/filepath"

	"github.com/sokinpui/coder/internal/engine/commands"
	"github.com/sokinpui/coder/internal/engine/token"
)

func (s *Server) handlePDFAdd(req Request) {
	var params AddPDFParams
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

func (s *Server) handleContextAdd(req Request) {
	var params ContextModifyParams
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

func (s *Server) handleContextExclude(req Request) {
	var params ContextModifyParams
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

func (s *Server) handleContextGet(req Request) {
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
