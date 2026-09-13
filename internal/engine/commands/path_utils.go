package commands

import (
	"os"
	"path/filepath"
	"strings"
)

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

func ExpandPaths(patterns []string) (expanded []string, invalid []string) {
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

func AppendUnique(original []string, newItems []string) []string {
	lookup := make(map[string]struct{}, len(original))
	for _, item := range original {
		lookup[item] = struct{}{}
	}

	result := original
	for _, item := range newItems {
		if _, exists := lookup[item]; !exists {
			result = append(result, item)
			lookup[item] = struct{}{}
		}
	}
	return result
}
