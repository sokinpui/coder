package server

import (
	"encoding/json"

	"github.com/sokinpui/coder/internal/engine/commands"
)

func (s *Server) handleItfApply(req Request) {
	var params ApplyItfParams
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

func (s *Server) handleItfUndo(req Request) {
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
