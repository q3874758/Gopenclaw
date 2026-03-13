package plugin

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"plugin"
	"strings"
	"sync"
)

// Plugin 插件接口
type Plugin interface {
	// Name 返回插件名称
	Name() string
	// Description 返回插件描述
	Description() string
	// Version 返回插件版本
	Version() string
	// Init 初始化插件
	Init(ctx context.Context, config json.RawMessage) error
	// Start 启动插件
	Start(ctx context.Context) error
	// Stop 停止插件
	Stop() error
}

// HookType 钩子类型
type HookType string

const (
	HookGatewayStart    HookType = "gateway.start"
	HookGatewayStop     HookType = "gateway.stop"
	HookAgentInvoke     HookType = "agent.invoke"
	HookAgentResponse   HookType = "agent.response"
	HookToolExecute     HookType = "tool.execute"
	HookMessageReceive HookType = "message.receive"
	HookMessageSend    HookType = "message.send"
	HookCronJobRun     HookType = "cron.job.run"
	HookWebhookReceive HookType = "webhook.receive"
)

// HookHandler 钩子处理函数
type HookHandler func(ctx context.Context, data interface{}) (interface{}, error)

// PluginInfo 插件信息
type Info struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Version     string          `json:"version"`
	Author      string          `json:"author"`
	Hooks       []HookType      `json:"hooks"`
	Config      json.RawMessage `json:"config,omitempty"`
}

// Manager 插件管理器
type Manager struct {
	mu       sync.RWMutex
	plugins  map[string]Plugin
	hooks    map[HookType][]HookHandler
	pluginDir string
	configDir string
}

// New 创建插件管理器
func New(pluginDir, configDir string) *Manager {
	return &Manager{
		plugins:  make(map[string]Plugin),
		hooks:    make(map[HookType][]HookHandler),
		pluginDir: pluginDir,
		configDir: configDir,
	}
}

// LoadPlugin 加载插件
func (m *Manager) LoadPlugin(ctx context.Context, name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 检查插件是否已加载
	if _, ok := m.plugins[name]; ok {
		slog.Info("plugin already loaded", "name", name)
		return nil
	}

	// 查找插件文件
	pluginPath := filepath.Join(m.pluginDir, fmt.Sprintf("%s.so", name))
	if _, err := os.Stat(pluginPath); os.IsNotExist(err) {
		// 尝试查找 .go 文件（开发模式）
		goPath := filepath.Join(m.pluginDir, name, "plugin.go")
		if _, err := os.Stat(goPath); os.IsNotExist(err) {
			return fmt.Errorf("plugin %q not found", name)
		}
		slog.Info("plugin source found, compilation required", "path", goPath)
		return fmt.Errorf("plugin %q requires compilation", name)
	}

	// 加载 .so 插件
	p, err := plugin.Open(pluginPath)
	if err != nil {
		return fmt.Errorf("open plugin failed: %w", err)
	}

	// 查找插件符号
	symPlugin, err := p.Lookup("Plugin")
	if err != nil {
		return fmt.Errorf("lookup Plugin symbol failed: %w", err)
	}

	// 类型断言
	pl, ok := symPlugin.(Plugin)
	if !ok {
		return fmt.Errorf("Plugin symbol is not a Plugin instance")
	}

	// 加载配置
	configPath := filepath.Join(m.configDir, fmt.Sprintf("%s.json", name))
	var config json.RawMessage
	if data, err := os.ReadFile(configPath); err == nil {
		config = data
	}

	// 初始化插件
	if err := pl.Init(ctx, config); err != nil {
		return fmt.Errorf("plugin init failed: %w", err)
	}

	// 启动插件
	if err := pl.Start(ctx); err != nil {
		return fmt.Errorf("plugin start failed: %w", err)
	}

	// 注册插件
	m.plugins[name] = pl

	slog.Info("plugin loaded", "name", name, "version", pl.Version())
	return nil
}

