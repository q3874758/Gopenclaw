package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Config 与 openclaw 兼容的配置结构（最小集，后续按协议扩展）
type Config struct {
	HomeDir    string          `json:"homeDir,omitempty"` // 配置目录
	Extensions string          `json:"extensions,omitempty"` // 扩展目录 (OPENCLAW_EXTENSIONS)
	StateDir   string          `json:"stateDir,omitempty"` // 状态目录 (支持 OPENCLAW_STATE_DIR)
	Agent      AgentConfig     `json:"agent,omitempty"`
	Gateway    GatewayConfig   `json:"gateway,omitempty"`
	Browser    BrowserConfig   `json:"browser,omitempty"` // Browser 配置
	Channels   ChannelsConfig  `json:"channels,omitempty"`
	Hooks      HooksConfig     `json:"hooks,omitempty"`
	TTS        TTSConfig       `json:"tts,omitempty"`
	Skills     SkillsConfig    `json:"skills,omitempty"`
	Talk       TalkConfig      `json:"talk,omitempty"`
	Messages   MessagesConfig  `json:"messages,omitempty"`
	Routing    RoutingConfig   `json:"routing,omitempty"`
	Cron       CronConfig      `json:"cron,omitempty"`
	Telegram   TelegramConfig  `json:"telegram,omitempty"`
	Discord    DiscordConfig   `json:"discord,omitempty"`
	Slack      SlackConfig     `json:"slack,omitempty"`
	WhatsApp   WhatsAppConfig  `json:"whatsapp,omitempty"`
	Feishu     FeishuConfig    `json:"feishu,omitempty"`
	MiniMax    MiniMaxConfig  `json:"minimax,omitempty"`
	Qwen       QwenConfig     `json:"qwen,omitempty"`
	HuggingFace HuggingFaceConfig `json:"huggingface,omitempty"`
	Voyage     VoyageConfig   `json:"voyage,omitempty"`
	Codex      CodexConfig    `json:"codex,omitempty"`
	vLLM       vLLMConfig    `json:"vllm,omitempty"`
	Device     DeviceConfig   `json:"device,omitempty"`
	Mistral    MistralConfig  `json:"mistral,omitempty"`
	Update     UpdateConfig   `json:"update,omitempty"`
	KiloCode   KiloCodeConfig `json:"kilocode,omitempty"`
	VercelAI   VercelAIConfig `json:"vercelai,omitempty"`
	Security   SecurityConfig `json:"security,omitempty"`
}

// HeartbeatConfig 心跳配置
type HeartbeatConfig struct {
	Every        string `json:"every,omitempty"` // 间隔（如 "30m"）
	Target       string `json:"target,omitempty"` // 目标
	To           string `json:"to,omitempty"` // 发送目标
	DirectPolicy string `json:"directPolicy,omitempty"` // DM 投递策略 (allow/deny)
}

type AgentConfig struct {
	Model            string            `json:"model,omitempty"` // 如 "openai/gpt-4o"
	ImageModel       string            `json:"imageModel,omitempty"` // 图像模型
	ModelShorts     map[string]string `json:"modelShorts,omitempty"` // 模型别名 (opus, sonnet, gpt 等)
	Heartbeat       *HeartbeatConfig  `json:"heartbeat,omitempty"` // 心跳配置
	MaxConcurrent   int               `json:"maxConcurrent,omitempty"` // 最大并行数
	IdleTimeoutDays int              `json:"idleTimeoutDays,omitempty"` // 会话空闲过期天数（默认7天）
	DailyResetHour  int              `json:"dailyResetHour,omitempty"` // 每日重置小时 (0-23, -1 禁用)
	ThinkingLevel   string           `json:"thinkingLevel,omitempty"` // thinking level (off/normal/high)
	Context1M       bool             `json:"context1m,omitempty"`   // 1M context beta 支持
	Tools           *AgentToolsConfig `json:"tools,omitempty"` // 工具配置
	IncludeDateTime bool              `json:"includeDateTime,omitempty"` // 是否在 system prompt 中包含当前日期时间
	Subagents       *SubagentConfig   `json:"subagents,omitempty"` // Subagent 配置
	ContextDiagnostics bool          `json:"contextDiagnostics,omitempty"` // 启用 context diagnostics
	MCP             *MCPConfig       `json:"mcp,omitempty"` // Model Context Protocol 配置
	ThinkingDefault string          `json:"thinkingDefault,omitempty"` // thinking 默认值 (adaptive)
}

