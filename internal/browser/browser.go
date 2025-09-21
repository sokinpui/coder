package browser

import (
	"fmt"
	"os/exec"
	"runtime"
)

// Open opens the specified URL in the default browser of the user.
func Open(url string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "linux":
		cmd = exec.Command("xdg-open", url)
	case "windows":
		// Using 'start' is more reliable and handles spaces in URLs correctly.
		cmd = exec.Command("cmd", "/c", "start", url)
	case "darwin":
		cmd = exec.Command("open", url)
	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}
	// Use Start() to launch the browser in a non-blocking way.
	return cmd.Start()
}
