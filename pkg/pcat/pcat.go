package pcat

import (
	"bufio"
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"sort"
	"strings"
	"sync"

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

type formattedFile struct {
	index int
	text  string
}

func formatFiles(files []string, withLineNumbers bool) (string, error) {
	if len(files) == 0 {
		return "", nil
	}

	workerCount := min(len(files), runtime.NumCPU())
	if workerCount < 1 {
		workerCount = 1
	}

	jobs := make(chan int, len(files))
	for i := range files {
		jobs <- i
	}
	close(jobs)

	results := make(chan formattedFile, len(files))
	var wg sync.WaitGroup

	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for idx := range jobs {
				text := formatSingleFile(files[idx], withLineNumbers)
				results <- formattedFile{index: idx, text: text}
			}
		}()
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	ordered := make([]string, len(files))
	for res := range results {
		ordered[res.index] = res.text
	}

	var out strings.Builder
	for _, text := range ordered {
		if text == "" {
			continue
		}
		out.WriteString(text)
	}
	result := strings.TrimSuffix(out.String(), "\n")
	if result == "" {
		return "", nil
	}
	return result + "\n---\n", nil
}

func formatSingleFile(file string, withLineNumbers bool) string {
	content, err := os.ReadFile(file)
	if err != nil || bytes.Contains(content, []byte{0}) {
		return ""
	}

	var out strings.Builder
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

	return out.String()
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

type resolvedEntry struct {
	index int
	path  string
}

func deduplicate(paths []string) []string {
	if len(paths) <= 1 {
		return paths
	}

	workerCount := min(len(paths), runtime.NumCPU())
	if workerCount < 1 {
		workerCount = 1
	}

	jobs := make(chan int, len(paths))
	for i := range paths {
		jobs <- i
	}
	close(jobs)

	results := make(chan resolvedEntry, len(paths))
	var wg sync.WaitGroup

	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for idx := range jobs {
				resolved := resolveForDedup(paths[idx])
				results <- resolvedEntry{index: idx, path: resolved}
			}
		}()
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	resolvedPaths := make([]string, len(paths))
	for res := range results {
		resolvedPaths[res.index] = res.path
	}

	var uniquePaths []string
	seen := make(map[string]struct{}, len(paths))
	for i, resolved := range resolvedPaths {
		if _, ok := seen[resolved]; ok {
			continue
		}
		seen[resolved] = struct{}{}
		uniquePaths = append(uniquePaths, paths[i])
	}
	return uniquePaths
}

func resolveForDedup(p string) string {
	resolvedPath, err := filepath.EvalSymlinks(p)
	if os.IsNotExist(err) {
		absPath, _ := filepath.Abs(p)
		return absPath
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not resolve path %s: %v\n", p, err)
		return p
	}
	return resolvedPath
}

func filterExcluded(paths []string, excludePatterns []string) ([]string, error) {
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
