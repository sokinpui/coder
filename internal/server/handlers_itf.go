package server

import (
	"encoding/json"
	"log"

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
		log.Printf("itf apply requested with args=%s", params.Args)
		cmdOut, _, _ := commands.ProcessCommand("/itf "+params.Args, s.session)
		s.sendResult(req.ID, map[string]any{"summary": cmdOut.Payload})
		return
	}

	log.Printf("itf apply custom content requested with args=%s", params.Args)
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

	log.Printf("itf undo requested")
	cmdOut, _, success := commands.ProcessCommand("/undo", s.session)
	s.sendResult(req.ID, map[string]any{
		"success": success,
		"summary": cmdOut.Payload,
	})
}
