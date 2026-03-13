package tools

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"io/ioutil"
	"path/filepath"
	"net/http"
	"strings"
)

// BashExecutor 执行 Shell 命令
type BashExecutor struct {
	BaseExecutor
}

func NewBashExecutor() *BashExecutor {
	return &BashExecutor{
		BaseExecutor: BaseExecutor{
			name:        "bash",
			description: "Execute a bash/shell command and return the output",
		},
	}
}

func (e *BashExecutor) Execute(ctx context.Context, args map[string]any) (string, error) {
	command, ok := args["command"]
	if !ok {
		return "", fmt.Errorf("missing required argument: command")
	}
	
	cmd := exec.CommandContext(ctx, "sh", "-c", fmt.Sprintf("%v", command))
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("%s: %s", err.Error(), string(output))
	}
	return string(output), nil
}

func (e *BashExecutor) GetSchema() map[string]any {
	return map[string]any{
		"type": "function",
		"function": map[string]any{
			"name":        "bash",
			"description": "Execute a bash/shell command and return the output",
			"parameters": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"command": map[string]any{
						"type":        "string",
						"description": "The command to execute",
					},
				},
				"required": []string{"command"},
			},
		},
	}
}

// ReadFileExecutor 读取文件
type ReadFileExecutor struct {
	BaseExecutor
}

func NewReadFileExecutor() *ReadFileExecutor {
	return &ReadFileExecutor{
		BaseExecutor: BaseExecutor{
			name:        "read_file",
			description: "Read the contents of a file",
		},
	}
}

func (e *ReadFileExecutor) Execute(ctx context.Context, args map[string]any) (string, error) {
	path, ok := args["path"]
	if !ok {
		return "", fmt.Errorf("missing required argument: path")
	}
	
	pathStr := fmt.Sprintf("%v", path)
	content, err := ioutil.ReadFile(pathStr)
	if err != nil {
		return "", fmt.Errorf("read file error: %w", err)
	}
	return string(content), nil
}

func (e *ReadFileExecutor) GetSchema() map[string]any {
	return map[string]any{
		"type": "function",
		"function": map[string]any{
			"name":        "read_file",
			"description": "Read the contents of a file",
			"parameters": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"path": map[string]any{
						"type":        "string",
						"description": "The path to the file to read",
					},
				},
				"required": []string{"path"},
			},
		},
	}
}

// WriteFileExecutor 写入文件
type WriteFileExecutor struct {
	BaseExecutor
}

func NewWriteFileExecutor() *WriteFileExecutor {
	return &WriteFileExecutor{
		BaseExecutor: BaseExecutor{
			name:        "write_file",
			description: "Write content to a file, creating or overwriting it",
		},
	}
}

func (e *WriteFileExecutor) Execute(ctx context.Context, args map[string]any) (string, error) {
	path, ok := args["path"]
	content, contentOk := args["content"]
	if !ok || !contentOk {
		return "", fmt.Errorf("missing required arguments: path, content")
	}
	
	pathStr := fmt.Sprintf("%v", path)
	contentStr := fmt.Sprintf("%v", content)
	
	// 确保目录存在
	dir := filepath.Dir(pathStr)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("create directory error: %w", err)
	}
	
	if err := ioutil.WriteFile(pathStr, []byte(contentStr), 0644); err != nil {
		return "", fmt.Errorf("write file error: %w", err)
	}
	
	return fmt.Sprintf("File written to %s", pathStr), nil
}

func (e *WriteFileExecutor) GetSchema() map[string]any {
	return map[string]any{
		"type": "function",
		"function": map[string]any{
			"name":        "write_file",
			"description": "Write content to a file, creating or overwriting it",
			"parameters": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"path": map[string]any{
						"type":        "string",
						"description": "The path to the file to write",
					},
					"content": map[string]any{
						"type":        "string",
						"description": "The content to write to the file",
					},
				},
				"required": []string{"path", "content"},
			},
		},
	}
}

// WebFetchExecutor 获取网页
type WebFetchExecutor struct {
	BaseExecutor
}

func NewWebFetchExecutor() *WebFetchExecutor {
	return &WebFetchExecutor{
		BaseExecutor: BaseExecutor{
			name:        "web_fetch",
			description: "Fetch the content of a URL",
		},
	}
}

func (e *WebFetchExecutor) Execute(ctx context.Context, args map[string]any) (string, error) {
	url, ok := args["url"]
	if !ok {
		return "", fmt.Errorf("missing required argument: url")
	}
	
	urlStr := fmt.Sprintf("%v", url)
	
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, urlStr, nil)
	if err != nil {
		return "", fmt.Errorf("create request error: %w", err)
	}
	
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("fetch error: %w", err)
	}
	defer resp.Body.Close()
	
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read response error: %w", err)
	}
	
	// 截断过长的内容
	content := string(body)
	if len(content) > 10000 {
		content = content[:10000] + "\n... (truncated)"
	}
	
	return content, nil
}

func (e *WebFetchExecutor) GetSchema() map[string]any {
	return map[string]any{
		"type": "function",
		"function": map[string]any{
			"name":        "web_fetch",
			"description": "Fetch the content of a URL",
			"parameters": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"url": map[string]any{
						"type":        "string",
						"description": "The URL to fetch",
					},
				},
				"required": []string{"url"},
			},
		},
	}
}

// SessionsListExecutor 列出会话
type SessionsListExecutor struct {
	BaseExecutor
}

func NewSessionsListExecutor() *SessionsListExecutor {
	return &SessionsListExecutor{
		BaseExecutor: BaseExecutor{
			name:        "sessions_list",
			description: "List all active sessions",
		},
	}
}

