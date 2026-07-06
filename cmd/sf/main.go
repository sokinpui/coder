package main

import (
	"fmt"
	"os"

	"github.com/sokinpui/coder/pkg/sf"
	"github.com/sokinpui/coder/pkg/version"
	"github.com/spf13/cobra"
)

func main() {
	var fileType string
	var excludes []string
	var showHidden bool

	rootCmd := &cobra.Command{
		Use:     "sf [path]",
		Short:   "A fast directory walker",
		Version: version.Get(),
		Args:    cobra.ArbitraryArgs,
		Run: func(cmd *cobra.Command, args []string) {
			results := sf.Run(args, fileType, excludes, showHidden)
			for _, path := range results {
				fmt.Println(path)
			}
		},
	}

	rootCmd.Flags().StringVarP(&fileType, "type", "t", "", "Filter by type: file, dir")
	rootCmd.Flags().StringSliceVarP(&excludes, "exclude", "E", []string{}, "Exclude patterns")
	rootCmd.Flags().BoolVarP(&showHidden, "hidden", "H", false, "Search hidden files")

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
