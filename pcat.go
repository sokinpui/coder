package pcat

import (
	"bufio"
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
)

type Config struct {
	Directories     []string
	Extensions      []string
	SpecificFiles   []string
	ExcludePatterns []string
	WithLineNumbers bool
	Hidden          bool
	ListOnly        bool
	ToClipboard     bool
}

// Read reads a list of files and returns a formatted string with their contents.
func Read(files []string, config Config) (string, error) {
	if len(files) == 0 {
		return "", nil
	}

	var out strings.Builder
	filesFormatted := 0

	for _, file := range files {
		content, err := os.ReadFile(file)
		if err != nil {
			continue
		}
		if bytes.Contains(content, []byte{0}) {
			continue
		}
		filesFormatted++

		out.WriteString(fmt.Sprintf("`%s`\n", file))

		lang := strings.TrimPrefix(filepath.Ext(file), ".")
		if lang == "" {
			lang = "txt"
		}

		fence := "```"
		if lang == "md" || lang == "markdown" {
			fence = "````"
			lang = "markdown"
		}

		out.WriteString(fmt.Sprintf("%s%s\n", fence, lang))

		if config.WithLineNumbers {
			scanner := bufio.NewScanner(strings.NewReader(string(content)))
			for i := 1; scanner.Scan(); i++ {
				out.WriteString(fmt.Sprintf("%4d | %s\n", i, scanner.Text()))
			}
		} else {
			out.Write(content)
		}

		if len(content) > 0 && !strings.HasSuffix(string(content), "\n") {
			out.WriteString("\n")
		}
		out.WriteString(fmt.Sprintf("%s\n\n", fence))
	}

	if filesFormatted == 0 {
		return "", nil
	}

	result := strings.TrimSuffix(out.String(), "\n")
	return result + "\n---\n", nil
}

type App struct {
	config Config
}

func New(config Config) *App {
	return &App{config: config}
}

func (a *App) Run() (string, error) {
	directoryFiles, err := a.findDirectoryFiles()
	if err != nil {
		return "", fmt.Errorf("finding files: %w", err)
	}

	allFiles := append(directoryFiles, a.config.SpecificFiles...)
	uniqueFiles := deduplicate(allFiles)

	filteredFiles, err := a.filterExcluded(uniqueFiles)
	if err != nil {
		return "", fmt.Errorf("filtering files: %w", err)
	}

	if a.config.ListOnly {
		if len(filteredFiles) == 0 {
			return "", nil
		}
		return strings.Join(filteredFiles, "\n") + "\n", nil
	}

	return a.formatOutput(filteredFiles)
}

func (a *App) findDirectoryFiles() ([]string, error) {
	fileSet := make(map[string]struct{})

	for _, dir := range a.config.Directories {
		err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}

			if !a.config.Hidden && isHidden(path, dir) {
				if d.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}

			if d.IsDir() {
				return nil
			}

			if hasValidExtension(path, a.config.Extensions) {
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
	for _, part := range strings.Split(relPath, string(filepath.Separator)) {
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
	for _, ext := range extensions {
		if fileExt == ext {
			return true
		}
	}
	return false
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

func (a *App) filterExcluded(paths []string) ([]string, error) {
	if len(a.config.ExcludePatterns) == 0 {
		return paths, nil
	}

	var filtered []string
	for _, path := range paths {
		isExcluded := false
		posixPath := filepath.ToSlash(path)
		for _, pattern := range a.config.ExcludePatterns {
			match, err := doublestar.Match(pattern, posixPath)
			if err != nil {
				return nil, fmt.Errorf("invalid exclude pattern '%s': %w", pattern, err)
			}
			if match {
				isExcluded = true
				break
			}
		}
		if !isExcluded {
			filtered = append(filtered, path)
		}
	}
	return filtered, nil
}

func (a *App) formatOutput(files []string) (string, error) {
	return Read(files, a.config)
}
