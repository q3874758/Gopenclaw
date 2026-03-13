package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Message 会话消息
type Message struct {
	Role      string    `json:"role"`
	Content   string    `json:"content"`
	Timestamp time.Time `json:"timestamp"`
	ToolCalls []struct {
		ID       string `json:"id"`
		Name     string `json:"name"`
		Output   string `json:"output,omitempty"`
	} `json:"tool_calls,omitempty"`
}

// Session 会话
type Session struct {
	ID               string    `json:"id"`
	Label            string    `json:"label"`
	Messages         []Message `json:"messages"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
	SessionIntroSent bool      `json:"session_intro_sent,omitempty"` // 系统提示是否已发送
	Model            string    `json:"model,omitempty"`              // 当前使用的模型
	TraceID          string    `json:"trace_id,omitempty"`          // 会话追踪 ID
	DailyResetAt     time.Time `json:"daily_reset_at,omitempty"`    // 每日重置时间
	ThinkingLevel    string    `json:"thinking_level,omitempty"`   // thinking level (off/normal/high)
}

// Storage 会话存储
type Storage struct {
	mu       sync.RWMutex
	sessions map[string]*Session
	baseDir  string
}

// New 创建存储
func New(baseDir string) (*Storage, error) {
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return nil, err
	}
	return &Storage{
		sessions: make(map[string]*Session),
		baseDir:  baseDir,
	}, nil
}

// Load 加载所有会话
func (s *Storage) Load() error {
	files, err := os.ReadDir(s.baseDir)
	if err != nil {
		return err
	}

	for _, f := range files {
		if f.IsDir() || filepath.Ext(f.Name()) != ".json" {
			continue
		}
		path := filepath.Join(s.baseDir, f.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		var session Session
		if err := json.Unmarshal(data, &session); err != nil {
			continue
		}
		s.sessions[session.ID] = &session
	}
	return nil
}

// Save 保存会话到文件
func (s *Storage) Save(session *Session) error {
	session.UpdatedAt = time.Now()
	data, err := json.MarshalIndent(session, "", "  ")
	if err != nil {
		return err
	}
	path := filepath.Join(s.baseDir, session.ID+".json")
	return os.WriteFile(path, data, 0644)
}

// CreateSession 创建新会话
func (s *Storage) CreateSession(id, label string) *Session {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 生成 trace ID
	traceID := generateTraceID()

	session := &Session{
		ID:        id,
		Label:     label,
		Messages:  make([]Message, 0),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		TraceID:   traceID,
	}
	s.sessions[id] = session
	_ = s.Save(session)
	return session
}

// generateTraceID 生成唯一的追踪 ID
func generateTraceID() string {
	return fmt.Sprintf("tr-%d-%s", time.Now().UnixMilli(), randomString(8))
}

// randomString 生成随机字符串
func randomString(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[time.Now().UnixNano()%int64(len(letters))]
	}
	return string(b)
}

// GetSession 获取会话
func (s *Storage) GetSession(id string) (*Session, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	session, ok := s.sessions[id]
	return session, ok
}

// AddMessage 添加消息
func (s *Storage) AddMessage(sessionID string, msg Message) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	session, ok := s.sessions[sessionID]
	if !ok {
		return fmt.Errorf("session %q not found", sessionID)
	}

	msg.Timestamp = time.Now()
	session.Messages = append(session.Messages, msg)
	session.UpdatedAt = time.Now()

	go s.Save(session) // 异步保存
	return nil
}

// ListSessions 列出所有会话
func (s *Storage) ListSessions() []Session {
	s.mu.RLock()
	defer s.mu.RUnlock()

	list := make([]Session, 0, len(s.sessions))
	for _, s := range s.sessions {
		list = append(list, *s)
	}
	return list
}

// DeleteSession 删除会话
func (s *Storage) DeleteSession(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.sessions[id]; !ok {
		return fmt.Errorf("session %q not found", id)
	}

	delete(s.sessions, id)
	path := filepath.Join(s.baseDir, id+".json")
	return os.Remove(path)
}

// CleanupExpiredSessions 删除过期的会话
func (s *Storage) CleanupExpiredSessions(idleDays int) (int, error) {
	if idleDays <= 0 {
		idleDays = 7 // 默认7天
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	cutoff := time.Now().AddDate(0, 0, -idleDays)
	deleted := 0

	for id, session := range s.sessions {
		if session.UpdatedAt.Before(cutoff) {
			delete(s.sessions, id)
			path := filepath.Join(s.baseDir, id+".json")
			if err := os.Remove(path); err != nil {
				return deleted, err
			}
			deleted++
		}
	}

	return deleted, nil
}

// EnsureFreshContext 确保会话上下文是新鲜的（模型改变时清除缓存）
func (s *Storage) EnsureFreshContext(sessionID string, newModel string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	session, ok := s.sessions[sessionID]
	if !ok {
		return true // 新会话，默认是新鲜的
	}

	// 检查模型是否改变
	if session.Model != "" && session.Model != newModel {
		// 模型改变了，清除消息历史
		session.Messages = nil
		session.Model = newModel
		return false // 上下文被清除
	}

	// 记录当前模型
	session.Model = newModel
	return true // 上下文保留
}

// ContextKey 上下文键
type ContextKey string

const (
	StorageKey ContextKey = "storage"
)

// FromContext 从上下文获取存储
func FromContext(ctx context.Context) (*Storage, bool) {
	s, ok := ctx.Value(StorageKey).(*Storage)
	return s, ok
}

// WithContext 将存储添加到上下文
func WithContext(ctx context.Context, storage *Storage) context.Context {
	return context.WithValue(ctx, StorageKey, storage)
}
