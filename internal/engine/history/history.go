package history

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/sokinpui/coder/internal/project"
	"github.com/sokinpui/coder/internal/types"
	_ "modernc.org/sqlite"
)

const historyDbName = ".coder/history.db"

type Metadata struct {
	Title            string
	Mode             string
	CreatedAt        time.Time
	ModifiedAt       time.Time
	ContextFiles     []string
	ContextDocuments []string
	Exclusions       []string
	WorkingDir       string
}

type ConversationInfo struct {
	ID         string    `json:"id"`
	Filename   string    `json:"filename"`
	Mode       string    `json:"mode,omitempty"`
	Title      string    `json:"title"`
	CreatedAt  time.Time `json:"createdAt"`
	ModifiedAt time.Time `json:"modifiedAt"`
}

type ConversationData struct {
	Filename         string
	Title            string
	Mode             string
	CreatedAt        time.Time
	Messages         []types.Message
	ContextFiles     []string
	ContextDocuments []string
	Exclusions       []string
	WorkingDir       string
}

type Manager struct {
	dbPath string
	db     *sql.DB
	mu     sync.Mutex
}

var (
	dbPoolMu sync.Mutex
	dbPool   = make(map[string]*sql.DB)
)

func getDB(dbPath string) (*sql.DB, error) {
	dbPoolMu.Lock()
	defer dbPoolMu.Unlock()

	if db, ok := dbPool[dbPath]; ok {
		return db, nil
	}

	if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
		return nil, fmt.Errorf("could not create database directory: %w", err)
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open sqlite database: %w", err)
	}

	db.SetMaxOpenConns(1)

	pragmas := []string{
		"PRAGMA journal_mode = WAL;",
		"PRAGMA busy_timeout = 5000;",
		"PRAGMA foreign_keys = ON;",
		"PRAGMA synchronous = NORMAL;",
	}
	for _, pragma := range pragmas {
		if _, err := db.Exec(pragma); err != nil {
			db.Close()
			return nil, fmt.Errorf("failed to apply %s: %w", pragma, err)
		}
	}

	if err := initSchema(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to initialize database schema: %w", err)
	}

	dbPool[dbPath] = db
	return db, nil
}

func initSchema(db *sql.DB) error {
	schema := `
	CREATE TABLE IF NOT EXISTS sessions (
		id TEXT PRIMARY KEY,
		title TEXT NOT NULL,
		mode TEXT NOT NULL,
		created_at TIMESTAMP NOT NULL,
		modified_at TIMESTAMP NOT NULL,
		working_dir TEXT NOT NULL DEFAULT '',
		context_files TEXT NOT NULL DEFAULT '[]',
		context_documents TEXT NOT NULL DEFAULT '[]',
		exclusions TEXT NOT NULL DEFAULT '[]'
	);

	CREATE TABLE IF NOT EXISTS messages (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		session_id TEXT NOT NULL,
		seq INTEGER NOT NULL,
		type INTEGER NOT NULL,
		content TEXT NOT NULL,
		tool_calls TEXT NOT NULL DEFAULT '[]',
		tool_call_id TEXT NOT NULL DEFAULT '',
		FOREIGN KEY (session_id) REFERENCES sessions(id) ON DELETE CASCADE
	);

	CREATE INDEX IF NOT EXISTS idx_messages_session_seq ON messages(session_id, seq);
	CREATE INDEX IF NOT EXISTS idx_sessions_modified_at ON sessions(modified_at DESC);
	CREATE INDEX IF NOT EXISTS idx_sessions_mode ON sessions(mode);
	`
	_, err := db.Exec(schema)
	return err
}

func NewManager() (*Manager, error) {
	repoRoot := project.Root()
	dbPath := filepath.Join(repoRoot, historyDbName)
	db, err := getDB(dbPath)
	if err != nil {
		return nil, err
	}

	return &Manager{
		dbPath: dbPath,
		db:     db,
	}, nil
}

