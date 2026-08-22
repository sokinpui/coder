package utils

import (
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/atotto/clipboard"
)

func Copy(content, customCmd string) error {
	if customCmd == "" {
		return clipboard.WriteAll(content)
	}

	parts := strings.Fields(customCmd)
	if len(parts) == 0 {
		return fmt.Errorf("empty copy command")
	}

	cmd := exec.Command(parts[0], parts[1:]...)
	cmd.Stdin = strings.NewReader(content)
	return cmd.Run()
}

func CopyImage(imagePath string, data []byte) error {
	absPath := imagePath
	if absPath != "" && !filepath.IsAbs(absPath) {
		absPath = filepath.Join(GetProjectRoot(), absPath)
	}

	if absPath != "" {
		if _, err := os.Stat(absPath); err == nil {
			return copyImageFile(absPath)
		}
	}

	if len(data) == 0 {
		return fmt.Errorf("no image path or data provided")
	}

	tmpFile, err := os.CreateTemp("", "coder-copy-*.png")
	if err != nil {
		return err
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.Write(data); err != nil {
		tmpFile.Close()
		return err
	}
	if err := tmpFile.Close(); err != nil {
		return err
	}

	return copyImageFile(tmpFile.Name())
}

func copyImageFile(absPath string) error {
	switch runtime.GOOS {
	case "darwin":
		return copyDarwinImage(absPath)
	case "linux":
		return copyLinuxImage(absPath)
	default:
		return fmt.Errorf("image copy not supported on %s", runtime.GOOS)
	}
}

func copyDarwinImage(absPath string) error {
	if _, err := exec.LookPath("osascript"); err != nil {
		return fmt.Errorf("osascript not found")
	}

	classType := "«class PNGf»"
	ext := strings.ToLower(filepath.Ext(absPath))
	if ext == ".jpg" || ext == ".jpeg" {
		classType = "JPEG picture"
	}

	script := fmt.Sprintf("set the clipboard to (read (POSIX file (item 1 of argv)) as %s)", classType)
	cmd := exec.Command("osascript", "-e", "on run argv", "-e", script, "-e", "end run", absPath)
	return cmd.Run()
}

func copyLinuxImage(absPath string) error {
	isWayland := os.Getenv("WAYLAND_DISPLAY") != "" || os.Getenv("XDG_SESSION_TYPE") == "wayland"
	if isWayland {
		return copyWaylandImage(absPath)
	}
	return copyX11Image(absPath)
}

func copyWaylandImage(absPath string) error {
	if _, err := exec.LookPath("wl-copy"); err != nil {
		return fmt.Errorf("wl-copy not found")
	}

	mimeType := getMimeType(absPath)
	file, err := os.Open(absPath)
	if err != nil {
		return err
	}
	defer file.Close()

	cmd := exec.Command("wl-copy", "--type", mimeType)
	cmd.Stdin = file
	return cmd.Run()
}

func copyX11Image(absPath string) error {
	if _, err := exec.LookPath("xclip"); err != nil {
		return fmt.Errorf("xclip not found")
	}

	mimeType := getMimeType(absPath)
	cmd := exec.Command("xclip", "-selection", "clipboard", "-t", mimeType, "-i", absPath)
	return cmd.Run()
}

func getMimeType(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".webp":
		return "image/webp"
	default:
		return "image/png"
	}
}

func PasteText() (string, error) {
	return clipboard.ReadAll()
}

func PasteCustom(customCmd string) ([]byte, string, error) {
	parts := strings.Fields(customCmd)
	if len(parts) == 0 {
		return nil, "", fmt.Errorf("empty paste command")
	}

	output, err := exec.Command(parts[0], parts[1:]...).Output()
	if err != nil {
		return nil, "", err
	}

	return output, http.DetectContentType(output), nil
}

func GetImageFromClipboard() ([]byte, string, error) {
	switch runtime.GOOS {
	case "darwin":
		return getDarwinImage()
	case "linux":
		return getLinuxImage()
	default:
		return nil, "", fmt.Errorf("image paste not supported on %s", runtime.GOOS)
	}
}

func getDarwinImage() ([]byte, string, error) {
	if _, err := exec.LookPath("pngpaste"); err != nil {
		return nil, "", fmt.Errorf("pngpaste not found")
	}

	tempPath := os.TempDir() + "/coder-paste.png"
	defer os.Remove(tempPath)

	if err := exec.Command("pngpaste", tempPath).Run(); err != nil {
		return nil, "", err
	}

	data, err := os.ReadFile(tempPath)
	if err != nil {
		return nil, "", err
	}

	return data, "image/png", nil
}

func getLinuxImage() ([]byte, string, error) {
	isWayland := os.Getenv("WAYLAND_DISPLAY") != "" || os.Getenv("XDG_SESSION_TYPE") == "wayland"

	if isWayland {
		return getWaylandImage()
	}
	return getX11Image()
}

func getWaylandImage() ([]byte, string, error) {
	if _, err := exec.LookPath("wl-paste"); err != nil {
		return nil, "", fmt.Errorf("wl-paste not found")
	}

	output, err := exec.Command("wl-paste", "-t", "image/png").Output()
	if err != nil {
		return nil, "", err
	}

	return output, "image/png", nil
}

func getX11Image() ([]byte, string, error) {
	if _, err := exec.LookPath("xclip"); err != nil {
		return nil, "", fmt.Errorf("xclip not found")
	}

	targets, err := exec.Command("xclip", "-selection", "clipboard", "-t", "TARGETS", "-o").Output()
	if err != nil || !strings.Contains(string(targets), "image/png") {
		return nil, "", fmt.Errorf("no image/png on clipboard")
	}

	output, err := exec.Command("xclip", "-selection", "clipboard", "-t", "image/png", "-o").Output()
	if err != nil {
		return nil, "", err
	}

	return output, "image/png", nil
}
