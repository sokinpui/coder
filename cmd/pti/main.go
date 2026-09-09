package main

import (
	"fmt"
	"os"

	"github.com/sokinpui/coder/pkg/pti"
	"github.com/sokinpui/coder/pkg/version"
)

func main() {
	pti.SetVersion(version.Get())
	if err := pti.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
