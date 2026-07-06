package sf

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"

	"github.com/monochromegane/go-gitignore"
)

type Matcher struct {
	ignoreMatchers []gitignore.IgnoreMatcher
	excludes       []string
	root           string
	showHidden     bool
}

func NewMatcher(root string, excludes []string, showHidden bool) *Matcher {
	baseDir := root
	if info, err := os.Stat(root); err == nil && !info.IsDir() {
		baseDir = filepath.Dir(root)
	}

	ignorePath := filepath.Join(baseDir, ".gitignore")

	m := &Matcher{
		excludes:   excludes,
		root:       root,
		showHidden: showHidden,
	}

	if globalPath := getGlobalGitIgnorePath(); globalPath != "" {
		if gi, err := gitignore.NewGitIgnore(globalPath, baseDir); err == nil {
			m.ignoreMatchers = append(m.ignoreMatchers, gi)
		}
	}

	gi, _ := gitignore.NewGitIgnore(ignorePath, baseDir)
	m.ignoreMatchers = append(m.ignoreMatchers, gi)

	return m
}

func (m *Matcher) ShouldSkip(path string, info os.DirEntry) bool {
	if !m.showHidden && m.isHidden(info.Name()) {
		return true
	}

	if m.isGitDir(info) {
		return true
	}

	for _, matcher := range m.ignoreMatchers {
		if matcher != nil && matcher.Match(path, info.IsDir()) {
			return true
		}
	}

	if m.matchExcludes(path, info) {
		return true
	}

	return false
}

func (m *Matcher) matchExcludes(path string, info os.DirEntry) bool {
	if len(m.excludes) == 0 {
		return false
	}

	rel := m.getRelativePath(path)
	name := info.Name()

	for _, pattern := range m.excludes {
		pattern = filepath.ToSlash(pattern)

		if matched, _ := filepath.Match(pattern, name); matched {
			return true
		}

		if matched, _ := filepath.Match(pattern, rel); matched {
			return true
		}

		if strings.Contains(pattern, "/") && m.matchSegments(rel, pattern) {
			return true
		}
	}
	return false
}

func (m *Matcher) matchSegments(rel, pattern string) bool {
	segments := strings.Split(rel, "/")
	for i := 1; i < len(segments); i++ {
		subPath := strings.Join(segments[i:], "/")
		if matched, _ := filepath.Match(pattern, subPath); matched {
			return true
		}
	}
	return false
}

func (m *Matcher) isHidden(name string) bool {
	return len(name) > 1 && strings.HasPrefix(name, ".")
}

func (m *Matcher) isGitDir(info os.DirEntry) bool {
	return info.IsDir() && info.Name() == ".git"
}

func (m *Matcher) getRelativePath(path string) string {
	rel, err := filepath.Rel(m.root, path)
	if err != nil {
		return filepath.ToSlash(path)
	}
	return filepath.ToSlash(rel)
}

func getGlobalGitIgnorePath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}

	gitConfigPaths := []string{
		filepath.Join(home, ".gitconfig"),
		filepath.Join(home, ".config", "git", "config"),
	}

	for _, p := range gitConfigPaths {
		if path := parseGitConfigForExcludes(p); path != "" {
			return expandHome(path, home)
		}
	}

	xdgPath := filepath.Join(home, ".config", "git", "ignore")
	if _, err := os.Stat(xdgPath); err == nil {
		return xdgPath
	}

	return ""
}

func parseGitConfigForExcludes(configPath string) string {
	file, err := os.Open(configPath)
	if err != nil {
		return ""
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	inCoreSection := false
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
			continue
		}

		if strings.HasPrefix(line, "[") {
			inCoreSection = strings.Contains(line, "[core]")
			continue
		}

		if inCoreSection && strings.Contains(line, "excludesfile") {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				return strings.TrimSpace(parts[1])
			}
		}
	}
	return ""
}

func expandHome(path, home string) string {
	if strings.HasPrefix(path, "~/") {
		return filepath.Join(home, path[2:])
	}
	return path
}
