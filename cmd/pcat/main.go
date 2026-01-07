package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"github.com/sokinpui/pcat"

	"github.com/spf13/cobra"
)

var (
	extensions, excludePatterns, paths             []string
	withLineNumbers, hidden, listOnly, toClipboard bool
	completionShell                                string
)

var rootCmd = &cobra.Command{
	Use:   "pcat",
	Short: "Concatenate and print files from specified paths (files and directories).",
	Long: `Concatenate and print files from specified paths (files and directories).
If no paths are provided, paths are read from stdin.`,
	Example: `  pcat -p ./src -p ./README.md # Process all files in ./src and the specific file
  pcat -p ./src -e 'py js'      # Process .py and .js files in ./src
  pcat -p . --hidden            # Process all files in current dir, including hidden ones
  pcat -p ./src --not '*_test.py'  # Exclude test files
  fd . -e py | pcat         # Process python files found by fd from stdin`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if completionShell != "" {
			switch completionShell {
			case "bash":
				return cmd.Root().GenBashCompletion(os.Stdout)
			case "zsh":
				return cmd.Root().GenZshCompletion(os.Stdout)
			case "fish":
				return cmd.Root().GenFishCompletion(os.Stdout, true)
			case "powershell":
				return cmd.Root().GenPowerShellCompletionWithDesc(os.Stdout)
			default:
				return fmt.Errorf("unsupported shell for completion: %s. Must be one of bash, zsh, fish, or powershell", completionShell)
			}
		}
		allPaths := append(paths, args...)
		return run(allPaths)
	},
}

func getPaths(args []string) ([]string, error) {
	if len(args) > 0 {
		return args, nil
	}

	stat, _ := os.Stdin.Stat()
	if (stat.Mode() & os.ModeCharDevice) == 0 {
		paths := make([]string, 0)
		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line != "" {
				paths = append(paths, line)
			}
		}
		return paths, scanner.Err()
	}

	return nil, nil
}

func isBrokenPipe(err error) bool {
	// This check is OS-dependent but covers common cases for Linux/macOS.
	return strings.Contains(err.Error(), "broken pipe")
}

func init() {
	rootCmd.Flags().StringVar(&completionShell, "completion", "", "Generate completion script for a specified shell (bash|zsh|fish|powershell)")
	rootCmd.Flags().StringSliceVarP(&paths, "path", "p", nil, "Specify paths (files and directories). Can be repeated.")
	rootCmd.Flags().StringSliceVarP(&extensions, "extension", "e", nil, "Filter by file extensions (e.g., 'py', 'js' or 'py js'). Can be repeated.")
	rootCmd.Flags().StringSliceVar(&excludePatterns, "not", nil, "Exclude files matching glob patterns. Can be repeated.")
	rootCmd.Flags().BoolVarP(&withLineNumbers, "with-line-numbers", "n", false, "Include line numbers for each file.")
	rootCmd.Flags().BoolVar(&hidden, "hidden", false, "Include hidden files and directories.")
	rootCmd.Flags().BoolVarP(&listOnly, "list", "l", false, "List the files that would be processed, without printing content.")
	rootCmd.Flags().BoolVarP(&toClipboard, "clipboard", "c", false, "Copy the output to the clipboard instead of printing to stdout.")
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func run(args []string) error {
	var processedExtensions []string
	for _, ext := range extensions {
		processedExtensions = append(processedExtensions, strings.Fields(ext)...)
	}
	extensions = processedExtensions

	paths, err := getPaths(args)
	if err != nil {
		return err
	}

	if paths == nil {
		paths = []string{"."}
	}

	var directories, specificFiles []string
	for _, path := range paths {
		info, err := os.Stat(path)
		if err != nil {
			return fmt.Errorf("invalid path '%s': %w", path, err)
		}
		if info.IsDir() {
			directories = append(directories, path)
		} else {
			specificFiles = append(specificFiles, path)
		}
	}

	if len(directories) > 0 && len(extensions) == 0 {
		extensions = []string{"any"}
	}

	config := pcat.Config{
		Directories:     directories,
		Extensions:      extensions,
		SpecificFiles:   specificFiles,
		ExcludePatterns: excludePatterns,
		WithLineNumbers: withLineNumbers,
		Hidden:          hidden,
		ListOnly:        listOnly,
		ToClipboard:     toClipboard,
	}

	app := pcat.New(config)
	output, err := app.Run()
	if err != nil {
		return err
	}

	if config.ToClipboard {
		if err := pcat.Write(output); err != nil {
			return err
		}
		fmt.Fprintln(os.Stderr, "pcat: Copied to clipboard.")
	} else if output != "" {
		if _, err := fmt.Fprint(os.Stdout, output); err != nil && !isBrokenPipe(err) {
			return err
		}
	}

	return nil
}
