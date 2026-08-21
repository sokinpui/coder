package itf

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"
)

type Config struct {
	Undo       bool
	Redo       bool
	Extensions []string
	Files      []string
}

type ProgressUpdate func(current, total int)

type App struct {
	cfg              *Config
	stateManager     *StateManager
	pathResolver     *PathResolver
	sourceProvider   *SourceProvider
	fileManager      *FileManager
	progressCallback ProgressUpdate
}

type DetailedError struct {
	Err   error
	Stack []byte
}

func (e *DetailedError) Error() string { return e.Err.Error() }

func NewApp(cfg *Config) (*App, error) {
	sm, err := NewStateManager()
	if err != nil {
		return nil, err
	}

	pr, err := NewPathResolver()
	if err != nil {
		return nil, err
	}

	return &App{
		cfg:            cfg,
		stateManager:   sm,
		pathResolver:   pr,
		sourceProvider: NewSourceProvider(),
		fileManager:    NewFileManager(),
	}, nil
}

func SetVersion(v string) {
	rootCmd.Version = v
}

func (a *App) SetProgressCallback(cb ProgressUpdate) { a.progressCallback = cb }

func (a *App) Execute() (summary Summary, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = &DetailedError{Err: fmt.Errorf("panic: %v", r), Stack: debug.Stack()}
		}
	}()

	switch {
	case a.cfg.Undo:
		return a.undoLastOperation()
	case a.cfg.Redo:
		return a.redoLastOperation()
	default:
		return a.processContent()
	}
}

func (a *App) processContent() (Summary, error) {
	c, err := a.sourceProvider.GetContent()
	if err != nil || c == "" {
		return Summary{Message: "Empty source"}, err
	}
	return a.processAndApply(c)
}

func (a *App) processAndApply(content string) (Summary, error) {
	a.stateManager.Sync()
	plan, err := CreatePlan(content, a.pathResolver, a.cfg.Extensions, a.cfg.Files)
	if err != nil {
		return Summary{}, err
	}
	if len(plan.Actions) == 0 {
		if len(plan.Failed) > 0 {
			s := Summary{Failed: plan.Failed}
			a.relativizeSummaryPaths(&s)
			return s, nil
		}
		return Summary{Message: "Nothing to do"}, nil
	}

	CreateDirs(plan.DirsToCreate)
	return a.applyChanges(plan)
}

func (a *App) applyChanges(plan *ExecutionPlan) (Summary, error) {
	totalOps := len(plan.Actions)
	currentOp := 0
	oldHashes := make(map[string]string)

	var created, modified, deleted, renamedSuccess []string
	var failedCreate, failedModify, failedDeletes, failedRenames []string
	renamedMap := make(map[string]string)

	progress := func() {
		currentOp++
		a.reportProgress(currentOp, totalOps)
	}

	trash := filepath.Join(a.stateManager.StateDir, TrashDir)

	for _, action := range plan.Actions {
		switch action.Type {
		case "write":
			isCreate := plan.FileActions[action.Change.Path] == "create"
			if !isCreate {
				a.backupFileState(action.Change.Path, oldHashes)
			}

			upd, fail := a.fileManager.WriteChanges([]FileChange{*action.Change}, nil)
			if len(fail) > 0 {
				if isCreate {
					failedCreate = append(failedCreate, fail...)
				} else {
					failedModify = append(failedModify, fail...)
				}
			} else if len(upd) > 0 {
				if isCreate {
					created = append(created, upd...)
				} else {
					modified = append(modified, upd...)
				}
			}

		case "rename":
			r := action.Rename
			a.backupFileState(r.OldPath, oldHashes)
			if err := os.Rename(r.OldPath, r.NewPath); err == nil {
				renamedMap[r.OldPath] = r.NewPath
				renamedSuccess = append(renamedSuccess, r.OldPath)
			} else {
				failedRenames = append(failedRenames, fmt.Sprintf("%s -> %s: %v", r.OldPath, r.NewPath, err))
			}

		case "delete":
			p := action.Path
			a.backupFileState(p, oldHashes)
			if err := TrashFile(p, trash, a.stateManager.ProjectRoot); err == nil {
				deleted = append(deleted, p)
			} else {
				failedDeletes = append(failedDeletes, fmt.Sprintf("%s: %v", p, err))
			}
		}
		progress()
	}

	a.recordHistory(created, modified, deleted, renamedSuccess, renamedMap, plan, oldHashes)

	return a.createSummary(
		created,
		modified,
		deleted,
		renamedMap,
		append(failedCreate, failedModify...),
		failedDeletes,
		failedRenames,
		plan.Failed,
	)
}

