package skills

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
)

// Manager 技能管理器
type Manager struct {
	mu      sync.RWMutex
	skills  map[string]*Skill
	binDir  string
}

// Skill 技能
type Skill struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	BinPath     string   `json:"binPath"`
	Commands    []string `json:"commands,omitempty"`
	Installed   bool     `json:"installed"`
	Version     string   `json:"version,omitempty"`
}

// New 创建技能管理器
func New(binDir string) *Manager {
	return &Manager{
		skills: make(map[string]*Skill),
		binDir: binDir,
	}
}

// List 列出所有技能
func (m *Manager) List() []*Skill {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]*Skill, 0, len(m.skills))
	for _, s := range m.skills {
		result = append(result, s)
	}
	return result
}

// Get 获取技能
func (m *Manager) Get(name string) (*Skill, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	s, ok := m.skills[name]
	return s, ok
}

// Install 安装技能
func (m *Manager) Install(ctx context.Context, name, repoURL string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	slog.Info("installing skill", "name", name, "repo", repoURL)

	// 确保 bin 目录存在
	if err := os.MkdirAll(m.binDir, 0755); err != nil {
		return fmt.Errorf("create bin dir failed: %w", err)
	}

	// 检查是否有 git
	if _, err := exec.LookPath("git"); err != nil {
		return fmt.Errorf("git not found: %w", err)
	}

	// 克隆仓库
	skillPath := filepath.Join(m.binDir, name)
	if _, err := os.Stat(skillPath); err == nil {
		// 已存在，执行 git pull
		cmd := exec.CommandContext(ctx, "git", "pull")
		cmd.Dir = skillPath
		if output, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("git pull failed: %w, output: %s", err, string(output))
		}
	} else {
		cmd := exec.CommandContext(ctx, "git", "clone", repoURL, skillPath)
		if output, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("git clone failed: %w, output: %s", err, string(output))
		}
	}

	// 检查是否有安装脚本
	installScript := filepath.Join(skillPath, "install.sh")
	if _, err := os.Stat(installScript); err == nil {
		cmd := exec.CommandContext(ctx, "bash", installScript)
		cmd.Dir = skillPath
		if output, err := cmd.CombinedOutput(); err != nil {
			slog.Warn("install script failed", "output", string(output))
		}
	}

	// 注册技能
	m.skills[name] = &Skill{
		Name:        name,
		Description: fmt.Sprintf("Skill from %s", repoURL),
		BinPath:     skillPath,
		Installed:   true,
	}

	slog.Info("skill installed", "name", name)
	return nil
}

// Update 更新技能
func (m *Manager) Update(ctx context.Context, name string) error {
	m.mu.RLock()
	skill, ok := m.skills[name]
	m.mu.RUnlock()

	if !ok {
		return fmt.Errorf("skill %q not found", name)
	}

	slog.Info("updating skill", "name", name)

	// 执行 git pull
	cmd := exec.CommandContext(ctx, "git", "pull")
	cmd.Dir = skill.BinPath
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git pull failed: %w, output: %s", err, string(output))
	}

	slog.Info("skill updated", "name", name)
	return nil
}

// Uninstall 卸载技能
func (m *Manager) Uninstall(ctx context.Context, name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	skill, ok := m.skills[name]
	if !ok {
		return fmt.Errorf("skill %q not found", name)
	}

	// 检查是否有卸载脚本
	uninstallScript := filepath.Join(skill.BinPath, "uninstall.sh")
	if _, err := os.Stat(uninstallScript); err == nil {
		cmd := exec.CommandContext(ctx, "bash", uninstallScript)
		cmd.Dir = skill.BinPath
		if output, err := cmd.CombinedOutput(); err != nil {
			slog.Warn("uninstall script failed", "output", string(output))
		}
	}

	// 移除目录
	if err := os.RemoveAll(skill.BinPath); err != nil {
		return fmt.Errorf("remove skill dir failed: %w", err)
	}

	// 移除注册
	delete(m.skills, name)

	slog.Info("skill uninstalled", "name", name)
	return nil
}

// Execute 执行技能命令
func (m *Manager) Execute(ctx context.Context, name string, args []string) (string, error) {
	m.mu.RLock()
	skill, ok := m.skills[name]
	m.mu.RUnlock()

	if !ok {
		return "", fmt.Errorf("skill %q not found", name)
	}

	// 查找可执行文件
	binName := name
	if len(skill.Commands) > 0 {
		binName = skill.Commands[0]
	}

	binPath := filepath.Join(skill.BinPath, "bin", binName)
	if _, err := os.Stat(binPath); os.IsNotExist(err) {
		// 尝试直接在技能目录查找
		binPath = filepath.Join(skill.BinPath, binName)
		if _, err := os.Stat(binPath); os.IsNotExist(err) {
			return "", fmt.Errorf("skill binary not found: %s", binName)
		}
	}

	cmd := exec.CommandContext(ctx, binPath, args...)
	cmd.Dir = skill.BinPath
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("execute failed: %w, output: %s", err, string(output))
	}

	return string(output), nil
}

// Bins 列出所有可用的二进制文件（跨所有技能）
func (m *Manager) Bins() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	bins := []string{}
	for _, skill := range m.skills {
		binPath := filepath.Join(skill.BinPath, "bin")
		if entries, err := os.ReadDir(binPath); err == nil {
			for _, entry := range entries {
				if entry.IsDir() {
					continue
				}
				bins = append(bins, entry.Name())
			}
		}
	}
	return bins
}