// MCPConfig Model Context Protocol 配置
type MCPConfig struct {
	Enabled  bool             `json:"enabled,omitempty"`
	Servers  []MCPServerConfig `json:"servers,omitempty"` // MCP 服务器列表
}

// MCPServerConfig MCP 服务器配置
type MCPServerConfig struct {
	Name    string   `json:"name,omitempty"`   // 服务器名称
	Command string   `json:"command,omitempty"` // 启动命令
	Args    []string `json:"args,omitempty"`   // 命令参数
	Env     []string `json:"env,omitempty"`    // 环境变量
}

// SubagentConfig Subagent 配置
type SubagentConfig struct {
	MaxSpawnDepth    int `json:"maxSpawnDepth,omitempty"`    // 最大嵌套深度 (default 2)
	MaxChildrenPerAgent int `json:"maxChildrenPerAgent,omitempty"` // 每个 agent 最大子 agent 数 (default 5)
}

// AgentToolsConfig Agent 工具配置
type AgentToolsConfig struct {
	Allow     []string          `json:"allow,omitempty"`  // 允许的工具列表
	Deny      []string          `json:"deny,omitempty"`  // 拒绝的工具列表
	URLAllowList []string       `json:"urlAllowList,omitempty"` // Web 工具 URL 白名单
}

type GatewayConfig struct {
	Port int    `json:"port,omitempty"` // Gopenclaw 默认 11999
	Bind string `json:"bind,omitempty"` // 默认 "127.0.0.1"
	ControlUI *ControlUIConfig `json:"controlUi,omitempty"` // Control UI 配置
	NoAuth   bool   `json:"noAuth,omitempty"` // 禁用 HTTP 认证
	HSTS     bool   `json:"hsts,omitempty"` // 启用 Strict-Transport-Security
}

// ControlUIConfig Control UI 配置
type ControlUIConfig struct {
	BasePath string `json:"basePath,omitempty"` // 基础路径
}

// BrowserConfig Browser 控制配置
type BrowserConfig struct {
	Target         string   `json:"target,omitempty"`       // sandbox/host/custom
	ExecutablePath string   `json:"executablePath,omitempty"` // Chrome 可执行文件路径
	RemoteURL      string   `json:"remoteURL,omitempty"`     // 远程 CDP URL
	NoSandbox     bool     `json:"noSandbox,omitempty"`    // 禁用沙箱
	UserDataDir   string   `json:"userDataDir,omitempty"`  // 用户数据目录
	Headless      bool     `json:"headless,omitempty"`     // 无头模式
	Width         int      `json:"width,omitempty"`        // 窗口宽度
	Height        int      `json:"height,omitempty"`       // 窗口高度
	RelayBindHost string   `json:"relayBindHost,omitempty"` // 中继绑定地址
	ExtraArgs     []string `json:"extraArgs,omitempty"`    // 额外浏览器参数
}

type ChannelsConfig struct {
	Telegram map[string]interface{} `json:"telegram,omitempty"`
	Discord  map[string]interface{} `json:"discord,omitempty"`
	Slack    map[string]interface{} `json:"slack,omitempty"`
	WhatsApp map[string]interface{} `json:"whatsapp,omitempty"`
	// 其他通道后续按需添加
}

