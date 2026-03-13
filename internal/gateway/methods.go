package gateway

import (
	"encoding/json"
	"time"

	"gopenclaw/internal/config"
	"gopenclaw/internal/protocol"

	"github.com/gorilla/websocket"
)

// MethodHandler 方法处理器函数类型
type MethodHandler func(g *Gateway, msg *protocol.Message, conn *websocket.Conn) *protocol.Message

// methodHandlers 方法处理器映射
var methodHandlers = map[string]MethodHandler{
	// 配置相关
	"config.get":           handleConfigGet,
	"config.set":           handleConfigSet,
	"config.patch":         handleConfigPatch,
	"config.apply":         handleConfigApply,
	"config.schema":         handleConfigSchema,

	// 会话相关
	"sessions.list":       handleSessionsList,
	"sessions.get":        handleSessionsGet,
	"sessions.patch":      handleSessionsPatch,
	"sessions.delete":     handleSessionsDelete,
	"sessions.reset":      handleSessionsReset,
	"sessions.compact":    handleSessionsCompact,
	"sessions.preview":    handleSessionsPreview,
	"sessions.history":    handleSessionsHistory,
	"sessions.send":       handleSessionsSend,

	// 节点相关
	"nodes.list":          handleNodesList,
	"nodes.get":           handleNodesGet,
	"nodes.describe":      handleNodesDescribe,

	// Agent 相关
	"agent.invoke":        handleAgentInvoke,
	"agent":              handleAgent,
	"agent.identity.get": handleAgentIdentityGet,
	"agent.wait":         handleAgentWait,

	// 工具相关
	"tools.catalog":      handleToolsCatalog,

	// Agent 管理
	"agents.list":        handleAgentsList,
	"agents.create":      handleAgentsCreate,
	"agents.update":      handleAgentsUpdate,
	"agents.delete":      handleAgentsDelete,

	// 通道相关
	"channels.status":    handleChannelsStatus,
	"channels.logout":    handleChannelsLogout,

	// 订阅/事件
	"wake":               handleWake,
	"system-event":       handleSystemEvent,
	"set-heartbeats":     handleSetHeartbeats,
	"last-heartbeat":     handleLastHeartbeat,

	// Cron 相关
	"cron.list":          handleCronList,
	"cron.status":        handleCronStatus,
	"cron.add":           handleCronAdd,
	"cron.update":        handleCronUpdate,
	"cron.remove":        handleCronRemove,
	"cron.run":           handleCronRun,
	"cron.runs":          handleCronRuns,

	// TTS 相关
	"tts.status":         handleTTSStatus,
	"tts.providers":      handleTTSProviders,
	"tts.enable":         handleTTSEnable,
	"tts.disable":        handleTTSDisable,
	"tts.convert":       handleTTSConvert,
	"tts.setProvider":   handleTTSSetProvider,

	// 执行审批
	"exec.approvals.get":   handleExecApprovalsGet,
	"exec.approvals.set":   handleExecApprovalsSet,

	// 向导
	"wizard.start":       handleWizardStart,
	"wizard.next":         handleWizardNext,
	"wizard.cancel":       handleWizardCancel,
	"wizard.status":       handleWizardStatus,

	// 技能
	"skills.status":      handleSkillsStatus,
	"skills.bins":        handleSkillsBins,
	"skills.install":     handleSkillsInstall,
	"skills.update":      handleSkillsUpdate,

	// 模型
	"models.list":        handleModelsList,

	// 使用量
	"usage.status":       handleUsageStatus,
	"usage.cost":         handleUsageCost,

	// 日志
	"logs.tail":          handleLogsTail,

	// 设备配对
	"device.pair.list":   handleDevicePairList,
	"device.pair.approve": handleDevicePairApprove,
	"device.pair.reject":  handleDevicePairReject,
	"device.pair.remove":  handleDevicePairRemove,

	// 健康检查
	"health":             handleHealth,
	"doctor.memory.status": handleDoctorMemoryStatus,
}

// dispatch 分发消息到对应方法处理器
func (g *Gateway) dispatch(conn *websocket.Conn, msg *protocol.Message) *protocol.Message {
	handler, ok := methodHandlers[msg.Method]
	if !ok {
		return g.methodNotImplemented(msg)
	}
	return handler(g, msg, conn)
}

