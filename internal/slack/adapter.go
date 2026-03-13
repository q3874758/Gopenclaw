package slack

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

// Config Slack 配置
type Config struct {
	Token   string `json:"token"`
	AppToken string `json:"appToken"` // Socket Mode 用
	WebhookURL string `json:"webhookUrl"`
	SigningSecret string `json:"signingSecret"`
}

// Adapter Slack 通道适配器
type Adapter struct {
	cfg    *Config
	client *http.Client
	status channels.ChannelStatus
	mu     sync.RWMutex
	ctx    context.Context
	cancel context.CancelFunc

	onMessage func(channels.InboundMessage) error
}

// New 创建 Slack 适配器
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

func (a *Adapter) ID() channels.ChannelID   { return "slack" }
func (a *Adapter) Name() string            { return "Slack" }
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

	slog.Info("slack adapter started")
	return nil
}

// Stop 停止适配器
func (a *Adapter) Stop() error {
	a.cancel()

	a.mu.Lock()
	a.status = channels.ChannelStatusDisconnected
	a.mu.Unlock()

	slog.Info("slack adapter stopped")
	return nil
}

// Send 发送消息
func (a *Adapter) Send(ctx context.Context, msg channels.OutboundMessage) error {
	// 使用 Incoming Webhook 或 chat.postMessage API
	if a.cfg.WebhookURL != "" {
		return a.sendWebhook(ctx, msg)
	}
	return a.sendAPI(ctx, msg)
}

func (a *Adapter) sendWebhook(ctx context.Context, msg channels.OutboundMessage) error {
	payload := map[string]interface{}{
		"text": msg.Content,
		"channel": msg.To,
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

func (a *Adapter) sendAPI(ctx context.Context, msg channels.OutboundMessage) error {
	// 使用 chat.postMessage API
	apiURL := "https://slack.com/api/chat.postMessage"
	payload := map[string]interface{}{
		"channel": msg.To,
		"text":   msg.Content,
	}

	data, _ := json.Marshal(payload)
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, strings.NewReader(string(data)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+a.cfg.Token)

	resp, err := a.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var result APIResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return err
	}

	if !result.Ok {
		return fmt.Errorf("slack API error: %s", result.Error)
	}
	return nil
}

// HandleWebhook 处理 Slack events
func (a *Adapter) HandleWebhook(w http.ResponseWriter, r *http.Request) {
	// 验证 signing secret
	if a.cfg.SigningSecret != "" {
		if !a.verifyRequest(r) {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
	}

	body, _ := io.ReadAll(r.Body)
	defer r.Body.Close()

	var event Event
	if err := json.Unmarshal(body, &event); err != nil {
		slog.Error("parse event failed", "err", err)
		http.Error(w, "parse error", http.StatusBadRequest)
		return
	}

	// URL 验证
	if event.Type == "url_verification" {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"challenge":"` + event.Challenge + `"}`))
		return
	}

	// 消息事件
	if event.Type == "event_callback" && event.Event != nil {
		msg := event.Event

		// 忽略机器人消息
		if msg.SubType == "bot_message" {
			w.WriteHeader(http.StatusOK)
			return
		}

		inbound := channels.InboundMessage{
			ID:       msg.Ts,
			Content:  msg.Text,
			Type:     "text",
			Source: channels.MessageSource{
				Channel: "slack",
				ChannelID: "slack",
				From:    msg.User,
				To:      msg.Channel,
				ThreadID: msg.ThreadTS,
			},
			Timestamp: time.Now().UnixMilli(),
		}

		if a.onMessage != nil {
			a.onMessage(inbound)
		}
	}

	w.WriteHeader(http.StatusOK)
}

// verifyRequest 验证请求签名
func (a *Adapter) verifyRequest(r *http.Request) bool {
	// TODO: 实现 Slack signing secret 验证
	return true
}

// ============ Slack API 类型 ============

type Event struct {
	Type      string   `json:"type"`
	Challenge string   `json:"challenge,omitempty"`
	Event    *MessageEvent `json:"event,omitempty"`
}

type MessageEvent struct {
	Type      string `json:"type"`
	Channel  string `json:"channel"`
	User     string `json:"user"`
	Text     string `json:"text"`
	Ts       string `json:"ts"`
	ThreadTS string `json:"thread_ts,omitempty"`
	SubType  string `json:"subtype,omitempty"`
}

type APIResponse struct {
	Ok   bool   `json:"ok"`
	Error string `json:"error,omitempty"`
}
