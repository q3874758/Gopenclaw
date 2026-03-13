package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// backupCmd 返回 backup 命令
func backupCmd() *cobra.Command {
	var opts struct {
		onlyConfig        bool
		noIncludeWorkspace bool
		outputPath       string
		verify           bool
	}

	cmd := &cobra.Command{
		Use:   "backup",
		Short: "Create or verify local state archives",
		Long: `Create or verify local state archives.

Examples:
  # Create full backup
  gopenclaw backup create

  # Create config-only backup
  gopenclaw backup create --only-config

  # Create backup to specific directory
  gopenclaw backup create --output-path /path/to/backups

  # Verify a backup archive
  gopenclaw backup verify backup.zip
`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if opts.verify {
				// 验证模式
				if len(args) < 1 {
					return fmt.Errorf("backup file path required")
				}
				result, err := VerifyBackup(args[0])
				if err != nil {
					return err
				}
				PrintVerifyResult(result)
				return nil
			}

			// 创建模式
			cfg := &BackupConfig{
				OnlyConfig:        opts.onlyConfig,
				NoIncludeWorkspace: opts.noIncludeWorkspace,
				OutputPath:       opts.outputPath,
			}

			result, err := CreateBackup(cfg)
			if err != nil {
				return err
			}
			PrintBackupResult(result)
			return nil
		},
	}

	// create 子命令
	createCmd := &cobra.Command{
		Use:   "create",
		Short: "Create a backup archive",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := &BackupConfig{
				OnlyConfig:        opts.onlyConfig,
				NoIncludeWorkspace: opts.noIncludeWorkspace,
				OutputPath:       opts.outputPath,
			}

			result, err := CreateBackup(cfg)
			if err != nil {
				return err
			}
			PrintBackupResult(result)
			return nil
		},
	}
	createCmd.Flags().BoolVar(&opts.onlyConfig, "only-config", false, "Backup config only")
	createCmd.Flags().BoolVar(&opts.noIncludeWorkspace, "no-include-workspace", false, "Do not include workspace files")
	createCmd.Flags().StringVarP(&opts.outputPath, "output-path", "o", "", "Output directory for backup file")

	// verify 子命令
	verifyCmd := &cobra.Command{
		Use:   "verify",
		Short: "Verify a backup archive",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			result, err := VerifyBackup(args[0])
			if err != nil {
				return err
			}
			PrintVerifyResult(result)
			return nil
		},
	}

	cmd.AddCommand(createCmd)
	cmd.AddCommand(verifyCmd)

	// 父命令标志（用于 create 子命令）
	cmd.Flags().BoolVar(&opts.onlyConfig, "only-config", false, "Backup config only")
	cmd.Flags().BoolVar(&opts.noIncludeWorkspace, "no-include-workspace", false, "Do not include workspace files")
	cmd.Flags().StringVarP(&opts.outputPath, "output-path", "o", "", "Output directory for backup file")

	// 检查环境变量
	if os.Getenv("DEBUG") != "" {
		fmt.Println("[DEBUG] backup command initialized")
	}

	return cmd
}
