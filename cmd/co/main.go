package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/sokinpui/coder/internal/config"
	coagentui "github.com/sokinpui/coder/internal/ui/coagent"
	"github.com/sokinpui/coder/pkg/version"
	"github.com/spf13/cobra"
)

var (
	initialPrompt     string
	customInstruction string
	runModel          string
)

func main() {
	rootCmd := &cobra.Command{
		Use:     "co [flags] [prompt]",
		Short:   "Autonomous AI coding agent",
		Long:    "Co is an autonomous terminal coding agent powered by LLM tool-calling loops.",
		Version: version.Get(),
		Example: `  co
  co "Fix the failing test in pkg/itf"
  co -p "Refactor generator to use interface"
  co -m gpt-4o`,
		Args: cobra.ArbitraryArgs,
		Run: func(cmd *cobra.Command, args []string) {
			prompt := initialPrompt
			if len(args) > 0 {
				posPrompt := strings.Join(args, " ")
				if prompt != "" {
					prompt = prompt + "\n" + posPrompt
				} else {
					prompt = posPrompt
				}
			}

			cfg, err := config.Load()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error loading configuration: %v\n", err)
				os.Exit(1)
			}

			if runModel != "" {
				cfg.Generation.ModelCode = runModel
			}

			if err := coagentui.Start(cfg, prompt); err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
		},
	}

	rootCmd.Flags().StringVarP(&initialPrompt, "prompt", "p", "", "Initial prompt/task to run")
	rootCmd.Flags().StringVarP(&customInstruction, "instruction", "i", "", "Custom system instructions for the agent")
	rootCmd.Flags().StringVarP(&runModel, "model", "m", "", "Model code to use for agent generation")

	rootCmd.CompletionOptions.DisableDefaultCmd = true

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
