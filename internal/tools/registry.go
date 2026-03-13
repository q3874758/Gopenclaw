package tools

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
)

// Executor 执行器接口
type Executor interface {
	Execute(ctx context.Context, args map[string]any) (string, error)
	Name() string
	Description() string
}

// Registry 工具注册表
type Registry struct {
	executors map[string]Executor
}

// New 创建新的工具注册表
func New() *Registry {
	return &Registry{
		executors: make(map[string]Executor),
	}
}

// Register 注册一个执行器
func (r *Registry) Register(exec Executor) {
	r.executors[exec.Name()] = exec
}

// Execute 执行指定工具，返回结果
func (r *Registry) Execute(ctx context.Context, name string, args map[string]any) (string, error) {
	exec, ok := r.executors[name]
	if !ok {
		return "", fmt.Errorf("tool %q not found", name)
	}
	return exec.Execute(ctx, args)
}

// GetToolDefinitions 获取所有工具的 JSON Schema 定义
func (r *Registry) GetToolDefinitions() []map[string]any {
	defs := make([]map[string]any, 0, len(r.executors))
	for _, exec := range r.executors {
		// 默认的 tool schema
		def := map[string]any{
			"type": "function",
			"function": map[string]any{
				"name":        exec.Name(),
				"description": exec.Description(),
				"parameters": map[string]any{
					"type": "object",
					"properties": map[string]any{},
				},
			},
		}
		defs = append(defs, def)
	}
	return defs
}

// BaseExecutor 基类，实现默认的 Name/Description
type BaseExecutor struct {
	name        string
	description string
}

func (b *BaseExecutor) Name() string        { return b.name }
func (b *BaseExecutor) Description() string { return b.description }

// JSONArgs 解析 JSON 参数
func JSONArgs(raw string) (map[string]any, error) {
	if raw == "" {
		return make(map[string]any), nil
	}
	var args map[string]any
	if err := json.Unmarshal([]byte(raw), &args); err != nil {
		return nil, errors.New("invalid JSON arguments")
	}
	return args, nil
}
