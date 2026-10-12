package templates

import (
	"embed"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/sokinpui/coder/internal/project"
	"gopkg.in/yaml.v3"
)

type Template struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
	Path        string `yaml:"path"`
	Content     string `yaml:"content"`
}

type TemplateMetadata struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
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

	for _, t := range Discover() {
		if strings.EqualFold(t.Name, name) {
			return t, true
		}
	}

	return Template{}, false
}

func Discover() []Template {
	EnsureBuiltinTemplates()

	dirs := getTemplateDirs()
	seen := make(map[string]struct{})
	var result []Template

	for _, dir := range dirs {
		templatesInDir := scanDir(dir)
		for _, t := range templatesInDir {
			norm := strings.ToLower(t.Name)
			if _, ok := seen[norm]; ok {
				continue
			}
			seen[norm] = struct{}{}
			result = append(result, t)
		}
	}

	return result
}

func scanDir(dir string) []Template {
	if _, err := os.Stat(dir); err != nil {
		return nil
	}

	var templates []Template
	_ = filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}

		name := d.Name()
		if path != dir && strings.HasPrefix(name, ".") {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		if d.IsDir() {
			return nil
		}

		if !strings.EqualFold(filepath.Ext(name), ".md") {
			return nil
		}

		defaultName := strings.TrimSuffix(name, filepath.Ext(name))
		tmpl, err := loadTemplate(path, defaultName)
		if err == nil {
			templates = append(templates, tmpl)
		}
		return nil
	})

	return templates
}

func loadTemplate(filePath, defaultName string) (Template, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return Template{}, err
	}

	meta, body, _ := parseFrontmatter(data)
	name := meta.Name
	if name == "" {
		name = defaultName
	}

	displayPath := relativizePath(filePath)

	return Template{
		Name:        name,
		Description: meta.Description,
		Path:        displayPath,
		Content:     strings.TrimSpace(body),
	}, nil
}

func parseFrontmatter(data []byte) (TemplateMetadata, string, bool) {
	content := string(data)
	if !strings.HasPrefix(content, "---") {
		return TemplateMetadata{}, content, false
	}

	rest := strings.TrimPrefix(content, "---")
	if strings.HasPrefix(rest, "\r\n") {
		rest = rest[2:]
	} else if strings.HasPrefix(rest, "\n") {
		rest = rest[1:]
	} else {
		return TemplateMetadata{}, content, false
	}

	marker := "\n---"
	endIdx := strings.Index(rest, marker)
	if endIdx == -1 {
		marker = "\n..."
		endIdx = strings.Index(rest, marker)
	}
	if endIdx == -1 {
		marker = "\r\n---"
		endIdx = strings.Index(rest, marker)
	}
	if endIdx == -1 {
		marker = "\r\n..."
		endIdx = strings.Index(rest, marker)
	}
	if endIdx == -1 {
		return TemplateMetadata{}, content, false
	}

	frontmatter := rest[:endIdx]
	body := rest[endIdx+len(marker):]
	body = strings.TrimLeft(body, "\r\n")

	var meta TemplateMetadata
	if err := yaml.Unmarshal([]byte(frontmatter), &meta); err != nil {
		return TemplateMetadata{}, content, false
	}
	return meta, body, true
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
