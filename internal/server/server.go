package server

import (
	"encoding/json"
	"fmt"
	"io"
	"sync"

	"github.com/sokinpui/coder/internal/config"
	"github.com/sokinpui/coder/internal/rpc"
	"github.com/sokinpui/coder/internal/session"
	"github.com/sokinpui/coder/internal/utils"
)

type Server struct {
	cfg      *config.Config
	session  *session.Session
	writerMu sync.Mutex
	writer   io.Writer
}

func New(cfg *config.Config) *Server {
	return &Server{
		cfg: cfg,
	}
}

func (s *Server) dispatch(req rpc.Request) {
	switch req.Method {
	case "ping":
		s.sendResult(req.ID, map[string]any{"pong": true, "version": utils.GetVersion()})
	case "session/init":
		s.handleInit(req)
	case "session/prompt":
		s.handlePrompt(req)
	case "session/cancel":
		s.handleCancel(req)
	case "session/context/add":
		s.handleContextAdd(req)
	case "session/context/exclude":
		s.handleContextExclude(req)
	case "session/context/get":
		s.handleContextGet(req)
	case "session/model":
		s.handleModelSet(req)
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

	sess, err := session.New(s.cfg, session.ModeCoding, "", nil)
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
	s.send(rpc.Response{
		JSONRPC: "2.0",
		ID:      id,
		Result:  result,
	})
}

func (s *Server) sendError(id *json.RawMessage, code int, msg string) {
	s.send(rpc.Response{
		JSONRPC: "2.0",
		ID:      id,
		Error:   &rpc.ResponseError{Code: code, Message: msg},
	})
}

func (s *Server) sendNotification(method string, params any) {
	s.send(rpc.Notification{
		JSONRPC: "2.0",
		Method:  method,
		Params:  params,
	})
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
