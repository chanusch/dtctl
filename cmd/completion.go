package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

// completionCmd represents the completion command
var completionCmd = &cobra.Command{
	Use:   "completion [bash|zsh|fish|powershell]",
	Short: "Generate shell completion scripts",
	Long: `Generate shell completion scripts for dtctl.

Examples:
  # bash (temporary)
  source <(dtctl completion bash)

  # bash (permanent)
  sudo cp <(dtctl completion bash) /etc/bash_completion.d/dtctl

  # zsh
  mkdir -p ~/.zsh/completions
  dtctl completion zsh > ~/.zsh/completions/_dtctl
  # Add to ~/.zshrc: fpath=(~/.zsh/completions $fpath)
  # Then: rm -f ~/.zcompdump* && autoload -U compinit && compinit

  # fish
  dtctl completion fish > ~/.config/fish/completions/dtctl.fish

  # powershell
  dtctl completion powershell | Out-String | Invoke-Expression
`,
	ValidArgs: []string{"bash", "zsh", "fish", "powershell"},
	Args:      cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
	Run: func(cmd *cobra.Command, args []string) {
		switch args[0] {
		case "bash":
			_ = cmd.Root().GenBashCompletion(os.Stdout)
		case "zsh":
			_ = cmd.Root().GenZshCompletion(os.Stdout)
		case "fish":
			_ = cmd.Root().GenFishCompletion(os.Stdout, true)
		case "powershell":
			_ = cmd.Root().GenPowerShellCompletionWithDesc(os.Stdout)
		}
	},
}

func init() {
	rootCmd.AddCommand(completionCmd)
}
