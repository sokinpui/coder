package itf

import (
	"fmt"
	"strings"
)

func getTargetBlock(diff []string) (block []string, deletedOnly []string, deletedOnlyOffset int) {
	deletedOnlyOffset = -1
	for _, line := range diff {
		if !strings.HasPrefix(line, "-") && !strings.HasPrefix(line, " ") {
			continue
		}

		content := line[1:]
		block = append(block, content)

		if strings.HasPrefix(line, "-") {
			if deletedOnlyOffset == -1 {
				deletedOnlyOffset = len(block) - 1
			}
			deletedOnly = append(deletedOnly, content)
		}
	}
	return block, deletedOnly, deletedOnlyOffset
}

func matchBlock(source, block []string, startLine int) (int, int) {
	if len(block) == 0 {
		return len(source) + 1, len(source)
	}

	normalizedSource := normalizeLines(source)
	normalizedBlock := normalizeLines(block)
	startIndex := max(0, startLine-1)

	for i := startIndex; i <= len(normalizedSource)-len(normalizedBlock); i++ {
		if isMatch(normalizedSource[i:i+len(normalizedBlock)], normalizedBlock) {
			return i + 1, i + len(normalizedBlock)
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

func correctDiffHunks(sourceLines []string, raw, path string) (string, error) {
	var hunks [][]string
	var ch []string
	for _, l := range strings.Split(raw, "\n") {
		if strings.HasPrefix(l, "---") || strings.HasPrefix(l, "+++") {
			continue
		}
		if strings.HasPrefix(l, "@@") {
			if len(ch) > 0 {
				hunks = append(hunks, ch)
			}
			ch = nil
			continue
		}
		if strings.HasPrefix(l, "+") || strings.HasPrefix(l, "-") || strings.HasPrefix(l, " ") || (l == "" && len(ch) > 0) {
			if l == "" {
				ch = append(ch, " ")
			} else {
				ch = append(ch, l)
			}
		}
	}
	if len(ch) > 0 {
		hunks = append(hunks, ch)
	}

	if len(hunks) == 0 {
		return "", nil
	}

	var cp []string
	cp = append(cp, fmt.Sprintf("--- a/%s\n+++ b/%s\n", path, path))
	offset, last := 0, 0
	for _, h := range hunks {
		fullBlock, deletedOnly, deletedOnlyOffset := getTargetBlock(h)

		os, me := matchBlock(sourceLines, fullBlock, last+1)

		if os == -1 && len(deletedOnly) > 0 && isSubstantial(deletedOnly) {
			// Fallback: try to match only the deleted lines if the LLM hallucinated context
			dos, dme := matchBlock(sourceLines, deletedOnly, last+1)
			if dos != -1 {
				os = dos - deletedOnlyOffset
				me = dme + (len(fullBlock) - 1 - (deletedOnlyOffset + len(deletedOnly) - 1))
			}
		}

		if os == -1 {
			return "", fmt.Errorf("failed match")
		}

		last = me

		ac, rc := 0, 0
		for _, l := range h {
			if strings.HasPrefix(l, "+") {
				ac++
			} else if strings.HasPrefix(l, "-") {
				rc++
			}
		}
		ol, nl := (len(h) - ac), (len(h) - rc)
		cp = append(cp, fmt.Sprintf("@@ -%d,%d +%d,%d @@\n", os, ol, os+offset, nl))

		srcLineOffset := 0
		for _, l := range h {
			if strings.HasPrefix(l, " ") {
				lineIdx := os + srcLineOffset - 1
				content := l[1:]
				if lineIdx >= 0 && lineIdx < len(sourceLines) {
					content = sourceLines[lineIdx]
				}
				cp = append(cp, " "+content+"\n")
				srcLineOffset++
			} else if strings.HasPrefix(l, "-") {
				lineIdx := os + srcLineOffset - 1
				content := l[1:]
				if lineIdx >= 0 && lineIdx < len(sourceLines) {
					content = sourceLines[lineIdx]
				}
				cp = append(cp, "-"+content+"\n")
				srcLineOffset++
			} else {
				cp = append(cp, l+"\n")
			}
		}
		offset += nl - ol
	}
	return strings.Join(cp, ""), nil
}

func normalizeLines(lines []string) []string {
	normalized := make([]string, len(lines))
	for i, l := range lines {
		normalized[i] = strings.TrimRight(l, " \t\r\n")
	}
	return normalized
}
