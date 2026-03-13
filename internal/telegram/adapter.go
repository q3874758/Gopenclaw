package telegram

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"gopenclaw/internal/channels"
)

// Config Telegram 配置
type Config struct {
	Token         string   `json:"token"`
	APIURL        string   `json:"apiUrl"` // 可选，自定义 API 地址
	WebhookURL    string   `json:"webhookUrl"`
	Secret       string   `json:"secret"`
	AllowFrom    []string `json:"allowFrom"`    // 白名单用户
	AllowGroups  []string `json:"allowGroups"`  // 白名单群组
	RequireMention bool   `json:"requireMention"` // 是否需要 @ 提及
	DM            *DMConfig `json:"dm,omitempty"` // DM 配置
}

// DMConfig DM 配置
type DMConfig struct {
	Enabled       bool     `json:"enabled"`        // 是否启用 DM
	AllowList    []string `json:"allowList"`     // DM 白名单
	DedupeWindow int      `json:"dedupeWindow"`  // 去重时间窗口（毫秒）
}

// seenMessage 已发送消息记录（用于去重和 echo 检测）
type seenMessage struct {
	Text     string
	ExpireAt time.Time
}

// Adapter Telegram 通道适配器
type Adapter struct {
	cfg    *Config
	client *http.Client
	status channels.ChannelStatus
	mu     sync.RWMutex
	ctx    context.Context
	cancel context.CancelFunc

	// 消息处理
	onMessage func(channels.InboundMessage) error

	// Bot API 地址
	apiURL string

	// 已发送消息记录（去重和 echo 检测）
	seenMessages    map[string]seenMessage
	seenMessagesMu sync.Mutex
}

// New 创建 Telegram 适配器
func New(cfg *Config) *Adapter {
	if cfg.APIURL == "" {
		cfg.APIURL = "https://api.telegram.org"
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &Adapter{
		cfg:    cfg,
		client: &http.Client{Timeout: 30 * time.Second},
		status: channels.ChannelStatusDisconnected,
		ctx:    ctx,
		cancel: cancel,
		apiURL: cfg.APIURL + "/bot" + cfg.Token,
		seenMessages: make(map[string]seenMessage),
	}
}

func (a *Adapter) ID() channels.ChannelID    { return "telegram" }
func (a *Adapter) Name() string             { return "Telegram" }
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

	// 如果配置了 webhook URL，设置 webhook
	if a.cfg.WebhookURL != "" {
		if err := a.setWebhook(ctx, a.cfg.WebhookURL); err != nil {
			slog.Warn("set webhook failed", "err", err)
		}
	}

	slog.Info("telegram adapter started")
	return nil
}

// Stop 停止适配器
func (a *Adapter) Stop() error {
	a.cancel()

	a.mu.Lock()
	a.status = channels.ChannelStatusDisconnected
	a.mu.Unlock()

	slog.Info("telegram adapter stopped")
	return nil
}

// Send 发送消息
func (a *Adapter) Send(ctx context.Context, msg channels.OutboundMessage) error {
	method := "sendMessage"
	params := map[string]interface{}{
		"chat_id": msg.To,
		"text":   msg.Content,
	}

	if msg.ReplyTo != "" {
		params["reply_to_message_id"] = msg.ReplyTo
	}

	// 记录发送的消息，用于 echo 检测（Same-phone mode）
	if a.cfg.DM != nil && a.cfg.DM.DedupeWindow > 0 {
		a.mu.Lock()
		msgKey := fmt.Sprintf("sent:%s:%d", msg.To, time.Now().UnixNano())
		a.seenMessages[msgKey] = seenMessage{
			Text:     msg.Content,
			ExpireAt: time.Now().Add(time.Duration(a.cfg.DM.DedupeWindow) * time.Millisecond),
		}
		a.mu.Unlock()
	}

	return a.callAPI(ctx, method, params)
}

