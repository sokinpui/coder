package itf

import (
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"
)

var filePathRegex = regexp.MustCompile(`(?m)^\+\+\+ b/(?P<path>.*?)(\s|$)`)

func ExtractPathFromDiff(content string) string {
	if match := filePathRegex.FindStringSubmatch(content); len(match) > 1 {
		return strings.TrimSpace(match[1])
	}
	return ""
}

type diffHunk struct {
	target      []string
	replacement []string
	deletedOnly []string
	addedOnly   []string
	delOffset   int
}

type hunkPatch struct {
	startIdx    int
	endIdx      int
	replacement []string
}

func ApplyDiffToPath(sourcePath, rawDiff string) ([]string, error) {
	var sourceLines []string
	if sourcePath != "" {
		content, err := os.ReadFile(sourcePath)
		if err != nil && !os.IsNotExist(err) {
			return nil, fmt.Errorf("failed to read source file: %w", err)
		}
		if len(content) > 0 {
			sourceLines = strings.Split(strings.TrimSuffix(strings.ReplaceAll(string(content), "\r\n", "\n"), "\n"), "\n")
		}
	}
	return ApplyDiff(sourceLines, rawDiff)
}

func ApplyDiff(sourceLines []string, rawDiff string) ([]string, error) {
	hunks := parseDiffHunks(rawDiff)
	if len(hunks) == 0 {
		return nil, fmt.Errorf("no valid diff hunks found")
	}

	var patches []hunkPatch
	searchStart := 0

	for i, h := range hunks {
		startIdx, endIdx := matchHunk(sourceLines, h, searchStart)
		if startIdx == -1 {
			if isAlreadyApplied(sourceLines, h) {
				return nil, fmt.Errorf("hunk #%d already applied (added lines are already present in file)", i+1)
			}
			return nil, fmt.Errorf("failed to match hunk #%d near %s", i+1, hunkPreview(h.target))
		}
		patches = append(patches, hunkPatch{
			startIdx:    startIdx,
			endIdx:      endIdx,
			replacement: h.replacement,
		})
		searchStart = endIdx
	}

	sort.Slice(patches, func(i, j int) bool {
		return patches[i].startIdx > patches[j].startIdx
	})

	result := make([]string, len(sourceLines))
	copy(result, sourceLines)

	for _, p := range patches {
		if p.startIdx > len(result) || p.endIdx > len(result) || p.startIdx > p.endIdx {
			return nil, fmt.Errorf("invalid patch bounds [%d:%d] for length %d", p.startIdx, p.endIdx, len(result))
		}
		result = append(result[:p.startIdx], append(p.replacement, result[p.endIdx:]...)...)
	}

	return result, nil
}

func parseDiffHunks(raw string) []diffHunk {
	var hunks []diffHunk
	var currentLines []string
	inHunk := !strings.Contains(raw, "@@")

	for _, line := range strings.Split(raw, "\n") {
		if strings.HasPrefix(line, "@@") {
			inHunk = true
			if len(currentLines) > 0 {
				if h, ok := buildDiffHunk(currentLines); ok {
					hunks = append(hunks, h)
				}
			}
			currentLines = nil
			continue
		}

		if !inHunk {
			continue
		}

		if !strings.Contains(raw, "@@") && (strings.HasPrefix(line, "---") || strings.HasPrefix(line, "+++")) && len(currentLines) == 0 {
			continue
		}

		if strings.HasPrefix(line, "+") || strings.HasPrefix(line, "-") || strings.HasPrefix(line, " ") || (line == "" && len(currentLines) > 0) {
			if line == "" {
				currentLines = append(currentLines, " ")
			} else {
				currentLines = append(currentLines, line)
			}
		}
	}
	if len(currentLines) > 0 {
		if h, ok := buildDiffHunk(currentLines); ok {
			hunks = append(hunks, h)
		}
	}
	return hunks
}

func buildDiffHunk(lines []string) (diffHunk, bool) {
	var h diffHunk
	h.delOffset = -1

	for _, line := range lines {
		prefix := line[0]
		content := line[1:]

		switch prefix {
		case ' ':
			h.target = append(h.target, content)
			h.replacement = append(h.replacement, content)
		case '-':
			if h.delOffset == -1 {
				h.delOffset = len(h.target)
			}
			h.target = append(h.target, content)
			h.deletedOnly = append(h.deletedOnly, content)
		case '+':
			h.replacement = append(h.replacement, content)
			h.addedOnly = append(h.addedOnly, content)
		}
	}
	if len(h.target) == 0 && len(h.replacement) == 0 {
		return h, false
	}
	return h, true
}

func isAlreadyApplied(source []string, h diffHunk) bool {
	if len(h.addedOnly) == 0 {
		return false
	}
	if len(h.replacement) > 0 {
		if start, _ := matchBlock(source, h.replacement, 0); start != -1 {
			return true
		}
	}
	if isSubstantial(h.addedOnly) {
		if start, _ := matchBlock(source, h.addedOnly, 0); start != -1 {
			return true
		}
	}
	return false
}

func matchHunk(source []string, h diffHunk, searchStart int) (int, int) {
	if len(h.target) == 0 {
		return len(source), len(source)
	}

	startIdx, endIdx := matchBlock(source, h.target, searchStart)
	if startIdx != -1 {
		return startIdx, endIdx
	}

	if len(h.deletedOnly) > 0 && isSubstantial(h.deletedOnly) {
		dos, dme := matchBlock(source, h.deletedOnly, searchStart)
		if dos != -1 {
			os := dos - h.delOffset
			me := dme + (len(h.target) - (h.delOffset + len(h.deletedOnly)))
			if os >= 0 && me <= len(source) {
				return os, me
			}
		}
	}
	return -1, -1
}

func matchBlock(source, target []string, startIdx int) (int, int) {
	if len(target) == 0 {
		return len(source), len(source)
	}

	normalizedSource := normalizeLines(source)
	normalizedTarget := normalizeLines(target)
	start := max(0, startIdx)

	for i := start; i <= len(normalizedSource)-len(normalizedTarget); i++ {
		if isMatch(normalizedSource[i:i+len(normalizedTarget)], normalizedTarget) {
			return i, i + len(normalizedTarget)
		}
	}
	return -1, -1
}

func isMatch(source, target []string) bool {
	for i := range target {
		if !linesMatch(source[i], target[i]) {
			return false
		}
	}
	return true
}

func linesMatch(s1, s2 string) bool {
	if s1 == s2 {
		return true
	}
	return strings.TrimSpace(s1) == strings.TrimSpace(s2)
}

func isSubstantial(lines []string) bool {
	count := 0
	for _, l := range lines {
		trimmed := strings.TrimSpace(l)
		if len(trimmed) > 1 && trimmed != "}" && trimmed != "{" && trimmed != ")," && trimmed != ");" {
			return true
		}
		count++
	}
	return count > 3
}

func normalizeLines(lines []string) []string {
	normalized := make([]string, len(lines))
	for i, l := range lines {
		normalized[i] = strings.TrimRight(l, " \t\r\n")
	}
	return normalized
}

func hunkPreview(lines []string) string {
	for _, l := range lines {
		trimmed := strings.TrimSpace(l)
		if trimmed != "" {
			if len(trimmed) > 40 {
				trimmed = trimmed[:37] + "..."
			}
			return fmt.Sprintf("%q", trimmed)
		}
	}
	return "empty target"
}
