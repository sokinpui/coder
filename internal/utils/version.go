package utils

import (
	"runtime/debug"
)

var Version = "devel"

func GetVersion() string {
	if Version != "devel" {
		return Version
	}

	info, ok := debug.ReadBuildInfo()
	if !ok {
		return "unknown"
	}

	if info.Main.Version != "" && info.Main.Version != "(devel)" {
		return info.Main.Version
	}

	return getVCSVersion(info)
}

func getVCSVersion(info *debug.BuildInfo) string {
	var rev string
	var modified bool

	for _, s := range info.Settings {
		switch s.Key {
		case "vcs.revision":
			rev = s.Value
		case "vcs.modified":
			modified = s.Value == "true"
		}
	}

	if rev == "" {
		return "devel"
	}
	if len(rev) > 7 {
		rev = rev[:7]
	}
	version := "devel-" + rev
	if modified {
		version += "-dirty"
	}
	return version
}
