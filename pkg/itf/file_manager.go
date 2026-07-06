package itf

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type FileManager struct{}

func NewFileManager() *FileManager {
	return &FileManager{}
}

func (m *FileManager) WriteChanges(changes []FileChange, progressCb func(int)) (updated, failed []string) {
	for i, change := range changes {
		content := strings.Join(change.Content, "\n")
		if len(change.Content) > 0 {
			content += "\n"
		}

		if err := os.WriteFile(change.Path, []byte(content), 0644); err != nil {
			failed = append(failed, change.Path)
			continue
		}

		updated = append(updated, change.Path)
		if progressCb != nil {
			progressCb(i + 1)
		}
	}
	return updated, failed
}

func (m *FileManager) Undo(ops []Operation, stateDir string, projectRoot string) Summary {
	var s Summary
	for _, op := range ops {
		if !m.undoFile(op, stateDir, projectRoot) {
			s.Failed = append(s.Failed, op.Path)
			continue
		}

		switch op.Action {
		case "create":
			s.Deleted = append(s.Deleted, op.Path)
		case "delete":
			s.Created = append(s.Created, op.Path)
		case "modify":
			s.Modified = append(s.Modified, op.Path)
		case "rename":
			s.Renamed = append(s.Renamed, fmt.Sprintf("%s -> %s", op.NewPath, op.Path))
		}
	}
	return s
}

func (m *FileManager) undoFile(op Operation, stateDir string, projectRoot string) bool {
	currentPath := op.Path
	if op.Action == "rename" {
		currentPath = op.NewPath
	}

	checkPath := currentPath
	if op.Action == "delete" {
		rel, _ := filepath.Rel(projectRoot, op.Path)
		checkPath = filepath.Join(stateDir, TrashDir, rel)
	}

	actualHash, _ := GetFileSHA256(checkPath)
	if actualHash != op.ContentHash {
		return false
	}

	if op.Action == "rename" {
		return os.Rename(op.NewPath, op.Path) == nil
	}

	if op.Action == "create" {
		return os.Remove(op.Path) == nil
	}

	if op.Action == "delete" {
		return RestoreFileFromTrash(op.Path, filepath.Join(stateDir, TrashDir), projectRoot) == nil
	}

	content, err := ReadBlob(stateDir, op.OldContentHash)
	if err != nil {
		return false
	}

	return os.WriteFile(op.Path, content, 0644) == nil
}

func (m *FileManager) Redo(ops []Operation, stateDir string, projectRoot string) Summary {
	var s Summary
	for _, op := range ops {
		if !m.redoFile(op, stateDir, projectRoot) {
			s.Failed = append(s.Failed, op.Path)
			continue
		}

		switch op.Action {
		case "create":
			s.Created = append(s.Created, op.Path)
		case "delete":
			s.Deleted = append(s.Deleted, op.Path)
		case "modify":
			s.Modified = append(s.Modified, op.Path)
		case "rename":
			s.Renamed = append(s.Renamed, fmt.Sprintf("%s -> %s", op.Path, op.NewPath))
		}
	}
	return s
}

func (m *FileManager) redoFile(op Operation, stateDir string, projectRoot string) bool {
	actualHash, _ := GetFileSHA256(op.Path)
	if actualHash != op.OldContentHash {
		return false
	}

	if op.Action == "rename" {
		return os.Rename(op.Path, op.NewPath) == nil
	}

	if op.Action == "delete" {
		return TrashFile(op.Path, filepath.Join(stateDir, TrashDir), projectRoot) == nil
	}

	content, err := ReadBlob(stateDir, op.ContentHash)
	if err != nil {
		return false
	}

	_ = os.MkdirAll(filepath.Dir(op.Path), 0755)
	return os.WriteFile(op.Path, content, 0644) == nil
}