// ============ 配置相关 ============

func handleConfigGet(g *Gateway, msg *protocol.Message, _ *websocket.Conn) *protocol.Message {
	return &protocol.Message{
		ID:     msg.ID,
		Result: protocol.ConfigGetResult{Config: g.cfg},
	}
}

func handleConfigSet(g *Gateway, msg *protocol.Message, _ *websocket.Conn) *protocol.Message {
	var params protocol.ConfigSetParams
	if err := decodeParams(msg.Params, &params); err != nil {
		return errorResult(msg.ID, -32602, err.Error())
	}

	if g.cfg == nil {
		g.cfg = config.Default()
	}

	// 简单实现：直接设置值
	if params.Key != "" && params.Value != nil {
		jsonData, _ := json.Marshal(g.cfg)
		var m map[string]interface{}
		json.Unmarshal(jsonData, &m)
		m[params.Key] = params.Value
		g.cfg = unmarshalConfig(m)
	}

	return &protocol.Message{
		ID:     msg.ID,
		Result: protocol.ConfigSetResult{OK: true},
	}
}

func handleConfigPatch(g *Gateway, msg *protocol.Message, _ *websocket.Conn) *protocol.Message {
	var params protocol.ConfigPatchParams
	if err := decodeParams(msg.Params, &params); err != nil {
		return errorResult(msg.ID, -32602, err.Error())
	}

	// TODO: 实现配置 patch
	return &protocol.Message{
		ID:     msg.ID,
		Result: protocol.ConfigSetResult{OK: true},
	}
}

func handleConfigApply(g *Gateway, msg *protocol.Message, _ *websocket.Conn) *protocol.Message {
	var params protocol.ConfigApplyParams
	if err := decodeParams(msg.Params, &params); err != nil {
		return errorResult(msg.ID, -32602, err.Error())
	}

	// TODO: 实现配置应用
	return &protocol.Message{
		ID:     msg.ID,
		Result: protocol.ConfigSetResult{OK: true},
	}
}

func handleConfigSchema(g *Gateway, msg *protocol.Message, _ *websocket.Conn) *protocol.Message {
	var params protocol.ConfigSchemaParams
	if err := decodeParams(msg.Params, &params); err != nil {
		return errorResult(msg.ID, -32602, err.Error())
	}

	// 返回简单 schema
	schema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"gateway": map[string]string{"type": "object"},
			"agent":   map[string]string{"type": "object"},
			"channels": map[string]string{"type": "object"},
		},
	}

	return &protocol.Message{
		ID:     msg.ID,
		Result: protocol.ConfigSchemaResult{Schema: schema},
	}
}

// ============ 会话相关 ============

func handleSessionsList(g *Gateway, msg *protocol.Message, _ *websocket.Conn) *protocol.Message {
	g.mu.RLock()
	list := make([]protocol.SessionSummary, 0, len(g.sessions))
	for _, s := range g.sessions {
		list = append(list, s)
	}
	g.mu.RUnlock()
	return &protocol.Message{
		ID:     msg.ID,
		Result: protocol.SessionsListResult{Sessions: list, Total: len(list)},
	}
}

func handleSessionsGet(g *Gateway, msg *protocol.Message, _ *websocket.Conn) *protocol.Message {
	var params protocol.SessionsGetParams
	if err := decodeParams(msg.Params, &params); err != nil {
		return errorResult(msg.ID, -32602, err.Error())
	}

	g.mu.RLock()
	session, ok := g.sessions[params.ID]
	g.mu.RUnlock()

	if !ok {
		return errorResult(msg.ID, -32602, "session not found")
	}

	return &protocol.Message{
		ID:     msg.ID,
		Result: protocol.SessionDetail{
			ID:         session.ID,
			Key:        session.ID,
			Label:      session.Label,
			CreatedAt:  time.Now().UnixMilli(),
			UpdatedAt:  time.Now().UnixMilli(),
		},
	}
}

