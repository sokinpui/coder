package utils

import (
	"fmt"
	"net/http"
	"os"
	"os/exec"
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
