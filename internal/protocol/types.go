package protocol

import "time"

// WS 消息与 OpenClaw Gateway 协议对齐（完整版）

// Message 客户端或服务端单条 WS 消息（JSON-RPC 风格）
type Message struct {
	ID     interface{} `json:"id,omitempty"`
	Method string      `json:"method,omitempty"`
	Params interface{} `json:"params,omitempty"`
	Result interface{} `json:"result,omitempty"`
	Error  *RPCError   `json:"error,omitempty"`
}

// RPCError 协议错误
type RPCError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// ============ 配置相关 ============

// ConfigGetParams config.get 请求参数
type ConfigGetParams struct {
	Key string `json:"key,omitempty"`
}

// ConfigSetParams config.set 请求参数
type ConfigSetParams struct {
	Key   string      `json:"key"`
	Value interface{} `json:"value"`
}

// ConfigPatchParams config.patch 请求参数
type ConfigPatchParams struct {
	Patch map[string]interface{} `json:"patch"`
}

// ConfigApplyParams config.apply 请求参数
type ConfigApplyParams struct {
	Partial bool `json:"partial,omitempty"`
}

// ConfigSchemaParams config.schema 请求参数
type ConfigSchemaParams struct {
	Key string `json:"key,omitempty"`
}

// ConfigGetResult config.get 响应
type ConfigGetResult struct {
	Config interface{} `json:"config"`
}

// ConfigSetResult config.set 响应
type ConfigSetResult struct {
	OK bool `json:"ok"`
}

// ConfigSchemaResult config.schema 响应
type ConfigSchemaResult struct {
	Schema interface{} `json:"schema"`
}

// ============ 会话相关 ============

// SessionsListParams sessions.list 请求参数
type SessionsListParams struct {
	Limit  int    `json:"limit,omitempty"`
	Offset int    `json:"offset,omitempty"`
	Filter string `json:"filter,omitempty"`
}

// SessionsGetParams sessions.get 请求参数
type SessionsGetParams struct {
	ID string `json:"id"`
}

// SessionsPatchParams sessions.patch 请求参数
type SessionsPatchParams struct {
	ID    string                 `json:"id"`
	Patch map[string]interface{} `json:"patch"`
}

// SessionsDeleteParams sessions.delete 请求参数
type SessionsDeleteParams struct {
	ID string `json:"id"`
}

// SessionsResetParams sessions.reset 请求参数
type SessionsResetParams struct {
	ID string `json:"id"`
}

// SessionsCompactParams sessions.compact 请求参数
type SessionsCompactParams struct {
	ID string `json:"id,omitempty"`
}

// SessionsPreviewParams sessions.preview 请求参数
type SessionsPreviewParams struct {
	ID      string `json:"id"`
	Limit   int    `json:"limit,omitempty"`
	Include string `json:"include,omitempty"`
}

// SessionsHistoryParams sessions.history 请求参数
type SessionsHistoryParams struct {
	ID     string `json:"id"`
	Limit  int    `json:"limit,omitempty"`
	After  string `json:"after,omitempty"`
	Before string `json:"before,omitempty"`
}

// SessionsSendParams sessions.send 请求参数
type SessionsSendParams struct {
	ID      string `json:"id"`
	Message string `json:"message"`
}

// SessionsListResult sessions.list 响应
type SessionsListResult struct {
	Sessions []SessionSummary `json:"sessions"`
	Total    int              `json:"total"`
}

// SessionSummary 会话摘要
type SessionSummary struct {
	ID           string                 `json:"id"`
	Key          string                 `json:"key,omitempty"`
	Label        string                 `json:"label,omitempty"`
	Channel      string                 `json:"channel,omitempty"`
	Model        string                 `json:"model,omitempty"`
	UpdatedAt    int64                  `json:"updatedAt"`
	CreatedAt    int64                  `json:"createdAt,omitempty"`
	Messages     int                    `json:"messages,omitempty"`
	InputTokens  int                    `json:"inputTokens,omitempty"`
	OutputTokens int                    `json:"outputTokens,omitempty"`
}