// TelegramConfig Telegram 配置
type TelegramConfig struct {
	Enabled        bool              `json:"enabled"`
	BotToken       string            `json:"botToken"`
	AllowFrom     []string          `json:"allowFrom,omitempty"`      // 允许的用户 ID
	AllowGroups   bool              `json:"allowGroups,omitempty"`     // 允许群组
	RequireMention bool             `json:"requireMention,omitempty"`  // 需要 @ 提及
	Topics        map[string]string `json:"topics,omitempty"`         // topic ID -> agent ID 映射
	PairCommand   bool              `json:"pairCommand,omitempty"`    // 启用 /pair 命令
	DirectPolicy  string            `json:"directPolicy,omitempty"`   // DM 策略: "direct" | "topic"
	Streaming     string            `json:"streaming,omitempty"`      // 流式传输: "partial" | "full" | "off"
}

// DiscordConfig Discord 配置
type DiscordConfig struct {
	Enabled     bool     `json:"enabled"`
	BotToken    string   `json:"botToken"`
	AppID       string   `json:"appId"`
	PublicKey   string   `json:"publicKey"`
	GuildID     string   `json:"guildId"`
	AllowBots   string   `json:"allowBots,omitempty"` // "mentions" 允许机器人提及
	Presence    DiscordPresenceConfig `json:"presence,omitempty"` // 自定义状态
}

// DiscordPresenceConfig Discord 自定义状态配置
type DiscordPresenceConfig struct {
	Status string `json:"status,omitempty"` // "online", "idle", "dnd", "invisible"
	Activity string `json:"activity,omitempty"` // 活动名称
	ActivityType string `json:"activityType,omitempty"` // "playing", "streaming", "listening", "watching"
	URL       string `json:"url,omitempty"` // streaming URL
}

// SlackConfig Slack 配置
type SlackConfig struct {
	Enabled        bool     `json:"enabled"`
	BotToken       string   `json:"botToken"`
	SigningSecret  string   `json:"signingSecret"`
	AppID          string   `json:"appId"`
	TypingReaction string   `json:"typingReaction,omitempty"` // typing indicator 反馈
}

// WhatsAppConfig WhatsApp 配置
type WhatsAppConfig struct {
	Enabled    bool   `json:"enabled"`
	AccountSID string `json:"accountSid"`
	AuthToken  string `json:"authToken"`
	PhoneNumber string `json:"phoneNumber"`
}

// FeishuConfig 飞书配置
type FeishuConfig struct {
	Enabled      bool     `json:"enabled"`
	AppID        string   `json:"appId"`
	AppSecret    string   `json:"appSecret"`
	Verification string   `json:"verification,omitempty"`
	EncryptKey   string   `json:"encryptKey,omitempty"`
	AllowFrom    []string `json:"allowFrom,omitempty"`
}

// MiniMaxConfig MiniMax OAuth 配置
type MiniMaxConfig struct {
	Enabled       bool     `json:"enabled"`
	ClientID      string   `json:"clientId,omitempty"`
	ClientSecret  string   `json:"clientSecret,omitempty"`
	APIKey        string   `json:"apiKey,omitempty"` // 直接 API Key（可选，用于开发）
	GroupID       string   `json:"groupId,omitempty"` // 企业版 group ID
}

// QwenConfig Qwen (通义千问) OAuth 配置
type QwenConfig struct {
	Enabled      bool     `json:"enabled"`
	ClientID     string   `json:"clientId,omitempty"`
	ClientSecret string   `json:"clientSecret,omitempty"`
	APIKey       string   `json:"apiKey,omitempty"` // 直接 API Key（可选）
}

// HuggingFaceConfig Hugging Face 配置
type HuggingFaceConfig struct {
	Enabled  bool   `json:"enabled"`
	APIKey   string `json:"apiKey,omitempty"`
	Endpoint string `json:"endpoint,omitempty"` // 自定义端点
	Model    string `json:"model,omitempty"`   // 默认模型
}

