package commands

import (
	"path/filepath"
	"strings"
)

func ExpandPaths(patterns []string) (expanded []string, invalid []string) {
	for _, p := range patterns {
		if !strings.ContainsAny(p, "*?[]") {
			expanded = append(expanded, p)
			continue
		}

		matches, err := filepath.Glob(p)
		if err != nil || len(matches) == 0 {
			invalid = append(invalid, p)
			continue
		}
		expanded = append(expanded, matches...)
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