// SessionDetail 会话详情
type SessionDetail struct {
	ID              string                 `json:"id"`
	Key             string                 `json:"key"`
	Label           string                 `json:"label,omitempty"`
	Channel         string                 `json:"channel,omitempty"`
	Model           string                 `json:"model,omitempty"`
	Provider        string                 `json:"provider,omitempty"`
	Origin          *SessionOrigin         `json:"origin,omitempty"`
	CreatedAt       int64                  `json:"createdAt"`
	UpdatedAt       int64                  `json:"updatedAt"`
	Messages        []Message              `json:"messages,omitempty"`
	InputTokens     int                    `json:"inputTokens,omitempty"`
	OutputTokens    int                    `json:"outputTokens,omitempty"`
	TotalTokens     int                    `json:"totalTokens,omitempty"`
	Runtime         *SessionRuntime        `json:"runtime,omitempty"`
	ACL             *SessionACL            `json:"acl,omitempty"`
}

// SessionOrigin 会话来源
type SessionOrigin struct {
	Label      string `json:"label,omitempty"`
	Provider   string `json:"provider,omitempty"`
	Surface    string `json:"surface,omitempty"`
	ChatType   string `json:"chatType,omitempty"`
	From       string `json:"from,omitempty"`
	To         string `json:"to,omitempty"`
	AccountID  string `json:"accountId,omitempty"`
	ThreadID   string `json:"threadId,omitempty"`
}

// SessionRuntime 会话运行时
type SessionRuntime struct {
	Mode              string `json:"mode,omitempty"`
	Status            string `json:"status,omitempty"`
	LastActivityAt    int64  `json:"lastActivityAt,omitempty"`
	LastError         string `json:"lastError,omitempty"`
	Cwd               string `json:"cwd,omitempty"`
	TimeoutSeconds    int    `json:"timeoutSeconds,omitempty"`
	PermissionProfile string `json:"permissionProfile,omitempty"`
}

// SessionACL 会话访问控制
type SessionACL struct {
	SendPolicy    string   `json:"sendPolicy,omitempty"`
	GroupID       string   `json:"groupId,omitempty"`
	GroupActivation string `json:"groupActivation,omitempty"`
	Allowlist     []string `json:"allowlist,omitempty"`
	Blocklist     []string `json:"blocklist,omitempty"`
}

// ============ 节点相关 ============

// NodesListParams nodes.list 请求参数
type NodesListParams struct {
	Status string `json:"status,omitempty"`
}

// NodesGetParams nodes.get 请求参数
type NodesGetParams struct {
	ID string `json:"id"`
}

// NodesInvokeParams nodes.invoke 请求参数
type NodesInvokeParams struct {
	NodeID   string                 `json:"nodeId"`
	Method   string                 `json:"method"`
	Params   map[string]interface{} `json:"params,omitempty"`
	Timeout  int                    `json:"timeout,omitempty"`
}

// NodesListResult nodes.list 响应
type NodesListResult struct {
	Nodes []NodeSummary `json:"nodes"`
}

// NodeSummary 节点摘要
type NodeSummary struct {
	ID          string            `json:"id"`
	Name        string            `json:"name,omitempty"`
	Status      string            `json:"status,omitempty"`
	Type        string            `json:"type,omitempty"`
	Version     string            `json:"version,omitempty"`
	LastSeenAt  int64             `json:"lastSeenAt,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

// NodeDetail 节点详情
type NodeDetail struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name,omitempty"`
	Status      string                 `json:"status"`
	Type        string                 `json:"type,omitempty"`
	Version     string                 `json:"version,omitempty"`
	CreatedAt   int64                  `json:"createdAt"`
	LastSeenAt  int64                  `json:"lastSeenAt"`
	Capabilities []string              `json:"capabilities,omitempty"`
	Metadata    map[string]string      `json:"metadata,omitempty"`
	Config      map[string]interface{} `json:"config,omitempty"`
}

// NodePairRequestParams node.pair.request 请求参数
type NodePairRequestParams struct {
	Name   string `json:"name"`
	Type   string `json:"type"`
	Code   string `json:"code,omitempty"`
}

// NodePairApproveParams node.pair.approve 请求参数
type NodePairApproveParams struct {
	RequestID string `json:"requestId"`
}

// NodePairRejectParams node.pair.reject 请求参数
type NodePairRejectParams struct {
	RequestID string `json:"requestId"`
	Reason    string `json:"reason,omitempty"`
}

// NodePairListParams node.pair.list 请求参数
type NodePairListParams struct {
	Status string `json:"status,omitempty"`
}

// ============ Agent 相关 ============

