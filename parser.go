package itf

import (
	"fmt"
	"path/filepath"
	"slices"
	"strings"
)

type ExecutionPlan struct {
	Actions      []PlannedAction
	FileActions  map[string]string
	DirsToCreate map[string]struct{}
	Failed       []string
}

func CreatePlan(content string, resolver *PathResolver, extensions []string, files []string) (*ExecutionPlan, error) {
	allowedFiles := make(map[string]struct{})
	for _, f := range files {
		allowedFiles[resolver.Resolve(f)] = struct{}{}
	}

	allBlocks, err := ExtractCodeBlocks([]byte(content))
	if err != nil {
		return nil, err
	}

	var actions []PlannedAction
	var failed []string
	
	// Track renames as we go to resolve diff sources correctly
	renameDestSet := make(map[string]struct{})
	renameDestToSource := make(map[string]string)

	for _, b := range allBlocks {
		switch b.Lang {
		case "rename":
			parsed := parseRenameBlock(b, resolver, allowedFiles)
			for _, r := range parsed {
				actions = append(actions, PlannedAction{Type: "rename", Rename: &r})
				renameDestSet[r.NewPath] = struct{}{}
				renameDestToSource[r.NewPath] = r.OldPath
			}
		case "delete":
			paths := parseDeleteBlock(b, resolver, allowedFiles)
			for _, p := range paths {
				actions = append(actions, PlannedAction{Type: "delete", Path: p})
			}
		case "diff":
			raw := strings.Trim(b.Content, "\n")
			path := ExtractPathFromDiff(raw)
			if path == "" || !isAllowed(resolver.Resolve(path), allowedFiles) {
				continue
			}
			
			d := DiffBlock{FilePath: path, RawContent: raw}
			abs := resolver.Resolve(d.FilePath)
			sourcePath := abs
			if s, ok := renameDestToSource[abs]; ok {
				sourcePath = s
			}

			if len(extensions) > 0 && !HasAllowedExtension(d.FilePath, extensions) {
				continue
			}

			patched, err := CorrectDiff(d, resolver, extensions, sourcePath)
			if err != nil {
				failed = append(failed, abs)
				continue
			}

			applied := applyPatch(sourcePath, patched)
			actions = append(actions, PlannedAction{
				Type: "write",
				Change: &FileChange{
					Path:     abs,
					Content:  applied,
					Source:   "diff",
					RawBlock: fmt.Sprintf("```diff\n%s\n```", d.RawContent),
				},
			})
		default:
			if len(extensions) == 1 && extensions[0] == ".diff" {
				continue
			}
			change := parseFileBlock(b, resolver, extensions, allowedFiles)
			if change != nil {
				actions = append(actions, PlannedAction{Type: "write", Change: change})
			}
		}
	}

	targetPaths := collectTargetPaths(actions)
	fileActions, dirs := GetFileActionsAndDirs(targetPaths, renameDestSet)
	
	for _, a := range actions {
		switch a.Type {
		case "delete":
			fileActions[a.Path] = "delete"
		case "rename":
			fileActions[a.Rename.OldPath] = "rename"
		}
	}

	return &ExecutionPlan{
		Actions:      actions,
		FileActions:  fileActions,
		DirsToCreate: dirs,
		Failed:       failed,
	}, nil
}

func parseFileBlock(b CodeBlock, resolver *PathResolver, extensions []string, allowed map[string]struct{}) *FileChange {
	path := ExtractPathFromHint(b.Hint)
	if path == "" {
		return nil
	}
	abs := resolver.Resolve(path)
	if !isAllowed(abs, allowed) {
		return nil
	}
	if !HasAllowedExtension(path, extensions) {
		return nil
	}

	trimmed := strings.TrimRight(b.Content, "\n")
	lines := strings.Split(trimmed, "\n")
	if len(lines) == 1 && lines[0] == "" {
		lines = []string{}
	}

	return &FileChange{
		Path:     abs,
		Content:  lines,
		Source:   "codeblock",
		RawBlock: fmt.Sprintf("```%s\n%s\n```", b.Lang, trimmed),
	}
}

func ExtractDiffBlocks(content string, resolver *PathResolver, files []string) []DiffBlock {
	blocks, _ := ExtractCodeBlocks([]byte(content))
	allowed := make(map[string]struct{})
	for _, f := range files {
		allowed[resolver.Resolve(f)] = struct{}{}
	}
	return extractDiffBlocksFromParsed(blocks, resolver, allowed)
}

func extractDiffBlocksFromParsed(blocks []CodeBlock, resolver *PathResolver, allowed map[string]struct{}) []DiffBlock {
	var diffs []DiffBlock
	for _, b := range blocks {
		if b.Lang != "diff" {
			continue
		}
		raw := strings.Trim(b.Content, "\n")
		path := ExtractPathFromDiff(raw)
		if path == "" {
			continue
		}
		if len(allowed) > 0 {
			if _, ok := allowed[resolver.Resolve(path)]; !ok {
				continue
			}
		}
		diffs = append(diffs, DiffBlock{FilePath: path, RawContent: raw})
	}
	return diffs
}

func ExtractPathFromHint(hint string) string {
	hint = strings.TrimSpace(hint)
	hint = strings.TrimLeft(hint, "# ")
	hint = strings.Trim(hint, "*")
	hint = strings.Trim(hint, "`")

	path := strings.TrimSpace(hint)
	if path != "" && !strings.Contains(path, " ") {
		return path
	}
	return ""
}

func HasAllowedExtension(path string, extensions []string) bool {
	if len(extensions) == 0 {
		return true
	}
	return slices.Contains(extensions, filepath.Ext(path))
}

func parseDeleteBlock(b CodeBlock, resolver *PathResolver, allowed map[string]struct{}) []string {
	var paths []string
	for line := range strings.SplitSeq(b.Content, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		abs := resolver.Resolve(trimmed)
		if !isAllowed(abs, allowed) {
			continue
		}
		paths = append(paths, abs)
	}
	return paths
}

func parseRenameBlock(b CodeBlock, resolver *PathResolver, allowed map[string]struct{}) []FileRename {
	var renames []FileRename
	for line := range strings.SplitSeq(b.Content, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		parts := strings.Fields(trimmed)
		if len(parts) != 2 {
			continue
		}
		oldAbs, newAbs := resolver.Resolve(parts[0]), resolver.Resolve(parts[1])
		if len(allowed) > 0 {
			_, ok1 := allowed[oldAbs]
			_, ok2 := allowed[newAbs]
			if !ok1 && !ok2 {
				continue
			}
		}
		renames = append(renames, FileRename{OldPath: oldAbs, NewPath: newAbs})
	}
	return renames
}

func isAllowed(path string, allowed map[string]struct{}) bool {
	if len(allowed) == 0 {
		return true
	}
	_, ok := allowed[path]
	return ok
}

func collectTargetPaths(actions []PlannedAction) []string {
	var paths []string
	seen := make(map[string]struct{})
	for _, a := range actions {
		p := ""
		switch a.Type {
		case "write":
			p = a.Change.Path
		case "rename":
			if _, ok := seen[a.Rename.OldPath]; !ok {
				paths = append(paths, a.Rename.OldPath)
				seen[a.Rename.OldPath] = struct{}{}
			}
			p = a.Rename.NewPath
		case "delete":
			p = a.Path
		}
		if p != "" {
			if _, ok := seen[p]; !ok {
				paths = append(paths, p)
				seen[p] = struct{}{}
			}
		}
	}
	return paths
}
