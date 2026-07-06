package utils

import "github.com/sokinpui/coder/pkg/version"

func GetVersion() string {
	return version.Get()
}