func (e *SessionsListExecutor) Execute(ctx context.Context, args map[string]any) (string, error) {
	// TODO: 从 Gateway 获取会话列表
	return "[]", nil
}

func (e *SessionsListExecutor) GetSchema() map[string]any {
	return map[string]any{
		"type": "function",
		"function": map[string]any{
			"name":        "sessions_list",
			"description": "List all active sessions",
			"parameters": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"limit": map[string]any{
						"type":        "number",
						"description": "Maximum number of sessions to return",
					},
				},
			},
		},
	}
}

// SessionsHistoryExecutor 获取会话历史
type SessionsHistoryExecutor struct {
	BaseExecutor
}

func NewSessionsHistoryExecutor() *SessionsHistoryExecutor {
	return &SessionsHistoryExecutor{
		BaseExecutor: BaseExecutor{
			name:        "sessions_history",
			description: "Get message history for a session",
		},
	}
}

func (e *SessionsHistoryExecutor) Execute(ctx context.Context, args map[string]any) (string, error) {
	sessionId, ok := args["sessionId"]
	if !ok {
		return "", fmt.Errorf("missing required argument: sessionId")
	}
	
	// TODO: 从 Gateway 获取会话历史
	return fmt.Sprintf("History for session %v", sessionId), nil
}

func (e *SessionsHistoryExecutor) GetSchema() map[string]any {
	return map[string]any{
		"type": "function",
		"function": map[string]any{
			"name":        "sessions_history",
			"description": "Get message history for a session",
			"parameters": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"sessionId": map[string]any{
						"type":        "string",
						"description": "The session ID",
					},
					"limit": map[string]any{
						"type":        "number",
						"description": "Maximum number of messages to return",
					},
				},
				"required": []string{"sessionId"},
			},
		},
	}
}

// SessionsSendExecutor 发送消息
type SessionsSendExecutor struct {
	BaseExecutor
}

func NewSessionsSendExecutor() *SessionsSendExecutor {
	return &SessionsSendExecutor{
		BaseExecutor: BaseExecutor{
			name:        "sessions_send",
			description: "Send a message to a session",
		},
	}
}

func (e *SessionsSendExecutor) Execute(ctx context.Context, args map[string]any) (string, error) {
	sessionId, ok := args["sessionId"]
	message, msgOk := args["message"]
	if !ok || !msgOk {
		return "", fmt.Errorf("missing required arguments: sessionId, message")
	}
	
	// TODO: 发送到 Gateway
	return fmt.Sprintf("Message sent to session %v: %v", sessionId, message), nil
}

func (e *SessionsSendExecutor) GetSchema() map[string]any {
	return map[string]any{
		"type": "function",
		"function": map[string]any{
			"name":        "sessions_send",
			"description": "Send a message to a session",
			"parameters": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"sessionId": map[string]any{
						"type":        "string",
						"description": "The session ID",
					},
					"message": map[string]any{
						"type":        "string",
						"description": "The message to send",
					},
				},
				"required": []string{"sessionId", "message"},
			},
		},
	}
}

// ListDirectoryExecutor 列出目录
type ListDirectoryExecutor struct {
	BaseExecutor
}

func NewListDirectoryExecutor() *ListDirectoryExecutor {
	return &ListDirectoryExecutor{
		BaseExecutor: BaseExecutor{
			name:        "ls",
			description: "List files in a directory",
		},
	}
}

func (e *ListDirectoryExecutor) Execute(ctx context.Context, args map[string]any) (string, error) {
	path := "."
	if p, ok := args["path"]; ok {
		path = fmt.Sprintf("%v", p)
	}
	
	files, err := ioutil.ReadDir(path)
	if err != nil {
		return "", fmt.Errorf("read dir error: %w", err)
	}
	
	var result strings.Builder
	for _, f := range files {
		result.WriteString(f.Name())
		if f.IsDir() {
			result.WriteString("/")
		}
		result.WriteString("\n")
	}
	
	return result.String(), nil
}

func (e *ListDirectoryExecutor) GetSchema() map[string]any {
	return map[string]any{
		"type": "function",
		"function": map[string]any{
			"name":        "ls",
			"description": "List files in a directory",
			"parameters": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"path": map[string]any{
						"type":        "string",
						"description": "The directory path (default: current directory)",
					},
				},
			},
		},
	}
}

// MakeDirectoryExecutor 创建目录
type MakeDirectoryExecutor struct {
	BaseExecutor
}

func NewMakeDirectoryExecutor() *MakeDirectoryExecutor {
	return &MakeDirectoryExecutor{
		BaseExecutor: BaseExecutor{
			name:        "mkdir",
			description: "Create a directory",
		},
	}
}

func (e *MakeDirectoryExecutor) Execute(ctx context.Context, args map[string]any) (string, error) {
	path, ok := args["path"]
	if !ok {
		return "", fmt.Errorf("missing required argument: path")
	}
	
	pathStr := fmt.Sprintf("%v", path)
	if err := os.MkdirAll(pathStr, 0755); err != nil {
		return "", fmt.Errorf("mkdir error: %w", err)
	}
	
	return fmt.Sprintf("Directory created: %s", pathStr), nil
}

func (e *MakeDirectoryExecutor) GetSchema() map[string]any {
	return map[string]any{
		"type": "function",
		"function": map[string]any{
			"name":        "mkdir",
			"description": "Create a directory",
			"parameters": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"path": map[string]any{
						"type":        "string",
						"description": "The directory path to create",
					},
				},
				"required": []string{"path"},
			},
		},
	}
}
