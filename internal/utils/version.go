package utils

import (
	"runtime/debug"
)

// GetVersion returns the version of the application.
// It tries to read the version from build info, falling back to the VCS revision if available.
func GetVersion() string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return "devel"
	}

	if info.Main.Version != "" && info.Main.Version != "(devel)" {
		return info.Main.Version
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
