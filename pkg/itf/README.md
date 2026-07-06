Golang rewrite of python [itf](https://github.com/sokinpui/itf)

# ITF: Insert To File

`itf` is a command-line tool that parses markdown content from stdin or your clipboard and applies the changes to your local files. It's designed to streamline workflows with Large Language Models (LLMs) by eliminating the need to manually copy and paste code snippets.

It can create, overwrite, delete, rename, or edit files based on the content of the markdown, also support redo and undo operations.

## Features

- **Clipboard & Pipe Integration**: Reads content directly from your clipboard or standard input.
- **File & Diff Block Parsing**: Intelligently parses markdown to identify file paths and content for file creation/modification, as well as diff hunks for patching.
- **Direct File Modification**: Applies changes directly to your local filesystem.
- **Undo/Redo**: Supports undoing and redoing file operations.
- **Interactive TUI**: Provides real-time feedback on the operations being performed.
- **Tool Call Extraction**: Can extract and print `tool` code blocks.
- **Extensible as a Library**: Can be used as a Go library in other projects.

## Installation

### Using `go install`

```bash
go install github.com/sokinpui/itf/cmd/itf@latest
```

### From Source

```bash
git clone https://github.com/sokinpui/itf.git
cd itf
go build ./cmd/itf
mv itf /usr/local/bin/
```

## Usage

### Basic Workflow

1.  Copy markdown content containing file blocks or diff blocks.
2.  Run `itf` in your terminal.

```bash
# Read from clipboard
itf

# Read from stdin
cat content.md | itf
pbpaste | itf # on macOS
```

### Input Formats

#### File Blocks

A file block is a standard markdown code block preceded by a line containing the file path in backticks.

**Example: Creating/Overwriting a code file**
`path/to/hello.go`

```go
package main

func main() {
    println("Hello, ITF!")
}
```

**Example: Creating/Overwriting a Markdown file**

````markdown
`path/to/hello.go`

```go
package main

func main() {
    println("Hello, ITF!")
}
```
````

#### Diff Blocks

A diff block is a code block with the language identifier `diff`.

```diff
--- a/src/main.go
+++ b/src/main.go
@@ -1,5 +1,6 @@
 package main

 func main() {
-	println("Hello, ITF!")
+	println("Hello, world!")
 }
```

#### Delete Blocks

A code block with the language identifier `delete` containing a list of file paths.

```delete
path/to/obsolete_file.go
```

#### Rename Blocks

A code block with the language identifier `rename` containing old and new file paths.

```rename
src/old_name.go src/new_name.go
```

### Command-Line Flags

| Flag                | Shorthand | Description                                                                 |
| ------------------- | --------- | --------------------------------------------------------------------------- |
| `--extension`       | `-e`      | Filter by file extension (e.g., `-e go`). Use `-e diff` for diff-only mode. |
| `--undo`            | `-u`      | Undo the last operation.                                                    |
| `--redo`            | `-r`      | Redo the last undo operation.                                               |
| `--output-diff-fix` | `-o`      | Print a corrected version of the diffs found in the input.                  |
| `--no-animation`    |           | Disable the loading spinner and progress updates.                           |

## Library Usage (API)

`itf` can be used as a Go library:

```go
import "github.com/sokinpui/itf"

config := itf.Config{Extensions: []string{".go"}}
results, err := itf.Apply(markdown, config)
```

## Developer Guide

### Project Structure

- `cmd/itf/`: Entry point.
- `itf/`: Core logic and public API.
- `internal/`: Internal packages (parser, patcher, source, state, tui, etc.).