func handleSessionsPatch(g *Gateway, msg *protocol.Message, _ *websocket.Conn) *protocol.Message {
	var params protocol.SessionsPatchParams
	if err := decodeParams(msg.Params, &params); err != nil {
		return errorResult(msg.ID, -32602, err.Error())
	}

	g.mu.Lock()
	defer g.mu.Unlock()

	if _, ok := g.sessions[params.ID]; !ok {
		return errorResult(msg.ID, -32602, "session not found")
	}

	// TODO: 应用 patch
	return &protocol.Message{
		ID:     msg.ID,
		Result: protocol.ConfigSetResult{OK: true},
	}
}

func handleSessionsDelete(g *Gateway, msg *protocol.Message, _ *websocket.Conn) *protocol.Message {
	var params protocol.SessionsDeleteParams
	if err := decodeParams(msg.Params, &params); err != nil {
		return errorResult(msg.ID, -32602, err.Error())
	}

	g.mu.Lock()
	defer g.mu.Unlock()

	if _, ok := g.sessions[params.ID]; !ok {
		return errorResult(msg.ID, -32602, "session not found")
	}

	delete(g.sessions, params.ID)
	return &protocol.Message{
		ID:     msg.ID,
		Result: protocol.ConfigSetResult{OK: true},
	}
}

func handleSessionsReset(g *Gateway, msg *protocol.Message, _ *websocket.Conn) *protocol.Message {
	var params protocol.SessionsResetParams
	if err := decodeParams(msg.Params, &params); err != nil {
		return errorResult(msg.ID, -32602, err.Error())
	}

	// TODO: 实现重置
	return &protocol.Message{
		ID:     msg.ID,
		Result: protocol.ConfigSetResult{OK: true},
	}
}

func handleSessionsCompact(g *Gateway, msg *protocol.Message, _ *websocket.Conn) *protocol.Message {
	var params protocol.SessionsCompactParams
	if err := decodeParams(msg.Params, &params); err != nil {
		return errorResult(msg.ID, -32602, err.Error())
	}

	// TODO: 实现压缩
	return &protocol.Message{
		ID:     msg.ID,
		Result: protocol.ConfigSetResult{OK: true},
	}
}

func handleSessionsPreview(g *Gateway, msg *protocol.Message, _ *websocket.Conn) *protocol.Message {
	var params protocol.SessionsPreviewParams
	if err := decodeParams(msg.Params, &params); err != nil {
		return errorResult(msg.ID, -32602, err.Error())
	}

	// TODO: 实现预览
	return &protocol.Message{
		ID:     msg.ID,
		Result: protocol.SessionsListResult{Sessions: []protocol.SessionSummary{}, Total: 0},
	}
}

func handleSessionsHistory(g *Gateway, msg *protocol.Message, _ *websocket.Conn) *protocol.Message {
	var params protocol.SessionsHistoryParams
	if err := decodeParams(msg.Params, &params); err != nil {
		return errorResult(msg.ID, -32602, err.Error())
	}

	// TODO: 实现历史
	return &protocol.Message{
		ID:     msg.ID,
		Result: map[string]interface{}{"messages": []interface{}{}},
	}
}

func handleSessionsSend(g *Gateway, msg *protocol.Message, _ *websocket.Conn) *protocol.Message {
	var params protocol.SessionsSendParams
	if err := decodeParams(msg.Params, &params); err != nil {
		return errorResult(msg.ID, -32602, err.Error())
	}

	// TODO: 实现发送
	return &protocol.Message{
		ID:     msg.ID,
		Result: protocol.ConfigSetResult{OK: true},
	}
}

// ============ 节点相关 ============

func handleNodesList(g *Gateway, msg *protocol.Message, _ *websocket.Conn) *protocol.Message {
	return &protocol.Message{
		ID:     msg.ID,
		Result: protocol.NodesListResult{Nodes: []protocol.NodeSummary{}},
	}
}

func handleNodesGet(g *Gateway, msg *protocol.Message, _ *websocket.Conn) *protocol.Message {
	var params protocol.NodesGetParams
	if err := decodeParams(msg.Params, &params); err != nil {
		return errorResult(msg.ID, -32602, err.Error())
	}

	return errorResult(msg.ID, -32601, "node not found")
}

