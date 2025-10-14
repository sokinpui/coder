package source

import (
	"coder/internal/config"
	"fmt"
	"os/exec"
	"strings"
)

// LoadProjectSource executes `fd` and pipes it to `pcat` to get formatted source code
// of files in the current directory, respecting .gitignore.
func LoadProjectSource(sources *config.FileSources) (string, error) {
	exclusions := []string{
		// common exclusions:
		"*-lock.json",
		"go.sum",
		".git",
		".coder",
		".vscode",
		".idea",
		"dist",
		"bin",
		".env*",
		"*.log",
		"*.svg",
		"*.png",
		"*.jpg",
		"*.wasm",
		"*.png",
		"*.jpg",
		"*.jpeg",
		"*.mp3",
		"*.mp4",
		"*.docx",
		"*.doc",
		"*.xlsx",
		"*.wav",
		"*.gif",
		"*.psd",
		"*.pdf",
		"*.tiff",
		"*.avif",
		"*.jfif",
		"*.pjeg",
		"*.pjp",
		"*.svg",
		"*.wbep",
		"*.bmp",
		"*.ico",
		"*.cur",
		"*.tif",
		"*.mov",
		"*.avi",
		"*.wmv",
		"*.flv",
		"*.mkv",
		"*.webm",
		"*.aac",
		"*.flac",
		"*.aif",
		"*.m4a",
		"__pycache__",
		"*.ogg",
		// ignore binary:
		"*.exe",
		"*.dll",
		"*.so",
		"*.dylib",
		"*.bin",
		"*.class",
		"*.o",
		"*.a",
		"*.lib",
		"*.obj",
		"*.pdb",
		"*.elf",
		"*.img",
		"*.iso",
		"*.dmg",
		// ingore compressed:
		"*.zip",
		"*.tar",
		"*.gz",
		"*.bz2",
		"*.xz",
		"*.7z",
		"*.rar",
		"*.zst",
	}

	var filesFromDirs []string
	if len(sources.FileDirs) > 0 {
		var quotedDirs []string
		for _, d := range sources.FileDirs {
			// quoting for directory paths with spaces or special characters.
			quotedDirs = append(quotedDirs, fmt.Sprintf("'%s'", strings.ReplaceAll(d, "'", "'\\''")))
		}

		var commandBuilder strings.Builder
		commandBuilder.WriteString(fmt.Sprintf("fd . %s --type=file --hidden", strings.Join(quotedDirs, " ")))

		for _, exclusion := range exclusions {
			commandBuilder.WriteString(fmt.Sprintf(" -E '%s'", exclusion))
		}

		command := commandBuilder.String()

		cmd := exec.Command("sh", "-c", command)
		output, err := cmd.Output()
		if err != nil {
			cmdForErr := exec.Command("sh", "-c", command)
			combinedOutput, _ := cmdForErr.CombinedOutput()
			return "", fmt.Errorf("failed to list files with fd: %w\nOutput: %s", err, string(combinedOutput))
		}

		lines := strings.Split(strings.TrimSpace(string(output)), "\n")
		for _, line := range lines {
			if line != "" {
				filesFromDirs = append(filesFromDirs, line)
			}
		}
	}

	allFiles := append(sources.FilePaths, filesFromDirs...)
	if len(allFiles) == 0 {
		return "", nil
	}

	pcatArgs := append([]string{"--no-header"}, allFiles...)
	cmd := exec.Command("pcat", pcatArgs...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to load project source with pcat: %w\nOutput: %s", err, string(output))
	}
	return string(output), nil
}
