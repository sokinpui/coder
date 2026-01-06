golang rewrite of python [pcat](https://github.com/sokinpui/pcat)

# pcat: code to prompt

```sh
find Project/ | xargs cat
```

A tool to print source code from multiple directories and files in markdown format.

Don't relay on `upload files` feature of LLM providers' web UI. Why AI studio doesn't support `.js` files upload?

```
Usage: pcat [flags] [PATH...]
Concatenate and print files from specified paths (files and directories).
If no paths are provided, paths are read from stdin.

Options:
  -c, --clipboard           Copy the output to the clipboard instead of printing to stdout.
  -e, --extension strings   Filter by file extensions (e.g., 'py', 'js'). Can be repeated.
  --hidden              Include hidden files and directories.
  -l, --list                List the files that would be processed, without printing content.
  --not strings         Exclude files matching glob patterns. Can be repeated.
  -n, --with-line-numbers   Include line numbers for each file.

Examples:
  pcat -p ./src -p ./README.md    # Process all files in ./src and the specific file
  pcat -p ./src -e 'py js'     # Process .py and .js files in ./src
  pcat -p . --hidden           # Process all files in current dir, including hidden ones
  pcat -p ./src --not '*_test.py' # Exclude test files
  fd . -e py | pcat         # Process python files found by fd from stdin

```

# Shell Completion

`pcat` supports generating completion scripts for Bash, Zsh, Fish, and PowerShell.

Use the `pcat --completion [shell]` flag. For example, to enable completion for Bash, add the following to your `.bashrc` or `.bash_profile`:

```sh
source <(pcat --completion bash)
```

# Installatoin

```
go install github.com/sokinpui/pcat.go/cmd/pcat@latest
```

locally:

```
git clone https://github.com/sokinpui/pcat.go.git
cd pcat.go
git install ./cmd/pcat
```
