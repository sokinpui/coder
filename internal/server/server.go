package server

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"sync"

	"github.com/sokinpui/coder/internal/commands"
	"github.com/sokinpui/coder/internal/config"
	"github.com/sokinpui/coder/internal/rpc"
	"github.com/sokinpui/coder/internal/session"
	"github.com/sokinpui/coder/internal/token"
	"github.com/sokinpui/coder/internal/types"
	"github.com/sokinpui/coder/internal/utils"
)

type Server struct {
	cfg        *config.Config
	session    *session.Session
	mu         sync.Mutex
	writerMu   sync.Mutex
	writer     io.Writer
	cancelFunc context.CancelFunc
}

func New(cfg *config.Config) *Server {
	return &Server{
		cfg: cfg,
	}
}

func (s *Server) ServeStdio() error {
	s.writer = os.Stdout
	return s.handleReader(os.Stdin)
}

func (s *Server) ServeListener(listener net.Listener) error {
	for {
		conn, err := listener.Accept()
		if err != nil {
			return err
		}
		go func(c net.Conn) {
			defer c.Close()
			srv := New(s.cfg)
			srv.writer = c
			_ = srv.handleReader(c)
		}(conn)
	}
}

func (s *Server) handleReader(reader io.Reader) error {
	scanner := bufio.NewScanner(reader)
	// Buffer size up to 10MB for large code snippets
	buf := make([]byte, 1024*1024)
	scanner.Buffer(buf, 10*1024*1024)

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var req rpc.Request
		if err := json.Unmarshal(line, &req); err != nil {
			s.sendError(nil, -32700, fmt.Sprintf("Parse error: %v", err))
			continue
		}

		go s.dispatch(req)
	}
	return scanner.Err()
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
	case "session/itf/apply":
		s.handleItfApply(req)
	case "session/itf/undo":
		s.handleItfUndo(req)
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
	s.session = sess
	return s.session.LoadContext()
}

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
	} else {
		s.sendNotification("session/chunk", rpc.StreamChunkNotification{Done: true})
		_ = s.session.SaveConversation()
	}
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

func (s *Server) handleItfApply(req rpc.Request) {
	var params rpc.ApplyItfParams
	if len(req.Params) > 0 {
		_ = json.Unmarshal(req.Params, &params)
	}

	if err := s.ensureSession(); err != nil {
		s.sendError(req.ID, -32603, err.Error())
		return
	}

	var res commands.ItfResult
	if params.Content != "" {
		res = commands.ExecuteItf(params.Content, params.Args)
	} else {
		cmdOut, _, _ := commands.ProcessCommand("/itf "+params.Args, s.session)
		s.sendResult(req.ID, map[string]any{"summary": cmdOut.Payload})
		return
	}

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
	s.writerMu.Lock()
	defer s.writerMu.Unlock()

	data, err := json.Marshal(v)
	if err != nil {
		return
	}
	data = append(data, '\n')
	_, _ = s.writer.Write(data)
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