func handleNodesDescribe(g *Gateway, msg *protocol.Message, _ *websocket.Conn) *protocol.Message {
	var params protocol.NodesGetParams
	if err := decodeParams(msg.Params, &params); err != nil {
		return errorResult(msg.ID, -32602, err.Error())
	}

	return errorResult(msg.ID, -32601, "node not found")
}

// ============ Agent 相关 ============

func handleAgentInvoke(g *Gateway, msg *protocol.Message, conn *websocket.Conn) *protocol.Message {
	return g.handleAgentInvoke(conn, msg)
}

func handleAgent(g *Gateway, msg *protocol.Message, conn *websocket.Conn) *protocol.Message {
	var params protocol.AgentRunParams
	if err := decodeParams(msg.Params, &params); err != nil {
		return errorResult(msg.ID, -32602, err.Error())
	}

	// TODO: 实现 agent
	return &protocol.Message{
		ID:     msg.ID,
		Result: protocol.AgentInvokeResult{Text: "agent not implemented"},
	}
}

func handleAgentIdentityGet(g *Gateway, msg *protocol.Message, _ *websocket.Conn) *protocol.Message {
	return &protocol.Message{
		ID: msg.ID,
		Result: protocol.AgentIdentityGetResult{
			ID:       "main",
			Name:     "Gopenclaw",
			Model:    g.cfg.Agent.Model,
			Provider: "openai",
		},
	}
}

func handleAgentWait(g *Gateway, msg *protocol.Message, _ *websocket.Conn) *protocol.Message {
	// TODO: 实现等待
	return &protocol.Message{
		ID:     msg.ID,
		Result: map[string]interface{}{"status": "ok"},
	}
}

// ============ 工具相关 ============

func handleToolsCatalog(g *Gateway, msg *protocol.Message, _ *websocket.Conn) *protocol.Message {
	// 从 tools.Registry 获取工具列表
	toolDefs := g.tools.GetToolDefinitions()

	tools := make([]protocol.ToolSummary, 0, len(toolDefs))
	for _, td := range toolDefs {
		fn, ok := td["function"].(map[string]any)
		if !ok {
			continue
		}
		name, _ := fn["name"].(string)
		desc, _ := fn["description"].(string)
		tools = append(tools, protocol.ToolSummary{
			Name:        name,
			Description: desc,
			Category:    "builtin",
			Parameters:  fn["parameters"],
			BuiltIn:     true,
		})
	}

	return &protocol.Message{
		ID:     msg.ID,
		Result: protocol.ToolsCatalogResult{Tools: tools},
	}
}

// ============ Agent 管理 ============

func handleAgentsList(g *Gateway, msg *protocol.Message, _ *websocket.Conn) *protocol.Message {
	return &protocol.Message{
		ID: msg.ID,
		Result: protocol.AgentListResult{
			Agents: []protocol.AgentSummary{
				{ID: "main", Name: "Main Agent", Model: g.cfg.Agent.Model, Status: "active", BuiltIn: true},
			},
		},
	}
}

func handleAgentsCreate(g *Gateway, msg *protocol.Message, _ *websocket.Conn) *protocol.Message {
	var params protocol.AgentCreateParams
	if err := decodeParams(msg.Params, &params); err != nil {
		return errorResult(msg.ID, -32602, err.Error())
	}

	// TODO: 实现创建
	return errorResult(msg.ID, -32601, "not implemented")
}

func handleAgentsUpdate(g *Gateway, msg *protocol.Message, _ *websocket.Conn) *protocol.Message {
	var params protocol.AgentUpdateParams
	if err := decodeParams(msg.Params, &params); err != nil {
		return errorResult(msg.ID, -32602, err.Error())
	}

	return errorResult(msg.ID, -32601, "not implemented")
}

func handleAgentsDelete(g *Gateway, msg *protocol.Message, _ *websocket.Conn) *protocol.Message {
	var params protocol.AgentDeleteParams
	if err := decodeParams(msg.Params, &params); err != nil {
		return errorResult(msg.ID, -32602, err.Error())
	}

	return errorResult(msg.ID, -32601, "not implemented")
}

// ============ 通道相关 ============

