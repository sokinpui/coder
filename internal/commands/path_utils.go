package commands

import (
	"path/filepath"
	"strings"
)

// ExpandPaths handles glob patterns and returns a list of expanded paths and a list of invalid patterns.
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

// AppendUnique adds items to a slice only if they don't already exist.
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