// VoyageConfig Voyage AI 配置（用于 embeddings）
type VoyageConfig struct {
	Enabled  bool   `json:"enabled"`
	APIKey   string `json:"apiKey,omitempty"`
	Endpoint string `json:"endpoint,omitempty"` // 自定义端点
	Model    string `json:"model,omitempty"`     // 默认 embedding 模型
}

// OpenAI Codex 配置
type CodexConfig struct {
	Enabled  bool   `json:"enabled"`
	APIKey   string `json:"apiKey,omitempty"`
	ClientID string `json:"clientId,omitempty"`     // OAuth client ID
}

// vLLMConfig vLLM 配置
type vLLMConfig struct {
	Enabled  bool   `json:"enabled"`
	APIKey   string `json:"apiKey,omitempty"`
	Endpoint string `json:"endpoint,omitempty"` // vLLM 服务端点
	Model    string `json:"model,omitempty"`   // 默认模型
}

// DeviceConfig 设备配置
type DeviceConfig struct {
	PairingEnabled bool `json:"pairingEnabled,omitempty"` // 启用设备配对
}

// MistralConfig Mistral AI 配置
type MistralConfig struct {
	Enabled  bool   `json:"enabled"`
	APIKey   string `json:"apiKey,omitempty"`
	Endpoint string `json:"endpoint,omitempty"` // 自定义端点
	Model    string `json:"model,omitempty"`   // 默认模型
}

// UpdateConfig 自动更新配置
type UpdateConfig struct {
	Enabled  bool   `json:"enabled,omitempty"`
	AutoCheck bool  `json:"autoCheck,omitempty"` // 自动检查更新
	Channel  string `json:"channel,omitempty"`   // 更新通道 (stable/beta)
}

// KiloCodeConfig Kilo Code 配置
type KiloCodeConfig struct {
	Enabled  bool   `json:"enabled"`
	APIKey   string `json:"apiKey,omitempty"`
	Endpoint string `json:"endpoint,omitempty"` // 自定义端点
	Model    string `json:"model,omitempty"`   // 默认模型
}

// VercelAIConfig Vercel AI Gateway 配置
type VercelAIConfig struct {
	Enabled  bool   `json:"enabled"`
	APIKey   string `json:"apiKey,omitempty"`
	Endpoint string `json:"endpoint,omitempty"` // 自定义端点
}

// SecurityConfig 安全配置
type SecurityConfig struct {
	TrustModel *TrustModelConfig `json:"trustModel,omitempty"` // 信任模型配置
}

// TrustModelConfig 信任模型配置
type TrustModelConfig struct {
	MultiUserHeuristic bool `json:"multiUserHeuristic,omitempty"` // 检测共享用户入口
}

type HooksConfig struct {
	Enabled                    bool     `json:"enabled"`
	Token                      string   `json:"token"`
	Path                       string   `json:"path"`
	AllowedAgentIDs             []string `json:"allowedAgentIds"`
	DefaultSessionKey          string   `json:"defaultSessionKey"`
	AllowRequestSessionKey     bool     `json:"allowRequestSessionKey"`
	AllowedSessionKeyPrefixes  []string `json:"allowedSessionKeyPrefixes"`
	LLMInputHook               bool     `json:"llmInputHook,omitempty"`   // 启用 llm_input hook
	LLMOutputHook              bool     `json:"llmOutputHook,omitempty"`  // 启用 llm_output hook
}

// TTSConfig TTS 配置
type TTSConfig struct {
	Enabled         bool             `json:"enabled"`
	DefaultProvider string           `json:"defaultProvider"`
	OpenAI          *TTSOpenAIConfig `json:"openai,omitempty"`
	AWS             *TTSAWSConfig    `json:"aws,omitempty"`
	Azure           *TTSAzureConfig  `json:"azure,omitempty"`
}

// TTSOpenAIConfig OpenAI TTS 配置
type TTSOpenAIConfig struct {
	APIKey  string `json:"apiKey,omitempty"`
	BaseURL string `json:"baseUrl,omitempty"` // 兼容端点支持
	Model   string `json:"model,omitempty"`   // tts-1, tts-1-hd
	Voice   string `json:"voice,omitempty"`   // alloy, echo, fable, onyx, nova, shimmer
}

