package cli

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"gopenclaw/internal/config"
	"gopenclaw/internal/cron"
	"gopenclaw/internal/gateway"
	"gopenclaw/internal/storage"
	"gopenclaw/internal/webhook"

	"github.com/spf13/cobra"
)

func gatewayCmd(ctx context.Context) *cobra.Command {
	var port int
	var bind string

	cmd := &cobra.Command{
		Use:   "gateway",
		Short: "启动 Gateway（WS + HTTP）",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := config.Default()
			if port != 0 {
				cfg.Gateway.Port = port
			}
			if bind != "" {
				cfg.Gateway.Bind = bind
			}

			// 初始化存储
			storageDir := config.OpenClawHome() + "/sessions"
			store, err := storage.New(storageDir)
			if err != nil {
				return fmt.Errorf("init storage: %w", err)
			}
			if err := store.Load(); err != nil {
				fmt.Fprintf(os.Stderr, "warning: load sessions: %v\n", err)
			}

			// 创建 Gateway
			gw := gateway.New(cfg)

			// 启动 Gateway
			go func() {
				if err := gw.Start(ctx); err != nil {
					fmt.Fprintf(os.Stderr, "Gateway error: %v\n", err)
				}
			}()

			// 初始化 cron 调度器
			scheduler, err := cron.New(cron.Options{
				StorePath: config.OpenClawHome() + "/cron.json",
				TimeZone:  "Local",
				Handler: func(ctx context.Context, job *cron.CronJob) error {
					fmt.Printf("Cron job executed: %s (%s)\n", job.Name, job.ID)
					return nil
				},
			})
			if err != nil {
				return fmt.Errorf("init cron: %w", err)
			}
			scheduler.Start()
			defer scheduler.Stop()

			// 初始化 webhook 处理器
			hookCfg := &webhook.Config{
				Enabled:               cfg.Hooks.Enabled,
				Token:                 cfg.Hooks.Token,
				Path:                  cfg.Hooks.Path,
				AllowedAgentIDs:       cfg.Hooks.AllowedAgentIDs,
				DefaultSessionKey:     cfg.Hooks.DefaultSessionKey,
				AllowRequestSessionKey: cfg.Hooks.AllowRequestSessionKey,
				AllowedSessionKeyPrefixes: cfg.Hooks.AllowedSessionKeyPrefixes,
			}
			hookHandler := webhook.New(hookCfg)
			webhookAddr := fmt.Sprintf("127.0.0.1:%d", cfg.Gateway.Port+1)
			go func() {
				if err := hookHandler.Start(ctx, webhookAddr); err != nil {
					fmt.Fprintf(os.Stderr, "Webhook error: %v\n", err)
				}
			}()

			// 等待中断信号
			sigCh := make(chan os.Signal, 1)
			signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
			<-sigCh

			fmt.Println("Shutting down...")
			return nil
		},
	}

	cmd.Flags().IntVarP(&port, "port", "p", 0, "Gateway 端口（默认 11999）")
	cmd.Flags().StringVar(&bind, "bind", "", "绑定地址（默认 127.0.0.1）")

	return cmd
}
