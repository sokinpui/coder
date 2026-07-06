# pcat

`pcat` (Prompt Cat) is a CLI tool designed to concatenate source code files into a single, formatted output optimized for prompting Large Language Models (LLMs). It handles file extensions, ignores binary files, and can copy the output directly to your clipboard.

## Installation

Ensure you have Go installed, then run:

```sh
go install github.com/sokinpui/coder/cmd/pcat@latest
```

## Usage

### Basic Command

```sh
# Concatenate all files in the current directory
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
# Copy the result directly to the clipboard instead of stdout
pcat src/ -c

# Include line numbers for each file
pcat main.go -n

# Just list the files that would be processed without printing content
pcat -l
```

### Pipe Support

`pcat` reads paths from stdin if no paths are provided via flags or arguments:

```sh
fd . -e go | pcat -c
```

## Command Line Flags

- `-p, --path`: Specify paths (files or directories). Can be repeated.
- `-e, --extension`: Filter by file extensions (e.g., 'go', 'js').
- `--not`: Exclude files matching glob patterns.
- `-n, --with-line-numbers`: Include line numbers in the output.
- `-c, --clipboard`: Copy output to clipboard.
- `-l, --list`: List files instead of printing content.
- `--hidden`: Include hidden files and directories.
- `--completion`: Generate shell completion script (bash, zsh, fish, powershell).

## Library Usage

`pcat` can be integrated into your Go projects as a library. See [docs/Usage/README.md](./docs/Usage/README.md) for API details and examples.
