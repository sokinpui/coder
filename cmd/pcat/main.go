package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/sokinpui/coder/pkg/pcat"
	"github.com/sokinpui/coder/pkg/version"
	"github.com/spf13/cobra"
)

var (
	extensions, excludePatterns, paths             []string
	withLineNumbers, hidden, listOnly, toClipboard bool
	completionShell                                string
)

var rootCmd = &cobra.Command{
	Use:     "pcat",
	Version: version.Get(),
	Short:   "Concatenate and print files from specified paths (files and directories).",
	Long: `Concatenate and print files from specified paths (files and directories).
If no paths are provided, paths are read from stdin.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if completionShell != "" {
			return handleCompletion(cmd)
		}
		allPaths := append(paths, args...)
		return run(allPaths)
	},
}

func handleCompletion(cmd *cobra.Command) error {
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
		return fmt.Errorf("unsupported shell for completion: %s", completionShell)
	}
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

func init() {
	rootCmd.Flags().StringVar(&completionShell, "completion", "", "Generate completion script")
	rootCmd.Flags().StringSliceVarP(&paths, "path", "p", nil, "Specify paths")
	rootCmd.Flags().StringSliceVarP(&extensions, "extension", "e", nil, "Filter by extensions")
	rootCmd.Flags().StringSliceVar(&excludePatterns, "not", nil, "Exclude patterns")
	rootCmd.Flags().BoolVarP(&withLineNumbers, "with-line-numbers", "n", false, "Include line numbers")
	rootCmd.Flags().BoolVar(&hidden, "hidden", false, "Include hidden files")
	rootCmd.Flags().BoolVarP(&listOnly, "list", "l", false, "List files only")
	rootCmd.Flags().BoolVarP(&toClipboard, "clipboard", "c", false, "Copy to clipboard")
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func run(args []string) error {
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

	output, err := pcat.Run(specificFiles, directories, extensions, excludePatterns, withLineNumbers, hidden, listOnly)
	if err != nil {
		return err
	}

	if toClipboard {
		if err := pcat.Write(output); err != nil {
			return err
		}
	} else if output != "" {
		fmt.Fprint(os.Stdout, output)
	}
	return nil
}
