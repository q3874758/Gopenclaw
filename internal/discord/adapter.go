package discord

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"gopenclaw/internal/channels"
)

// Config Discord 配置
type Config struct {
	Token       string `json:"token"`
	ApplicationID string `json:"applicationId"`
	PublicKey   string `json:"publicKey"`
	WebhookURL  string `json:"webhookUrl"`
}

// Adapter Discord 通道适配器
type Adapter struct {
	cfg     *Config
	client  *http.Client
	status channels.ChannelStatus
	mu     sync.RWMutex
	ctx    context.Context
	cancel context.CancelFunc

	onMessage func(channels.InboundMessage) error
}

// New 创建 Discord 适配器
func New(cfg *Config) *Adapter {
	ctx, cancel := context.WithCancel(context.Background())

	return &Adapter{
		cfg:     cfg,
		client:  &http.Client{Timeout: 30 * time.Second},
		status:  channels.ChannelStatusDisconnected,
		ctx:    ctx,
		cancel:  cancel,
	}
}

func (a *Adapter) ID() channels.ChannelID   { return "discord" }
func (a *Adapter) Name() string         { return "Discord" }
func (a *Adapter) Status() channels.ChannelStatus {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.status
}

// SetMessageHandler 设置消息处理函数
func (a *Adapter) SetMessageHandler(handler func(channels.InboundMessage) error) {
	a.onMessage = handler
}

// OnMessage 设置消息处理回调（实现 ChannelAdapter 接口）
func (a *Adapter) OnMessage(handler func(channels.InboundMessage) error) {
	a.onMessage = handler
}

// Start 启动适配器
func (a *Adapter) Start(ctx context.Context) error {
	a.mu.Lock()
	a.status = channels.ChannelStatusConnected
	a.mu.Unlock()

	slog.Info("discord adapter started")
	return nil
}

// Stop 停止适配器
func (a *Adapter) Stop() error {
	a.cancel()

	a.mu.Lock()
	a.status = channels.ChannelStatusDisconnected
	a.mu.Unlock()

	slog.Info("discord adapter stopped")
	return nil
}

// Send 发送消息
func (a *Adapter) Send(ctx context.Context, msg channels.OutboundMessage) error {
	// 使用 Discord webhook 或 REST API
	if a.cfg.WebhookURL != "" {
		return a.sendWebhook(ctx, msg)
	}
	return a.sendREST(ctx, msg)
}

func (a *Adapter) sendWebhook(ctx context.Context, msg channels.OutboundMessage) error {
	payload := map[string]interface{}{
		"content": msg.Content,
	}

	data, _ := json.Marshal(payload)
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, a.cfg.WebhookURL, strings.NewReader(string(data)))
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("webhook error: %d", resp.StatusCode)
	}
	return nil
}

func (a *Adapter) sendREST(ctx context.Context, msg channels.OutboundMessage) error {
	// 使用 Discord REST API - channel.messages
	apiURL := fmt.Sprintf("https://discord.com/api/v10/channels/%s/messages", msg.To)

	// 自动检测并发送图片
	if msg.File == nil {
		msg.File = a.detectAndLoadImage(msg.Content)
	}

	// 如果有附件，使用 multipart/form-data
	if msg.File != nil && len(msg.File.Content) > 0 {
		return a.sendRESTWithFile(ctx, apiURL, msg)
	}

	data, _ := json.Marshal(map[string]interface{}{
		"content": msg.Content,
	})
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, strings.NewReader(string(data)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bot "+a.cfg.Token)

	resp, err := a.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("discord API error: %d", resp.StatusCode)
	}
	return nil
}

// detectAndLoadImage 检测消息内容中的图片路径并加载
func (a *Adapter) detectAndLoadImage(content string) *channels.File {
	// 图片扩展名
	extensions := []string{".png", ".jpg", ".jpeg", ".gif", ".webp", ".bmp"}

	// 常见的保存路径模式
	patterns := []string{
		"/workspace/",
		"saved to ",
		"written to ",
		"created at ",
	}

	// 先尝试从整个内容中提取路径
	for _, pattern := range patterns {
		// 查找 "patternxxx.png" 格式
		idx := strings.Index(content, pattern)
		if idx >= 0 {
			// 从 pattern 位置开始提取
			rest := content[idx+len(pattern):]
			// 找到第一个非路径字符（空格、引号、换行等）
			endIdx := len(rest)
			for i, c := range rest {
				if c == ' ' || c == '"' || c == '\'' || c == '`' || c == '\n' || c == '\r' {
					endIdx = i
					break
				}
			}
			path := strings.TrimSpace(rest[:endIdx])

			// 检查是否是图片文件
			for _, ext := range extensions {
				if strings.HasSuffix(strings.ToLower(path), ext) {
					// 尝试读取文件
					if data, err := os.ReadFile(path); err == nil {
						slog.Info("discord: detected and loading image", "path", path, "size", len(data))
						return &channels.File{
							Name:        filepath.Base(path),
							Content:     data,
							ContentType: a.getContentType(path),
						}
					}
				}
			}
		}
	}

	// 如果没有找到路径，尝试直接搜索 workspace 目录下的图片
	workspaceDirs := []string{"/workspace", ".", ".."}
	for _, dir := range workspaceDirs {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, entry := range entries {
			if !entry.IsDir() {
				name := entry.Name()
				for _, ext := range extensions {
					if strings.HasSuffix(strings.ToLower(name), ext) {
						path := filepath.Join(dir, name)
						if data, err := os.ReadFile(path); err == nil {
							// 检查文件是否刚刚修改（5秒内）
							info, _ := entry.Info()
							if time.Since(info.ModTime()) < 5*time.Second {
								slog.Info("discord: found recent image in workspace", "path", path, "size", len(data))
								return &channels.File{
									Name:        name,
									Content:     data,
									ContentType: a.getContentType(path),
								}
							}
						}
					}
				}
			}
		}
	}

	return nil
}

