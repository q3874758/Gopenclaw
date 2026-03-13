package whatsapp

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"gopenclaw/internal/channels"
)

// Config WhatsApp 配置（Twilio 兼容）
type Config struct {
	AccountSID  string `json:"accountSid"`
	AuthToken   string `json:"authToken"`
	PhoneNumber string `json:"phoneNumber"`
	WebhookURL  string `json:"webhookUrl"`
}

// Adapter WhatsApp 通道适配器
type Adapter struct {
	cfg     *Config
	client  *http.Client
	status  channels.ChannelStatus
	mu      sync.RWMutex
	ctx     context.Context
	cancel  context.CancelFunc

	onMessage func(channels.InboundMessage) error
}

// New 创建 WhatsApp 适配器
func New(cfg *Config) *Adapter {
	ctx, cancel := context.WithCancel(context.Background())

	return &Adapter{
		cfg:    cfg,
		client: &http.Client{Timeout: 30 * time.Second},
		status: channels.ChannelStatusDisconnected,
		ctx:    ctx,
		cancel: cancel,
	}
}

func (a *Adapter) ID() channels.ChannelID   { return "whatsapp" }
func (a *Adapter) Name() string       { return "WhatsApp" }
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

	slog.Info("whatsapp adapter started")
	return nil
}

// Stop 停止适配器
func (a *Adapter) Stop() error {
	a.cancel()

	a.mu.Lock()
	a.status = channels.ChannelStatusDisconnected
	a.mu.Unlock()

	slog.Info("whatsapp adapter stopped")
	return nil
}

// Send 发送消息
func (a *Adapter) Send(ctx context.Context, msg channels.OutboundMessage) error {
	// 使用 Twilio API
	apiURL := fmt.Sprintf("https://api.twilio.com/2010-04-01/Accounts/%s/Messages.json", a.cfg.AccountSID)

	// 构建 form data
	body := fmt.Sprintf("To=%s&From=%s&Body=%s",
		"whatsapp:"+msg.To,
		"whatsapp:"+a.cfg.PhoneNumber,
		urlEncode(msg.Content),
	)

	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, strings.NewReader(body))
	req.SetBasicAuth(a.cfg.AccountSID, a.cfg.AuthToken)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := a.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("twilio API error: %d", resp.StatusCode)
	}
	return nil
}

// HandleWebhook 处理 webhook 请求
func (a *Adapter) HandleWebhook(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, _ := io.ReadAll(r.Body)
	defer r.Body.Close()

	// 解析 form data
	values, err := parseForm(string(body))
	if err != nil {
		http.Error(w, "parse error", http.StatusBadRequest)
		return
	}

	// 只处理 WhatsApp 消息
	if values.Get("MessageSid") == "" {
		w.WriteHeader(http.StatusOK)
		return
	}

	inbound := channels.InboundMessage{
		ID:       values.Get("MessageSid"),
		Content:  values.Get("Body"),
		Type:     "text",
		Source: channels.MessageSource{
			Channel:  "whatsapp",
			ChannelID: "whatsapp",
			From:    strings.TrimPrefix(values.Get("From"), "whatsapp:"),
			To:      strings.TrimPrefix(values.Get("To"), "whatsapp:"),
		},
		Timestamp: time.Now().UnixMilli(),
	}

	if a.onMessage != nil {
		a.onMessage(inbound)
	}

	w.WriteHeader(http.StatusOK)
}

// ============ 辅助函数 ============

func urlEncode(s string) string {
	result := ""
	for _, c := range s {
		switch c {
		case ' ':
			result += "%20"
		case '\n':
			result += "%0A"
		case '&':
			result += "%26"
		case '=':
			result += "%3D"
		default:
			result += string(c)
		}
	}
	return result
}

func parseForm(body string) (url.Values, error) {
	return url.ParseQuery(body)
}
