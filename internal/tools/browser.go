package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os/exec"
	"strings"
	"sync"
	"time"
)

// Config 浏览器工具配置
type Config struct {
	ExecutablePath string `json:"executablePath,omitempty"` // Chrome/Chromium 路径
	RemoteURL      string `json:"remoteUrl,omitempty"`     // 远程 CDP URL
	NoSandbox     bool   `json:"noSandbox,omitempty"`    // 禁用沙箱
	UserDataDir   string `json:"userDataDir,omitempty"`  // 用户数据目录
	Headless     bool   `json:"headless,omitempty"`     // 无头模式
	Width        int    `json:"width,omitempty"`        // 窗口宽度
	Height       int    `json:"height,omitempty"`       // 窗口高度
}

// Browser 浏览器控制器
type Browser struct {
	cfg      *Config
	conn     *Conn
	mu       sync.RWMutex
	running  bool
	ctx      context.Context
	cancel   context.CancelFunc
}

// Conn CDP 连接
type Conn struct {
	conn net.Conn
}

// NewBrowser 创建浏览器控制器
func NewBrowser(cfg *Config) *Browser {
	if cfg.Width == 0 {
		cfg.Width = 1920
	}
	if cfg.Height == 0 {
		cfg.Height = 1080
	}

	ctx, cancel := context.WithCancel(context.Background())
	return &Browser{
		cfg:    cfg,
		running: false,
		ctx:    ctx,
		cancel: cancel,
	}
}

// Start 启动浏览器
func (b *Browser) Start(ctx context.Context) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.running {
		return nil
	}

	// 如果配置了远程 CDP URL，直接连接
	if b.cfg.RemoteURL != "" {
		conn, err := net.DialTimeout("tcp", b.cfg.RemoteURL, 10*time.Second)
		if err != nil {
			return fmt.Errorf("connect to remote CDP failed: %w", err)
		}
		b.conn = &Conn{conn: conn}
		b.running = true
		slog.Info("browser connected to remote", "url", b.cfg.RemoteURL)
		return nil
	}

	// 否则尝试启动本地 Chrome
	args := []string{
		"--remote-debugging-port=9222",
		fmt.Sprintf("--window-size=%d,%d", b.cfg.Width, b.cfg.Height),
	}

	if b.cfg.Headless {
		args = append(args, "--headless=new")
	}
	if b.cfg.NoSandbox {
		args = append(args, "--no-sandbox", "--disable-dev-shm-usage")
	}
	if b.cfg.UserDataDir != "" {
		args = append(args, "--user-data-dir="+b.cfg.UserDataDir)
	}

	// 启动 Chrome
	cmd := exec.CommandContext(ctx, b.cfg.ExecutablePath, args...)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start chrome failed: %w", err)
	}

	// 等待 CDP 端口
	time.Sleep(2 * time.Second)

	// 连接到 CDP
	conn, err := net.DialTimeout("tcp", "localhost:9222", 10*time.Second)
	if err != nil {
		return fmt.Errorf("connect to CDP failed: %w", err)
	}

	b.conn = &Conn{conn: conn}
	b.running = true
	slog.Info("browser started", "pid", cmd.Process.Pid)
	return nil
}

// Stop 停止浏览器
func (b *Browser) Stop() error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if !b.running {
		return nil
	}

	if b.conn != nil {
		b.conn.conn.Close()
	}
	b.cancel()

	b.running = false
	slog.Info("browser stopped")
	return nil
}

// IsRunning 检查是否运行中
func (b *Browser) IsRunning() bool {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.running
}

// ListTabs 列出所有标签页
func (b *Browser) ListTabs() ([]Tab, error) {
	if b.conn == nil {
		return nil, fmt.Errorf("browser not running")
	}

	resp, err := b.sendCommand("Target.getTargets", nil)
	if err != nil {
		return nil, err
	}

	var result struct {
		TargetInfos []struct {
			ID         string `json:"targetId"`
			Type      string `json:"type"`
			URL       string `json:"url"`
			Title     string `json:"title"`
		} `json:"targetInfos"`
	}

	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, err
	}

	var tabs []Tab
	for _, t := range result.TargetInfos {
		if t.Type == "page" {
			tabs = append(tabs, Tab{
				ID:    t.ID,
				URL:   t.URL,
				Title: t.Title,
			})
		}
	}

	return tabs, nil
}

