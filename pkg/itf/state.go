package itf

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	stateDirName    = ".itf"
	stateFileName   = "states.itf"
	TrashDir        = "trash"
	BlobsDir        = "blobs"
	entrySeparator  = "\n===\n"
	opSeparator     = "\n---\n"
	none            = "-"
)

type Operation struct {
	Timestamp      int64
	Action         string
	Path           string
	OldContentHash string
	ContentHash    string
	NewPath        string
}

type HistoryEntry struct {
	Operations []Operation
}

type State struct {
	History      []HistoryEntry
	CurrentIndex int
}

type StateManager struct {
	statePath   string
	state       *State
	StateDir    string
	ProjectRoot string
}

func findGitRoot() (string, error) {
	gitPath, err := exec.LookPath("git")
	if err != nil {
		return os.Getwd()
	}

	out, err := exec.Command(gitPath, "rev-parse", "--show-toplevel").Output()
	if err != nil {
		return os.Getwd()
	}
	return strings.TrimSpace(string(out)), nil
}

func NewStateManager() (*StateManager, error) {
	root, _ := findGitRoot()
	dir := filepath.Join(root, stateDirName)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}
	m := &StateManager{
		statePath:   filepath.Join(dir, stateFileName),
		StateDir:    dir,
		ProjectRoot: root,
	}
	m.state = &State{CurrentIndex: -1, History: []HistoryEntry{}}
	_ = m.load()
	return m, nil
}

func (m *StateManager) load() error {
	file, err := os.Open(m.statePath)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	if !scanner.Scan() {
		return nil
	}

	idx, _ := strconv.Atoi(strings.TrimSpace(scanner.Text()))
	m.state = &State{CurrentIndex: idx, History: []HistoryEntry{}}

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "===" {
			m.state.History = append(m.state.History, HistoryEntry{})
			continue
		}

		if line == "" || line == "---" || len(m.state.History) == 0 {
			continue
		}

		entry := &m.state.History[len(m.state.History)-1]
		op := Operation{Timestamp: parseTimestamp(line)}

		fields := []*string{&op.Action, &op.Path, &op.OldContentHash, &op.ContentHash, &op.NewPath}
		for _, f := range fields {
			if !scanner.Scan() {
				break
			}
			*f = strings.TrimSpace(scanner.Text())
		}

		op.Action = m.fromStoreValue(op.Action)
		op.Path = m.resolvePath(op.Path)
		op.OldContentHash = m.fromStoreValue(op.OldContentHash)
		op.ContentHash = m.fromStoreValue(op.ContentHash)
		op.NewPath = m.resolvePath(op.NewPath)

		entry.Operations = append(entry.Operations, op)
	}
	return scanner.Err()
}

func parseTimestamp(s string) int64 {
	ts, _ := strconv.ParseInt(strings.TrimSpace(s), 10, 64)
	return ts
}

func (m *StateManager) save() {
	file, err := os.Create(m.statePath)
	if err != nil {
		return
	}
	defer file.Close()

	writer := bufio.NewWriter(file)
	defer writer.Flush()

	fmt.Fprintf(writer, "%d", m.state.CurrentIndex)

	for _, e := range m.state.History {
		fmt.Fprint(writer, entrySeparator)
		for i, op := range e.Operations {
			fmt.Fprintf(writer, "%d\n%s\n%s\n%s\n%s\n%s",
				op.Timestamp,
				m.toStoreValue(op.Action),
				m.relativePath(op.Path),
				m.toStoreValue(op.OldContentHash),
				m.toStoreValue(op.ContentHash),
				m.relativePath(op.NewPath),
			)
			if i < len(e.Operations)-1 {
				fmt.Fprint(writer, opSeparator)
			}
		}
	}
}

func (m *StateManager) fromStoreValue(s string) string {
	if s == none {
		return ""
	}
	return s
}

func (m *StateManager) toStoreValue(s string) string {
	if s == "" {
		return none
	}
	return s
}

func (m *StateManager) relativePath(p string) string {
	if p == "" {
		return none
	}
	if rel, err := filepath.Rel(m.ProjectRoot, p); err == nil {
		return rel
	}
	return p
}

func (m *StateManager) resolvePath(p string) string {
	if p == "" || p == none {
		return ""
	}
	if filepath.IsAbs(p) {
		return p
	}
	return filepath.Join(m.ProjectRoot, p)
}

func (m *StateManager) Sync() {
	if m.state.CurrentIndex < 0 {
		return
	}

	for i := m.state.CurrentIndex; i >= 0; i-- {
		if m.matchState(i) {
			if i < m.state.CurrentIndex {
				m.state.History = m.state.History[:i+1]
				m.state.CurrentIndex = i
				m.save()
			}
			return
		}
	}

	m.state.History = []HistoryEntry{}
	m.state.CurrentIndex = -1
	m.save()
}

func (m *StateManager) matchState(idx int) bool {
	if idx < 0 || idx >= len(m.state.History) {
		return false
	}

	entry := m.state.History[idx]
	for _, op := range entry.Operations {
		path := op.Path
		if op.Action == "rename" {
			path = op.NewPath
		}
		
		currentHash, err := GetFileSHA256(path)
		if op.Action == "delete" {
			if err == nil {
				return false
			}
			continue
		}

		if err != nil || currentHash != op.ContentHash {
			return false
		}
	}
	return true
}

func (m *StateManager) Write(ops []Operation) {
	if m.state.CurrentIndex < len(m.state.History)-1 {
		m.state.History = m.state.History[:m.state.CurrentIndex+1]
	}
	m.state.History = append(m.state.History, HistoryEntry{Operations: ops})
	m.state.CurrentIndex++
	m.save()
}

func (m *StateManager) GetOperationsToUndo() []Operation {
	if m.state.CurrentIndex < 0 {
		return nil
	}
	ops := m.state.History[m.state.CurrentIndex].Operations
	m.state.CurrentIndex--
	m.save()
	return ops
}

func (m *StateManager) GetOperationsToRedo() []Operation {
	if m.state.CurrentIndex+1 >= len(m.state.History) {
		return nil
	}
	m.state.CurrentIndex++
	ops := m.state.History[m.state.CurrentIndex].Operations
	m.save()
	return ops
}

func (m *StateManager) CreateOperations(updated []string, actions map[string]string, renames []FileRename, oldHashes map[string]string) []Operation {
	var ops []Operation
	rm := make(map[string]string)
	for _, r := range renames {
		rm[r.OldPath] = r.NewPath
	}

	now := time.Now().UTC().Unix()
	for _, f := range updated {
		action := actions[f]
		checkPath, newPath := f, ""
		
		switch action {
		case "rename":
			newPath = rm[f]
			checkPath = newPath
		case "delete":
			rel, _ := filepath.Rel(m.ProjectRoot, f)
			checkPath = filepath.Join(m.StateDir, TrashDir, rel)
		}

		currentHash, _ := GetFileSHA256(checkPath)
		if action != "delete" && currentHash != "" {
			content, _ := os.ReadFile(checkPath)
			_ = WriteBlob(m.StateDir, currentHash, content)
		}

		ops = append(ops, Operation{
			Timestamp:      now,
			Path:           f,
			Action:         action,
			OldContentHash: oldHashes[f],
			ContentHash:    currentHash,
			NewPath:        newPath,
		})
	}
	sort.Slice(ops, func(i, j int) bool { return ops[i].Path < ops[j].Path })
	return ops
}
