package channels

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
)

// ChannelID 通道 ID
type ChannelID string

// ChannelStatus 通道状态
type ChannelStatus string

const (
	ChannelStatusConnected    ChannelStatus = "connected"
	ChannelStatusDisconnected ChannelStatus = "disconnected"
	ChannelStatusError      ChannelStatus = "error"
)

// MessageSource 消息来源
type MessageSource struct {
	ChannelID  string `json:"channelId"`
	Channel   string `json:"channel"`
	From      string `json:"from"`
	To        string `json:"to"`
	GroupID   string `json:"groupId,omitempty"`
	ThreadID string `json:"threadId,omitempty"`
	UserID   string `json:"userId,omitempty"`
}

// InboundMessage 入站消息
type InboundMessage struct {
	ID        string        `json:"id"`
	Source   MessageSource `json:"source"`
	Content  string       `json:"content"`
	Type     string       `json:"type"` // "text", "image", "audio", etc.
	Raw      interface{}   `json:"raw,omitempty"`
	Timestamp int64       `json:"timestamp"`
}

// OutboundMessage 出站消息
type OutboundMessage struct {
	To      string `json:"to"`
	Content string `json:"content"`
	Type    string `json:"type"` // "text", "image", "audio", etc.
	Thread  string `json:"thread,omitempty"`
	ReplyTo string `json:"replyTo,omitempty"`
	File    *File  `json:"file,omitempty"` // 附件文件
}

// File 附件文件
type File struct {
	Name        string `json:"name"` // 文件名
	Content     []byte `json:"content"` // 文件内容 (base64 解码后)
	ContentType string `json:"contentType"` // MIME 类型
	URL         string `json:"url"` // 文件 URL (可选)
}

// ChannelAdapter 通道适配器接口
type ChannelAdapter interface {
	// ID 返回通道 ID
	ID() ChannelID
	// Name 返回通道名称
	Name() string
	// Status 返回通道状态
	Status() ChannelStatus
	// Start 启动通道
	Start(ctx context.Context) error
	// Stop 停止通道
	Stop() error
	// Send 发送消息
	Send(ctx context.Context, msg OutboundMessage) error
	// OnMessage 设置消息处理回调
	OnMessage(handler func(InboundMessage) error)
}

// Handler 消息处理函数
type Handler func(InboundMessage) error

// Registry 通道注册表
type Registry struct {
	mu        sync.RWMutex
	adapters  map[ChannelID]ChannelAdapter
	handlers  map[ChannelID][]Handler
}

// New 创建通道注册表
func New() *Registry {
	return &Registry{
		adapters: make(map[ChannelID]ChannelAdapter),
		handlers: make(map[ChannelID][]Handler),
	}
}

// Register 注册通道适配器
func (r *Registry) Register(adapter ChannelAdapter) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.adapters[adapter.ID()] = adapter
	r.handlers[adapter.ID()] = []Handler{}
}

// Unregister 注销通道适配器
func (r *Registry) Unregister(id ChannelID) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.adapters, id)
	delete(r.handlers, id)
}

// Get 获取通道适配器
func (r *Registry) Get(id ChannelID) (ChannelAdapter, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	adapter, ok := r.adapters[id]
	return adapter, ok
}

// List 列出所有通道
func (r *Registry) List() []ChannelAdapter {
	r.mu.RLock()
	defer r.mu.RUnlock()

	list := make([]ChannelAdapter, 0, len(r.adapters))
	for _, adapter := range r.adapters {
		list = append(list, adapter)
	}
	return list
}

// GetStatus 获取所有通道状态
func (r *Registry) GetStatus() map[ChannelID]ChannelStatus {
	r.mu.RLock()
	defer r.mu.RUnlock()

	status := make(map[ChannelID]ChannelStatus)
	for id, adapter := range r.adapters {
		status[id] = adapter.Status()
	}
	return status
}

// SendToChannel 发送消息到指定通道
func (r *Registry) SendToChannel(ctx context.Context, channelID ChannelID, msg OutboundMessage) error {
	r.mu.RLock()
	adapter, ok := r.adapters[channelID]
	r.mu.RUnlock()

	if !ok {
		return fmt.Errorf("channel %q not found", channelID)
	}

	return adapter.Send(ctx, msg)
}

// Broadcast 广播消息到所有已连接通道
func (r *Registry) Broadcast(ctx context.Context, msg OutboundMessage) error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var lastErr error
	for _, adapter := range r.adapters {
		if adapter.Status() == ChannelStatusConnected {
			if err := adapter.Send(ctx, msg); err != nil {
				slog.Error("broadcast failed", "channel", adapter.ID(), "err", err)
				lastErr = err
			}
		}
	}
	return lastErr
}

// MessageRouter 消息路由器
type MessageRouter struct {
	mu       sync.RWMutex
	routes   map[ChannelID][]Handler
	channels *Registry
}

// NewMessageRouter 创建消息路由器
func NewMessageRouter(channels *Registry) *MessageRouter {
	return &MessageRouter{
		routes:   make(map[ChannelID][]Handler),
		channels: channels,
	}
}

// AddRoute 添加路由
func (r *MessageRouter) AddRoute(channelID ChannelID, handler Handler) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.routes[channelID] = append(r.routes[channelID], handler)
}

// Route 路由消息
func (r *MessageRouter) Route(msg InboundMessage) error {
	r.mu.RLock()
	handlers := r.routes[ChannelID(msg.Source.ChannelID)]
	r.mu.RUnlock()

	if len(handlers) == 0 {
		slog.Warn("no handler for message", "channel", msg.Source.ChannelID)
		return nil
	}

	for _, handler := range handlers {
		if err := handler(msg); err != nil {
			return err
		}
	}
	return nil
}

// Config 通道配置
type Config struct {
	Enabled bool                   `json:"enabled"`
	Token   string                 `json:"token,omitempty"`
	Secret string                 `json:"secret,omitempty"`
	APIKey  string                `json:"apiKey,omitempty"`
	Extra   map[string]interface{} `json:"extra,omitempty"`
}

// MarshalJSON 序列化配置（隐藏敏感信息）
func (c *Config) MarshalJSON() ([]byte, error) {
	type Alias Config
	return json.Marshal(&struct {
		*Alias
		Token   string `json:"token,omitempty"`
		Secret string `json:"secret,omitempty"`
		APIKey  string `json:"apiKey,omitempty"`
	}{
		Alias:   (*Alias)(c),
		Token:   "***",
		Secret: "***",
		APIKey:  "***",
	})
}
