package utils

import (
	"os/exec"
	"runtime/debug"
	"strings"
)

func GetVersion() string {
	info, ok := debug.ReadBuildInfo()
	if ok && info.Main.Version != "" && info.Main.Version != "(devel)" {
		return info.Main.Version
	}

	if tag := getGitTag(); tag != "" {
		return tag
	}

	return getRevision(info)
}

func getRevision(info *debug.BuildInfo) string {
	if info == nil {
		return "devel"
	}

	for _, setting := range info.Settings {
		if setting.Key == "vcs.revision" {
			rev := setting.Value
			if len(rev) > 7 {
				rev = rev[:7]
			}
			return "devel-" + rev
		}
	}

	return "devel"
}

func getGitTag() string {
	cmd := exec.Command("git", "describe", "--tags", "--always", "--dirty")
	output, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(output))
}