// HandleWebhook 处理 webhook 请求
func (a *Adapter) HandleWebhook(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 验证 secret
	if a.cfg.Secret != "" {
		secret := r.Header.Get("X-Telegram-Bot-Api-Secret-Token")
		if secret != a.cfg.Secret {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "read body error", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	var update Update
	if err := json.Unmarshal(body, &update); err != nil {
		http.Error(w, "parse error", http.StatusBadRequest)
		return
	}

	if update.Message == nil {
		w.WriteHeader(http.StatusOK)
		return
	}

	// 判断是否为 DM（私聊）
	isDM := update.Message.Chat.Type == "private"

	// ============ DM 去重逻辑 ============
	// 按 chat.id + message_id + text 生成唯一 key，防止同一 DM 触发重复回复
	msgKey := fmt.Sprintf("dm:%d:%d", update.Message.Chat.ID, update.Message.MessageID)
	if isDM && a.cfg.DM != nil && a.cfg.DM.DedupeWindow > 0 {
		a.mu.Lock()
		if _, exists := a.seenMessages[msgKey]; exists {
			a.mu.Unlock()
			slog.Debug("telegram dm duplicate, skipping", "msgId", update.Message.MessageID)
			w.WriteHeader(http.StatusOK)
			return
		}
		// 记录消息
		a.seenMessages[msgKey] = seenMessage{
			Text:     update.Message.Text,
			ExpireAt: time.Now().Add(time.Duration(a.cfg.DM.DedupeWindow) * time.Millisecond),
		}
		a.mu.Unlock()
	}

	// ============ Echo 检测（自聊天模式）============
	// Same-phone mode: 当 from == to 时检测 echo
	if isDM && update.Message.Text != "" {
		// 检查是否在已发送消息列表中
		a.mu.Lock()
		for key, seen := range a.seenMessages {
			if strings.HasPrefix(key, "sent:") && seen.Text == update.Message.Text {
				// 发现 echo，忽略
				delete(a.seenMessages, key)
				a.mu.Unlock()
				slog.Debug("telegram echo detected, skipping", "text", update.Message.Text)
				w.WriteHeader(http.StatusOK)
				return
			}
		}
		a.mu.Unlock()
	}

	// 转换为入站消息
	inbound := channels.InboundMessage{
		ID:      strconv.Itoa(update.Message.MessageID),
		Content: update.Message.Text,
		Type:    "text",
		Timestamp: update.Message.Date,
		Source: channels.MessageSource{
			Channel: "telegram",
			ChannelID: "telegram",
			From: update.Message.From.Username,
			To:   strconv.Itoa(update.Message.Chat.ID),
		},
		Raw: update,
	}

	// 调用消息处理函数
	if a.onMessage != nil {
		if err := a.onMessage(inbound); err != nil {
			slog.Error("handle message failed", "err", err)
		}
	}

	w.WriteHeader(http.StatusOK)
}

// callAPI 调用 Telegram Bot API
func (a *Adapter) callAPI(ctx context.Context, method string, params map[string]interface{}) error {
	data, err := json.Marshal(params)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, a.apiURL+"/"+method, strings.NewReader(string(data)))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := a.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("API error: %d", resp.StatusCode)
	}

	var result APIResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return err
	}

	if !result.Ok {
		return fmt.Errorf("API error: %d %s", result.ErrorCode, result.Description)
	}

	return nil
}

// setWebhook 设置 webhook
func (a *Adapter) setWebhook(ctx context.Context, url string) error {
	params := map[string]interface{}{
		"url":          url,
		"secret_token": a.cfg.Secret,
	}
	return a.callAPI(ctx, "setWebhook", params)
}

// ============ Telegram API 类型 ============

type Update struct {
	UpdateID int     `json:"update_id"`
	Message *Message `json:"message,omitempty"`
}

type Message struct {
	MessageID           int            `json:"message_id"`
	From               *User          `json:"from"`
	Date               int64          `json:"date"`
	Chat               *Chat          `json:"chat"`
	Text               string         `json:"text"`
	Entities           []MessageEntity `json:"entities,omitempty"`
}

type User struct {
	ID           int    `json:"id"`
	IsBot        bool   `json:"is_bot"`
	FirstName   string `json:"first_name"`
	LastName    string `json:"last_name,omitempty"`
	Username    string `json:"username,omitempty"`
	LanguageCode string `json:"language_code,omitempty"`
}

type Chat struct {
	ID        int    `json:"id"`
	Type      string `json:"type"` // "private", "group", "supergroup", "channel"
	Title     string `json:"title,omitempty"`
	Username  string `json:"username,omitempty"`
	FirstName string `json:"first_name,omitempty"`
	LastName  string `json:"last_name,omitempty"`
}

type MessageEntity struct {
	Type   string `json:"type"`
	Offset int    `json:"offset"`
	Length int    `json:"length"`
	URL    string `json:"url,omitempty"`
	User   *User  `json:"user,omitempty"`
}

type APIResponse struct {
	Ok          bool        `json:"ok"`
	ErrorCode   int         `json:"error_code,omitempty"`
	Description string      `json:"description,omitempty"`
	Result      interface{} `json:"result,omitempty"`
}
