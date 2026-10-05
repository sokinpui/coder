package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/sokinpui/coder/internal/config"
	"github.com/sokinpui/coder/internal/engine/coagent"
	"github.com/sokinpui/coder/internal/ui"
	"github.com/sokinpui/coder/pkg/version"
	"github.com/spf13/cobra"
)

var (
	initialPrompt     string
	customInstruction string
	runModel          string
	runProtocol       string
	completionShell   string
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
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			return nil, cobra.ShellCompDirectiveDefault
		},
		Run: func(cmd *cobra.Command, args []string) {
			if completionShell != "" {
				generateCompletion(cmd, completionShell)
				return
			}

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

			if cmd.Flags().Changed("protocol") && runProtocol != "" {
				cfg.Server.Protocol = runProtocol
			}

			if runModel != "" {
				cfg.Agent.ModelCode = runModel
			}

			sess, err := coagent.NewSession(cfg)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error creating agent session: %v\n", err)
				os.Exit(1)
			}
			if customInstruction != "" {
				sess.Instruction = customInstruction
			}

			if err := ui.Start(sess, prompt); err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
		},
	}

	rootCmd.Flags().StringVarP(&initialPrompt, "prompt", "p", "", "Initial prompt/task to run")
	rootCmd.Flags().StringVarP(&customInstruction, "instruction", "i", "", "Custom system instructions for the agent")
	rootCmd.Flags().StringVarP(&runModel, "model", "m", "", "Model code to use for agent generation")
	rootCmd.Flags().StringVarP(&runProtocol, "protocol", "P", "responses", "Protocol to use: chat or responses")
	rootCmd.Flags().StringVar(&completionShell, "completion", "", "Generate autocompletion script (bash, zsh, fish, powershell)")

	_ = rootCmd.RegisterFlagCompletionFunc("completion", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{"bash", "zsh", "fish", "powershell"}, cobra.ShellCompDirectiveNoFileComp
	})
	_ = rootCmd.RegisterFlagCompletionFunc("protocol", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{"chat", "responses"}, cobra.ShellCompDirectiveNoFileComp
	})

	rootCmd.CompletionOptions.DisableDefaultCmd = true

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func generateCompletion(cmd *cobra.Command, shell string) {
	var err error
	switch shell {
	case "bash":
		err = cmd.Root().GenBashCompletion(os.Stdout)
	case "zsh":
		err = cmd.Root().GenZshCompletion(os.Stdout)
	case "fish":
		err = cmd.Root().GenFishCompletion(os.Stdout, true)
	case "powershell":
		err = cmd.Root().GenPowerShellCompletionWithDesc(os.Stdout)
	default:
		fmt.Fprintf(os.Stderr, "Unsupported shell: %s\n", shell)
		os.Exit(1)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error generating completion: %v\n", err)
		os.Exit(1)
	}
}