func handleChannelsStatus(g *Gateway, msg *protocol.Message, _ *websocket.Conn) *protocol.Message {
	return &protocol.Message{
		ID: msg.ID,
		Result: protocol.ChannelsStatusResult{
			Channels: []protocol.ChannelStatus{
				{ID: "webchat", Name: "WebChat", Status: "connected", Connected: true},
			},
		},
	}
}

func handleChannelsLogout(g *Gateway, msg *protocol.Message, _ *websocket.Conn) *protocol.Message {
	var params protocol.ChannelsLogoutParams
	if err := decodeParams(msg.Params, &params); err != nil {
		return errorResult(msg.ID, -32602, err.Error())
	}

	return &protocol.Message{
		ID:     msg.ID,
		Result: protocol.ConfigSetResult{OK: true},
	}
}

// ============ 订阅/事件 ============

func handleWake(g *Gateway, msg *protocol.Message, _ *websocket.Conn) *protocol.Message {
	var params protocol.WakeParams
	if err := decodeParams(msg.Params, &params); err != nil {
		return errorResult(msg.ID, -32602, err.Error())
	}

	// TODO: 实现 wake
	return &protocol.Message{
		ID:     msg.ID,
		Result: map[string]interface{}{"status": "ok"},
	}
}

func handleSystemEvent(g *Gateway, msg *protocol.Message, _ *websocket.Conn) *protocol.Message {
	var params protocol.SystemEventParams
	if err := decodeParams(msg.Params, &params); err != nil {
		return errorResult(msg.ID, -32602, err.Error())
	}

	// TODO: 实现系统事件
	return &protocol.Message{
		ID:     msg.ID,
		Result: map[string]interface{}{"status": "ok"},
	}
}

func handleSetHeartbeats(g *Gateway, msg *protocol.Message, _ *websocket.Conn) *protocol.Message {
	var params protocol.SetHeartbeatsParams
	if err := decodeParams(msg.Params, &params); err != nil {
		return errorResult(msg.ID, -32602, err.Error())
	}

	// TODO: 实现设置 heartbeats
	return &protocol.Message{
		ID:     msg.ID,
		Result: map[string]interface{}{"status": "ok"},
	}
}

func handleLastHeartbeat(g *Gateway, msg *protocol.Message, _ *websocket.Conn) *protocol.Message {
	// TODO: 实现
	return &protocol.Message{
		ID: msg.ID,
		Result: protocol.HeartbeatEvent{
			SessionID: "main",
			Text:      "",
			Timestamp: time.Now().UnixMilli(),
		},
	}
}

// ============ Cron 相关 ============

func handleCronList(g *Gateway, msg *protocol.Message, _ *websocket.Conn) *protocol.Message {
	// TODO: 从 cron scheduler 获取
	return &protocol.Message{
		ID:     msg.ID,
		Result: protocol.CronListResult{Jobs: []protocol.CronJobSummary{}},
	}
}

func handleCronStatus(g *Gateway, msg *protocol.Message, _ *websocket.Conn) *protocol.Message {
	var params protocol.CronStatusParams
	if err := decodeParams(msg.Params, &params); err != nil {
		return errorResult(msg.ID, -32602, err.Error())
	}

	return errorResult(msg.ID, -32601, "job not found")
}

func handleCronAdd(g *Gateway, msg *protocol.Message, _ *websocket.Conn) *protocol.Message {
	var params protocol.CronAddParams
	if err := decodeParams(msg.Params, &params); err != nil {
		return errorResult(msg.ID, -32602, err.Error())
	}

	// TODO: 添加 cron 任务
	return &protocol.Message{
		ID:     msg.ID,
		Result: map[string]interface{}{"id": "new-job-id"},
	}
}

func handleCronUpdate(g *Gateway, msg *protocol.Message, _ *websocket.Conn) *protocol.Message {
	var params protocol.CronUpdateParams
	if err := decodeParams(msg.Params, &params); err != nil {
		return errorResult(msg.ID, -32602, err.Error())
	}

	return &protocol.Message{
		ID:     msg.ID,
		Result: protocol.ConfigSetResult{OK: true},
	}
}