// AgentInvokeParams agent.invoke 请求参数
type AgentInvokeParams struct {
	Message       string                 `json:"message"`
	SessionID     string                 `json:"sessionId,omitempty"`
	Stream        bool                   `json:"stream,omitempty"`
	Model         string                 `json:"model,omitempty"`
	Provider      string                 `json:"provider,omitempty"`
	Thinking      string                 `json:"thinking,omitempty"`
	Timeout       int                    `json:"timeoutSeconds,omitempty"`
	Context       map[string]interface{} `json:"context,omitempty"`
}

// AgentRunParams agent 请求参数（用于 cron/webhook）
type AgentRunParams struct {
	Message       string `json:"message"`
	SessionKey    string `json:"sessionKey,omitempty"`
	Model         string `json:"model,omitempty"`
	Thinking      string `json:"thinking,omitempty"`
	Timeout       int    `json:"timeoutSeconds,omitempty"`
	WakeMode      string `json:"wakeMode,omitempty"`
	AgentID       string `json:"agentId,omitempty"`
	Deliver       *bool  `json:"deliver,omitempty"`
	Channel       string `json:"channel,omitempty"`
	To            string `json:"to,omitempty"`
}

// AgentInvokeResult agent.invoke 响应
type AgentInvokeResult struct {
	Text        string           `json:"text"`
	SessionID   string           `json:"sessionId,omitempty"`
	Usage       *UsageSummary    `json:"usage,omitempty"`
	ToolCalls   []ExecutedTool  `json:"toolCalls,omitempty"`
	Error       string          `json:"error,omitempty"`
}

// AgentStreamChunk 流式返回时的单块内容
type AgentStreamChunk struct {
	Chunk      string           `json:"chunk"`
	ToolCall   *InProgressTool  `json:"toolCall,omitempty"`
	Finished   bool             `json:"finished,omitempty"`
}

// InProgressTool 进行中的工具调用
type InProgressTool struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Args   string `json:"args,omitempty"`
}

// ExecutedTool 已执行的工具调用
type ExecutedTool struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Output    string `json:"output,omitempty"`
	Error     string `json:"error,omitempty"`
	Duration  int64  `json:"durationMs,omitempty"`
}

// AgentListParams agents.list 请求参数
type AgentListParams struct {
	IncludeBuiltIn bool `json:"includeBuiltIn,omitempty"`
}

// AgentCreateParams agents.create 请求参数
type AgentCreateParams struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	Config      map[string]interface{} `json:"config,omitempty"`
}

// AgentUpdateParams agents.update 请求参数
type AgentUpdateParams struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name,omitempty"`
	Description string                 `json:"description,omitempty"`
	Config      map[string]interface{} `json:"config,omitempty"`
}

// AgentDeleteParams agents.delete 请求参数
type AgentDeleteParams struct {
	ID string `json:"id"`
}

// AgentListResult agents.list 响应
type AgentListResult struct {
	Agents []AgentSummary `json:"agents"`
}

// AgentSummary Agent 摘要
type AgentSummary struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Model       string `json:"model,omitempty"`
	Status      string `json:"status,omitempty"`
	BuiltIn     bool   `json:"builtIn,omitempty"`
}

// AgentIdentityGetResult agent.identity.get 响应
type AgentIdentityGetResult struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Model    string `json:"model"`
	Provider string `json:"provider"`
}

// ============ 工具相关 ============

// ToolsCatalogParams tools.catalog 请求参数
type ToolsCatalogParams struct {
	IncludeBuiltIn bool `json:"includeBuiltIn,omitempty"`
	Category      string `json:"category,omitempty"`
}

// ToolsCatalogResult tools.catalog 响应
type ToolsCatalogResult struct {
	Tools []ToolSummary `json:"tools"`
}

// ToolSummary 工具摘要
type ToolSummary struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Category    string `json:"category,omitempty"`
	Parameters  any    `json:"parameters,omitempty"`
	BuiltIn     bool   `json:"builtIn,omitempty"`
}

// ============ 通道相关 ============

// ChannelsStatusParams channels.status 请求参数
type ChannelsStatusParams struct {
	Channel string `json:"channel,omitempty"`
}

// ChannelsLogoutParams channels.logout 请求参数
type ChannelsLogoutParams struct {
	Channel   string `json:"channel"`
	AccountID string `json:"accountId,omitempty"`
}

// ChannelsStatusResult channels.status 响应
type ChannelsStatusResult struct {
	Channels []ChannelStatus `json:"channels"`
}

