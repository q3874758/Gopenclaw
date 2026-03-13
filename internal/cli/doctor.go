package cli

import (
	"fmt"
	"net"
	"net/http"
	"os"

	"github.com/spf13/cobra"

	"gopenclaw/internal/config"
)

func doctorCmd() *cobra.Command {
	var port int
	cmd := &cobra.Command{
		Use:   "doctor",
		Short: "检查配置、环境与 Gateway 连通性",
		RunE: func(cmd *cobra.Command, args []string) error {
			out := cmd.OutOrStdout()
			cfg, _ := config.Load()
			if port == 0 {
				port = cfg.Gateway.Port
				if port == 0 {
					port = 11999
				}
			}

			// 配置目录与文件
			home := config.OpenClawHome()
			path := config.ConfigPath()
			_, _ = fmt.Fprintf(out, "OPENCLAW_HOME / 配置目录: %s\n", home)
			if _, err := os.Stat(home); err != nil {
				_, _ = fmt.Fprintf(out, "  状态: 目录不存在（首次运行将自动创建）\n")
			} else {
				_, _ = fmt.Fprintf(out, "  状态: 存在\n")
			}
			_, _ = fmt.Fprintf(out, "配置文件: %s\n", path)
			if _, err := os.Stat(path); err != nil {
				_, _ = fmt.Fprintf(out, "  状态: 不存在，将使用默认配置\n")
			} else {
				_, _ = fmt.Fprintf(out, "  状态: 存在\n")
			}

			// API Key
			_, _ = fmt.Fprintf(out, "OPENAI_API_KEY: ")
			if os.Getenv("OPENAI_API_KEY") == "" {
				_, _ = fmt.Fprintf(out, "未设置（Agent 调用 LLM 需设置）\n")
			} else {
				_, _ = fmt.Fprintf(out, "已设置\n")
			}

			// Gateway 端口
			_, _ = fmt.Fprintf(out, "Gateway 端口: %d\n", port)
			ln, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
			if err != nil {
				_, _ = fmt.Fprintf(out, "  端口状态: 可能已被占用或无法绑定\n")
			} else {
				_ = ln.Close()
				_, _ = fmt.Fprintf(out, "  端口状态: 可用\n")
			}

			// 若 Gateway 已在运行，尝试 /health
			_, _ = fmt.Fprintf(out, "Gateway 健康检查: ")
			resp, err := http.Get(fmt.Sprintf("http://127.0.0.1:%d/health", port))
			if err != nil {
				_, _ = fmt.Fprintf(out, "未运行或不可达\n")
			} else {
				_ = resp.Body.Close()
				if resp.StatusCode == 200 {
					_, _ = fmt.Fprintf(out, "正常 (HTTP 200)\n")
				} else {
					_, _ = fmt.Fprintf(out, "HTTP %d\n", resp.StatusCode)
				}
			}
			return nil
		},
	}
	cmd.Flags().IntVar(&port, "port", 0, "Gateway 端口（0=使用配置）")
	return cmd
}
