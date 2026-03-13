# Gopenclaw Gateway 方法清单

通过 WebSocket 连接 `ws://<host>:11999/ws` 后，客户端发送 JSON 消息，`method` 字段指定方法名。

## 通用消息格式

- **请求**：`{ "id": optional, "method": "method.name", "params": { ... } }`
- **响应**：`{ "id": optional, "result": { ... } }` 或 `{ "id": optional, "error": { "code": number, "message": string } }`

## 方法对照表

| 类别 | 方法 | 说明 |
|------|------|------|
| 配置 | config.get | 获取配置 |
| 配置 | config.set | 设置配置项 |
| 配置 | config.patch | 部分更新配置 |
| 配置 | config.apply | 应用配置 |
| 配置 | config.schema | 获取配置 schema |
| 会话 | sessions.list | 列出会话 |
| 会话 | sessions.get | 获取会话详情 |
| 会话 | sessions.patch | 更新会话 |
| 会话 | sessions.delete | 删除会话 |
| 会话 | sessions.reset | 重置会话 |
| 会话 | sessions.compact | 压缩会话历史 |
| 会话 | sessions.preview | 会话预览 |
| 会话 | sessions.history | 会话历史 |
| 会话 | sessions.send | 发送消息到会话 |
| 节点 | nodes.list | 列出节点 |
| 节点 | nodes.get | 获取节点 |
| 节点 | nodes.describe | 节点描述 |
| Agent | agent.invoke | 调用 Agent（支持流式） |
| Agent | agent | 别名 |
| Agent | agent.identity.get | 获取 Agent 身份 |
| Agent | agent.wait | 等待 Agent |
| 工具 | tools.catalog | 获取工具目录 |
| 通道 | channels.status | 通道状态 |
| 通道 | channels.logout | 通道登出 |
| 订阅 | wake | 唤醒 |
| 订阅 | system-event | 系统事件 |
| 订阅 | set-heartbeats | 设置心跳 |
| 订阅 | last-heartbeat | 最后心跳 |
| Cron | cron.list | 列出定时任务 |
| Cron | cron.status | 任务状态 |
| Cron | cron.add | 添加任务 |
| Cron | cron.update | 更新任务 |
| Cron | cron.remove | 移除任务 |
| Cron | cron.run | 立即执行 |
| Cron | cron.runs | 运行记录 |
| TTS | tts.status | TTS 状态 |
| TTS | tts.providers | TTS 提供商列表 |
| TTS | tts.enable | 启用 TTS |
| TTS | tts.disable | 禁用 TTS |
| TTS | tts.convert | 文本转语音 |
| TTS | tts.setProvider | 设置提供商 |
| 执行审批 | exec.approvals.get | 获取审批配置 |
| 执行审批 | exec.approvals.set | 设置审批配置 |
| 向导 | wizard.start | 启动向导 |
| 向导 | wizard.next | 下一步 |
| 向导 | wizard.cancel | 取消 |
| 向导 | wizard.status | 向导状态 |
| 技能 | skills.status | 技能状态 |
| 技能 | skills.bins | 技能二进制列表 |
| 技能 | skills.install | 安装技能 |
| 技能 | skills.update | 更新技能 |
| 模型 | models.list | 列出模型 |
| 使用量 | usage.status | 使用量状态 |
| 使用量 | usage.cost | 费用查询 |
| 日志 | logs.tail | 日志尾读 |
| 设备配对 | device.pair.list | 配对列表 |
| 设备配对 | device.pair.approve | 批准配对 |
| 设备配对 | device.pair.reject | 拒绝配对 |
| 设备配对 | device.pair.remove | 移除配对 |
| 健康 | health | 健康检查（也可 HTTP GET /health） |
| 健康 | doctor.memory.status | 内存诊断 |

## HTTP 端点

| 路径 | 方法 | 说明 |
|------|------|------|
| / | GET | WebChat 静态页 |
| /ws | GET | WebSocket 升级 |
| /health | GET | 健康检查 |

## 与官方 OpenClaw 的兼容性

Gopenclaw 的协议与官方 OpenClaw Gateway 保持兼容，方法名与参数结构对齐。部分方法当前为桩实现，返回默认值，可按需扩展。
