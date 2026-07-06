package sf_test

import (
	"flag"
	"fmt"
	"testing"

	"github.com/sokinpui/sf"
)

var path = flag.String("path", ".", "the directory path to search")

func TestRun(t *testing.T) {
	if !flag.Parsed() {
		flag.Parse()
	}

	roots := []string{*path}
	fileType := ""
	excludes := []string{}
	showHidden := false

	results := sf.Run(roots, fileType, excludes, showHidden)

	for _, p := range results {
		fmt.Println(p)
	}

	t.Logf("Found %d entries", len(results))
}