// ChannelStatus 通道状态
type ChannelStatus struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Status       string `json:"status"`
	Connected    bool   `json:"connected"`
	LastActivity int64 `json:"lastActivity,omitempty"`
	Error        string `json:"error,omitempty"`
	Account      string `json:"account,omitempty"`
}

// ============ 订阅/事件相关 ============

// PresenceEvent Presence 事件
type PresenceEvent struct {
	Type      string   `json:"type"`
	SessionID string   `json:"sessionId,omitempty"`
	Status    string   `json:"status"`
	Timestamp int64    `json:"timestamp"`
}

// HeartbeatEvent Heartbeat 事件
type HeartbeatEvent struct {
	SessionID string `json:"sessionId,omitempty"`
	Text      string `json:"text,omitempty"`
	Timestamp int64  `json:"timestamp"`
}

// SystemEventParams system-event 请求参数
type SystemEventParams struct {
	SessionID string `json:"sessionId,omitempty"`
	Text      string `json:"text"`
	Type      string `json:"type,omitempty"`
}

// WakeParams wake 请求参数
type WakeParams struct {
	SessionID string `json:"sessionId,omitempty"`
	Mode      string `json:"mode,omitempty"`
	Text      string `json:"text,omitempty"`
}

// SetHeartbeatsParams set-heartbeats 请求参数
type SetHeartbeatsParams struct {
	Heartbeats []HeartbeatConfig `json:"heartbeats"`
}

// HeartbeatConfig 心跳配置
type HeartbeatConfig struct {
	SessionID  string `json:"sessionId"`
	IntervalMs int    `json:"intervalMs"`
	Text       string `json:"text,omitempty"`
}

// ============ TTS 相关 ============

// TTSStatusParams tts.status 请求参数
type TTSStatusParams struct{}

// TTSProvidersParams tts.providers 请求参数
type TTSProvidersParams struct{}

// TTSEnableParams tts.enable 请求参数
type TTSEnableParams struct {
	Provider string `json:"provider"`
	Config   map[string]interface{} `json:"config,omitempty"`
}

// TTSDisableParams tts.disable 请求参数
type TTSDisableParams struct {
	Provider string `json:"provider"`
}

// TTSConvertParams tts.convert 请求参数
type TTSConvertParams struct {
	Text     string `json:"text"`
	Provider string `json:"provider,omitempty"`
	Voice    string `json:"voice,omitempty"`
	Model    string `json:"model,omitempty"`
}

// TTSStatusResult tts.status 响应
type TTSStatusResult struct {
	Enabled   bool              `json:"enabled"`
	Provider  string            `json:"provider"`
	Voices    []string          `json:"voices,omitempty"`
	Config    map[string]string `json:"config,omitempty"`
}

// TTSProvidersResult tts.providers 响应
type TTSProvidersResult struct {
	Providers []TTSProvider `json:"providers"`
}

// TTSProvider TTS 提供商
type TTSProvider struct {
	ID      string   `json:"id"`
	Name    string   `json:"name"`
	Voices  []string `json:"voices,omitempty"`
	Default bool     `json:"default,omitempty"`
}

// TTSConvertResult tts.convert 响应
type TTSConvertResult struct {
	AudioData string `json:"audioData"` // base64
	MimeType  string `json:"mimeType"`
	Duration  int    `json:"durationMs,omitempty"`
}

// ============ Cron 相关 ============

// CronListParams cron.list 请求参数
type CronListParams struct {
	IncludeDisabled bool `json:"includeDisabled,omitempty"`
}

// CronAddParams cron.add 请求参数
type CronAddParams struct {
	Name          string                 `json:"name"`
	Enabled       bool                   `json:"enabled"`
	Schedule      map[string]interface{} `json:"schedule"`
	SessionTarget string                 `json:"sessionTarget"`
	WakeMode      string                 `json:"wakeMode"`
	Payload       map[string]interface{} `json:"payload"`
	Delivery      map[string]interface{} `json:"delivery,omitempty"`
}

// CronUpdateParams cron.update 请求参数
type CronUpdateParams struct {
	ID            string                 `json:"id"`
	Enabled       *bool                  `json:"enabled,omitempty"`
	Schedule      map[string]interface{} `json:"schedule,omitempty"`
	SessionTarget string                 `json:"sessionTarget,omitempty"`
	WakeMode      string                 `json:"wakeMode,omitempty"`
	Payload       map[string]interface{} `json:"payload,omitempty"`
	Delivery      map[string]interface{} `json:"delivery,omitempty"`
}

