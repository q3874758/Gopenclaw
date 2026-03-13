package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"

	"gopenclaw/internal/config"
)

// migrateFromOfficialCmd 从官方 OpenClaw (~/.openclaw) 迁移配置到 Gopenclaw (~/.gopenclaw)
// 当前仅迁移 openclaw.json（config）；凭证与会话后续可扩展。
func migrateFromOfficialCmd() *cobra.Command {
	var (
		dryRun    bool
		noBackup  bool
		backupDir string
	)
	cmd := &cobra.Command{
		Use:   "migrate-from-official",
		Short: "从官方 OpenClaw (~/.openclaw) 迁移配置到 Gopenclaw",
		RunE: func(cmd *cobra.Command, args []string) error {
			out := cmd.OutOrStdout()
			srcHome := officialHome()
			srcCfg := filepath.Join(srcHome, "openclaw.json")
			dstHome := config.OpenClawHome()
			dstCfg := config.ConfigPath()

			_, _ = fmt.Fprintf(out, "官方配置目录: %s\n", srcHome)
			_, _ = fmt.Fprintf(out, "官方配置文件: %s\n", srcCfg)
			_, _ = fmt.Fprintf(out, "目标配置目录: %s\n", dstHome)
			_, _ = fmt.Fprintf(out, "目标配置文件: %s\n", dstCfg)

			if _, err := os.Stat(srcCfg); err != nil {
				return fmt.Errorf("未找到官方 openclaw.json（%s），请确认已安装并配置官方 OpenClaw", srcCfg)
			}

			if dryRun {
				_, _ = fmt.Fprintln(out, "\n[Dry-run] 仅预览，不会写入任何文件。")
				return nil
			}

			// 备份现有 Gopenclaw 配置
			if !noBackup {
				if _, err := os.Stat(dstCfg); err == nil {
					if backupDir == "" {
						backupDir = dstHome
					}
					if err := os.MkdirAll(backupDir, 0700); err != nil {
						return err
					}
					suffix := time.Now().Format("20060102-150405")
					backupPath := filepath.Join(backupDir, "openclaw.json.backup."+suffix)
					if err := copyFile(dstCfg, backupPath); err != nil {
						return err
					}
					_, _ = fmt.Fprintf(out, "已备份当前 Gopenclaw 配置到: %s\n", backupPath)
				}
			}

			// 读取官方配置并转换为 Gopenclaw 配置结构
			data, err := os.ReadFile(srcCfg)
			if err != nil {
				return err
			}
			var src map[string]interface{}
			if err := json.Unmarshal(data, &src); err != nil {
				return err
			}

			// 尝试直接按 Gopenclaw 的 Config 结构解析
			var dstCfgStruct config.Config
			if err := json.Unmarshal(data, &dstCfgStruct); err != nil {
				// 若结构不完全兼容，至少保留 gateway.port 等核心字段
				dstCfgStruct = *config.Default()
				if gw, ok := src["gateway"].(map[string]interface{}); ok {
					if p, ok := gw["port"].(float64); ok {
						dstCfgStruct.Gateway.Port = int(p)
					}
					if b, ok := gw["bind"].(string); ok && b != "" {
						dstCfgStruct.Gateway.Bind = b
					}
				}
				if ag, ok := src["agent"].(map[string]interface{}); ok {
					if m, ok := ag["model"].(string); ok && m != "" {
						dstCfgStruct.Agent.Model = m
					}
				}
			}

			// 避免端口冲突：若迁入端口是 18789/19018，则改为 11999
			if dstCfgStruct.Gateway.Port == 0 ||
				dstCfgStruct.Gateway.Port == 18789 ||
				dstCfgStruct.Gateway.Port == 19018 {
				dstCfgStruct.Gateway.Port = 11999
			}

			if err := os.MkdirAll(dstHome, 0700); err != nil {
				return err
			}
			enc, err := json.MarshalIndent(&dstCfgStruct, "", "  ")
			if err != nil {
				return err
			}
			if err := os.WriteFile(dstCfg, enc, 0600); err != nil {
				return err
			}
			_, _ = fmt.Fprintln(out, "已将官方配置迁移到 Gopenclaw。")
			return nil
		},
	}
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "仅预览，不写入任何文件")
	cmd.Flags().BoolVar(&noBackup, "no-backup", false, "不备份现有 Gopenclaw 配置")
	cmd.Flags().StringVar(&backupDir, "backup-dir", "", "备份目录（默认使用 Gopenclaw 配置目录）")
	return cmd
}

func officialHome() string {
	if h := os.Getenv("OPENCLAW_OFFICIAL_HOME"); h != "" {
		return h
	}
	if h := os.Getenv("OPENCLAW_HOME"); h != "" {
		return h
	}
	dir, _ := os.UserHomeDir()
	return filepath.Join(dir, ".openclaw")
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	defer out.Close()
	if _, err := out.ReadFrom(in); err != nil {
		return err
	}
	return nil
}