func (m *Manager) SaveConversation(data *ConversationData) error {
	if m == nil || m.db == nil {
		return fmt.Errorf("database not initialized")
	}
	if data == nil {
		return fmt.Errorf("conversation data cannot be nil")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	ctxFilesJSON, err := json.Marshal(data.ContextFiles)
	if err != nil {
		return err
	}
	ctxDocsJSON, err := json.Marshal(data.ContextDocuments)
	if err != nil {
		return err
	}
	exclusionsJSON, err := json.Marshal(data.Exclusions)
	if err != nil {
		return err
	}

	now := time.Now()
	tx, err := m.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	upsertSession := `
	INSERT INTO sessions (id, title, mode, created_at, modified_at, working_dir, context_files, context_documents, exclusions)
	VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	ON CONFLICT(id) DO UPDATE SET
		title = excluded.title,
		mode = excluded.mode,
		modified_at = excluded.modified_at,
		working_dir = excluded.working_dir,
		context_files = excluded.context_files,
		context_documents = excluded.context_documents,
		exclusions = excluded.exclusions;
	`
	_, err = tx.Exec(
		upsertSession,
		data.Filename,
		data.Title,
		data.Mode,
		data.CreatedAt,
		now,
		data.WorkingDir,
		string(ctxFilesJSON),
		string(ctxDocsJSON),
		string(exclusionsJSON),
	)
	if err != nil {
		return err
	}

	if _, err := tx.Exec("DELETE FROM messages WHERE session_id = ?", data.Filename); err != nil {
		return err
	}

	insertMsgStmt, err := tx.Prepare(`
	INSERT INTO messages (session_id, seq, type, content, tool_calls, tool_call_id)
	VALUES (?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return err
	}
	defer insertMsgStmt.Close()

	seq := 0
	for _, msg := range data.Messages {
		if !msg.Type.IsHistory() {
			continue
		}
		if msg.Type == types.InstructionMessage || msg.Type == types.SourceCodeMessage {
			continue
		}

		toolCallsJSON := "[]"
		if len(msg.ToolCalls) > 0 {
			b, err := json.Marshal(msg.ToolCalls)
			if err == nil {
				toolCallsJSON = string(b)
			}
		}

		_, err = insertMsgStmt.Exec(
			data.Filename,
			seq,
			int(msg.Type),
			msg.Content,
			toolCallsJSON,
			msg.ToolCallID,
		)
		if err != nil {
			return err
		}
		seq++
	}

	return tx.Commit()
}

func (m *Manager) LoadConversation(id string) (*Metadata, []types.Message, error) {
	if m == nil || m.db == nil {
		return nil, nil, fmt.Errorf("database not initialized")
	}
	if strings.TrimSpace(id) == "" {
		return nil, nil, fmt.Errorf("session ID is required")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	cleanID := strings.TrimSuffix(id, ".md")

	row := m.db.QueryRow(`
	SELECT id, title, mode, created_at, modified_at, working_dir, context_files, context_documents, exclusions
	FROM sessions
	WHERE id = ? OR id = ? OR id = ?
	LIMIT 1
	`, id, cleanID, cleanID+".md")

	var (
		sessID, title, mode, workingDir, ctxFilesStr, ctxDocsStr, exclusionsStr string
		createdAt, modifiedAt                                                   time.Time
	)

	err := row.Scan(&sessID, &title, &mode, &createdAt, &modifiedAt, &workingDir, &ctxFilesStr, &ctxDocsStr, &exclusionsStr)
	if err != nil {
		return nil, nil, fmt.Errorf("could not find session %s: %w", id, err)
	}

	var ctxFiles, ctxDocs, exclusions []string
	_ = json.Unmarshal([]byte(ctxFilesStr), &ctxFiles)
	_ = json.Unmarshal([]byte(ctxDocsStr), &ctxDocs)
	_ = json.Unmarshal([]byte(exclusionsStr), &exclusions)

	metadata := &Metadata{
		Title:            title,
		Mode:             mode,
		CreatedAt:        createdAt,
		ModifiedAt:       modifiedAt,
		ContextFiles:     ctxFiles,
		ContextDocuments: ctxDocs,
		Exclusions:       exclusions,
		WorkingDir:       workingDir,
	}

	rows, err := m.db.Query(`
	SELECT type, content, tool_calls, tool_call_id
	FROM messages
	WHERE session_id = ?
	ORDER BY seq ASC
	`, sessID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to query messages for session %s: %w", sessID, err)
	}
	defer rows.Close()

	var messages []types.Message
	for rows.Next() {
		var (
			msgType       int
			content       string
			toolCallsJSON string
			toolCallID    string
		)
		if err := rows.Scan(&msgType, &content, &toolCallsJSON, &toolCallID); err != nil {
			return nil, nil, err
		}

		var toolCalls []types.ToolCall
		if toolCallsJSON != "" && toolCallsJSON != "[]" {
			_ = json.Unmarshal([]byte(toolCallsJSON), &toolCalls)
		}

		messages = append(messages, types.Message{
			Type:       types.MessageType(msgType),
			Content:    content,
			ToolCalls:  toolCalls,
			ToolCallID: toolCallID,
		})
	}

	if err := rows.Err(); err != nil {
		return nil, nil, err
	}

	return metadata, messages, nil
}

func (m *Manager) ListConversations() ([]ConversationInfo, error) {
	return m.ListConversationsByMode("")
}

func (m *Manager) ListConversationsByMode(modeFilter string) ([]ConversationInfo, error) {
	if m == nil || m.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	normFilter := ""
	if modeFilter != "" {
		normFilter = normalizeMode(modeFilter)
	}

	rows, err := m.db.Query(`
	SELECT id, title, mode, created_at, modified_at
	FROM sessions
	ORDER BY modified_at DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to query sessions: %w", err)
	}
	defer rows.Close()

	var conversations []ConversationInfo
	for rows.Next() {
		var (
			id, title, mode       string
			createdAt, modifiedAt time.Time
		)
		if err := rows.Scan(&id, &title, &mode, &createdAt, &modifiedAt); err != nil {
			return nil, err
		}

		normMode := normalizeMode(mode)
		if normFilter != "" && normMode != normFilter {
			continue
		}

		conversations = append(conversations, ConversationInfo{
			ID:         id,
			Filename:   id,
			Mode:       normMode,
			Title:      title,
			CreatedAt:  createdAt,
			ModifiedAt: modifiedAt,
		})
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return conversations, nil
}

func normalizeMode(mode string) string {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case "chat":
		return "chat"
	case "agent":
		return "agent"
	default:
		return "coder"
	}
}
