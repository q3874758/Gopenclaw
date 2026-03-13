# Gopenclaw 兼容性测试说明

本文档描述与官方 OpenClaw Gateway 协议的兼容性测试思路与用例范围。

## 测试范围

### 1. WebSocket 连接

- 连接 `ws://127.0.0.1:11999/ws` 成功
- 发送无效 JSON 时返回合理错误

### 2. 核心方法

| 方法 | 测试要点 |
|------|----------|
| config.get | 无 params 或空 params，返回 config 对象 |
| sessions.list | 返回 sessions 数组（可为空） |
| nodes.list | 返回 nodes 数组（可为空） |
| agent.invoke | 带 message、可选 sessionId、stream，返回 result.text 或流式 chunk |

### 3. 消息格式

- 请求：`{ "method": "config.get", "params": {} }`
- 成功响应：`{ "result": { ... } }`
- 错误响应：`{ "error": { "code": number, "message": string } }`

### 4. 流式响应

- `agent.invoke` 且 `stream: true` 时，服务端推送多条带 `result.chunk` 的消息，最后一条可为完整 `result`。

## 运行方式（建议）

- 使用 Go 测试：在 `internal/gateway` 或新建 `test/e2e` 下写集成测试，启动 Gateway 后通过 gorilla/websocket 发送请求并断言响应。
- 或使用脚本（如 Node 或 Python）连接 WS，按上表发送请求并校验响应字段。

## 已实现用例（0.6）

自动化测试见 `internal/gateway/compatibility_test.go`，运行：`go test ./internal/gateway/... -v`

- [x] WebSocket 连接成功
- [x] 无效 JSON 返回 Parse error（-32700）
- [x] config.get 请求/响应格式（result.config）
- [x] sessions.list 请求/响应格式（result.sessions）
- [x] nodes.list 请求/响应格式（result.nodes）
- [x] agent.invoke 缺少 message 时返回错误（-32602）
- [x] HTTP GET /health 返回 200 与 `{ "ok": true }`
- [x] agent.invoke 非流式请求/响应（mock LLM，无需 API Key）
- [x] agent.invoke 流式 chunk 格式（mock SSE）

## 待扩展用例

- [ ] 更多错误码与官方对齐（如 -32600 无效请求）
