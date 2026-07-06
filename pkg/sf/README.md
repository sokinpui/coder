 # sf

A simple, fast find tool written in Go. It is a lightweight alternative to `fd`, designed to provide file context for development tools.

## Installation

```bash
go install github.com/sokinpui/sf/cmd/sf@latest
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

Search for all files and directories in the current directory:
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

You can use `sf` as a library in your Go projects.

```go
package main

import (
	"fmt"
	"github.com/sokinpui/sf"
)

func main() {
	results := sf.Run([]string{"."}, "file", []string{"vendor/*"}, false)
	for _, path := range results {
		fmt.Println(path)
	}
}
```