// TTSAWSConfig AWS Polly 配置
type TTSAWSConfig struct {
	Region    string `json:"region,omitempty"`
	AccessKey string `json:"accessKey,omitempty"`
	SecretKey string `json:"secretKey,omitempty"`
}

// TTSAzureConfig Azure TTS 配置
type TTSAzureConfig struct {
	Key    string `json:"key,omitempty"`
	Region string `json:"region,omitempty"`
}

// SkillsConfig 技能配置
type SkillsConfig struct {
	Enabled      bool     `json:"enabled"`
	BinDir       string   `json:"binDir,omitempty"`
	AutoInstall  []string `json:"autoInstall,omitempty"`
	SessionLogsDir string `json:"sessionLogsDir,omitempty"` // 会话日志目录（从 .clawdbot 迁移到 .gopenclaw）
}

// TalkConfig Talk mode 配置
type TalkConfig struct {
	SilenceTimeoutMs int64 `json:"silenceTimeoutMs,omitempty"` // 静默超时（毫秒）
	VoiceEnabled    bool   `json:"voiceEnabled,omitempty"`    // 是否启用语音
	VoiceProvider   string `json:"voiceProvider,omitempty"`   // 语音提供商
}

// MessagesConfig 消息配置
type MessagesConfig struct {
	ResponsePrefix string            `json:"responsePrefix,omitempty"` // 回复前缀（如 emoji）
	ShowTimestamps bool             `json:"showTimestamps,omitempty"` // 显示时间戳
	TextChunkLimit int              `json:"textChunkLimit,omitempty"` // 文本块限制
	SessionIntro   string          `json:"sessionIntro,omitempty"`   // 系统提示只发一次（可选）
	MessagePrefix  string          `json:"messagePrefix,omitempty"`   // 每条消息的前缀
	HistoryLimit  map[string]int   `json:"historyLimit,omitempty"`  // per-provider/per-account 历史消息限制
	TrustedMessageID bool          `json:"trustedMessageID,omitempty"` // 包含 trusted inbound message_id
	AckReaction   map[string]string `json:"ackReaction,omitempty"` // per-channel ack reaction overrides
	Maintenance  *SessionMaintenanceConfig `json:"maintenance,omitempty"` // Session 维护配置
	TTS          *MessagesTTSConfig `json:"tts,omitempty"` // TTS 配置
}

// MessagesTTSConfig 消息 TTS 配置
type MessagesTTSConfig struct {
	OpenAI *TTSOpenAIConfig `json:"openai,omitempty"` // OpenAI TTS 配置
}

// SessionMaintenanceConfig Session 维护配置
type SessionMaintenanceConfig struct {
	MaxDiskBytes int64 `json:"maxDiskBytes,omitempty"` // 最大磁盘占用（字节）
}

// RoutingConfig 路由配置
type RoutingConfig struct {
	Queue            string            `json:"queue,omitempty"` // "interrupt" | "queue"
	QueuePerSurface  map[string]string `json:"queuePerSurface,omitempty"` // per-surface 覆盖
	GroupChat       *GroupChatConfig `json:"groupChat,omitempty"` // 群组聊天配置
	DMScope         string            `json:"dmScope,omitempty"` // 多用户 DM 隔离范围 (device/user/all)
}

// GroupChatConfig 群组聊天配置
type GroupChatConfig struct {
	RequireMention bool     `json:"requireMention"` // 是否需要 @ 提及
	AllowList     []string `json:"allowList,omitempty"` // 允许的群组列表
}

