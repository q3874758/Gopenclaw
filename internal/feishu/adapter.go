package feishu

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"sync"
)

// Config 飞书配置
type Config struct {
	AppID        string   `json:"appId"`
	AppSecret    string   `json:"appSecret"`
	Verification string   `json:"verification,omitempty"`
	EncryptKey   string   `json:"encryptKey,omitempty"`
	AllowFrom    []string `json:"allowFrom,omitempty"` // 允许发送消息的用户 ID 列表
}

// Adapter 飞书适配器
type Adapter struct {
	cfg         *Config
	client      *http.Client
	accessToken string
	tokenMu     sync.RWMutex
	status      string
	handler     func(interface{}) error
}

// New 创建飞书适配器
func New(cfg *Config) *Adapter {
	return &Adapter{
		cfg:    cfg,
		client: &http.Client{},
		status: "stopped",
	}
}

// ID 返回适配器 ID
func (a *Adapter) ID() string {
	return "feishu"
}

// Name 返回适配器名称
func (a *Adapter) Name() string {
	return "飞书"
}

// Status 返回通道状态
func (a *Adapter) Status() string {
	return a.status
}

// Start 启动通道
func (a *Adapter) Start(ctx context.Context) error {
	if err := a.ensureToken(ctx); err != nil {
		return err
	}
	a.status = "running"
	slog.Info("feishu: started")
	return nil
}

// Stop 停止通道
func (a *Adapter) Stop() error {
	a.status = "stopped"
	slog.Info("feishu: stopped")
	return nil
}

// OnMessage 设置消息处理回调
func (a *Adapter) OnMessage(handler func(interface{}) error) {
	a.handler = handler
}

// Send 发送消息
func (a *Adapter) Send(ctx context.Context, msg interface{}) error {
	outbound, ok := msg.(struct {
		To      string `json:"to"`
		Content string `json:"content"`
		Type    string `json:"type"`
		Thread  string `json:"thread,omitempty"`
	})
	if !ok {
		// 尝试解析 OutboundMessage 格式
		var m struct {
			To      string `json:"to"`
			Content string `json:"content"`
			Type    string `json:"type"`
			Thread  string `json:"thread,omitempty"`
		}
		data, _ := json.Marshal(msg)
		json.Unmarshal(data, &m)
		outbound = m
	}

	// 确保有 access token
	if err := a.ensureToken(ctx); err != nil {
		return err
	}

	// 发送消息到飞书
	content := map[string]interface{}{
		"text": outbound.Content,
	}

	// 支持富文本消息
	if outbound.Type == "interactive" {
		content = map[string]interface{}{
			"card": buildCard(outbound.Content),
		}
	}

	payload := map[string]interface{}{
		"receive_id": outbound.To,
		"msg_type":   "text",
		"content":    content,
	}

	apiURL := fmt.Sprintf("https://open.feishu.cn/open-apis/im/v1/messages?receive_id_type=open_id")
	return a.sendREST(ctx, apiURL, payload)
}

// sendREST 发送 REST API 请求
func (a *Adapter) sendREST(ctx context.Context, apiURL string, payload map[string]interface{}) error {
	a.tokenMu.RLock()
	token := a.accessToken
	a.tokenMu.RUnlock()

	body, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, strings.NewReader(string(body)))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := a.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("feishu API error: %d %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return err
	}

	if result.Code != 0 {
		return fmt.Errorf("feishu API error: %d %s", result.Code, result.Msg)
	}

	return nil
}

// ensureToken 确保有有效的 access token
func (a *Adapter) ensureToken(ctx context.Context) error {
	a.tokenMu.Lock()
	defer a.tokenMu.Unlock()

	if a.accessToken != "" {
		return nil
	}

	// 获取 access token
	payload := map[string]interface{}{
		"app_id":     a.cfg.AppID,
		"app_secret": a.cfg.AppSecret,
	}

	body, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		"https://open.feishu.cn/open-apis/auth/v3/tenant_access_token/internal",
		strings.NewReader(string(body)))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := a.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var result struct {
		Code              int    `json:"code"`
		Msg               string `json:"msg"`
		TenantAccessToken string `json:"tenant_access_token"`
		Expire            int    `json:"expire"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return err
	}

	if result.Code != 0 {
		return fmt.Errorf("feishu auth error: %d %s", result.Code, result.Msg)
	}

	a.accessToken = result.TenantAccessToken
	slog.Info("feishu: got access token", "expire", result.Expire)

	return nil
}

// HandleWebhook 处理飞书 webhook
func (a *Adapter) HandleWebhook(data []byte) error {
	// 解析飞书 webhook payload
	var payload struct {
		Type    string `json:"type"`
		Challenge string `json:"challenge,omitempty"`
		Event    struct {
			Type      string `json:"type"`
			Message   struct {
				MessageID string `json:"message_id"`
				OpenID    string `json:"open_id"`
				ChatID    string `json:"chat_id"`
				Text      string `json:"text"`
			} `json:"message"`
		} `json:"event"`
	}

	if err := json.Unmarshal(data, &payload); err != nil {
		return err
	}

	// URL 验证
	if payload.Challenge != "" {
		// 返回 challenge
		return nil
	}

	// 处理消息事件
	if payload.Event.Type == "message" {
		msg := payload.Event.Message

		// 检查是否在允许列表中
		if len(a.cfg.AllowFrom) > 0 {
			allowed := false
			for _, id := range a.cfg.AllowFrom {
				if id == msg.OpenID || id == "*" {
					allowed = true
					break
				}
			}
			if !allowed {
				slog.Info("feishu: message from non-allowed user", "open_id", msg.OpenID)
				return nil
			}
		}

		// 发送到 Gateway
		slog.Info("feishu: received message", "open_id", msg.OpenID, "chat_id", msg.ChatID, "text", msg.Text)
		// 这里应该触发 Gateway 处理
		// 具体实现需要通过回调或 channel 发送
	}

	return nil
}

// buildCard 构建飞书卡片消息
func buildCard(content string) string {
	card := map[string]interface{}{
		"config": map[string]interface{}{
			"wide_screen_mode": true,
		},
		"header": map[string]interface{}{
			"title": map[string]interface{}{
				"tag":  "plain_text",
				"content": "Gopenclaw 消息",
			},
			"template": "blue",
		},
		"elements": []interface{}{
			map[string]interface{}{
				"tag": "div",
				"text": map[string]interface{}{
					"tag":  "plain_text",
					"content": content,
				},
			},
		},
	}
	cardJSON, _ := json.Marshal(card)
	return string(cardJSON)
}

// decodeMessage 解码消息（兼容旧格式）
func decodeMessage(msg interface{}, target interface{}) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, target)
}
