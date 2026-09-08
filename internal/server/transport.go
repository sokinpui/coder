package server

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strings"

	"github.com/sokinpui/coder/internal/rpc"
	"golang.org/x/net/websocket"
)

type wsWriter struct {
	ws *websocket.Conn
}

func (w *wsWriter) Write(p []byte) (n int, err error) {
	trimmed := strings.TrimSpace(string(p))
	if trimmed == "" {
		return len(p), nil
	}
	if err := websocket.Message.Send(w.ws, trimmed); err != nil {
		return 0, err
	}
	return len(p), nil
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

func (s *Server) ServeHTTP(listener net.Listener) error {
	server := &http.Server{
		Handler: websocket.Server{
			Handshake: func(cfg *websocket.Config, req *http.Request) error {
				return nil
			},
			Handler: func(ws *websocket.Conn) {
				clientSrv := New(s.cfg)
				clientSrv.writer = &wsWriter{ws: ws}
				clientSrv.handleWebSocket(ws)
			},
		},
	}
	return server.Serve(listener)
}

func (s *Server) handleWebSocket(ws *websocket.Conn) {
	defer ws.Close()
	defer func() {
		s.mu.Lock()
		if s.cancelFunc != nil {
			s.cancelFunc()
			s.cancelFunc = nil
		}
		s.mu.Unlock()
	}()

	for {
		var raw string
		if err := websocket.Message.Receive(ws, &raw); err != nil {
			return
		}

		raw = strings.TrimSpace(raw)
		if raw == "" {
			continue
		}

		var req rpc.Request
		if err := json.Unmarshal([]byte(raw), &req); err == nil {
			go s.dispatch(req)
			continue
		}

		scanner := bufio.NewScanner(strings.NewReader(raw))
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "" {
				continue
			}
			var r rpc.Request
			if err := json.Unmarshal([]byte(line), &r); err != nil {
				s.sendError(nil, -32700, fmt.Sprintf("Parse error: %v", err))
				continue
			}
			go s.dispatch(r)
		}
	}
}

func (s *Server) handleReader(reader io.Reader) error {
	scanner := bufio.NewScanner(reader)
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
