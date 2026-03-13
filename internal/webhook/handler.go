package webhook

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"
)

// Config webhook 配置
type Config struct {
	Enabled               bool     `json:"enabled"`
	Token                 string   `json:"token"`
	Path                  string   `json:"path"` // 默认 "/hooks"
	AllowedAgentIDs       []string `json:"allowedAgentIds"`
	DefaultSessionKey     string   `json:"defaultSessionKey"`
	AllowRequestSessionKey bool   `json:"allowRequestSessionKey"`
	AllowedSessionKeyPrefixes []string `json:"allowedSessionKeyPrefixes"`
}

// HookEvent webhook 事件
type HookEvent struct {
	// wake 事件
	Text string `json:"text"`
	Mode string `json:"mode"` // "now" | "next-heartbeat"

	// agent 事件
	Message  string `json:"message"`
	Name     string `json:"name"`
	AgentID  string `json:"agentId"`
	SessionKey string `json:"sessionKey"`
	WakeMode string `json:"wakeMode"`
	Deliver *bool  `json:"deliver"`
	Channel  string `json:"channel"`
	To      string `json:"to"`
	Model    string `json:"model"`
	Thinking string `json:"thinking"`
	TimeoutSeconds int `json:"timeoutSeconds"`
}

// EventHandler 事件处理器
type EventHandler interface {
	HandleWake(ctx context.Context, event HookEvent) error
	HandleAgent(ctx context.Context, event HookEvent) (string, error)
}

// Handler webhook 处理器
type Handler struct {
	mu       sync.RWMutex
	config   *Config
	server   *http.Server
	handlers map[string]func(ctx context.Context, payload map[string]any) (string, error)
	clientIPRateLimit map[string]time.Time // IP -> 最后一次失败时间
	rateLimitLock sync.Mutex
}

// New 创建 webhook 处理器
func New(config *Config) *Handler {
	if config.Path == "" {
		config.Path = "/hooks"
	}
	return &Handler{
		config:            config,
		handlers:          make(map[string]func(ctx context.Context, payload map[string]any) (string, error)),
		clientIPRateLimit: make(map[string]time.Time),
	}
}

// SetHandler 设置事件处理器
func (h *Handler) SetHandler(handler EventHandler) {
	h.mu.Lock()
	defer h.mu.Unlock()
	// 内部处理
}

// RegisterMapping 注册自定义映射
func (h *Handler) RegisterMapping(name string, fn func(ctx context.Context, payload map[string]any) (string, error)) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.handlers[name] = fn
}

// Start 启动 webhook 服务器
func (h *Handler) Start(ctx context.Context, addr string) error {
	if !h.config.Enabled {
		slog.Info("webhook disabled in config")
		return nil
	}

	mux := http.NewServeMux()
	
	// /hooks/wake
	mux.HandleFunc(h.config.Path+"/wake", h.handleWake)
	// /hooks/agent
	mux.HandleFunc(h.config.Path+"/agent", h.handleAgent)
	// /hooks/<name> 自定义映射
	mux.HandleFunc(h.config.Path+"/", h.handleMapping)
	// health
	mux.HandleFunc("/health", h.handleHealth)

	h.server = &http.Server{
		Addr:         addr,
		Handler:      h,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	go func() {
		<-ctx.Done()
		_ = h.server.Shutdown(context.Background())
	}()

	slog.Info("webhook server started", "addr", addr, "path", h.config.Path)
	return h.server.ListenAndServe()
}

// ServeHTTP 实现 http.Handler
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h.config.Enabled {
		h.authAndServe(w, r)
	} else {
		http.Error(w, "webhook disabled", http.StatusForbidden)
	}
}

// authAndServe 认证并服务请求
func (h *Handler) authAndServe(w http.ResponseWriter, r *http.Request) {
	// 检查是否启用
	if !h.config.Enabled {
		http.Error(w, "webhook disabled", http.StatusForbidden)
		return
	}

	// 检查认证
	if err := h.authenticate(r); err != nil {
		h.rateLimit(r)
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	// 继续处理
	if h.server.Handler != nil {
		h.server.Handler.ServeHTTP(w, r)
	}
}

// authenticate 认证请求
func (h *Handler) authenticate(r *http.Request) error {
	if h.config.Token == "" {
		return fmt.Errorf("webhook token not configured")
	}

	// 优先从 Authorization header 获取
	auth := r.Header.Get("Authorization")
	if auth != "" {
		if strings.HasPrefix(auth, "Bearer ") {
			token := auth[7:]
			if hmac.Equal([]byte(token), []byte(h.config.Token)) {
				return nil
			}
		}
	}

	// 尝试 x-openclaw-token
	if token := r.Header.Get("x-openclaw-token"); token != "" {
		if hmac.Equal([]byte(token), []byte(h.config.Token)) {
			return nil
		}
	}

	// 拒绝 query string token
	if r.URL.Query().Get("token") != "" {
		return fmt.Errorf("query-string tokens are rejected")
	}

	return fmt.Errorf("invalid token")
}

// rateLimit 速率限制
func (h *Handler) rateLimit(r *http.Request) {
	h.rateLimitLock.Lock()
	defer h.rateLimitLock.Unlock()

	ip := strings.Split(r.RemoteAddr, ":")[0]
	h.clientIPRateLimit[ip] = time.Now()

	// 清理旧记录（超过 1 小时）
	cutoff := time.Now().Add(-1 * time.Hour)
	for k, v := range h.clientIPRateLimit {
		if v.Before(cutoff) {
			delete(h.clientIPRateLimit, k)
		}
	}
}

// isRateLimited 检查是否被速率限制
func (h *Handler) isRateLimited(r *http.Request) bool {
	h.rateLimitLock.Lock()
	defer h.rateLimitLock.Unlock()

	ip := strings.Split(r.RemoteAddr, ":")[0]
	if last, ok := h.clientIPRateLimit[ip]; ok {
		// 5 分钟内超过 3 次失败
		if time.Since(last) < 5*time.Minute {
			return true
		}
	}
	return false
}

// handleHealth 健康检查
func (h *Handler) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"ok":true}`))
}

// handleWake 处理 wake 事件
func (h *Handler) handleWake(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	payload, err := h.parsePayload(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var event HookEvent
	if err := json.Unmarshal([]byte(payload), &event); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}

	// 验证必填字段
	if event.Text == "" {
		http.Error(w, "text is required", http.StatusBadRequest)
		return
	}

	// 验证 mode
	if event.Mode == "" {
		event.Mode = "now"
	}
	if event.Mode != "now" && event.Mode != "next-heartbeat" {
		http.Error(w, "mode must be 'now' or 'next-heartbeat'", http.StatusBadRequest)
		return
	}

	// TODO: 调用实际的 wake 处理逻辑
	slog.Info("webhook wake event", "text", event.Text, "mode", event.Mode)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"ok"}`))
}

