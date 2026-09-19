package commands

import "github.com/sokinpui/coder/internal/engine/source"

func ExpandHome(path string) string {
	return source.ExpandHome(path)
}

func RelativizeToCWD(path string) string {
	return source.RelativizeToCWD(path)
}

func ExpandPaths(patterns []string) (expanded []string, invalid []string) {
	return source.ExpandPaths(patterns)
}

func AppendUnique(original []string, newItems []string) []string {
	return source.AppendUnique(original, newItems)
}

func FilterPaths(original []string, toRemove map[string]struct{}) []string {
	return source.FilterPaths(original, toRemove)
}

func filterPaths(original []string, toRemove map[string]struct{}) []string {
	return source.FilterPaths(original, toRemove)
}
