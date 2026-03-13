package signal

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"gopenclaw/internal/channels"
)

// Config Signal 配置
type Config struct {
	PhoneNumber string `json:"phoneNumber"`
	APIURL      string `json:"apiUrl"` // Signal API 服务器地址
	AuthToken   string `json:"authToken"`
}

// Adapter Signal 通道适配器
type Adapter struct {
	cfg     *Config
	client  *http.Client
	status  channels.ChannelStatus
	mu      sync.RWMutex
	ctx     context.Context
	cancel  context.CancelFunc
}

// New 创建 Signal 适配器
func New(cfg *Config) *Adapter {
	if cfg.APIURL == "" {
		cfg.APIURL = "http://localhost:8080"
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &Adapter{
		cfg:     cfg,
		client:  &http.Client{Timeout: 30 * time.Second},
		status:  channels.ChannelStatusDisconnected,
		ctx:     ctx,
		cancel:  cancel,
	}
}

// ID 返回通道 ID
func (a *Adapter) ID() channels.ChannelID {
	return "signal"
}

// Name 返回通道名称
func (a *Adapter) Name() string {
	return "Signal"
}

// Status 返回通道状态
func (a *Adapter) Status() channels.ChannelStatus {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.status
}

// Start 启动适配器
func (a *Adapter) Start(ctx context.Context) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.status == channels.ChannelStatusConnected {
		return nil
	}

	// 测试连接
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, a.cfg.APIURL+"/v1/health", nil)
	if err != nil {
		return err
	}

	resp, err := a.client.Do(req)
	if err != nil {
		a.status = channels.ChannelStatusError
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		a.status = channels.ChannelStatusError
		return fmt.Errorf("signal API health check failed: %d", resp.StatusCode)
	}

	a.status = channels.ChannelStatusConnected
	slog.Info("signal adapter started")
	return nil
}

// Stop 停止适配器
func (a *Adapter) Stop() error {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.cancel()
	a.status = channels.ChannelStatusDisconnected
	slog.Info("signal adapter stopped")
	return nil
}

// Send 发送消息
func (a *Adapter) Send(ctx context.Context, msg channels.OutboundMessage) error {
	a.mu.RLock()
	defer a.mu.RUnlock()

	if a.status != channels.ChannelStatusConnected {
		return fmt.Errorf("signal adapter not connected")
	}

	// 构造 Signal API 请求
	payload := map[string]interface{}{
		"recipient": msg.To,
		"message":   msg.Content,
	}

	data, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, a.cfg.APIURL+"/v2/send", strings.NewReader(string(data)))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+a.cfg.AuthToken)

	resp, err := a.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("send failed: %d %s", resp.StatusCode, string(body))
	}

	return nil
}

// OnMessage 设置消息处理回调
func (a *Adapter) OnMessage(handler func(channels.InboundMessage) error) {
	// Signal 通常使用 webhook 接收消息
	// 这里只保留接口，实际实现在 webhook handler 中
}

// HandleWebhook 处理 webhook
func (a *Adapter) HandleWebhook(data []byte) error {
	// 解析 Signal webhook 数据
	var payload struct {
		Envelope struct {
			Source     string `json:"source"`
			SourceNumber string `json:"sourceNumber"`
			Message    string `json:"message"`
		} `json:"envelope"`
	}

	if err := json.Unmarshal(data, &payload); err != nil {
		return err
	}

	msg := channels.InboundMessage{
		ID:       "",
		Source:   channels.MessageSource{
			Channel: "signal",
			From:    payload.Envelope.Source,
			To:      a.cfg.PhoneNumber,
		},
		Content:  payload.Envelope.Message,
		Type:     "text",
		Timestamp: 0,
	}

	return nil
}