// UnloadPlugin 卸载插件
func (m *Manager) UnloadPlugin(ctx context.Context, name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	pl, ok := m.plugins[name]
	if !ok {
		return fmt.Errorf("plugin %q not found", name)
	}

	if err := pl.Stop(); err != nil {
		slog.Warn("plugin stop failed", "name", name, "err", err)
	}

	delete(m.plugins, name)
	slog.Info("plugin unloaded", "name", name)
	return nil
}

// ListPlugins 列出已加载的插件
func (m *Manager) ListPlugins() []*Info {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]*Info, 0, len(m.plugins))
	for _, pl := range m.plugins {
		result = append(result, &Info{
			Name:        pl.Name(),
			Description: pl.Description(),
			Version:     pl.Version(),
		})
	}
	return result
}

// RegisterHook 注册钩子
func (m *Manager) RegisterHook(hookType HookType, handler HookHandler) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.hooks[hookType] = append(m.hooks[hookType], handler)
	slog.Debug("hook registered", "type", hookType)
}

// TriggerHook 触发钩子
func (m *Manager) TriggerHook(ctx context.Context, hookType HookType, data interface{}) (interface{}, error) {
	m.mu.RLock()
	handlers, ok := m.hooks[hookType]
	m.mu.RUnlock()

	if !ok || len(handlers) == 0 {
		return data, nil
	}

	result := data
	var lastErr error

	for _, handler := range handlers {
		result, lastErr = handler(ctx, result)
		if lastErr != nil {
			slog.Error("hook handler error", "type", hookType, "err", lastErr)
			return result, lastErr
		}
	}

	return result, nil
}

// ScanPlugins 扫描插件目录
func (m *Manager) ScanPlugins() ([]string, error) {
	entries, err := os.ReadDir(m.pluginDir)
	if err != nil {
		return nil, err
	}

	var plugins []string
	for _, entry := range entries {
		if entry.IsDir() {
			plugins = append(plugins, entry.Name())
		}
		// 检查 .so 文件
		if strings.HasSuffix(entry.Name(), ".so") {
			name := strings.TrimSuffix(entry.Name(), ".so")
			plugins = append(plugins, name)
		}
	}

	return plugins, nil
}

// LoadAll 加载所有插件
func (m *Manager) LoadAll(ctx context.Context) error {
	plugins, err := m.ScanPlugins()
	if err != nil {
		return err
	}

	for _, name := range plugins {
		if err := m.LoadPlugin(ctx, name); err != nil {
			slog.Warn("load plugin failed", "name", name, "err", err)
		}
	}

	return nil
}

// UnloadAll 卸载所有插件
func (m *Manager) UnloadAll(ctx context.Context) error {
	m.mu.RLock()
	names := make([]string, 0, len(m.plugins))
	for name := range m.plugins {
		names = append(names, name)
	}
	m.mu.RUnlock()

	for _, name := range names {
		if err := m.UnloadPlugin(ctx, name); err != nil {
			slog.Warn("unload plugin failed", "name", name, "err", err)
		}
	}

	return nil
}

// BasePlugin 基础插件实现
type BasePlugin struct {
	name        string
	description string
	version    string
	ctx        context.Context
	config     json.RawMessage
}

// NewBasePlugin 创建基础插件
func NewBasePlugin(name, description, version string) *BasePlugin {
	return &BasePlugin{
		name:        name,
		description: description,
		version:    version,
	}
}

func (p *BasePlugin) Name() string        { return p.name }
func (p *BasePlugin) Description() string { return p.description }
func (p *BasePlugin) Version() string    { return p.version }

func (p *BasePlugin) Init(ctx context.Context, config json.RawMessage) error {
	p.ctx = ctx
	p.config = config
	return nil
}

func (p *BasePlugin) Start(ctx context.Context) error { return nil }
func (p *BasePlugin) Stop() error                      { return nil }

// Config 获取配置
func (p *BasePlugin) Config() json.RawMessage {
	return p.config
}
