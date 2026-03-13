package agent

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"

	"gopenclaw/internal/config"
)

// Client 调用 OpenAI 兼容的 Chat Completions API，单轮对话
type Client struct {
	cfg          *config.Config
	apiKey       string
	baseURL      string // 默认 https://api.openai.com/v1
	client       *http.Client
	tools        []ToolDefinition
	sessionIntroSent bool // 当前会话是否已发送系统提示
}

// New 根据配置创建 Agent 客户端；API Key 从环境变量 OPENAI_API_KEY 读取
func New(cfg *config.Config) *Client {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		apiKey = os.Getenv("ANTHROPIC_API_KEY") // 可选：后续可扩展为 Anthropic 等
	}
	baseURL := os.Getenv("OPENAI_BASE_URL")
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}
	return &Client{
		cfg:     cfg,
		apiKey:  apiKey,
		baseURL: baseURL,
		client:  &http.Client{},
		tools:   make([]ToolDefinition, 0),
	}
}

// RegisterTool 注册一个工具
func (c *Client) RegisterTool(def ToolDefinition) {
	c.tools = append(c.tools, def)
}

// GetTools 返回已注册的工具列表
func (c *Client) GetTools() []ToolDefinition {
	return c.tools
}

// SetSessionIntroSent 设置系统提示已发送标志
func (c *Client) SetSessionIntroSent(sent bool) {
	c.sessionIntroSent = sent
}

// IsSessionIntroSent 检查系统提示是否已发送
func (c *Client) IsSessionIntroSent() bool {
	return c.sessionIntroSent
}

// openAI 请求/响应结构（最小集）
type chatRequest struct {
	Model           string          `json:"model"`
	Messages        []message       `json:"messages"`
	Stream          bool            `json:"stream,omitempty"`
	Tools           []ToolDefinition `json:"tools,omitempty"`
	ReasoningEffort string          `json:"reasoning_effort,omitempty"` // ultrathink 模式
}

type message struct {
	Role      string     `json:"role"`
	Content   string     `json:"content"`
	ToolCalls []toolCall `json:"tool_calls,omitempty"`
	ToolCallID string   `json:"tool_call_id,omitempty"`
}

type toolCall struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Function struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	} `json:"function"`
}

type chatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

// Invoke 发送单轮用户消息，返回助手回复文本；未配置 API Key 时返回错误
func (c *Client) Invoke(ctx context.Context, userMessage string) (string, error) {
	if c.apiKey == "" {
		return "", fmt.Errorf("OPENAI_API_KEY not set")
	}
	model := c.cfg.Agent.Model
	if model == "" {
		model = "gpt-4o-mini" // 默认
	}
	// 兼容 "openai/gpt-4o" 格式，只取后半段
	if len(model) > 7 && model[:7] == "openai/" {
		model = model[7:]
	}

	msgs := []message{}

	// sessionIntro：系统提示只发一次（可选）
	sessionIntro := c.cfg.Messages.SessionIntro
	if sessionIntro != "" && !c.sessionIntroSent {
		msgs = append(msgs, message{Role: "system", Content: sessionIntro})
		c.sessionIntroSent = true
	}

	// ultrathink 前缀支持：检测用户消息是否以 "ultrathink" 开头，如果是则添加 reasoning_effort
	userContent := userMessage
	var reasoningEffort string
	if strings.HasPrefix(userContent, "ultrathink ") {
		userContent = strings.TrimPrefix(userContent, "ultrathink ")
		reasoningEffort = "high"
	} else if strings.HasPrefix(userContent, "ultrathink\n") {
		userContent = strings.TrimPrefix(userContent, "ultrathink\n")
		reasoningEffort = "high"
	}

	// per-message 提示词前缀：给用户消息添加前缀
	messagePrefix := c.cfg.Messages.MessagePrefix
	if messagePrefix != "" {
		userContent = messagePrefix + userContent
	}

	msgs = append(msgs, message{Role: "user", Content: userContent})

	body := chatRequest{
		Model:    model,
		Messages: msgs,
		Stream:   false,
	}

	// 如果需要 ultrathink 模式，添加到 body
	if reasoningEffort != "" {
		// reasoning_effort 需要在请求中单独设置
		body.ReasoningEffort = reasoningEffort
	}
	data, err := json.Marshal(body)
	if err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/chat/completions", bytes.NewReader(data))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	resp, err := c.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API status %d", resp.StatusCode)
	}
	var out chatResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", err
	}
	if len(out.Choices) == 0 {
		return "", fmt.Errorf("no choices in response")
	}
	return out.Choices[0].Message.Content, nil
}