// CronRemoveParams cron.remove 请求参数
type CronRemoveParams struct {
	ID string `json:"id"`
}

// CronRunParams cron.run 请求参数
type CronRunParams struct {
	ID   string `json:"id"`
	Mode string `json:"mode,omitempty"` // "due" | "force"
}

// CronStatusParams cron.status 请求参数
type CronStatusParams struct {
	ID string `json:"id"`
}

// CronRunsParams cron.runs 请求参数
type CronRunsParams struct {
	ID     string `json:"id"`
	Limit  int    `json:"limit,omitempty"`
	Offset int    `json:"offset,omitempty"`
}

// CronListResult cron.list 响应
type CronListResult struct {
	Jobs []CronJobSummary `json:"jobs"`
}

// CronJobSummary Cron 任务摘要
type CronJobSummary struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	Enabled         bool   `json:"enabled"`
	Schedule        string `json:"schedule"`
	NextRunAtMs     int64  `json:"nextRunAtMs,omitempty"`
	LastRunAtMs    int64  `json:"lastRunAtMs,omitempty"`
	LastRunStatus  string `json:"lastRunStatus,omitempty"`
	LastRunDuration int64  `json:"lastRunDurationMs,omitempty"`
	RunCount       int    `json:"runCount"`
}

// CronStatusResult cron.status 响应
type CronStatusResult struct {
	Job CronJobDetail `json:"job"`
}

// CronJobDetail Cron 任务详情
type CronJobDetail struct {
	ID              string                 `json:"id"`
	Name            string                 `json:"name"`
	Enabled         bool                   `json:"enabled"`
	Schedule        map[string]interface{} `json:"schedule"`
	SessionTarget   string                 `json:"sessionTarget"`
	WakeMode        string                 `json:"wakeMode"`
	Payload         map[string]interface{} `json:"payload"`
	Delivery        map[string]interface{} `json:"delivery,omitempty"`
	State           map[string]interface{} `json:"state"`
	CreatedAtMs     int64                  `json:"createdAtMs"`
	UpdatedAtMs     int64                  `json:"updatedAtMs"`
}

// CronRunsResult cron.runs 响应
type CronRunsResult struct {
	Runs []CronRunSummary `json:"runs"`
	Total int             `json:"total"`
}

// CronRunSummary Cron 运行记录摘要
type CronRunSummary struct {
	ID          string `json:"id"`
	JobID       string `json:"jobId"`
	StartedAtMs int64 `json:"startedAtMs"`
	EndedAtMs   int64 `json:"endedAtMs,omitempty"`
	Status      string `json:"status"`
	Error       string `json:"error,omitempty"`
	Summary     string `json:"summary,omitempty"`
	SessionID   string `json:"sessionId,omitempty"`
}

// ============ 执行审批相关 ============

// ExecApprovalsGetParams exec.approvals.get 请求参数
type ExecApprovalsGetParams struct {
	NodeID string `json:"nodeId,omitempty"`
}

// ExecApprovalsSetParams exec.approvals.set 请求参数
type ExecApprovalsSetParams struct {
	NodeID   string   `json:"nodeId"`
	Commands []string `json:"commands"`
	Action   string   `json:"action"` // "allow" | "deny"
}

// ExecApprovalRequestParams exec.approval.request 请求参数
type ExecApprovalRequestParams struct {
	NodeID   string `json:"nodeId"`
	Command  string `json:"command"`
	Args     string `json:"args,omitempty"`
	Cwd      string `json:"cwd,omitempty"`
	Timeout  int    `json:"timeout,omitempty"`
}

// ExecApprovalResolveParams exec.approval.resolve 请求参数
type ExecApprovalResolveParams struct {
	RequestID string `json:"requestId"`
	Approved  bool   `json:"approved"`
	Output    string `json:"output,omitempty"`
}

// ExecApprovalsResult exec.approvals 响应
type ExecApprovalsResult struct {
	NodeID     string   `json:"nodeId"`
	Allowed    []string `json:"allowed"`
	Denied     []string `json:"denied"`
	ExactOnly  bool     `json:"exactOnly"`
}

// ExecApprovalRequest 执行审批请求
type ExecApprovalRequest struct {
	ID        string `json:"id"`
	NodeID    string `json:"nodeId"`
	Command   string `json:"command"`
	Args      string `json:"args,omitempty"`
	Cwd       string `json:"cwd,omitempty"`
	Status    string `json:"status"` // "pending" | "approved" | "denied"
	RequestedAt int64 `json:"requestedAt"`
}

