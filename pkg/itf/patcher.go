package itf

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
)

var filePathRegex = regexp.MustCompile(`(?m)^\+\+\+ b/(?P<path>.*?)(\s|$)`)

func ExtractPathFromDiff(content string) string {
	if match := filePathRegex.FindStringSubmatch(content); len(match) > 1 {
		return strings.TrimSpace(match[1])
	}
	return ""
}

func GeneratePatchedContents(diffs []DiffBlock, resolver *PathResolver, extensions []string, renameMap map[string]string) ([]FileChange, []string, error) {
	var changes []FileChange
	var failed []string
	for _, d := range diffs {
		abs := resolver.Resolve(d.FilePath)
		sourcePath := abs
		if renameMap != nil {
			if s, ok := renameMap[abs]; ok {
				sourcePath = s
			}
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

		changes = append(changes, FileChange{
			Path:     abs,
			Content:  applied,
			Source:   "diff",
			RawBlock: fmt.Sprintf("```diff\n%s\n```", d.RawContent),
		})
	}
	return changes, failed, nil
}

func CorrectDiff(diff DiffBlock, resolver *PathResolver, extensions []string, sourcePath string) (string, error) {
	src := ""
	if sourcePath != "" {
		if _, err := os.Stat(sourcePath); err == nil {
			src = sourcePath
		}
	}

	var lines []string
	if src != "" {
		if content, err := os.ReadFile(src); err == nil {
			lines = strings.Split(string(content), "\n")
		}
	}
	return correctDiffHunks(lines, diff.RawContent, diff.FilePath)
}

func applyPatch(sourcePath, patch string) []string {
	var sourceLines []string
	if sourcePath != "" {
		srcPath := sourcePath
		if content, err := os.ReadFile(srcPath); err == nil {
			if len(content) > 0 {
				sourceLines = strings.Split(strings.TrimSuffix(string(content), "\n"), "\n")
			}
		}
	}
	return applyUnifiedDiff(sourceLines, patch)
}

func applyUnifiedDiff(source []string, patch string) []string {
	patchLines := strings.Split(patch, "\n")
	var result []string
	srcIdx := 0

	for i := 0; i < len(patchLines); i++ {
		line := patchLines[i]
		if !strings.HasPrefix(line, "@@ -") {
			continue
		}

		parts := strings.Split(line, " ")
		if len(parts) < 2 {
			continue
		}

		rangePart := strings.TrimPrefix(parts[1], "-")
		rangeSplit := strings.Split(rangePart, ",")
		if len(rangeSplit) == 0 || rangeSplit[0] == "" {
			continue
		}
		start, _ := strconv.Atoi(rangeSplit[0])

		startIdx := max(0, start-1)

		for srcIdx < startIdx && srcIdx < len(source) {
			result = append(result, source[srcIdx])
			srcIdx++
		}

		i++
		for i < len(patchLines) {
			hunkLine := patchLines[i]
			if strings.HasPrefix(hunkLine, "@@") || strings.HasPrefix(hunkLine, "---") || strings.HasPrefix(hunkLine, "+++") {
				i--
				break
			}

			if strings.HasPrefix(hunkLine, "+") {
				result = append(result, hunkLine[1:])
			} else if strings.HasPrefix(hunkLine, "-") {
				srcIdx++
			} else if strings.HasPrefix(hunkLine, " ") || hunkLine == "" {
				if srcIdx < len(source) {
					result = append(result, source[srcIdx])
				}
				srcIdx++
			}
			i++
		}
	}

	for srcIdx < len(source) {
		result = append(result, source[srcIdx])
		srcIdx++
	}

	return result
}
