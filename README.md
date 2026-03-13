# Gopenclaw

OpenClaw 的 Go 重写版：零 Node，性能消耗最低，功能与 [openclaw/openclaw](https://github.com/openclaw/openclaw) 对齐。

## 要求

- Go 1.21+

## 构建与运行

```bash
go build -o gopenclaw ./cmd/openclaw
./gopenclaw gateway
```

## 项目结构

- `cmd/openclaw` — CLI 入口（Cobra）
- `internal/config` — 配置（~/.gopenclaw/openclaw.json）
- `internal/protocol` — Gateway WS 协议类型
- `internal/gateway` — Gateway（WS + HTTP）、RPC 分发、兼容性测试
- `internal/agent` — Agent（LLM 调用 + 工具循环）
- `internal/tools` — 工具注册与执行器（echo、bash、read_file、write_file、ls、mkdir、web_fetch、sessions_*、tool、grep、find、curl、glob）
- `internal/cron`、`internal/webhook`、`internal/storage` — 定时任务、Webhook、会话持久化
- `internal/channels`、`internal/telegram`、`internal/discord`、`internal/slack`、`internal/whatsapp` — 通道适配器
- `internal/routing`、`internal/tts`、`internal/skills`、`internal/plugin` — 路由策略、TTS、技能、插件
- `ui/` — WebChat 静态页；`docs/` — 文档；`contrib/` — systemd 等

## 配置

Gopenclaw 与官方 OpenClaw **使用不同配置目录**，同机运行互不干扰：

- **默认**：`~/.gopenclaw/openclaw.json`，默认端口 **11999**
- 若设置 **GOPENCLAW_HOME**：使用该目录下的 `openclaw.json`
- 若设置 **OPENCLAW_HOME**：使用该目录（兼容从官方迁移时的路径）

官方 OpenClaw 默认端口 **18789**，常用 `~/.openclaw`；Gopenclaw 默认端口 **11999**、配置目录 `~/.gopenclaw`，同机运行互不占用。

Agent 调用 LLM 需设置 **OPENAI_API_KEY**；模型在配置中 `agent.model`（如 `openai/gpt-4o`），未配置时默认 `gpt-4o-mini`。

## 使用 Agent（单轮对话）

1. 启动 Gateway：`./gopenclaw gateway`（默认端口 11999）
2. 另开终端，设置 API Key 后发送消息：`$env:OPENAI_API_KEY="sk-..."; ./gopenclaw agent --message "你好"`

## 文档

- [Gateway 方法清单](docs/gateway-methods.md) — 全量 RPC 方法对照表
- [Gateway OpenAPI](docs/gateway-openapi.yaml) — WebSocket API 的 OpenAPI 3.0 描述
- [兼容性测试说明](docs/compatibility-tests.md) — 与官方协议的兼容性测试用例
- [架构说明](docs/architecture.md) — 组件与数据目录
- [开发与测试](docs/development.md) — 构建、测试、添加工具与 RPC 方法
