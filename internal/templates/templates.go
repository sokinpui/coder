package templates

import (
	"embed"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/sokinpui/coder/internal/project"
)

type Template struct {
	Name string
	Path string
}

//go:embed builtin
var builtinFS embed.FS

var (
	ensureBuiltinOnce sync.Once
)

func getGlobalTemplatesDir() string {
	home, err := os.UserHomeDir()
	if err == nil && home != "" {
		return filepath.Join(home, ".config", "coder", "templates")
	}
	return filepath.Join(os.TempDir(), "coder", "templates")
}

func EnsureBuiltinTemplates() {
	ensureBuiltinOnce.Do(func() {
		targetDir := getGlobalTemplatesDir()
		if targetDir == "" {
			return
		}
		_ = installBuiltinTemplates(targetDir)
	})
}

func installBuiltinTemplates(targetDir string) error {
	return fs.WalkDir(builtinFS, "builtin", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel("builtin", path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		dest := filepath.Join(targetDir, rel)
		if d.IsDir() {
			return os.MkdirAll(dest, 0755)
		}
		if _, err := os.Stat(dest); err == nil {
			return nil
		}
		data, err := builtinFS.ReadFile(path)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
			return err
		}
		return os.WriteFile(dest, data, 0644)
	})
}

func Find(name string) (Template, bool) {
	EnsureBuiltinTemplates()

	for _, dir := range getTemplateDirs() {
		candidate := filepath.Join(dir, name+".md")
		if _, err := os.Stat(candidate); err == nil {
			return Template{
				Name: name,
				Path: relativizePath(candidate),
			}, true
		}

		candidateRaw := filepath.Join(dir, name)
		if info, err := os.Stat(candidateRaw); err == nil && !info.IsDir() {
			return Template{
				Name: name,
				Path: relativizePath(candidateRaw),
			}, true
		}
	}

	return Template{}, false
}

func getTemplateDirs() []string {
	var dirs []string
	if root := project.Root(); root != "" {
		dirs = append(dirs, filepath.Join(root, ".coder", "templates"))
	}
	if gDir := getGlobalTemplatesDir(); gDir != "" {
		dirs = append(dirs, gDir)
	}
	return dirs
}

func relativizePath(path string) string {
	abs, err := filepath.Abs(path)
	if err != nil {
		return filepath.ToSlash(path)
	}

	cwd, err := os.Getwd()
	if err == nil {
		if rel, err := filepath.Rel(cwd, abs); err == nil && !strings.HasPrefix(rel, "..") {
			return filepath.ToSlash(rel)
		}
	}

	if root := project.Root(); root != "" {
		if rel, err := filepath.Rel(root, abs); err == nil && !strings.HasPrefix(rel, "..") {
			return filepath.ToSlash(rel)
		}
	}

	return filepath.ToSlash(project.ShortenPath(abs))
}