func handleCronRemove(g *Gateway, msg *protocol.Message, _ *websocket.Conn) *protocol.Message {
	var params protocol.CronRemoveParams
	if err := decodeParams(msg.Params, &params); err != nil {
		return errorResult(msg.ID, -32602, err.Error())
	}

	return &protocol.Message{
		ID:     msg.ID,
		Result: protocol.ConfigSetResult{OK: true},
	}
}

func handleCronRun(g *Gateway, msg *protocol.Message, _ *websocket.Conn) *protocol.Message {
	var params protocol.CronRunParams
	if err := decodeParams(msg.Params, &params); err != nil {
		return errorResult(msg.ID, -32602, err.Error())
	}

	// TODO: 运行 cron 任务
	return &protocol.Message{
		ID:     msg.ID,
		Result: map[string]interface{}{"status": "queued"},
	}
}

func handleCronRuns(g *Gateway, msg *protocol.Message, _ *websocket.Conn) *protocol.Message {
	var params protocol.CronRunsParams
	if err := decodeParams(msg.Params, &params); err != nil {
		return errorResult(msg.ID, -32602, err.Error())
	}

	return &protocol.Message{
		ID:     msg.ID,
		Result: protocol.CronRunsResult{Runs: []protocol.CronRunSummary{}, Total: 0},
	}
}

// ============ TTS 相关 ============

func handleTTSStatus(g *Gateway, msg *protocol.Message, _ *websocket.Conn) *protocol.Message {
	return &protocol.Message{
		ID: msg.ID,
		Result: protocol.TTSStatusResult{
			Enabled:  false,
			Provider: "",
			Voices:   []string{},
		},
	}
}

func handleTTSProviders(g *Gateway, msg *protocol.Message, _ *websocket.Conn) *protocol.Message {
	return &protocol.Message{
		ID: msg.ID,
		Result: protocol.TTSProvidersResult{
			Providers: []protocol.TTSProvider{},
		},
	}
}

func handleTTSEnable(g *Gateway, msg *protocol.Message, _ *websocket.Conn) *protocol.Message {
	var params protocol.TTSEnableParams
	if err := decodeParams(msg.Params, &params); err != nil {
		return errorResult(msg.ID, -32602, err.Error())
	}

	return &protocol.Message{
		ID:     msg.ID,
		Result: protocol.ConfigSetResult{OK: true},
	}
}

func handleTTSDisable(g *Gateway, msg *protocol.Message, _ *websocket.Conn) *protocol.Message {
	var params protocol.TTSDisableParams
	if err := decodeParams(msg.Params, &params); err != nil {
		return errorResult(msg.ID, -32602, err.Error())
	}

	return &protocol.Message{
		ID:     msg.ID,
		Result: protocol.ConfigSetResult{OK: true},
	}
}

func handleTTSConvert(g *Gateway, msg *protocol.Message, _ *websocket.Conn) *protocol.Message {
	var params protocol.TTSConvertParams
	if err := decodeParams(msg.Params, &params); err != nil {
		return errorResult(msg.ID, -32602, err.Error())
	}

	return errorResult(msg.ID, -32601, "TTS not enabled")
}

func handleTTSSetProvider(g *Gateway, msg *protocol.Message, _ *websocket.Conn) *protocol.Message {
	// TODO: 实现
	return &protocol.Message{
		ID:     msg.ID,
		Result: protocol.ConfigSetResult{OK: true},
	}
}

// ============ 执行审批相关 ============

func handleExecApprovalsGet(g *Gateway, msg *protocol.Message, _ *websocket.Conn) *protocol.Message {
	return &protocol.Message{
		ID: msg.ID,
		Result: protocol.ExecApprovalsResult{
			NodeID:    "",
			Allowed:   []string{},
			Denied:    []string{},
			ExactOnly: false,
		},
	}
}

func handleExecApprovalsSet(g *Gateway, msg *protocol.Message, _ *websocket.Conn) *protocol.Message {
	var params protocol.ExecApprovalsSetParams
	if err := decodeParams(msg.Params, &params); err != nil {
		return errorResult(msg.ID, -32602, err.Error())
	}

	return &protocol.Message{
		ID:     msg.ID,
		Result: protocol.ConfigSetResult{OK: true},
	}
}

// ============ 向导相关 ============

