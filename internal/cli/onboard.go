package cli

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"gopenclaw/internal/config"
)

// onboardCmd 引导用户完成最小配置：端口与模型（API Key 仍通过环境变量设置）
func onboardCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "onboard",
		Short: "引导完成 Gopenclaw 的最小配置",
		RunE: func(cmd *cobra.Command, args []string) error {
			out := cmd.OutOrStdout()
			in := bufio.NewReader(cmd.InOrStdin())

			cfg, err := config.Load()
			if err != nil {
				return err
			}

			_, _ = fmt.Fprintf(out, "配置文件路径: %s\n", config.ConfigPath())

			// 配置 Gateway 端口
			_, _ = fmt.Fprintf(out, "\n当前 Gateway 端口: %d（回车保留，或输入新端口，例如 11999）: ", cfg.Gateway.Port)
			line, _ := in.ReadString('\n')
			line = strings.TrimSpace(line)
			if line != "" {
				var p int
				if _, err := fmt.Sscanf(line, "%d", &p); err == nil && p > 0 && p < 65536 {
					cfg.Gateway.Port = p
				} else {
					_, _ = fmt.Fprintf(out, "端口无效，保持原值 %d\n", cfg.Gateway.Port)
				}
			}

			// 配置 Agent 模型
			defaultModel := cfg.Agent.Model
			if defaultModel == "" {
				defaultModel = "openai/gpt-4o-mini"
			}
			_, _ = fmt.Fprintf(out, "\nAgent 模型（当前: %q，回车使用默认 %q）: ", cfg.Agent.Model, defaultModel)
			line, _ = in.ReadString('\n')
			line = strings.TrimSpace(line)
			if line == "" {
				cfg.Agent.Model = defaultModel
			} else {
				cfg.Agent.Model = line
			}

			// 提示 API Key 设置方式
			_, _ = fmt.Fprintln(out, "\n注意：Agent 调用 LLM 需要设置 OPENAI_API_KEY 环境变量，例如：")
			if isWindows() {
				_, _ = fmt.Fprintln(out, `  setx OPENAI_API_KEY "sk-xxxxx"  # 永久设置（新终端生效）`)
				_, _ = fmt.Fprintln(out, `  $env:OPENAI_API_KEY="sk-xxxxx"  # 当前 PowerShell 会话`)
			} else {
				_, _ = fmt.Fprintln(out, `  export OPENAI_API_KEY="sk-xxxxx"`)
			}

			if err := config.Save(cfg); err != nil {
				return err
			}
			_, _ = fmt.Fprintln(out, "\n已写入配置。你可以运行以下命令开始使用：")
			_, _ = fmt.Fprintln(out, "  gopenclaw gateway")
			_, _ = fmt.Fprintln(out, `  OPENAI_API_KEY="sk-..." gopenclaw agent --message "你好"`)
			return nil
		},
	}
	return cmd
}

func isWindows() bool {
	return strings.Contains(strings.ToLower(os.Getenv("OS")), "windows")
}

