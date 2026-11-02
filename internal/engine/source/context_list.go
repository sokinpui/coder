package source

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func Add(currentFiles, inputs, exclusions []string) ([]string, []string) {
	expanded, invalid := ExpandPaths(inputs)

	var dirs []string
	var codeFiles []string

	for _, p := range expanded {
		if strings.EqualFold(filepath.Ext(p), ".pdf") {
			continue
		}
		info, err := os.Stat(p)
		if err != nil {
			invalid = append(invalid, p)
			continue
		}

		if info.IsDir() {
			dirs = append(dirs, filepath.ToSlash(p))
			continue
		}

		codeFiles = append(codeFiles, filepath.ToSlash(p))
	}

	var resolvedFiles []string
	if len(dirs) > 0 || len(codeFiles) > 0 {
		resolvedFiles, _ = ResolveFileList(dirs, codeFiles, exclusions)
	}

	newFiles := AppendUnique(currentFiles, resolvedFiles)
	return newFiles, invalid
}

func Exclude(currentFiles, targets []string) ([]string, []string) {
	expanded, _ := ExpandPaths(targets)
	toRemove := make(map[string]struct{}, len(expanded))
	for _, p := range expanded {
		toRemove[filepath.ToSlash(filepath.Clean(p))] = struct{}{}
	}

	newFiles := make([]string, 0, len(currentFiles))
	var removedFiles []string
	for _, f := range currentFiles {
		if ShouldRemovePath(f, toRemove) {
			removedFiles = append(removedFiles, f)
			continue
		}
		newFiles = append(newFiles, f)
	}

	return newFiles, removedFiles
}

func FilterPaths(original []string, toRemove map[string]struct{}) []string {
	filtered := make([]string, 0, len(original))
	for _, p := range original {
		if ShouldRemovePath(p, toRemove) {
			continue
		}
		filtered = append(filtered, p)
	}
	return filtered
}

func ShouldRemovePath(path string, toRemove map[string]struct{}) bool {
	pathKey := path
	cleanTarget := filepath.ToSlash(filepath.Clean(pathKey))
	if _, found := toRemove[cleanTarget]; found {
		return true
	}
	if _, found := toRemove[path]; found {
		return true
	}

	for removePath := range toRemove {
		cleanRemove := strings.TrimSuffix(filepath.ToSlash(filepath.Clean(removePath)), "/")
		if cleanRemove == "" {
			continue
		}
		if cleanTarget == cleanRemove {
			return true
		}
		if cleanRemove == "." {
			if !strings.HasPrefix(cleanTarget, "../") && !filepath.IsAbs(cleanTarget) {
				return true
			}
		}
		if strings.HasPrefix(cleanTarget, cleanRemove+"/") {
			return true
		}
	}
	return false
}

func ContextSuggestions(files []string) []string {
	if len(files) == 0 {
		return nil
	}

	seen := make(map[string]struct{})
	var suggestions []string

	addPath := func(p string) {
		p = filepath.ToSlash(filepath.Clean(p))
		if p == "." || p == "" {
			return
		}

		for i := 0; i < len(p); i++ {
			if p[i] == '/' {
				dir := p[:i+1]
				if dir != "/" {
					if _, ok := seen[dir]; !ok {
						seen[dir] = struct{}{}
						suggestions = append(suggestions, dir)
					}
				}
			}
		}

		if _, ok := seen[p]; !ok {
			seen[p] = struct{}{}
			suggestions = append(suggestions, p)
		}
	}

	for _, f := range files {
		addPath(f)
	}

	sort.Strings(suggestions)
	return suggestions
}

func AppendUnique(original []string, newItems []string) []string {
	seen := make(map[string]struct{}, len(original)+len(newItems))
	for _, item := range original {
		seen[item] = struct{}{}
	}

	result := append([]string{}, original...)
	for _, item := range newItems {
		if _, ok := seen[item]; !ok {
			seen[item] = struct{}{}
			result = append(result, item)
		}
	}
	return result
}

func ExpandHome(path string) string {
	if path != "~" && !strings.HasPrefix(path, "~/") && !strings.HasPrefix(path, "~\\") {
		return path
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return path
	}

	if path == "~" {
		return home
	}

	return filepath.Join(home, path[2:])
}

func RelativizeToCWD(path string) string {
	cwd, err := os.Getwd()
	if err != nil {
		return path
	}

	rel, err := filepath.Rel(cwd, path)
	if err != nil || strings.HasPrefix(rel, "..") {
		return path
	}
	return rel
}

func ExpandPaths(patterns []string) ([]string, []string) {
	var expanded []string
	var invalid []string

	for _, p := range patterns {
		expandedPath := RelativizeToCWD(ExpandHome(p))
		if !strings.ContainsAny(expandedPath, "*?[]") {
			expanded = append(expanded, expandedPath)
			continue
		}

		matches, err := filepath.Glob(expandedPath)
		if err != nil || len(matches) == 0 {
			invalid = append(invalid, p)
			continue
		}
		for _, m := range matches {
			expanded = append(expanded, RelativizeToCWD(m))
		}
	}
	return expanded, invalid
}
