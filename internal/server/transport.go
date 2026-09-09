package server

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/sokinpui/coder/internal/rpc"
	"github.com/sokinpui/coder/internal/utils"
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
	mux := http.NewServeMux()

	wsHandler := websocket.Server{
		Handshake: func(cfg *websocket.Config, req *http.Request) error {
			return nil
		},
		Handler: func(ws *websocket.Conn) {
			clientSrv := New(s.cfg)
			clientSrv.writer = &wsWriter{ws: ws}
			clientSrv.handleWebSocket(ws)
		},
	}

	mux.Handle("/ws", wsHandler)
	mux.HandleFunc("/upload/pdf", s.handleHTTPUploadPDF)
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if strings.ToLower(r.Header.Get("Upgrade")) == "websocket" {
			wsHandler.ServeHTTP(w, r)
			return
		}
		http.NotFound(w, r)
	})

	server := &http.Server{Handler: mux}
	return server.Serve(listener)
}

func (s *Server) handleHTTPUploadPDF(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if err := r.ParseMultipartForm(64 << 20); err != nil {
		http.Error(w, fmt.Sprintf("Failed to parse form: %v", err), http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		file, header, err = r.FormFile("pdf")
	}
	if err != nil {
		http.Error(w, "Missing 'file' in multipart form", http.StatusBadRequest)
		return
	}
	defer file.Close()

	if !strings.EqualFold(filepath.Ext(header.Filename), ".pdf") {
		http.Error(w, "Only .pdf files are accepted", http.StatusBadRequest)
		return
	}

	repoRoot := utils.GetProjectRoot()
	docsDir := filepath.Join(repoRoot, ".coder", "documents")
	if err := os.MkdirAll(docsDir, 0755); err != nil {
		http.Error(w, fmt.Sprintf("Failed to create document dir: %v", err), http.StatusInternalServerError)
		return
	}

	cleanBase := filepath.Base(header.Filename)
	destPath := filepath.Join(docsDir, cleanBase)
	outFile, err := os.Create(destPath)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to save file: %v", err), http.StatusInternalServerError)
		return
	}
	defer outFile.Close()

	n, err := io.Copy(outFile, file)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to write file content: %v", err), http.StatusInternalServerError)
		return
	}

	relPath, err := filepath.Rel(repoRoot, destPath)
	if err != nil {
		relPath = destPath
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"success":  true,
		"path":     filepath.ToSlash(relPath),
		"filename": cleanBase,
		"size":     n,
	})
}

func (s *Server) handleWebSocket(ws *websocket.Conn) {
	defer ws.Close()
	defer func() {
		if s.session != nil {
			s.session.CancelGeneration()
		}
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
