# pcat (Prompt Cat)

`pcat` is a fast CLI tool designed to concatenate source code files into a single, beautifully formatted markdown code block representation. It is optimized for preparing rich, structured codebase contexts to feed into Large Language Models (LLMs).

## Features

- **Code Block Formatting**: Outputs files wrapped in standard markdown syntax highlighting blocks.
- **Language Detection**: Automatically infers standard markdown syntax languages from file extensions (e.g., `.go` -> `go`, `.rs` -> `rust`).
- **Safety Checks**: Automatically skips binary files to keep LLM context clean.
- **Clipboard Integration**: Pipe/concatenate content directly to your system clipboard (`-c`).
- **Flexible Exclude Rules**: Supports standard glob exclusion patterns (via `--not`).
- **Line Numbers**: Option to append neat, left-padded line numbers for referencing exact code coordinates.
- **Fuzzy/Interactive Pipeline**: Easily chains with tools like `find` or `fd` through stdin pipelines.

## Installation

### Via Go directly

```sh
go install github.com/sokinpui/coder/cmd/pcat@latest
```

### From Source (inside the Coder repository)

```bash
go build -o pcat ./cmd/pcat
mv pcat /usr/local/bin/ # Or any directory in your $PATH
```

## Usage

### Basic Commands

```sh
# Concatenate all files in the current directory (respects standard text files)
pcat

# Process specific files and directories
pcat -p src/main.go -p pkg/
```

### Filtering

```sh
# Filter by specific extensions (comma or space separated)
pcat -e "go md"

# Exclude files matching glob patterns
pcat --not "**/test_*.go" --not "vendor/**"
```

### Output Options

```sh
# Copy the formatted output directly to the system clipboard
pcat src/ -c

# Include line numbers for precise AI referencing
pcat main.go -n

# Just list files that would be processed (without printing contents)
pcat -l
```

### Pipe Support

`pcat` reads paths from stdin if no paths are provided via flags or arguments:

```sh
sf . -t file | pcat -c
```

## Command Line Flags

- `-p, --path`: Specify target files or directories (can be specified multiple times).
- `-e, --extension`: Filter files by specific extensions (e.g., `go`, `js`, `py`).
- `--not`: Glob patterns to exclude files from processing.
- `-n, --with-line-numbers`: Include formatted line numbers.
- `-c, --clipboard`: Write the generated output to the system clipboard.
- `-l, --list`: Print the list of matched file paths only.
- `--hidden`: Include hidden files and directories.
- `--completion`: Generate autocomplete script for your preferred shell.

## Library Usage

`pcat` can also be used as a library in other Go projects. See [docs/Usage/README.md](./docs/Usage/README.md) for more details.
