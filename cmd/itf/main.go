package main

import (
	"fmt"
	"os"

	"github.com/sokinpui/coder/pkg/itf"
	"github.com/sokinpui/coder/pkg/version"
)

func main() {
	itf.SetVersion(version.Get())
	if err := itf.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