// handleAgent 处理 agent 事件
func (h *Handler) handleAgent(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	payload, err := h.parsePayload(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var event HookEvent
	if err := json.Unmarshal([]byte(payload), &event); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}

	// 验证必填字段
	if event.Message == "" {
		http.Error(w, "message is required", http.StatusBadRequest)
		return
	}

	// 验证 sessionKey
	if event.SessionKey != "" && !h.config.AllowRequestSessionKey {
		http.Error(w, "sessionKey overrides are disabled", http.StatusBadRequest)
		return
	}

	// 验证 sessionKey 前缀
	if event.SessionKey != "" && len(h.config.AllowedSessionKeyPrefixes) > 0 {
		allowed := false
		for _, prefix := range h.config.AllowedSessionKeyPrefixes {
			if strings.HasPrefix(event.SessionKey, prefix) {
				allowed = true
				break
			}
		}
		if !allowed {
			http.Error(w, "sessionKey prefix not allowed", http.StatusBadRequest)
			return
		}
	}

	// 验证 agentId
	if event.AgentID != "" && len(h.config.AllowedAgentIDs) > 0 {
		allowed := false
		for _, id := range h.config.AllowedAgentIDs {
			if id == "*" || id == event.AgentID {
				allowed = true
				break
			}
		}
		if !allowed {
			http.Error(w, "agentId not allowed", http.StatusBadRequest)
			return
		}
	}

	// 验证 wakeMode
	if event.WakeMode == "" {
		event.WakeMode = "now"
	}

	// 验证 deliver 默认值
	if event.Deliver == nil {
		deliver := true
		event.Deliver = &deliver
	}

	// 验证 channel
	validChannels := map[string]bool{
		"last": true, "whatsapp": true, "telegram": true,
		"discord": true, "slack": true, "signal": true,
		"imessage": true, "msteams": true,
	}
	if event.Channel != "" && !validChannels[event.Channel] {
		http.Error(w, "invalid channel", http.StatusBadRequest)
		return
	}

	// TODO: 调用实际的 agent 处理逻辑
	slog.Info("webhook agent event", "message", event.Message, "name", event.Name, "agentId", event.AgentID)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"ok"}`))
}

// handleMapping 处理自定义映射
func (h *Handler) handleMapping(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 提取映射名称
	name := strings.TrimPrefix(r.URL.Path, h.config.Path+"/")
	if name == "" {
		http.Error(w, "mapping name required", http.StatusBadRequest)
		return
	}

	// 查找映射处理器
	h.mu.RLock()
	handler, ok := h.handlers[name]
	h.mu.RUnlock()

	if !ok {
		http.Error(w, "mapping not found", http.StatusNotFound)
		return
	}

	// 解析 payload
	payload, err := h.parsePayload(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var data map[string]any
	if err := json.Unmarshal([]byte(payload), &data); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}

	// 调用处理器
	result, err := handler(r.Context(), data)
	if err != nil {
		slog.Error("webhook mapping error", "name", name, "err", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if result == "" {
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	} else {
		_, _ = w.Write([]byte(result))
	}
}

// parsePayload 解析请求 payload
func (h *Handler) parsePayload(r *http.Request) (string, error) {
	// 检查 content-length
	if r.ContentLength > 1<<20 { // 1MB
		return "", fmt.Errorf("payload too large")
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		return "", fmt.Errorf("read body error")
	}
	defer r.Body.Close()

	return string(body), nil
}

// VerifySignature 验证 HMAC 签名（供外部使用）
func VerifySignature(payload []byte, token string, signature string) bool {
	mac := hmac.New(sha256.New, []byte(token))
	mac.Write(payload)
	expected := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(expected), []byte(signature))
}