func handleWizardStart(g *Gateway, msg *protocol.Message, _ *websocket.Conn) *protocol.Message {
	var params protocol.WizardStartParams
	if err := decodeParams(msg.Params, &params); err != nil {
		return errorResult(msg.ID, -32602, err.Error())
	}

	return &protocol.Message{
		ID: msg.ID,
		Result: protocol.WizardResult{
			ID:       "wizard-1",
			Type:     params.Type,
			Step:     "start",
			Status:   "in_progress",
			NextStep: "configure",
		},
	}
}

func handleWizardNext(g *Gateway, msg *protocol.Message, _ *websocket.Conn) *protocol.Message {
	var params protocol.WizardNextParams
	if err := decodeParams(msg.Params, &params); err != nil {
		return errorResult(msg.ID, -32602, err.Error())
	}

	return &protocol.Message{
		ID: msg.ID,
		Result: protocol.WizardResult{
			ID:       "wizard-1",
			Status:   "completed",
		},
	}
}

func handleWizardCancel(g *Gateway, msg *protocol.Message, _ *websocket.Conn) *protocol.Message {
	var params protocol.WizardCancelParams
	if err := decodeParams(msg.Params, &params); err != nil {
		return errorResult(msg.ID, -32602, err.Error())
	}

	return &protocol.Message{
		ID:     msg.ID,
		Result: protocol.ConfigSetResult{OK: true},
	}
}

func handleWizardStatus(g *Gateway, msg *protocol.Message, _ *websocket.Conn) *protocol.Message {
	var params protocol.WizardStatusParams
	if err := decodeParams(msg.Params, &params); err != nil {
		return errorResult(msg.ID, -32602, err.Error())
	}

	return &protocol.Message{
		ID: msg.ID,
		Result: protocol.WizardResult{
			Status: "not_started",
		},
	}
}

// ============ 技能相关 ============

func handleSkillsStatus(g *Gateway, msg *protocol.Message, _ *websocket.Conn) *protocol.Message {
	return &protocol.Message{
		ID:     msg.ID,
		Result: protocol.SkillsStatusResult{Skills: []protocol.SkillInfo{}},
	}
}

func handleSkillsBins(g *Gateway, msg *protocol.Message, _ *websocket.Conn) *protocol.Message {
	return &protocol.Message{
		ID:     msg.ID,
		Result: map[string]interface{}{"bins": []string{}},
	}
}

func handleSkillsInstall(g *Gateway, msg *protocol.Message, _ *websocket.Conn) *protocol.Message {
	var params protocol.SkillsInstallParams
	if err := decodeParams(msg.Params, &params); err != nil {
		return errorResult(msg.ID, -32602, err.Error())
	}

	return errorResult(msg.ID, -32601, "not implemented")
}

func handleSkillsUpdate(g *Gateway, msg *protocol.Message, _ *websocket.Conn) *protocol.Message {
	var params protocol.SkillsUpdateParams
	if err := decodeParams(msg.Params, &params); err != nil {
		return errorResult(msg.ID, -32602, err.Error())
	}

	return errorResult(msg.ID, -32601, "not implemented")
}

// ============ 模型相关 ============

func handleModelsList(g *Gateway, msg *protocol.Message, _ *websocket.Conn) *protocol.Message {
	return &protocol.Message{
		ID: msg.ID,
		Result: protocol.ModelsListResult{
			Models: []protocol.ModelInfo{
				{ID: "gpt-4o", Name: "GPT-4o", Provider: "openai", ContextWindow: 128000, MaxOutputTokens: 16384, SupportsVision: true, SupportsTools: true, Enabled: true},
				{ID: "gpt-4o-mini", Name: "GPT-4o Mini", Provider: "openai", ContextWindow: 128000, MaxOutputTokens: 16384, SupportsVision: true, SupportsTools: true, Enabled: true},
			},
		},
	}
}

// ============ 使用量相关 ============

