package pti

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/sokinpui/coder/pkg/version"
	"github.com/spf13/cobra"
)

var (
	dpiFlag       float64
	outputDirFlag string
	pagesFlag     string
)

var rootCmd = &cobra.Command{
	Use:     "pti [flags] <file...>",
	Short:   "Render PDF documents into images",
	Version: version.Get(),
	Args:    cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		opts := Options{
			DPI:       dpiFlag,
			OutputDir: outputDirFlag,
			Pages:     pagesFlag,
		}

		var files []string
		for _, arg := range args {
			matches, err := filepath.Glob(arg)
			if err != nil || len(matches) == 0 {
				files = append(files, arg)
				continue
			}
			files = append(files, matches...)
		}

		for _, file := range files {
			res, err := Convert(file, opts)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error processing %s: %v\n", file, err)
				continue
			}
			fmt.Printf("%s: rendered %d page(s) -> %s/\n", file, len(res.OutputFiles), res.OutputDir)
		}

		return nil
	},
}

func init() {
	rootCmd.Flags().Float64VarP(&dpiFlag, "dpi", "d", DefaultDPI, "DPI resolution for rendered images")
	rootCmd.Flags().StringVarP(&outputDirFlag, "output", "o", "", "Base directory for output document folders")
	rootCmd.Flags().StringVarP(&pagesFlag, "pages", "p", "", "Comma-separated page numbers or ranges (e.g. 1-3,5)")
}

func SetVersion(v string) {
	rootCmd.Version = v
}

func Execute() error {
	return rootCmd.Execute()
}