// getContentType 根据文件扩展名获取 MIME 类型
func (a *Adapter) getContentType(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".png":
		return "image/png"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".gif":
		return "image/gif"
	case ".webp":
		return "image/webp"
	case ".bmp":
		return "image/bmp"
	default:
		return "application/octet-stream"
	}
}

func (a *Adapter) sendRESTWithFile(ctx context.Context, apiURL string, msg channels.OutboundMessage) error {
	// 创建 multipart 请求
	var b bytes.Buffer
	writer := multipart.NewWriter(&b)

	// 添加 content 字段
	if msg.Content != "" {
		_ = writer.WriteField("content", msg.Content)
	}

	// 添加文件
	contentType := msg.File.ContentType
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	part, err := writer.CreateFormFile("file", msg.File.Name)
	if err != nil {
		return err
	}
	_, err = part.Write(msg.File.Content)
	if err != nil {
		return err
	}
	writer.Close()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, &b)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Authorization", "Bot "+a.cfg.Token)

	resp, err := a.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("discord API error: %d %s", resp.StatusCode, string(body))
	}
	return nil
}

// HandleWebhook 处理 Discord webhook (Interactions API)
func (a *Adapter) HandleWebhook(w http.ResponseWriter, r *http.Request) {
	// 验证请求签名
	if a.cfg.PublicKey != "" {
		if !a.verifyRequest(r) {
			http.Error(w, "invalid signature", http.StatusUnauthorized)
			return
		}
	}

	// 处理 interaction
	contentType := r.Header.Get("Content-Type")
	if strings.Contains(contentType, "application/json") {
		body, _ := io.ReadAll(r.Body)
		defer r.Body.Close()

		var interaction Interaction
		if err := json.Unmarshal(body, &interaction); err != nil {
			http.Error(w, "parse error", http.StatusBadRequest)
			return
		}

		// Ping Pong
		if interaction.Type == 1 { // Ping
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"type":1}`))
			return
		}

		// 处理消息组件
		if interaction.Type == 3 { // Message Component
			// 处理 button/select menu 点击
		}

		// 处理 slash command
		if interaction.Type == 2 && interaction.Data.Name != "" {
			options := ""
			if interaction.Data.Options != nil {
				for _, opt := range *interaction.Data.Options {
					options += " " + opt.Value
				}
			}

			inbound := channels.InboundMessage{
				ID:      interaction.ID,
				Content: "/" + interaction.Data.Name + options,
				Type:    "text",
				Source: channels.MessageSource{
					Channel: "discord",
					ChannelID: "discord",
					From:    interaction.Member.User.Username,
					To:      interaction.ChannelID,
				},
				Timestamp: time.Now().UnixMilli(),
			}

			if a.onMessage != nil {
				a.onMessage(inbound)
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"type":5}`)) // Ack with response
}

// verifyRequest 验证请求签名
func (a *Adapter) verifyRequest(r *http.Request) bool {
	// TODO: 实现 Discord 签名验证
	// https://discord.com/developers/docs/interactions/receiving-and-responding#security-and-authorization
	return true
}

// ============ Discord API 类型 ============

type Interaction struct {
	ID         string          `json:"id"`
	ApplicationID string       `json:"application_id"`
	Type      int             `json:"type"`
	Data      *InteractionData `json:"data,omitempty"`
	GuildID   string         `json:"guild_id,omitempty"`
	ChannelID string         `json:"channel_id,omitempty"`
	Member    *Member         `json:"member,omitempty"`
	User     *User           `json:"user,omitempty"`
}

type InteractionData struct {
	ID       string          `json:"id"`
	Name     string          `json:"name"`
	Options  *[]Option       `json:"options,omitempty"`
	CustomID string          `json:"custom_id,omitempty"`
}

type Option struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type Member struct {
	User      *User  `json:"user"`
	Nick     string `json:"nick,omitempty"`
	Roles    []string `json:"roles"`
	JoinedAt string `json:"joined_at"`
}

type User struct {
	ID        string `json:"id"`
	Username  string `json:"username"`
	Avatar    string `json:"avatar,omitempty"`
	Discriminator string `json:"discriminator,omitempty"`
}
