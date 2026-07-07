# sf (Search Fast)

`sf` is a simple, blazing-fast directory walking and find tool written in Go. It is designed to be a lightweight and highly efficient alternative to `fd`, primarily optimized to quickly gather project file listings to feed as context for developer AI tools.

## Features

- **Blazing Fast**: Uses highly parallelized directory walking (`runtime.NumCPU() * 2` workers).
- **Gitignore Respecting**: Automatically reads and respects local and global `.gitignore` patterns.
- **Filtering**: Easily filter search results by type (files or directories).
- **Flexible Exclusions**: Supports custom glob exclusion patterns.
- **Hidden File Control**: Toggle whether hidden files/folders should be searched.

## Installation

### Via Go directly

```sh
go install github.com/sokinpui/coder/cmd/sf@latest
```

### From Source (inside the Coder repository)

```bash
go build -o sf ./cmd/sf
mv sf /usr/local/bin/ # Or any directory in your $PATH
```

## Command Line Usage

```bash
sf [path] [flags]
```

### Flags
- `-t, --type <file|dir>`: Filter results by type.
- `-E, --exclude <pattern>`: Exclude entries matching the glob pattern (can be used multiple times).
- `-H, --hidden`: Include hidden files and directories in the search.
- `-h, --help`: Help for sf.

### Examples

Search for all files and directories in the current directory (respects `.gitignore`):
```bash
sf
```

Search for directories only in a specific path:
```bash
sf /path/to/search -t dir
```

Exclude specific patterns:
```bash
sf . -E "*.log" -E "node_modules/*"
```

Show hidden files:
```bash
sf . -H
```

## API Usage

You can import and use `sf` as a library in your own Go projects.

```go
package main

import (
	"fmt"
	"github.com/sokinpui/coder/pkg/sf"
)

func main() {
	// sf.Run(roots, fileType, excludes, showHidden)
	results := sf.Run([]string{"."}, "file", []string{"vendor/*"}, false)
	for _, path := range results {
		fmt.Println(path)
	}
}
```