// CreateTab 创建新标签页
func (b *Browser) CreateTab(url string) (string, error) {
	if b.conn == nil {
		return "", fmt.Errorf("browser not running")
	}

	resp, err := b.sendCommand("Target.createTarget", map[string]interface{}{
		"url": url,
	})
	if err != nil {
		return "", err
	}

	var result struct {
		TargetID string `json:"targetId"`
	}
	if err := json.Unmarshal(resp, &result); err != nil {
		return "", err
	}

	return result.TargetID, nil
}

// CloseTab 关闭标签页
func (b *Browser) CloseTab(tabID string) error {
	if b.conn == nil {
		return fmt.Errorf("browser not running")
	}

	_, err := b.sendCommand("Target.closeTarget", map[string]interface{}{
		"targetId": tabID,
	})
	return err
}

// Navigate 导航到 URL
func (b *Browser) Navigate(tabID, url string) error {
	if b.conn == nil {
		return fmt.Errorf("browser not running")
	}

	// 先附加到目标
	_, err := b.sendCommand("Target.attachToTarget", map[string]interface{}{
		"targetId": tabID,
		"flatten":  true,
	})
	if err != nil {
		return err
	}

	// 导航
	_, err = b.sendCommandOnTab(tabID, "Page.navigate", map[string]interface{}{
		"url": url,
	})
	return err
}

// GetHTML 获取页面 HTML
func (b *Browser) GetHTML(tabID string) (string, error) {
	if b.conn == nil {
		return "", fmt.Errorf("browser not running")
	}

	resp, err := b.sendCommandOnTab(tabID, "DOM.getDocument", nil)
	if err != nil {
		return "", err
	}

	var result struct {
		Root struct {
			NodeID int `json:"nodeId"`
		} `json:"root"`
	}
	if err := json.Unmarshal(resp, &result); err != nil {
		return "", err
	}

	// 获取 HTML
	resp, err = b.sendCommandOnTab(tabID, "DOM.getOuterHTML", map[string]interface{}{
		"nodeId": result.Root.NodeID,
	})
	if err != nil {
		return "", err
	}

	var htmlResult struct {
		OuterHTML string `json:"outerHTML"`
	}
	if err := json.Unmarshal(resp, &htmlResult); err != nil {
		return "", err
	}

	return htmlResult.OuterHTML, nil
}

// Screenshot 截图
func (b *Browser) Screenshot(tabID string, format string) (string, error) {
	if b.conn == nil {
		return "", fmt.Errorf("browser not running")
	}

	if format == "" {
		format = "png"
	}

	resp, err := b.sendCommandOnTab(tabID, "Page.captureScreenshot", map[string]interface{}{
		"format": format,
	})
	if err != nil {
		return "", err
	}

	var result struct {
		Data string `json:"data"`
	}
	if err := json.Unmarshal(resp, &result); err != nil {
		return "", err
	}

	return result.Data, nil
}

// Eval 在页面执行 JavaScript
func (b *Browser) Eval(tabID, script string) (string, error) {
	if b.conn == nil {
		return "", fmt.Errorf("browser not running")
	}

	resp, err := b.sendCommandOnTab(tabID, "Runtime.evaluate", map[string]interface{}{
		"expression":    script,
		"returnByValue": true,
	})
	if err != nil {
		return "", err
	}

	var result struct {
		Result struct {
			Type  string `json:"type"`
			Value any    `json:"value"`
		} `json:"result"`
	}
	if err := json.Unmarshal(resp, &result); err != nil {
		return "", err
	}

	if result.Result.Type == "undefined" {
		return "", nil
	}

	data, _ := json.Marshal(result.Result.Value)
	return string(data), nil
}

// sendCommand 发送 CDP 命令
func (b *Browser) sendCommand(method string, params interface{}) ([]byte, error) {
	id := time.Now().UnixNano()

	req := CDPRequest{
		ID:     id,
		Method: method,
		Params: params,
	}

	data, _ := json.Marshal(req)
	data = append(data, '\n')

	if _, err := b.conn.conn.Write(data); err != nil {
		return nil, err
	}

	// 读取响应
	buf := make([]byte, 8192)
	b.conn.conn.SetReadDeadline(time.Now().Add(30 * time.Second))
	n, err := b.conn.conn.Read(buf)
	if err != nil {
		return nil, err
	}

	return buf[:n], nil
}