// ============ 向导相关 ============

// WizardStartParams wizard.start 请求参数
type WizardStartParams struct {
	Type string `json:"type"`
}

// WizardNextParams wizard.next 请求参数
type WizardNextParams struct {
	Input map[string]interface{} `json:"input"`
}

// WizardCancelParams wizard.cancel 请求参数
type WizardCancelParams struct {
	ID string `json:"id,omitempty"`
}

// WizardStatusParams wizard.status 请求参数
type WizardStatusParams struct {
	ID string `json:"id,omitempty"`
}

// WizardResult wizard 响应
type WizardResult struct {
	ID       string                 `json:"id"`
	Type     string                 `json:"type"`
	Step     string                 `json:"step"`
	Status   string                 `json:"status"` // "in_progress" | "completed" | "cancelled"
	Data     map[string]interface{} `json:"data,omitempty"`
	NextStep string                 `json:"nextStep,omitempty"`
}

// ============ 技能相关 ============

// SkillsStatusParams skills.status 请求参数
type SkillsStatusParams struct {
	IncludeBuiltIn bool `json:"includeBuiltIn,omitempty"`
}

// SkillsInstallParams skills.install 请求参数
type SkillsInstallParams struct {
	Source string `json:"source"` // npm package or git URL
	Force  bool   `json:"force,omitempty"`
}

// SkillsUpdateParams skills.update 请求参数
type SkillsUpdateParams struct {
	Name string `json:"name"`
}

// SkillsBinsParams skills.bins 请求参数
type SkillsBinsParams struct{}

// SkillsStatusResult skills.status 响应
type SkillsStatusResult struct {
	Skills []SkillInfo `json:"skills"`
}

// SkillInfo 技能信息
type SkillInfo struct {
	Name        string `json:"name"`
	Version     string `json:"version"`
	Status      string `json:"status"` // "installed" | "available" | "error"
	Description string `json:"description,omitempty"`
	BuiltIn     bool   `json:"builtIn,omitempty"`
}

// ============ 模型相关 ============

// ModelsListParams models.list 请求参数
type ModelsListParams struct {
	Provider   string `json:"provider,omitempty"`
	IncludeDisabled bool `json:"includeDisabled,omitempty"`
}

// ModelsListResult models.list 响应
type ModelsListResult struct {
	Models []ModelInfo `json:"models"`
}

// ModelInfo 模型信息
type ModelInfo struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Provider    string   `json:"provider"`
	ContextWindow int   `json:"contextWindow,omitempty"`
	MaxOutputTokens int `json:"maxOutputTokens,omitempty"`
	SupportsVision bool `json:"supportsVision,omitempty"`
	SupportsTools  bool `json:"supportsTools,omitempty"`
	Enabled      bool  `json:"enabled,omitempty"`
}

// ============ 使用量相关 ============

// UsageStatusParams usage.status 请求参数
type UsageStatusParams struct {
	StartDate string `json:"startDate,omitempty"`
	EndDate   string `json:"endDate,omitempty"`
}

// UsageCostParams usage.cost 请求参数
type UsageCostParams struct {
	StartDate string `json:"startDate,omitempty"`
	EndDate   string `json:"endDate,omitempty"`
	GroupBy   string `json:"groupBy,omitempty"` // "day" | "model" | "session"
}

// UsageStatusResult usage.status 响应
type UsageStatusResult struct {
	Period      *UsagePeriod `json:"period"`
	TotalTokens int          `json:"totalTokens"`
	InputTokens int          `json:"inputTokens"`
	OutputTokens int         `json:"outputTokens"`
	TotalCost   float64      `json:"totalCost"`
	Breakdown   []UsageEntry `json:"breakdown,omitempty"`
}

// UsageCostResult usage.cost 响应
type UsageCostResult struct {
	TotalCost float64       `json:"totalCost"`
	Breakdown []UsageCostEntry `json:"breakdown,omitempty"`
}

// UsagePeriod 使用量周期
type UsagePeriod struct {
	Start string `json:"start"`
	End   string `json:"end"`
}

// UsageEntry 使用量条目
type UsageEntry struct {
	Model        string `json:"model"`
	InputTokens  int    `json:"inputTokens"`
	OutputTokens int    `json:"outputTokens"`
	TotalTokens  int    `json:"totalTokens"`
	Requests     int    `json:"requests"`
}

