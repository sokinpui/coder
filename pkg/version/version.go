package version

import "runtime/debug"

var Version = "devel"

func Get() string {
	if Version != "devel" {
		return Version
	}

	if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "" {
		return info.Main.Version
	}
	return Version
}