// sendCommandOnTab 在指定标签页发送命令
func (b *Browser) sendCommandOnTab(tabID, method string, params interface{}) ([]byte, error) {
	id := time.Now().UnixNano()

	req := CDPRequest{
		ID:     id,
		Method: method,
		Params: params,
	}

	data, _ := json.Marshal(req)
	data = append(data, '\n')

	// 使用 HTTP POST 到 CDP 端点
	client := &http.Client{Timeout: 30 * time.Second}
	url := "http://localhost:9222/json/send/" + tabID
	resp, err := client.Post(url, "application/json", strings.NewReader(string(data)))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return io.ReadAll(resp.Body)
}

// CDPRequest CDP 请求
type CDPRequest struct {
	ID     int64       `json:"id"`
	Method string      `json:"method"`
	Params interface{} `json:"params,omitempty"`
}

// Tab 标签页信息
type Tab struct {
	ID    string `json:"id"`
	URL   string `json:"url"`
	Title string `json:"title"`
}

// NewDefaultConfig 返回默认配置
func NewDefaultConfig() *Config {
	return &Config{
		ExecutablePath: "chrome",
		Headless:       true,
		Width:         1920,
		Height:        1080,
	}
}

// BrowserExecutor 浏览器工具执行器
type BrowserExecutor struct {
	BaseExecutor
	baseURL  string
	browser *Browser
	mu      sync.Mutex
}

// NewBrowserExecutor 创建浏览器执行器
func NewBrowserExecutor(cfg *Config) *BrowserExecutor {
	return &BrowserExecutor{
		BaseExecutor: BaseExecutor{
			name:        "browser",
			description: "Control browser: list tabs, navigate, screenshot, evaluate JS",
		},
		browser: NewBrowser(cfg),
	}
}

// Execute 执行浏览器操作
func (e *BrowserExecutor) Execute(ctx context.Context, args map[string]interface{}) (string, error) {
	action, _ := args["action"].(string)
	if action == "" {
		return "", fmt.Errorf("action required")
	}

	// 确保浏览器运行
	if !e.browser.IsRunning() {
		if err := e.browser.Start(ctx); err != nil {
			return "", err
		}
	}

	switch action {
	case "list":
		tabs, err := e.browser.ListTabs()
		if err != nil {
			return "", err
		}
		data, _ := json.Marshal(tabs)
		return string(data), nil

	case "create":
		url, _ := args["url"].(string)
		if url == "" {
			url = "about:blank"
		}
		tabID, err := e.browser.CreateTab(url)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf(`{"tabId":"%s"}`, tabID), nil

	case "close":
		tabID, _ := args["tabId"].(string)
		if tabID == "" {
			return "", fmt.Errorf("tabId required")
		}
		if err := e.browser.CloseTab(tabID); err != nil {
			return "", err
		}
		return `{"ok":true}`, nil

	case "navigate":
		tabID, _ := args["tabId"].(string)
		url, _ := args["url"].(string)
		if tabID == "" || url == "" {
			return "", fmt.Errorf("tabId and url required")
		}
		if err := e.browser.Navigate(tabID, url); err != nil {
			return "", err
		}
		return `{"ok":true}`, nil

	case "html":
		tabID, _ := args["tabId"].(string)
		if tabID == "" {
			return "", fmt.Errorf("tabId required")
		}
		html, err := e.browser.GetHTML(tabID)
		if err != nil {
			return "", err
		}
		return html, nil

	case "screenshot":
		tabID, _ := args["tabId"].(string)
		format, _ := args["format"].(string)
		if tabID == "" {
			return "", fmt.Errorf("tabId required")
		}
		data, err := e.browser.Screenshot(tabID, format)
		if err != nil {
			return "", err
		}
		// 返回 base64 编码的图片
		return fmt.Sprintf(`{"format":"%s","data":"%s"}`, format, data), nil

	case "eval":
		tabID, _ := args["tabId"].(string)
		script, _ := args["script"].(string)
		if tabID == "" || script == "" {
			return "", fmt.Errorf("tabId and script required")
		}
		result, err := e.browser.Eval(tabID, script)
		if err != nil {
			return "", err
		}
		return result, nil

	default:
		return "", fmt.Errorf("unknown action: %s", action)
	}
}

// Stop 停止浏览器
func (e *BrowserExecutor) Stop() error {
	return e.browser.Stop()
}
