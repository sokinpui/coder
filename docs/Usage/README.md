# Using pcat as a library

The `pcat` package can be integrated into other Go applications to find and format source code from the filesystem.

## Installation

To use the `pcat` package in your project, install it using `go get`:

```sh
go get github.com/sokinpui/pcat.go
```

## Example

Here is a basic example of how to use the `pcat` library to read all `.go` files from the current directory, excluding `go.mod`.

```go
package main

import (
	"fmt"
	"log"

	"github.com/sokinpui/pcat.go/pcat"
)

func main() {
	// 1. Configure pcat's behavior.
	config := pcat.Config{
		Directories:     []string{"."},
		Extensions:      []string{"go"},
		ExcludePatterns: []string{"go.mod"},
		WithLineNumbers: true,
	}

	// 2. Create a new pcat application instance and run it.
	// The Run() method finds files based on the configuration,
	// reads them, and returns the formatted output.
	app := pcat.New(config)
	output, err := app.Run()
	if err != nil {
		log.Fatalf("Failed to run pcat: %v", err)
	}

	fmt.Println(output)

	// --- Or, if you already have a list of files ---

	// You can use pcat.Read directly if you manage file discovery yourself.
	files := []string{"pcat/pcat.go", "cmd/pcat/main.go"}
	readConfig := pcat.Config{WithLineNumbers: true}

	output, err = pcat.Read(files, readConfig)
	if err != nil {
		log.Fatalf("Failed to read files: %v", err)
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
