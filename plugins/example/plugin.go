package example

import (
	"context"
	"encoding/json"
	"fmt"
)

// ExamplePlugin 示例插件
type ExamplePlugin struct {
	name        string
	description string
	version     string
	messageCount int
	config      json.RawMessage
}

// NewExamplePlugin 创建示例插件
func NewExamplePlugin() *ExamplePlugin {
	return &ExamplePlugin{
		name:        "example",
		description: "An example plugin demonstrating the plugin system",
		version:     "1.0.0",
		messageCount: 0,
	}
}

// Name 返回插件名称
func (p *ExamplePlugin) Name() string { return p.name }

// Description 返回插件描述
func (p *ExamplePlugin) Description() string { return p.description }

// Version 返回插件版本
func (p *ExamplePlugin) Version() string { return p.version }

// Init 初始化
func (p *ExamplePlugin) Init(ctx context.Context, config json.RawMessage) error {
	p.config = config
	fmt.Println("ExamplePlugin initialized")
	return nil
}

// Start 启动
func (p *ExamplePlugin) Start(ctx context.Context) error {
	fmt.Println("ExamplePlugin started")
	return nil
}

// Stop 停止
func (p *ExamplePlugin) Stop() error {
	fmt.Printf("ExamplePlugin stopped, processed %d messages\n", p.messageCount)
	return nil
}
