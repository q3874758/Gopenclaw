package tools

import (
	"context"
	"fmt"
)

// EchoExecutor Echo 工具，用于测试
type EchoExecutor struct {
	BaseExecutor
}

func NewEchoExecutor() *EchoExecutor {
	return &EchoExecutor{
		BaseExecutor: BaseExecutor{
			name:        "echo",
			description: "Echo back the input text. Useful for testing.",
		},
	}
}

func (e *EchoExecutor) Execute(ctx context.Context, args map[string]any) (string, error) {
	text, ok := args["text"]
	if !ok {
		return "", fmt.Errorf("missing required argument: text")
	}
	return fmt.Sprintf("%v", text), nil
}

// GetSchema 返回 echo 工具的完整 schema
func (e *EchoExecutor) GetSchema() map[string]any {
	return map[string]any{
		"type": "function",
		"function": map[string]any{
			"name":        "echo",
			"description": "Echo back the input text. Useful for testing.",
			"parameters": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"text": map[string]any{
						"type":        "string",
						"description": "Text to echo back",
					},
				},
				"required": []string{"text"},
			},
		},
	}
}
