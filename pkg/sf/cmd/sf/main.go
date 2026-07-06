package main

import (
	"fmt"
	"os"
	"runtime/debug"

	"github.com/sokinpui/sf"
	"github.com/spf13/cobra"
)

func main() {
	var fileType string
	var excludes []string
	var showHidden bool

	rootCmd := &cobra.Command{
		Use:     "sf [path]",
		Short:   "A fast directory walker",
		Version: getVersion(),
		Args:    cobra.ArbitraryArgs,
		Run: func(cmd *cobra.Command, args []string) {
			results := sf.Run(args, fileType, excludes, showHidden)

			for _, path := range results {
				fmt.Println(path)
			}
		},
	}

	rootCmd.Flags().StringVarP(&fileType, "type", "t", "", "Filter by type: file, dir")
	rootCmd.Flags().StringSliceVarP(&excludes, "exclude", "E", []string{}, "Exclude entries that match the given glob pattern")
	rootCmd.Flags().BoolVarP(&showHidden, "hidden", "H", false, "Search hidden files and directories")

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func getVersion() string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return "unknown"
	}

	if info.Main.Version != "" && info.Main.Version != "(devel)" {
		return info.Main.Version
	}

	return "devel"
}
