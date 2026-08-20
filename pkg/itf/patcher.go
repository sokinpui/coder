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
	target       []string
	replacement  []string
	deletedOnly  []string
	addedOnly    []string
	delOffset    int
	delSegments  [][]string
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
		if isAlreadyApplied(sourceLines, h) {
			return nil, fmt.Errorf("hunk #%d already applied (added lines are already present in file)", i+1)
		}

		startIdx, endIdx := matchHunk(sourceLines, h, searchStart)
		if startIdx == -1 {
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

	for line := range strings.SplitSeq(raw, "\n") {
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
	var currentDel []string

	for _, line := range lines {
		prefix := line[0]
		content := line[1:]

		switch prefix {
		case ' ':
			if len(currentDel) > 0 {
				h.delSegments = append(h.delSegments, currentDel)
				currentDel = nil
			}
			h.target = append(h.target, content)
			h.replacement = append(h.replacement, content)
		case '-':
			if h.delOffset == -1 {
				h.delOffset = len(h.target)
			}
			currentDel = append(currentDel, content)
			h.target = append(h.target, content)
			h.deletedOnly = append(h.deletedOnly, content)
		case '+':
			if len(currentDel) > 0 {
				h.delSegments = append(h.delSegments, currentDel)
				currentDel = nil
			}
			h.replacement = append(h.replacement, content)
			h.addedOnly = append(h.addedOnly, content)
		}
	}
	if len(h.target) == 0 && len(h.replacement) == 0 {
		return h, false
	}
	if len(currentDel) > 0 {
		h.delSegments = append(h.delSegments, currentDel)
	}
	return h, true
}

func isAlreadyApplied(source []string, h diffHunk) bool {
	if len(h.addedOnly) == 0 && len(h.deletedOnly) == 0 {
		return false
	}
	if len(h.replacement) > 0 {
		if start, _ := matchBlock(source, h.replacement, 0); start != -1 {
			if len(h.deletedOnly) > 0 {
				if tStart, _ := matchBlock(source, h.target, 0); tStart == -1 {
					return true
				}
			} else {
				return true
			}
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
		if searchStart <= len(source) {
			return searchStart, searchStart
		}
		return len(source), len(source)
	}

	if startIdx, endIdx := matchBlock(source, h.target, searchStart); startIdx != -1 {
		return startIdx, endIdx
	}

	if startIdx, endIdx := matchAnchor(source, h.target, searchStart); startIdx != -1 {
		return startIdx, endIdx
	}

	if startIdx, endIdx := matchFuzzyWindow(source, h.target, searchStart); startIdx != -1 {
		return startIdx, endIdx
	}

	if startIdx, endIdx := matchDeletedSegments(source, h, searchStart); startIdx != -1 {
		return startIdx, endIdx
	}

	return -1, -1
}

func matchAnchor(source, target []string, searchStart int) (int, int) {
	if len(target) < 3 {
		return -1, -1
	}

	topAnchor := extractAnchor(target, 2, false)
	botAnchor := extractAnchor(target, 2, true)
	if len(topAnchor) == 0 || len(botAnchor) == 0 {
		return -1, -1
	}

	start := max(0, searchStart)
	for i := start; i <= len(source)-len(topAnchor); i++ {
		if !isMatch(source[i:i+len(topAnchor)], topAnchor) {
			continue
		}

		searchBotStart := i + len(topAnchor)
		maxBotSearch := min(len(source), i+len(target)+8)
		for j := searchBotStart; j <= maxBotSearch-len(botAnchor); j++ {
			if !isMatch(source[j:j+len(botAnchor)], botAnchor) {
				continue
			}

			end := j + len(botAnchor)
			window := source[i:end]
			if similarityScore(window, target) >= 0.70 {
				return i, end
			}
		}
	}
	return -1, -1
}

func extractAnchor(lines []string, count int, fromEnd bool) []string {
	var anchor []string
	if fromEnd {
		for i := len(lines) - 1; i >= 0 && len(anchor) < count; i-- {
			if strings.TrimSpace(lines[i]) != "" {
				anchor = append([]string{lines[i]}, anchor...)
			}
		}
		return anchor
	}

	for i := 0; i < len(lines) && len(anchor) < count; i++ {
		if strings.TrimSpace(lines[i]) != "" {
			anchor = append(anchor, lines[i])
		}
	}
	return anchor
}

func matchFuzzyWindow(source, target []string, searchStart int) (int, int) {
	if len(source) == 0 || len(target) == 0 {
		return -1, -1
	}

	targetLen := len(target)
	bestScore := 0.0
	bestStart := -1
	bestEnd := -1

	start := max(0, searchStart)
	minWindow := max(1, targetLen-4)
	maxWindow := targetLen + 4

	for i := start; i < len(source); i++ {
		for w := minWindow; w <= maxWindow; w++ {
			if i+w > len(source) {
				break
			}
			window := source[i : i+w]
			score := similarityScore(window, target)
			if score > bestScore && score >= 0.70 {
				bestScore = score
				bestStart = i
				bestEnd = i + w
			}
		}
	}

	return bestStart, bestEnd
}

func matchDeletedSegments(source []string, h diffHunk, searchStart int) (int, int) {
	if len(h.delSegments) == 0 || h.delOffset < 0 {
		return -1, -1
	}

	pos := max(0, searchStart)
	firstStart := -1
	lastEnd := -1

	for _, seg := range h.delSegments {
		if !isSubstantial(seg) {
			continue
		}
		s, e := matchBlock(source, seg, pos)
		if s == -1 {
			return -1, -1
		}
		if firstStart == -1 {
			firstStart = s
		}
		lastEnd = e
		pos = e
	}

	if firstStart == -1 || lastEnd == -1 {
		return -1, -1
	}

	startIdx := max(0, firstStart-h.delOffset)
	endIdx := min(len(source), startIdx+len(h.target))
	if similarityScore(source[startIdx:endIdx], h.target) >= 0.60 {
		return startIdx, endIdx
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

func similarityScore(a, b []string) float64 {
	if len(a) == 0 && len(b) == 0 {
		return 1.0
	}
	if len(a) == 0 || len(b) == 0 {
		return 0.0
	}
	lcs := lcsLength(a, b)
	return (2.0 * float64(lcs)) / float64(len(a)+len(b))
}

func lcsLength(a, b []string) int {
	dp := make([]int, len(b)+1)
	for i := 1; i <= len(a); i++ {
		prev := 0
		for j := 1; j <= len(b); j++ {
			temp := dp[j]
			if linesMatch(a[i-1], b[j-1]) {
				dp[j] = prev + 1
			} else if dp[j-1] > dp[j] {
				dp[j] = dp[j-1]
			}
			prev = temp
		}
	}
	return dp[len(b)]
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
