package pcat

import (
	"bufio"
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
)

func Run(specificFiles, directories, extensions, excludePatterns []string, withLineNumbers, hidden, listOnly bool) (string, error) {
	if len(directories) > 0 && len(extensions) == 0 {
		extensions = []string{"any"}
	}

	directoryFiles, err := findDirectoryFiles(directories, extensions, hidden)
	if err != nil {
		return "", fmt.Errorf("finding files: %w", err)
	}

	allFiles := deduplicate(append(directoryFiles, specificFiles...))
	filteredFiles, err := filterExcluded(allFiles, excludePatterns)
	if err != nil {
		return "", fmt.Errorf("filtering files: %w", err)
	}

	if len(filteredFiles) == 0 {
		return "", nil
	}

	if listOnly {
		return strings.Join(filteredFiles, "\n") + "\n", nil
	}

	return formatFiles(filteredFiles, withLineNumbers)
}

func formatFiles(files []string, withLineNumbers bool) (string, error) {
	if len(files) == 0 {
		return "", nil
	}

	var out strings.Builder
	for _, file := range files {
		content, err := os.ReadFile(file)
		if err != nil || bytes.Contains(content, []byte{0}) {
			continue
		}

		out.WriteString(fmt.Sprintf("`%s`\n", file))
		lang := getLang(file)
		fence := getFence(lang)

		out.WriteString(fmt.Sprintf("%s%s\n", fence, lang))
		if withLineNumbers {
			writeWithLineNumbers(&out, content)
		} else {
			out.Write(content)
		}
		ensureNewline(&out, content)
		out.WriteString(fmt.Sprintf("%s\n\n", fence))
	}
	result := strings.TrimSuffix(out.String(), "\n")
	return result + "\n---\n", nil
}

func getLang(file string) string {
	lang := strings.TrimPrefix(filepath.Ext(file), ".")
	if lang == "" {
		return "txt"
	}
	return lang
}

func getFence(lang string) string {
	if lang == "md" || lang == "markdown" {
		return "````"
	}
	return "```"
}

func writeWithLineNumbers(out *strings.Builder, content []byte) {
	scanner := bufio.NewScanner(bytes.NewReader(content))
	for i := 1; scanner.Scan(); i++ {
		out.WriteString(fmt.Sprintf("%4d | %s\n", i, scanner.Text()))
	}
}

func ensureNewline(out *strings.Builder, content []byte) {
	if len(content) > 0 && !bytes.HasSuffix(content, []byte("\n")) {
		out.WriteString("\n")
	}
}

func findDirectoryFiles(directories, extensions []string, includeHidden bool) ([]string, error) {
	fileSet := make(map[string]struct{})

	for _, dir := range directories {
		err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}

			if !includeHidden && isHidden(path, dir) && d.IsDir() {
				return filepath.SkipDir
			}

			if !includeHidden && isHidden(path, dir) {
				return nil
			}

			if d.IsDir() {
				return nil
			}

			if hasValidExtension(path, extensions) {
				fileSet[path] = struct{}{}
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
	}

	files := make([]string, 0, len(fileSet))
	for file := range fileSet {
		files = append(files, file)
	}
	sort.Strings(files)
	return files, nil
}

func isHidden(path, baseDir string) bool {
	relPath, err := filepath.Rel(baseDir, path)
	if err != nil {
		return false
	}
	for part := range strings.SplitSeq(relPath, string(filepath.Separator)) {
		if strings.HasPrefix(part, ".") && part != "." && part != ".." {
			return true
		}
	}
	return false
}

func hasValidExtension(path string, extensions []string) bool {
	if len(extensions) == 0 {
		return false
	}
	if len(extensions) > 0 && extensions[0] == "any" {
		return true
	}
	fileExt := strings.TrimPrefix(filepath.Ext(path), ".")
	return slices.Contains(extensions, fileExt)
}

func deduplicate(paths []string) []string {
	var uniquePaths []string
	seen := make(map[string]struct{})

	for _, p := range paths {
		resolvedPath, err := filepath.EvalSymlinks(p)
		if os.IsNotExist(err) {
			resolvedPath, _ = filepath.Abs(p)
		} else if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: could not resolve path %s: %v\n", p, err)
			resolvedPath = p
		}

		if _, ok := seen[resolvedPath]; !ok {
			uniquePaths = append(uniquePaths, p)
			seen[resolvedPath] = struct{}{}
		}
	}
	return uniquePaths
}

func filterExcluded(paths []string, excludePatterns []string) ([]string, error) {
	if len(excludePatterns) == 0 {
		return paths, nil
	}

	var filtered []string
	for _, path := range paths {
		excluded := false
		posixPath := filepath.ToSlash(path)
		for _, pattern := range excludePatterns {
			match, err := doublestar.Match(pattern, posixPath)
			if err != nil {
				return nil, fmt.Errorf("invalid exclude pattern '%s': %w", pattern, err)
			}
			if match {
				excluded = true
				break
			}
		}
		if !excluded {
			filtered = append(filtered, path)
		}
	}
	return filtered, nil
}