// streamToolCall 流式响应中的工具调用
type streamToolCall struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Function struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	} `json:"function"`
}

// streamResponse 用于解析流式 SSE 的增量
type streamResponse struct {
	Choices []struct {
		Delta struct {
			Content   string          `json:"content"`
			ToolCalls []streamToolCall `json:"tool_calls"`
		} `json:"delta"`
	} `json:"choices"`
}

// ToolDefinition 定义一个可被 Agent 调用的工具
type ToolDefinition struct {
	Type     string `json:"type"` // "function"
	Function struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		Parameters  any    `json:"parameters"` // JSON Schema
	} `json:"function"`
}

// ToolExecutor 工具执行器接口
type ToolExecutor interface {
	Execute(ctx context.Context, name string, args map[string]any) (string, error)
}

// ToolCall 表示一次工具调用
type ToolCall struct {
	ID        string
	Name      string
	Arguments map[string]any
}

// jsonArgs 解析 JSON 参数字符串
func jsonArgs(raw string) (map[string]any, error) {
	if raw == "" {
		return make(map[string]any), nil
	}
	var args map[string]any
	if err := json.Unmarshal([]byte(raw), &args); err != nil {
		return nil, err
	}
	return args, nil
}

// Result 表示工具执行结果
type Result struct {
	ToolCallID string
	Output     string
	Error      string
}

// InvokeStream 流式调用 LLM，每收到一段内容就调用 onChunk；返回完整文本与错误
func (c *Client) InvokeStream(ctx context.Context, userMessage string, onChunk func(chunk string) error) (fullText string, err error) {
	if c.apiKey == "" {
		return "", fmt.Errorf("OPENAI_API_KEY not set")
	}
	model := c.cfg.Agent.Model
	if model == "" {
		model = "gpt-4o-mini"
	}
	if len(model) > 7 && model[:7] == "openai/" {
		model = model[7:]
	}
	body := chatRequest{
		Model: model,
		Messages: []message{
			{Role: "user", Content: userMessage},
		},
		Stream:   true,
		Tools:    c.tools,
	}
	data, err := json.Marshal(body)
	if err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/chat/completions", bytes.NewReader(data))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	resp, err := c.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API status %d", resp.StatusCode)
	}
	scanner := bufio.NewScanner(resp.Body)
	var full strings.Builder
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		payload := strings.TrimPrefix(line, "data: ")
		if payload == "[DONE]" {
			break
		}
		var chunk streamResponse
		if json.Unmarshal([]byte(payload), &chunk) != nil {
			continue
		}
		if len(chunk.Choices) == 0 || chunk.Choices[0].Delta.Content == "" {
			continue
		}
		text := chunk.Choices[0].Delta.Content
		full.WriteString(text)
		if onChunk != nil && onChunk(text) != nil {
			break
		}
	}
	return full.String(), scanner.Err()
}

