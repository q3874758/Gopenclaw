package cli

import (
	"context"
	"strings"

	"github.com/spf13/cobra"
)

// Execute 运行根命令及子命令
func Execute(ctx context.Context) error {
	// 版本信息
	var version = "dev"
	var commit = "unknown"

	root := &cobra.Command{
		Use:   "gopenclaw",
		Short: "Gopenclaw — OpenClaw 的 Go 重写版，零 Node",
		Version: version + " (" + commit + ")",
	}
	// 自定义 help 模板，确保 Usage 行显示 gopenclaw
	root.SetHelpTemplate(`{{.Short}}

Usage:
  gopenclaw [command]

Available Commands:
{{range .Commands}}{{if (and (not .Hidden) .Runnable)}}  {{rpad .Name .NamePadding}} {{.Short}}
{{end}}{{end}}

Flags:
{{.LocalFlags.FlagUsages | trimTrailingWhitespaces}}

Use "gopenclaw [command] --help" for more information about a command.
`)

	// 添加自定义 completion 命令，生成后将 openclaw 替换为 gopenclaw
	completionCmd := &cobra.Command{
		Use:   "completion [shell]",
		Short: "Generate the autocompletion script for gopenclaw",
		Long: `To load completions:

Bash:
  $ source <(gopenclaw completion bash)

# To load completions for each session, execute once:
# Linux:
$ gopenclaw completion bash > /etc/bash_completion.d/gopenclaw
# macOS:
$ gopenclaw completion bash > /usr/local/etc/bash_completion.d/gopenclaw

Zsh:
  # If shell completion is not already enabled in your environment,
  # you will need to enable it.  You can execute the following once:

  $ echo "autoload -U compinit; compinit" >> ~/.zshrc

  # To load completions for each session, execute once:
  $ gopenclaw completion zsh > "${fpath[1]}/_gopenclaw"

  # You will need to start a new shell for this setup to take effect.

Fish:
  $ gopenclaw completion fish | source

  # To load completions for each session, execute once:
  $ gopenclaw completion fish > ~/.config/fish/completions/gopenclaw.fish

PowerShell:
  PS> gopenclaw completion powershell | Out-String | Invoke-Expression

  # To load completions for every session, add the output of the above command
  # to your powershell profile.
`,
		DisableFlagsInUseLine: true,
		ValidArgs:             []string{"bash", "zsh", "fish", "powershell"},
		Args:                  cobra.ExactValidArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var buf strings.Builder
			var err error
			switch args[0] {
			case "bash":
				err = root.GenBashCompletionV2(&buf, true)
			case "zsh":
				err = root.GenZshCompletion(&buf)
			case "fish":
				err = root.GenFishCompletion(&buf, true)
			case "powershell":
				err = root.GenPowerShellCompletion(&buf)
			}
			if err != nil {
				return err
			}

			// 替换 openclaw 为 gopenclaw
			// 注意：先用占位符替换 gopenclaw，避免 gopenclaw 中的 openclaw 被重复替换
			output := buf.String()
			output = strings.ReplaceAll(output, "gopenclaw", "__GOPENCLAW_PLACEHOLDER__")
			output = strings.ReplaceAll(output, "openclaw", "gopenclaw")
			output = strings.ReplaceAll(output, "__GOPENCLAW_PLACEHOLDER__", "gopenclaw")
			_, err = cmd.OutOrStdout().Write([]byte(output))
			return err
		},
	}

	root.AddCommand(completionCmd)
	root.AddCommand(backupCmd())

	root.AddCommand(gatewayCmd(ctx))
	root.AddCommand(agentCmd())
	root.AddCommand(configCmd())
	root.AddCommand(doctorCmd())
	root.AddCommand(onboardCmd())
	root.AddCommand(migrateFromOfficialCmd())
	root.SetContext(ctx)
	return root.Execute()
}
