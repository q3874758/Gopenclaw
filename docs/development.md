# 开发与测试

## 环境要求

- Go 1.21+
- （可选）OPENAI_API_KEY，用于 Agent 真实调用与 E2E 测试

## 构建

```bash
go build -o gopenclaw ./cmd/openclaw
```

## 测试

### 单元与兼容性测试

```bash
# 运行所有测试
go test ./...

# 仅 Gateway 兼容性测试（无需 API Key）
go test ./internal/gateway/... -v

# 简短输出
go test ./internal/gateway/... -count=1
```

兼容性测试会启动内存中的 Gateway（随机端口），通过 WebSocket 调用 `config.get`、`sessions.list`、`nodes.list`，并校验无效 JSON 与 `agent.invoke` 参数错误等，详见 [兼容性测试说明](compatibility-tests.md)。

### 本地运行 Gateway

```bash
./gopenclaw gateway
# 默认 http://127.0.0.1:11999
```

设置 `OPENAI_API_KEY` 后可用 CLI 或浏览器 WebChat 与 Agent 对话：

```bash
./gopenclaw agent --message "你好"
```

## 项目结构（简要）

| 路径 | 说明 |
|------|------|
| `cmd/openclaw` | CLI 入口（Cobra） |
| `internal/config` | 配置加载与结构 |
| `internal/protocol` | WS 协议类型定义 |
| `internal/gateway` | Gateway HTTP/WS 与 RPC 分发 |
| `internal/agent` | LLM 调用与工具循环 |
| `internal/tools` | 工具注册与执行器 |
| `internal/cron` | 定时任务 |
| `internal/webhook` | Webhook 处理 |
| `internal/storage` | 会话持久化 |
| `internal/channels` | 通道抽象与路由 |
| `internal/telegram`、`discord`、`slack`、`whatsapp` | 各通道适配器 |
| `internal/routing` | 路由策略（allowlist/blocklist） |
| `internal/tts` | TTS 提供商 |
| `internal/skills` | 技能管理 |
| `internal/plugin` | 插件与钩子 |
| `ui/` | WebChat 静态页 |
| `docs/` | 文档与 OpenAPI |
| `contrib/` | systemd 等运维配置 |

## 添加新工具

1. 在 `internal/tools` 下实现 `Executor` 接口（`Execute`、`Name`、`Description`）。
2. 在 `internal/gateway/gateway.go` 的 `New()` 中 `toolReg.Register(NewXxxExecutor())`。
3. 若需向 Agent 暴露 schema，在对应 Executor 上实现 `GetSchema()` 或通过 Registry 的 `GetToolDefinitions()` 统一返回。

## 添加新 RPC 方法

1. 在 `internal/protocol/types.go` 中定义请求/响应类型（如需要）。
2. 在 `internal/gateway/methods.go` 的 `methodHandlers` 中注册方法名与处理函数。
3. 实现 handler：从 `msg.Params` 解码参数，返回 `&protocol.Message{ ID: msg.ID, Result: ... }` 或 `errorResult(...)`。

## 文档索引

- [Gateway 方法清单](gateway-methods.md)
- [Gateway OpenAPI](gateway-openapi.yaml)
- [兼容性测试说明](compatibility-tests.md)
- [架构说明](architecture.md)