// UsageCostEntry 成本条目
type UsageCostEntry struct {
	Date     string  `json:"date,omitempty"`
	Model    string  `json:"model,omitempty"`
	SessionID string `json:"sessionId,omitempty"`
	Cost     float64 `json:"cost"`
}

// UsageSummary 使用量摘要
type UsageSummary struct {
	InputTokens  int     `json:"input_tokens,omitempty"`
	OutputTokens int     `json:"output_tokens,omitempty"`
	TotalTokens  int     `json:"total_tokens,omitempty"`
	CacheRead    int     `json:"cache_read_tokens,omitempty"`
	CacheWrite   int     `json:"cache_write_tokens,omitempty"`
	Cost         float64 `json:"cost,omitempty"`
}

// ============ 日志相关 ============

// LogsTailParams logs.tail 请求参数
type LogsTailParams struct {
	SessionID string `json:"sessionId,omitempty"`
	Level     string `json:"level,omitempty"` // "debug" | "info" | "warn" | "error"
	Limit     int    `json:"limit,omitempty"`
	After     string `json:"after,omitempty"`
}

// LogsTailResult logs.tail 响应
type LogsTailResult struct {
	Entries []LogEntry `json:"entries"`
}

// LogEntry 日志条目
type LogEntry struct {
	ID        string    `json:"id"`
	Timestamp time.Time `json:"timestamp"`
	Level     string    `json:"level"`
	Message   string    `json:"message"`
	SessionID string    `json:"sessionId,omitempty"`
	Source    string    `json:"source,omitempty"`
}

// ============ 设备配对相关 ============

// DevicePairListParams device.pair.list 请求参数
type DevicePairListParams struct{}

// DevicePairApproveParams device.pair.approve 请求参数
type DevicePairApproveParams struct {
	RequestID string `json:"requestId"`
	DeviceName string `json:"deviceName,omitempty"`
}

// DevicePairRejectParams device.pair.reject 请求参数
type DevicePairRejectParams struct {
	RequestID string `json:"requestId"`
	Reason    string `json:"reason,omitempty"`
}

// DevicePairRemoveParams device.pair.remove 请求参数
type DevicePairRemoveParams struct {
	DeviceID string `json:"deviceId"`
}

// DevicePairListResult device.pair.list 响应
type DevicePairListResult struct {
	Devices []DeviceInfo `json:"devices"`
	Pending []PairRequest `json:"pending,omitempty"`
}

// DeviceInfo 设备信息
type DeviceInfo struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Type      string `json:"type"`
	Status    string `json:"status"`
	PairedAt  int64  `json:"pairedAt"`
	LastSeen  int64  `json:"lastSeen,omitempty"`
}

// PairRequest 配对请求
type PairRequest struct {
	ID        string `json:"id"`
	Code      string `json:"code"`
	Status    string `json:"status"` // "pending" | "approved" | "rejected"
	CreatedAt int64  `json:"createdAt"`
}

// ============ 工具调用相关 ============

// ToolCallInfo 工具调用信息
type ToolCallInfo struct {
	ID         string                 `json:"id"`
	Name       string                 `json:"name"`
	Arguments  map[string]any         `json:"arguments"`
}

// ToolResultInfo 工具执行结果
type ToolResultInfo struct {
	ToolCallID string `json:"tool_call_id"`
	Output     string `json:"output"`
	Error      string `json:"error,omitempty"`
}

// ============ 健康检查 ============

// HealthParams health 请求参数
type HealthParams struct{}

// HealthResult health 响应
type HealthResult struct {
	Status    string            `json:"status"` // "ok" | "degraded" | "down"
	Uptime    int64             `json:"uptimeMs"`
	Version   string            `json:"version"`
	Checks    map[string]Check `json:"checks,omitempty"`
}

// Check 健康检查项
type Check struct {
	Status  string `json:"status"` // "ok" | "warn" | "error"
	Message string `json:"message,omitempty"`
}

// DoctorMemoryStatusParams doctor.memory.status 请求参数
type DoctorMemoryStatusParams struct{}

// DoctorMemoryStatusResult doctor.memory.status 响应
type DoctorMemoryStatusResult struct {
	Total     int64 `json:"totalBytes"`
	Used      int64 `json:"usedBytes"`
	SessionCount int `json:"sessionCount"`
	GCEnabled bool  `json:"gcEnabled"`
}
