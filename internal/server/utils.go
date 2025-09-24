package server

import (
	"os"
	"strings"
)

// getShortCwd returns the current working directory, with the user's home directory replaced by '~'.
func getShortCwd() string {
	wd, err := os.Getwd()
	if err != nil {
		return "unknown directory"
	}
	home, err := os.UserHomeDir()
	if err == nil && home != "" && strings.HasPrefix(wd, home) {
		return "~" + strings.TrimPrefix(wd, home)
	}
	return wd
}