// CronConfig Cron 配置
type CronConfig struct {
	Enabled       bool     `json:"enabled,omitempty"`
	TimeZone      string   `json:"timeZone,omitempty"`      // 时区
	MaxConcurrent int      `json:"maxConcurrent,omitempty"` // 最大并发任务数
	DeliveryMode  string   `json:"deliveryMode,omitempty"` // 投递模式: "announce" | "none" | "auto"
	WebhookToken  string   `json:"webhookToken,omitempty"` // Webhook 认证 token
	Notify        bool     `json:"notify,omitempty"`       // 完成后是否发送 webhook 通知
	Stagger       int      `json:"stagger,omitempty"`     // 任务交错延迟（秒）
	Exact         bool     `json:"exact,omitempty"`       // 精确时间执行
}

// Default 返回默认配置；Gopenclaw 默认端口 11999，与官方 OpenClaw 默认 18789 区分
func Default() *Config {
	return &Config{
		Gateway: GatewayConfig{
			Port: 11999,
			Bind: "127.0.0.1",
		},
		Agent: AgentConfig{
			Model: "",
			ModelShorts: map[string]string{
				"opus":      "anthropic/claude-opus-4-5-20251105",
				"sonnet":    "anthropic/claude-sonnet-4-5-20251105",
				"gpt":       "openai/gpt-5",
				"gpt-mini":  "openai/gpt-4o-mini",
				"gemini":    "google/gemini-2.0-flash-exp",
				"gemini-flash": "google/gemini-2.0-flash",
			},
		},
	}
}

// OpenClawHome 返回 Gopenclaw 配置目录，与官方 OpenClaw 分离，避免同机冲突：
// 优先 GOPENCLAW_HOME，否则 OPENCLAW_HOME（兼容迁移），否则 ~/.gopenclaw（默认不与 ~/.openclaw 共用）
func OpenClawHome() string {
	if h := os.Getenv("GOPENCLAW_HOME"); h != "" {
		return h
	}
	if h := os.Getenv("OPENCLAW_HOME"); h != "" {
		return h
	}
	dir, _ := os.UserHomeDir()
	return filepath.Join(dir, ".gopenclaw")
}

// ConfigPath 返回 openclaw.json 路径
func ConfigPath() string {
	return filepath.Join(OpenClawHome(), "openclaw.json")
}

// Load 从默认路径加载配置；文件不存在时返回默认配置
func Load() (*Config, error) {
	path := ConfigPath()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return Default(), nil
		}
		return nil, err
	}
	cfg := Default()
	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, err
	}
	// 填充零值
	if cfg.Gateway.Port == 0 {
		cfg.Gateway.Port = 11999
	}
	if cfg.Gateway.Bind == "" {
		cfg.Gateway.Bind = "127.0.0.1"
	}
	// 环境变量覆盖
	loadEnvOverrides(cfg)
	return cfg, nil
}

// loadEnvOverrides 从环境变量加载配置覆盖
func loadEnvOverrides(cfg *Config) {
	// Extensions 目录
	if ext := os.Getenv("OPENCLAW_EXTENSIONS"); ext != "" {
		cfg.Extensions = ext
	}
	// Home 目录
	if home := os.Getenv("GOPENCLAW_HOME"); home != "" {
		cfg.HomeDir = home
	}
	// Agent 配置
	if model := os.Getenv("GOPENCLAW_MODEL"); model != "" {
		cfg.Agent.Model = model
	}
	// API Keys (按需添加更多)
	if apiKey := os.Getenv("OPENAI_API_KEY"); apiKey != "" {
		if cfg.Agent.Tools == nil {
			cfg.Agent.Tools = &AgentToolsConfig{}
		}
	}
	// TTS 配置
	if ttsKey := os.Getenv("OPENAI_API_KEY"); ttsKey != "" {
		if cfg.TTS.OpenAI == nil {
			cfg.TTS.OpenAI = &TTSOpenAIConfig{}
		}
		cfg.TTS.OpenAI.APIKey = ttsKey
	}
}

// Save 将配置写回 openclaw.json
func Save(cfg *Config) error {
	dir := OpenClawHome()
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(ConfigPath(), data, 0600)
}
