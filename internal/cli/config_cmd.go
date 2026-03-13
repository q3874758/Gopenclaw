package cli

import (
	"encoding/json"

	"github.com/spf13/cobra"

	"gopenclaw/internal/config"
)

func configCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "查看或修改配置（当前仅支持查看）",
	}
	cmd.AddCommand(configGetCmd())
	cmd.AddCommand(configPathCmd())
	return cmd
}

func configGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get",
		Short: "打印当前配置（JSON）",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			enc := json.NewEncoder(cmd.OutOrStdout())
			enc.SetIndent("", "  ")
			return enc.Encode(cfg)
		},
	}
}

func configPathCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "path",
		Short: "打印配置文件路径",
		RunE: func(cmd *cobra.Command, args []string) error {
			_, _ = cmd.OutOrStdout().Write([]byte(config.ConfigPath() + "\n"))
			return nil
		},
	}
}
