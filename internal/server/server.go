package server

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"sync"

	"github.com/sokinpui/coder/internal/config"
	"github.com/sokinpui/coder/internal/engine"
	"github.com/sokinpui/coder/internal/engine/coder"
	"github.com/sokinpui/coder/pkg/version"
)

type Server struct {
	cfg      *config.Config
	session  *coder.Session
	writerMu sync.Mutex
	writer   io.Writer
}

func New(cfg *config.Config) *Server {
	return &Server{
		cfg: cfg,
	}
}

func (s *Server) dispatch(req Request) {
	log.Printf("[req] method=%s id=%s", req.Method, formatRawID(req.ID))
	switch req.Method {
	case "ping":
		s.sendResult(req.ID, map[string]any{"pong": true, "version": version.Get()})
	case "session/init":
		s.handleInit(req)
	case "session/prompt":
		s.handlePrompt(req)
	case "session/compact":
		s.handleCompact(req)
	case "session/cancel":
		s.handleCancel(req)
	case "session/context/add":
		s.handleContextAdd(req)
	case "session/pdf/add":
		s.handlePDFAdd(req)
	case "session/context/exclude":
		s.handleContextExclude(req)
	case "session/context/get":
		s.handleContextGet(req)
	case "session/model":
		s.handleModelSet(req)
	case "session/reasoning":
		s.handleReasoningSet(req)
	case "session/rename":
		s.handleSessionRename(req)
	case "session/message/delete":
		s.handleMessageDelete(req)
	case "session/regenerate":
		s.handleRegenerate(req)
	case "session/branch":
		s.handleBranch(req)
	case "session/message/edit":
		s.handleMessageEdit(req)
	case "session/tokens":
		s.handleTokens(req)
	case "session/itf/apply":
		s.handleItfApply(req)
	case "session/itf/undo":
		s.handleItfUndo(req)
	case "models/list":
		s.handleModelsList(req)
	case "history/list":
		s.handleHistoryList(req)
	case "history/load":
		s.handleHistoryLoad(req)
	case "config/get":
		s.sendResult(req.ID, s.cfg)
	case "config/reload":
		s.handleConfigReload(req)
	default:
		s.sendError(req.ID, -32601, fmt.Sprintf("Method not found: %s", req.Method))
	}
}

func (s *Server) ensureSession() error {
	if s.session != nil {
		return nil
	}

	sess, err := coder.New(s.cfg, engine.ModeCoder, "", nil)
	if err != nil {
		return err
	}
	if err := sess.LoadContext(); err != nil {
		return err
	}

	s.session = sess
	return nil
}

func (s *Server) sendResult(id *json.RawMessage, result any) {
	log.Printf("[resp] id=%s status=ok", formatRawID(id))
	s.send(Response{
		JSONRPC: "2.0",
		ID:      id,
		Result:  result,
	})
}

func (s *Server) sendError(id *json.RawMessage, code int, msg string) {
	log.Printf("[resp] id=%s code=%d error=%s", formatRawID(id), code, msg)
	s.send(Response{
		JSONRPC: "2.0",
		ID:      id,
		Error:   &ResponseError{Code: code, Message: msg},
	})
}

func (s *Server) sendNotification(method string, params any) {
	s.logNotification(method, params)
	s.send(Notification{
		JSONRPC: "2.0",
		Method:  method,
		Params:  params,
	})
}

func (s *Server) logNotification(method string, params any) {
	switch method {
	case "session/chunk":
		chunk, ok := params.(StreamChunkNotification)
		if !ok {
			log.Printf("[event] %s", method)
			return
		}
		if chunk.Error != "" {
			log.Printf("[event] session/chunk done=true error=%s", chunk.Error)
			return
		}
		if chunk.Done {
			log.Printf("[event] session/chunk done=true tokens=%d", chunk.TokenCount)
			return
		}
		if chunk.ReasoningContent != "" {
			log.Printf("[event] session/chunk reasoning delta bytes=%d", len(chunk.ReasoningContent))
			return
		}
		log.Printf("[event] session/chunk content delta bytes=%d", len(chunk.Content))
	case "session/event":
		if evMap, ok := params.(map[string]any); ok {
			log.Printf("[event] session/event type=%v", evMap["type"])
			return
		}
		log.Printf("[event] session/event")
	default:
		log.Printf("[event] %s", method)
	}
}

func formatRawID(id *json.RawMessage) string {
	if id == nil {
		return "null"
	}
	return string(*id)
}

func (s *Server) send(v any) {
	data, err := json.Marshal(v)
	if err != nil {
		return
	}
	data = append(data, '\n')

	s.writerMu.Lock()
	defer s.writerMu.Unlock()
	if s.writer == nil {
		return
	}
	_, _ = s.writer.Write(data)
}