func handleUsageStatus(g *Gateway, msg *protocol.Message, _ *websocket.Conn) *protocol.Message {
	return &protocol.Message{
		ID: msg.ID,
		Result: protocol.UsageStatusResult{
			Period: &protocol.UsagePeriod{
				Start: time.Now().AddDate(0, 0, -30).Format("2006-01-02"),
				End:   time.Now().Format("2006-01-02"),
			},
			TotalTokens:  0,
			InputTokens:  0,
			OutputTokens: 0,
			TotalCost:    0,
		},
	}
}

func handleUsageCost(g *Gateway, msg *protocol.Message, _ *websocket.Conn) *protocol.Message {
	return &protocol.Message{
		ID: msg.ID,
		Result: protocol.UsageCostResult{
			TotalCost: 0,
		},
	}
}

// ============ 日志相关 ============

func handleLogsTail(g *Gateway, msg *protocol.Message, _ *websocket.Conn) *protocol.Message {
	var params protocol.LogsTailParams
	if err := decodeParams(msg.Params, &params); err != nil {
		return errorResult(msg.ID, -32602, err.Error())
	}

	return &protocol.Message{
		ID:     msg.ID,
		Result: protocol.LogsTailResult{Entries: []protocol.LogEntry{}},
	}
}

// ============ 设备配对相关 ============

func handleDevicePairList(g *Gateway, msg *protocol.Message, _ *websocket.Conn) *protocol.Message {
	return &protocol.Message{
		ID: msg.ID,
		Result: protocol.DevicePairListResult{
			Devices: []protocol.DeviceInfo{},
			Pending: []protocol.PairRequest{},
		},
	}
}

func handleDevicePairApprove(g *Gateway, msg *protocol.Message, _ *websocket.Conn) *protocol.Message {
	var params protocol.DevicePairApproveParams
	if err := decodeParams(msg.Params, &params); err != nil {
		return errorResult(msg.ID, -32602, err.Error())
	}

	return &protocol.Message{
		ID:     msg.ID,
		Result: protocol.ConfigSetResult{OK: true},
	}
}

func handleDevicePairReject(g *Gateway, msg *protocol.Message, _ *websocket.Conn) *protocol.Message {
	var params protocol.DevicePairRejectParams
	if err := decodeParams(msg.Params, &params); err != nil {
		return errorResult(msg.ID, -32602, err.Error())
	}

	return &protocol.Message{
		ID:     msg.ID,
		Result: protocol.ConfigSetResult{OK: true},
	}
}

func handleDevicePairRemove(g *Gateway, msg *protocol.Message, _ *websocket.Conn) *protocol.Message {
	var params protocol.DevicePairRemoveParams
	if err := decodeParams(msg.Params, &params); err != nil {
		return errorResult(msg.ID, -32602, err.Error())
	}

	return &protocol.Message{
		ID:     msg.ID,
		Result: protocol.ConfigSetResult{OK: true},
	}
}

// ============ 健康检查 ============

func handleHealth(g *Gateway, msg *protocol.Message, _ *websocket.Conn) *protocol.Message {
	return &protocol.Message{
		ID: msg.ID,
		Result: protocol.HealthResult{
			Status:  "ok",
			Uptime:  time.Since(g.startTime).Milliseconds(),
			Version: "0.1.0",
			Checks:  map[string]protocol.Check{},
		},
	}
}

func handleDoctorMemoryStatus(g *Gateway, msg *protocol.Message, _ *websocket.Conn) *protocol.Message {
	return &protocol.Message{
		ID: msg.ID,
		Result: protocol.DoctorMemoryStatusResult{
			Total:       0,
			Used:        0,
			SessionCount: len(g.sessions),
			GCEnabled:  true,
		},
	}
}

// ============ 辅助函数 ============

func decodeParams(params interface{}, target interface{}) error {
	if params == nil {
		return nil
	}
	data, err := json.Marshal(params)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, target)
}

func errorResult(id interface{}, code int, message string) *protocol.Message {
	return &protocol.Message{
		ID: id,
		Error: &protocol.RPCError{
			Code:    code,
			Message: message,
		},
	}
}

func unmarshalConfig(m map[string]interface{}) *config.Config {
	data, _ := json.Marshal(m)
	cfg := config.Default()
	json.Unmarshal(data, cfg)
	return cfg
}

// 添加启动时间
func (g *Gateway) SetStartTime(t time.Time) {
	g.startTime = t
}