// InvokeWithTools 发送消息列表给 LLM，支持工具调用；返回 tool_calls 列表、完整文本和错误
func (c *Client) InvokeWithTools(ctx context.Context, msgs []Message, stream bool, onChunk func(chunk string) error) ([]ToolCall, string, error) {
	if c.apiKey == "" {
		return nil, "", fmt.Errorf("OPENAI_API_KEY not set")
	}
	model := c.cfg.Agent.Model
	if model == "" {
		model = "gpt-4o-mini"
	}
	if len(model) > 7 && model[:7] == "openai/" {
		model = model[7:]
	}

	// 转换 internal/agent.message 到 chatRequest 的 message
	chatMsgs := make([]message, len(msgs))
	for i, m := range msgs {
		chatMsgs[i] = message{
			Role:      m.Role,
			Content:   m.Content,
			ToolCallID: m.ToolCallID,
		}
	}

	// sessionIntro：系统提示只发一次（可选）
	sessionIntro := c.cfg.Messages.SessionIntro
	if sessionIntro != "" && !c.sessionIntroSent {
		// 插入到消息列表最前面
		chatMsgs = append([]message{{Role: "system", Content: sessionIntro}}, chatMsgs...)
		c.sessionIntroSent = true
	}

	// ultrathink 前缀支持：检测第一条用户消息是否以 "ultrathink" 开头
	var reasoningEffort string
	if len(chatMsgs) > 0 && chatMsgs[0].Role == "user" {
		userContent := chatMsgs[0].Content
		if strings.HasPrefix(userContent, "ultrathink ") {
			chatMsgs[0].Content = strings.TrimPrefix(userContent, "ultrathink ")
			reasoningEffort = "high"
		} else if strings.HasPrefix(userContent, "ultrathink\n") {
			chatMsgs[0].Content = strings.TrimPrefix(userContent, "ultrathink\n")
			reasoningEffort = "high"
		}
	}

	// per-message 提示词前缀：给用户消息添加前缀
	messagePrefix := c.cfg.Messages.MessagePrefix
	if messagePrefix != "" {
		for i := range chatMsgs {
			if chatMsgs[i].Role == "user" {
				chatMsgs[i].Content = messagePrefix + chatMsgs[i].Content
			}
		}
	}

	body := chatRequest{
		Model:    model,
		Messages: chatMsgs,
		Stream:   stream,
		Tools:    c.tools,
	}

	// 如果需要 ultrathink 模式，添加到 body
	if reasoningEffort != "" {
		body.ReasoningEffort = reasoningEffort
	}
	data, err := json.Marshal(body)
	if err != nil {
		return nil, "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/chat/completions", bytes.NewReader(data))
	if err != nil {
		return nil, "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("API status %d", resp.StatusCode)
	}

	if stream {
		return c.handleStreamToolCalls(resp.Body, onChunk)
	}

	// 非流式处理
	var out chatResponseWithToolCalls
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, "", err
	}
	if len(out.Choices) == 0 {
		return nil, "", fmt.Errorf("no choices in response")
	}

	msg := out.Choices[0].Message
	var toolCalls []ToolCall
	for _, tc := range msg.ToolCalls {
		args, _ := jsonArgs(tc.Function.Arguments)
		toolCalls = append(toolCalls, ToolCall{
			ID:        tc.ID,
			Name:      tc.Function.Name,
			Arguments: args,
		})
	}
	return toolCalls, msg.Content, nil
}

// Message 对外使用的消息结构
type Message struct {
	Role      string `json:"role"`
	Content   string `json:"content"`
	ToolCallID string `json:"tool_call_id,omitempty"`
}

// chatResponseWithToolCalls 带工具调用的响应
type chatResponseWithToolCalls struct {
	Choices []struct {
		Message struct {
			Content   string     `json:"content"`
			ToolCalls []toolCall `json:"tool_calls"`
		} `json:"message"`
	} `json:"choices"`
}

// handleStreamToolCalls 处理流式响应，提取 tool_calls
func (c *Client) handleStreamToolCalls(body interface{ Read(p []byte) (n int, err error) }, onChunk func(chunk string) error) ([]ToolCall, string, error) {
	scanner := bufio.NewScanner(body)
	var full strings.Builder
	var toolCalls []streamToolCall

	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		payload := strings.TrimPrefix(line, "data: ")
		if payload == "[DONE]" {
			break
		}
		var chunk streamResponse
		if json.Unmarshal([]byte(payload), &chunk) != nil {
			continue
		}
		if len(chunk.Choices) == 0 {
			continue
		}
		delta := chunk.Choices[0].Delta

		// 收集 tool_calls
		if len(delta.ToolCalls) > 0 {
			toolCalls = append(toolCalls, delta.ToolCalls...)
		}

		if delta.Content != "" {
			text := delta.Content
			full.WriteString(text)
			if onChunk != nil && onChunk(text) != nil {
				break
			}
		}
	}

	// 转换 tool_calls
	var result []ToolCall
	for _, tc := range toolCalls {
		args, _ := jsonArgs(tc.Function.Arguments)
		result = append(result, ToolCall{
			ID:        tc.ID,
			Name:      tc.Function.Name,
			Arguments: args,
		})
	}
	return result, full.String(), scanner.Err()
}