func (a *App) recordHistory(created, modified, deleted, renamed []string, renamedMap map[string]string, plan *ExecutionPlan, oldHashes map[string]string) {
	successCount := len(created) + len(modified) + len(deleted) + len(renamed)
	if successCount == 0 {
		return
	}

	var renamesList []FileRename
	for oldPath, newPath := range renamedMap {
		renamesList = append(renamesList, FileRename{OldPath: oldPath, NewPath: newPath})
	}

	historyPaths := make([]string, 0, successCount)
	historyPaths = append(historyPaths, created...)
	historyPaths = append(historyPaths, modified...)
	historyPaths = append(historyPaths, deleted...)
	historyPaths = append(historyPaths, renamed...)

	ops := a.stateManager.CreateOperations(historyPaths, plan.FileActions, renamesList, oldHashes)
	a.stateManager.Write(ops)
}

func (a *App) backupFileState(path string, hashes map[string]string) {
	if _, ok := hashes[path]; ok {
		return // Already backed up
	}
	h, _ := GetFileSHA256(path)
	hashes[path] = h
	if h != "" {
		if content, err := os.ReadFile(path); err == nil {
			_ = WriteBlob(a.stateManager.StateDir, h, content)
		}
	}
}

func (a *App) reportProgress(current, total int) {
	if a.progressCallback != nil {
		a.progressCallback(current, total)
	}
}

func (a *App) createSummary(created, modified, deleted []string, renamed map[string]string, failedWrites, failedDeletes, failedRenames, failedPlan []string) (Summary, error) {
	var renamedPaths []string
	for oldPath, newPath := range renamed {
		renamedPaths = append(renamedPaths, fmt.Sprintf("%s -> %s", oldPath, newPath))
	}

	allFailed := append(failedWrites, append(failedDeletes, append(failedRenames, failedPlan...)...)...)
	s := Summary{
		Created:  created,
		Modified: modified,
		Deleted:  deleted,
		Renamed:  renamedPaths,
		Failed:   allFailed,
	}
	a.relativizeSummaryPaths(&s)
	return s, nil
}

func (a *App) undoLastOperation() (Summary, error) {
	ops := a.stateManager.GetOperationsToUndo()
	if len(ops) == 0 {
		return Summary{Message: "No undo"}, nil
	}
	s := a.fileManager.Undo(ops, a.stateManager.StateDir, a.stateManager.ProjectRoot)
	s.Message = "Undone"
	a.relativizeSummaryPaths(&s)
	return s, nil
}

func (a *App) redoLastOperation() (Summary, error) {
	ops := a.stateManager.GetOperationsToRedo()
	if len(ops) == 0 {
		return Summary{Message: "No redo"}, nil
	}
	s := a.fileManager.Redo(ops, a.stateManager.StateDir, a.stateManager.ProjectRoot)
	s.Message = "Redone"
	a.relativizeSummaryPaths(&s)
	return s, nil
}

func (a *App) relativizeSummaryPaths(s *Summary) {
	wd, _ := os.Getwd()
	relPath := func(p string) string {
		if r, err := filepath.Rel(wd, p); err == nil {
			return r
		}
		return p
	}

	relList := func(paths []string) []string {
		var res []string
		for _, p := range paths {
			if strings.Contains(p, ": ") {
				parts := strings.SplitN(p, ": ", 2)
				filePath := parts[0]
				errMsg := parts[1]
				if strings.Contains(filePath, " -> ") {
					arrowParts := strings.SplitN(filePath, " -> ", 2)
					res = append(res, fmt.Sprintf("%s -> %s: %s", relPath(arrowParts[0]), relPath(arrowParts[1]), errMsg))
				} else {
					res = append(res, fmt.Sprintf("%s: %s", relPath(filePath), errMsg))
				}
			} else if strings.Contains(p, " -> ") {
				parts := strings.SplitN(p, " -> ", 2)
				res = append(res, fmt.Sprintf("%s -> %s", relPath(parts[0]), relPath(parts[1])))
			} else {
				res = append(res, relPath(p))
			}
		}
		return res
	}
	s.Created = relList(s.Created)
	s.Modified = relList(s.Modified)
	s.Deleted = relList(s.Deleted)
	s.Renamed = relList(s.Renamed)
	s.Failed = relList(s.Failed)
}
