package main

import (
	"fmt"
	"os"

	"github.com/sokinpui/itf"
)

func main() {
	if err := itf.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
