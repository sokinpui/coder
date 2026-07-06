# Using pcat as a library

The `pcat` package can be integrated into other Go applications to find and format source code from the filesystem.

## Installation

To use the `pcat` package in your project, install it using `go get`:

```sh
go get github.com/sokinpui/coder
```

## Example

Here is a basic example of how to use the `pcat` library to read all `.go` files from the current directory, excluding `go.mod`.

```go
package main

import (
	"fmt"
	"log"

	"github.com/sokinpui/coder/pkg/pcat"
)

func main() {
	// Execute pcat logic through a single entry point.
	// pcat.Run(specificFiles, directories, extensions, excludePatterns, withLineNumbers, hidden, listOnly)
	output, err := pcat.Run(
		nil,               // specificFiles
		[]string{"."},     // directories
		[]string{"go"},    // extensions
		[]string{"go.mod"},// excludePatterns
		true,              // withLineNumbers
		false,             // hidden
		false,             // listOnly
	)

	if err != nil {
		log.Fatalf("Failed to run pcat: %v", err)
	}

	fmt.Println(output)
}
```

## API

### `pcat.Config`

This struct allows you to customize how files are discovered and formatted. Key fields include:

- `Directories`: A list of directory paths to search for files.
- `SpecificFiles`: A list of specific file paths to include.
- `Extensions`: A list of file extensions to filter by (e.g., "go", "md").
- `ExcludePatterns`: A list of glob patterns to exclude matching files.
- `WithLineNumbers`: A boolean to toggle line numbers in the output.

### `pcat.New(config)`

Creates a new `App` instance with the provided configuration. The `app.Run()` method then executes the file discovery and formatting process.

### `pcat.Read(files, config)`

A more direct function that reads and formats a predefined list of file paths according to the provided configuration. This is useful if you have your own file discovery logic.
