package tools

import (
	"context"
	"errors"
	"os/exec"
	"path/filepath"
	"strings"
)

// ErrMissingArg 缺少参数错误
var ErrMissingArg = errors.New("missing required argument")

// ToolExecutor 系统工具执行器
type ToolExecutor struct {
	BaseExecutor
}

// NewToolExecutor 创建系统工具执行器
func NewToolExecutor() *ToolExecutor {
	return &ToolExecutor{
		BaseExecutor: BaseExecutor{
			name:        "tool",
			description: "Execute system commands and queries",
		},
	}
}

// Execute 执行系统工具
func (e *ToolExecutor) Execute(ctx context.Context, args map[string]interface{}) (string, error) {
	command, _ := args["command"].(string)
	if command == "" {
		return "", ErrMissingArg
	}

	// 解析命令和参数
	parts := strings.Fields(command)
	if len(parts) == 0 {
		return "", ErrMissingArg
	}

	var cmd *exec.Cmd
	switch parts[0] {
	case "bash", "sh", "shell":
		if len(parts) < 2 {
			return "", ErrMissingArg
		}
		cmd = exec.CommandContext(ctx, "sh", "-c", strings.Join(parts[1:], " "))
	case "cmd", "powershell":
		if len(parts) < 2 {
			return "", ErrMissingArg
		}
		shell := "cmd.exe"
		if parts[0] == "powershell" {
			shell = "powershell"
		}
		cmd = exec.CommandContext(ctx, shell, "/c", strings.Join(parts[1:], " "))
	case "which":
		if len(parts) < 2 {
			return "", ErrMissingArg
		}
		cmd = exec.CommandContext(ctx, "which", parts[1])
	case "where":
		if len(parts) < 2 {
			return "", ErrMissingArg
		}
		cmd = exec.CommandContext(ctx, "where", parts[1])
	case "pwd":
		cmd = exec.CommandContext(ctx, "pwd")
	case "whoami":
		cmd = exec.CommandContext(ctx, "whoami")
	case "hostname":
		cmd = exec.CommandContext(ctx, "hostname")
	case "uname":
		args := parts[1:]
		if len(args) == 0 {
			args = []string{"-a"}
		}
		cmd = exec.CommandContext(ctx, "uname", args...)
	case "date":
		cmd = exec.CommandContext(ctx, "date")
	case "env":
		cmd = exec.CommandContext(ctx, "env")
	default:
		// 直接执行命令
		cmd = exec.CommandContext(ctx, parts[0], parts[1:]...)
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return string(output), err
	}

	return string(output), nil
}

// GrepExecutor grep 工具
type GrepExecutor struct {
	BaseExecutor
}

// NewGrepExecutor 创建 grep 执行器
func NewGrepExecutor() *GrepExecutor {
	return &GrepExecutor{
		BaseExecutor: BaseExecutor{
			name:        "grep",
			description: "Search for patterns in files",
		},
	}
}

// Execute 执行 grep
func (e *GrepExecutor) Execute(ctx context.Context, args map[string]interface{}) (string, error) {
	pattern, _ := args["pattern"].(string)
	if pattern == "" {
		return "", ErrMissingArg
	}

	path, _ := args["path"].(string)
	if path == "" {
		path = "."
	}

	cmd := exec.CommandContext(ctx, "grep", "-r", "-n", "--color=never", pattern, path)
	output, err := cmd.CombinedOutput()
	return string(output), err
}

// FindExecutor find 工具
type FindExecutor struct {
	BaseExecutor
}

// NewFindExecutor 创建 find 执行器
func NewFindExecutor() *FindExecutor {
	return &FindExecutor{
		BaseExecutor: BaseExecutor{
			name:        "find",
			description: "Find files in a directory hierarchy",
		},
	}
}

// Execute 执行 find
func (e *FindExecutor) Execute(ctx context.Context, args map[string]interface{}) (string, error) {
	path, _ := args["path"].(string)
	if path == "" {
		path = "."
	}

	name, _ := args["name"].(string)

	cmdArgs := []string{path}
	if name != "" {
		cmdArgs = append(cmdArgs, "-name", name)
	} else {
		cmdArgs = append(cmdArgs, "-type", "f")
	}

	cmd := exec.CommandContext(ctx, "find", cmdArgs...)
	output, err := cmd.CombinedOutput()
	return string(output), err
}

// CurlExecutor curl 工具
type CurlExecutor struct {
	BaseExecutor
}

// NewCurlExecutor 创建 curl 执行器
func NewCurlExecutor() *CurlExecutor {
	return &CurlExecutor{
		BaseExecutor: BaseExecutor{
			name:        "curl",
			description: "Perform HTTP requests",
		},
	}
}

// Execute 执行 curl
func (e *CurlExecutor) Execute(ctx context.Context, args map[string]interface{}) (string, error) {
	url, _ := args["url"].(string)
	if url == "" {
		return "", ErrMissingArg
	}

	method, _ := args["method"].(string)
	if method == "" {
		method = "GET"
	}

	data, _ := args["data"].(string)

	cmdArgs := []string{"-s", "-X", method, url}
	if data != "" {
		cmdArgs = append(cmdArgs, "-d", data)
	}

	cmd := exec.CommandContext(ctx, "curl", cmdArgs...)
	output, err := cmd.CombinedOutput()
	return string(output), err
}

// GlobExecutor 文件搜索工具
type GlobExecutor struct {
	BaseExecutor
}

// NewGlobExecutor 创建 glob 执行器
func NewGlobExecutor() *GlobExecutor {
	return &GlobExecutor{
		BaseExecutor: BaseExecutor{
			name:        "glob",
			description: "Find files matching a pattern",
		},
	}
}

// Execute 执行 glob
func (e *GlobExecutor) Execute(ctx context.Context, args map[string]interface{}) (string, error) {
	pattern, _ := args["pattern"].(string)
	if pattern == "" {
		return "", ErrMissingArg
	}

	// 使用 filepath.Glob
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return "", err
	}

	if len(matches) == 0 {
		return "No matches found", nil
	}

	return strings.Join(matches, "\n"), nil
}

// RegisterSystemTools 注册系统工具
func RegisterSystemTools(reg *Registry) {
	reg.Register(NewToolExecutor())
	reg.Register(NewGrepExecutor())
	reg.Register(NewFindExecutor())
	reg.Register(NewCurlExecutor())
	reg.Register(NewGlobExecutor())
	reg.Register(NewBrowserExecutor(NewDefaultConfig()))
}
