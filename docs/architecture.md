# Gopenclaw 架构说明

## 概览

Gopenclaw 是 OpenClaw 的 Go 重写版，采用单进程、零 Node 的架构，通过 WebSocket 提供与官方兼容的 Gateway 协议。

## 核心组件

```
┌─────────────┐     WS/HTTP      ┌──────────────┐     HTTP      ┌─────────────┐
│  CLI / Web  │ ◄──────────────► │   Gateway    │ ◄────────────► │   LLM API   │
│   Client    │                  │ (internal/   │                │ (OpenAI 等) │
└─────────────┘                  │  gateway)    │                └─────────────┘
                                 └──────┬───────┘
                                        │
                    ┌───────────────────┼───────────────────┐
                    ▼                   ▼                   ▼
             ┌─────────────┐    ┌─────────────┐    ┌─────────────┐
             │   Agent     │    │   Tools     │    │  Storage /  │
             │ (LLM+工具)  │    │ (Registry)  │    │ Cron/Webhook│
             └─────────────┘    └─────────────┘    └─────────────┘
```

- **Gateway**：HTTP 服务（`/`、`/health`）+ WebSocket（`/ws`），按 `method` 分发到各 RPC 处理函数。
- **Agent**：调用 OpenAI 兼容的 Chat Completions API，支持流式与非流式，支持 function calling（工具调用）。
- **Tools**：内置工具（echo、bash、read_file、write_file、ls、mkdir、web_fetch、sessions_*、tool、grep、find、curl、glob 等），由 Gateway 在 agent.invoke 时执行并回填结果。
- **Storage**：会话持久化（`~/.gopenclaw/sessions/`）。
- **Cron**：定时任务（`~/.gopenclaw/cron.json`）。
- **Webhook**：`/hooks/wake`、`/hooks/agent` 等，与配置中的 `hooks` 对齐。
- **Channels**：Telegram、Discord、Slack、WhatsApp 等适配器，通过 `internal/channels` 与各平台对接。
- **Plugin**：插件与钩子（gateway.start、agent.invoke、tool.execute 等），便于扩展。

## 配置与数据目录

- 配置：`~/.gopenclaw/openclaw.json`（或 `GOPENCLAW_HOME` / `OPENCLAW_HOME` 指定目录）。
- 会话：`~/.gopenclaw/sessions/`。
- Cron：`~/.gopenclaw/cron.json`。
- 默认端口：**11999**（与官方 18789 区分）。

## 协议

- 客户端连上 `ws://<host>:11999/ws` 后，发送 JSON 消息：`{ "method": "xxx", "params": { ... } }`。
- 服务端返回 `{ "result": { ... } }` 或 `{ "error": { "code", "message" } }`。
- 详见 [Gateway 方法清单](gateway-methods.md) 与 [OpenAPI](gateway-openapi.yaml)。
