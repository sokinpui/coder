package skills

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/sokinpui/coder/internal/project"
	"gopkg.in/yaml.v3"
)

type Skill struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
	Path        string `yaml:"path"`
}

type SkillMetadata struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
}

func Discover() []Skill {
	dirs := getSkillDirs()
	seen := make(map[string]struct{})
	var result []Skill

	for _, dir := range dirs {
		skillsInDir := scanDir(dir)
		for _, s := range skillsInDir {
			if _, ok := seen[s.Name]; ok {
				continue
			}
			seen[s.Name] = struct{}{}
			result = append(result, s)
		}
	}

	return result
}

func AppendSkillsPrompt(base string) string {
	if strings.Contains(base, "# Available Skills") {
		return base
	}

	skillList := Discover()
	if len(skillList) == 0 {
		return base
	}

	yamlData, err := yaml.Marshal(skillList)
	if err != nil {
		return base
	}

	var sb strings.Builder
	sb.WriteString(base)
	sb.WriteString("\n\n# Available Skills\n\n")
	sb.WriteString("When a task matches a skill's description, use the `read` tool to read the skill file at the given path before proceeding.\n\n")
	sb.WriteString("```yaml\n")
	sb.Write(yamlData)
	sb.WriteString("```")

	return sb.String()
}

func getSkillDirs() []string {
	var dirs []string
	if root := project.Root(); root != "" {
		dirs = append(dirs, filepath.Join(root, ".coder", "skills"))
	}
	if home, err := os.UserHomeDir(); err == nil && home != "" {
		dirs = append(dirs, filepath.Join(home, ".config", "coder", "skills"))
	}
	return dirs
}

func scanDir(dir string) []Skill {
	if _, err := os.Stat(dir); err != nil {
		return nil
	}

	var skills []Skill
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
			if path == dir {
				return nil
			}

			skillFile := findSkillFileInDir(path)
			if skillFile == "" {
				return nil
			}

			skill, err := loadSkill(skillFile, name)
			if err == nil {
				skills = append(skills, skill)
			}
			return filepath.SkipDir
		}

		if !strings.EqualFold(filepath.Ext(name), ".md") {
			return nil
		}

		defaultName := strings.TrimSuffix(name, filepath.Ext(name))
		if strings.EqualFold(name, "skill.md") {
			defaultName = filepath.Base(filepath.Dir(path))
		}

		skill, err := loadSkill(path, defaultName)
		if err == nil {
			skills = append(skills, skill)
		}
		return nil
	})

	return skills
}

func findSkillFileInDir(dir string) string {
	candidate := filepath.Join(dir, "SKILL.md")
	if _, err := os.Stat(candidate); err == nil {
		return candidate
	}
	candidateLower := filepath.Join(dir, "skill.md")
	if _, err := os.Stat(candidateLower); err == nil {
		return candidateLower
	}
	return ""
}

func loadSkill(filePath, defaultName string) (Skill, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return Skill{}, err
	}

	meta, _ := parseFrontmatter(data)
	name := meta.Name
	if name == "" {
		name = defaultName
	}

	displayPath := relativizePath(filePath)

	return Skill{
		Name:        name,
		Description: meta.Description,
		Path:        displayPath,
	}, nil
}

func parseFrontmatter(data []byte) (SkillMetadata, bool) {
	content := string(data)
	if !strings.HasPrefix(content, "---") {
		return SkillMetadata{}, false
	}

	rest := strings.TrimPrefix(content, "---")
	if strings.HasPrefix(rest, "\r\n") {
		rest = rest[2:]
	} else if strings.HasPrefix(rest, "\n") {
		rest = rest[1:]
	} else {
		return SkillMetadata{}, false
	}

	endIdx := strings.Index(rest, "\n---")
	if endIdx == -1 {
		endIdx = strings.Index(rest, "\n...")
	}
	if endIdx == -1 {
		return SkillMetadata{}, false
	}

	frontmatter := rest[:endIdx]
	var meta SkillMetadata
	if err := yaml.Unmarshal([]byte(frontmatter), &meta); err != nil {
		return SkillMetadata{}, false
	}
	return meta, true
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

	return filepath.ToSlash(abs)
}
