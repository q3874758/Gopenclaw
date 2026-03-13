package gateway

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"gopenclaw/internal/agent"
	"gopenclaw/internal/channels"
	"gopenclaw/internal/config"
	"gopenclaw/internal/cron"
	"gopenclaw/internal/protocol"
	"gopenclaw/internal/tools"

	"gopenclaw/internal/discord"
	"gopenclaw/internal/slack"
	"gopenclaw/internal/telegram"
	"gopenclaw/internal/whatsapp"

	"github.com/gorilla/websocket"
)

// uiIndex 从工作目录相对路径加载 ui/index.html（后续可改为 go:embed）
func uiIndex() ([]byte, error) {
	wd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	path := filepath.Join(wd, "ui", "index.html")
	return os.ReadFile(path)
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

// Gateway 网关服务：HTTP + WebSocket，协议路由与会话管理
type Gateway struct {
	cfg        *config.Config
	agent      *agent.Client
	tools      *tools.Registry
	cron       *cron.Scheduler
	channels   *channels.Registry
	server     *http.Server
	mu         sync.RWMutex
	sessions   map[string]protocol.SessionSummary // sessionId -> summary，agent.invoke 时写入
	startTime  time.Time

	// 消息队列
	msgQueue    chan *protocol.Message // 消息队列
	processing  bool                   // 是否正在处理消息
}

// New 根据配置创建 Gateway
func New(cfg *config.Config) *Gateway {
	toolReg := tools.New()
	toolReg.Register(tools.NewEchoExecutor())
	toolReg.Register(tools.NewBashExecutor())
	toolReg.Register(tools.NewReadFileExecutor())
	toolReg.Register(tools.NewWriteFileExecutor())
	toolReg.Register(tools.NewWebFetchExecutor())
	toolReg.Register(tools.NewSessionsListExecutor())
	toolReg.Register(tools.NewSessionsHistoryExecutor())
	toolReg.Register(tools.NewSessionsSendExecutor())
	toolReg.Register(tools.NewListDirectoryExecutor())
	toolReg.Register(tools.NewMakeDirectoryExecutor())

	// 注册系统工具
	tools.RegisterSystemTools(toolReg)

	// 注册 Web Search 工具（带 fallback：Grok > Kimi > DuckDuckGo）
	// 按字母顺序排列，优先尝试 Grok，再尝试 Kimi，最后 DuckDuckGo
	searchProvider := tools.NewSearchWithFallback(
		tools.NewGrokProvider(""),      // 优先 Grok (需要 xAI_API_KEY)
		tools.NewKimiProvider(""),       // 其次 Kimi (需要 MOONSHOT_API_KEY)
		tools.NewDuckDuckGoProvider(),   // 最后 DuckDuckGo (免费，无需 API Key)
	)
	toolReg.Register(tools.NewSearchExecutor(searchProvider))

	// 注册 Canvas 工具
	toolReg.Register(tools.NewCanvasExecutor())

	g := &Gateway{
		cfg:        cfg,
		agent:      agent.New(cfg),
		tools:      toolReg,
		sessions:   make(map[string]protocol.SessionSummary),
		startTime:  time.Now(),
		channels:   channels.New(),
	}

	// 初始化通道适配器
	g.initChannels(cfg)

	// 初始化 cron 调度器
	cronScheduler, err := cron.New(cron.Options{
		StorePath: filepath.Join(cfg.HomeDir, "cron.json"),
		TimeZone:  cfg.Cron.TimeZone,
		Handler:   g.handleCronJob,
	})
	if err != nil {
		slog.Error("failed to create cron scheduler", "err", err)
	} else {
		g.cron = cronScheduler
	}

	// 将工具注册到 agent
	for _, def := range toolReg.GetToolDefinitions() {
		if fn, ok := def["function"].(map[string]any); ok {
			name, _ := fn["name"].(string)
			desc, _ := fn["description"].(string)
			params, _ := fn["parameters"].(map[string]any)
			g.agent.RegisterTool(agent.ToolDefinition{
				Type: "function",
				Function: struct {
					Name        string `json:"name"`
					Description string `json:"description"`
					Parameters  any    `json:"parameters"`
				}{
					Name:        name,
					Description: desc,
					Parameters:  params,
				},
			})
		}
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/", g.handleIndex)
	mux.HandleFunc("/ws", g.handleWebSocket)
	mux.HandleFunc("/health", g.handleHealth)
	g.server = &http.Server{
		Addr:    listenAddr(cfg),
		Handler: mux,
	}
	return g
}

func listenAddr(cfg *config.Config) string {
	port := cfg.Gateway.Port
	if port == 0 {
		port = 11999
	}
	bind := cfg.Gateway.Bind
	if bind == "" || bind == "loopback" {
		bind = "127.0.0.1"
	}
	return fmtAddr(bind, port)
}

func fmtAddr(host string, port int) string {
	return fmt.Sprintf("%s:%d", host, port)
}

// Start 阻塞运行，直到 ctx 取消
func (g *Gateway) Start(ctx context.Context) error {
	// 启动 cron 调度器
	if g.cron != nil {
		g.cron.Start()
	}

	go func() {
		<-ctx.Done()
		_ = g.server.Shutdown(context.Background())
		// 停止 cron 调度器
		if g.cron != nil {
			g.cron.Stop()
		}
	}()
	slog.Info("gateway listening", "addr", g.server.Addr)
	return g.server.ListenAndServe()
}

// ServeListener 在给定 listener 上服务，用于测试（可获知实际端口）
func (g *Gateway) ServeListener(ctx context.Context, ln net.Listener) error {
	go func() {
		<-ctx.Done()
		_ = g.server.Shutdown(context.Background())
	}()
	slog.Info("gateway listening", "addr", ln.Addr().String())
	return g.server.Serve(ln)
}

func (g *Gateway) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(`{"ok":true}`))
}

func (g *Gateway) handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	data, err := uiIndex()
	if err != nil {
		http.Error(w, "ui not found", http.StatusInternalServerError)
		return
	}
	_, _ = w.Write(data)
}

func (g *Gateway) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		slog.Warn("ws upgrade failed", "err", err)
		return
	}
	defer conn.Close()
	g.runConn(conn)
}

func (g *Gateway) runConn(conn *websocket.Conn) {
	for {
		_, data, err := conn.ReadMessage()
		if err != nil {
			break
		}
		var msg protocol.Message
		if err := json.Unmarshal(data, &msg); err != nil {
			_ = g.sendError(conn, nil, -32700, "Parse error")
			continue
		}

		// 获取路由配置
		queueMode := g.cfg.Routing.Queue

		// 消息队列模式处理
		if queueMode == "queue" {
			g.mu.Lock()
			if g.processing {
				// 正在处理，将消息放入队列
				if g.msgQueue == nil {
					g.msgQueue = make(chan *protocol.Message, 100)
				}
				g.mu.Unlock()
				select {
				case g.msgQueue <- &msg:
					// 消息已入队
					_ = conn.WriteJSON(&protocol.Message{
						ID:     msg.ID,
						Result: map[string]interface{}{"status": "queued"},
					})
					continue
				default:
					// 队列已满
					g.mu.Unlock()
					_ = g.sendError(conn, msg.ID, -32000, "Queue full")
					continue
				}
			}
			g.processing = true
			g.mu.Unlock()

			// 处理消息
			resp := g.dispatch(conn, &msg)
			if resp != nil {
				_ = conn.WriteJSON(resp)
			}

			// 处理队列中的下一条消息
			g.processNextInQueue(conn)
		} else {
			// interrupt 模式：立即处理
			resp := g.dispatch(conn, &msg)
			if resp != nil {
				_ = conn.WriteJSON(resp)
			}
		}
	}
}

// processNextInQueue 处理队列中的下一条消息
func (g *Gateway) processNextInQueue(conn *websocket.Conn) {
	g.mu.Lock()
	queue := g.msgQueue
	g.mu.Unlock()

	if queue == nil {
		g.mu.Lock()
		g.processing = false
		g.mu.Unlock()
		return
	}

	select {
	case msg, ok := <-queue:
		if !ok {
			g.mu.Lock()
			g.processing = false
			g.mu.Unlock()
			return
		}
		// 处理队列中的消息
		resp := g.dispatch(conn, msg)
		if resp != nil {
			_ = conn.WriteJSON(resp)
		}
		// 递归处理下一条
		g.processNextInQueue(conn)
	default:
		// 队列为空，重置状态
		g.mu.Lock()
		g.processing = false
		g.mu.Unlock()
	}
}

func (g *Gateway) handleAgentInvoke(conn *websocket.Conn, msg *protocol.Message) *protocol.Message {
	var params protocol.AgentInvokeParams
	if msg.Params != nil {
		raw, _ := json.Marshal(msg.Params)
		_ = json.Unmarshal(raw, &params)
	}
	if params.Message == "" {
		return &protocol.Message{
			ID: msg.ID,
			Error: &protocol.RPCError{
				Code:    -32602,
				Message: "params.message required",
			},
		}
	}
	sid := params.SessionID
	if sid == "" {
		sid = "main"
	}
	g.mu.Lock()
	g.sessions[sid] = protocol.SessionSummary{ID: sid, Label: sid}
	g.mu.Unlock()

	// 处理工具调用循环
	messages := []agent.Message{
		{Role: "user", Content: params.Message},
	}

	maxIterations := 5
	for iter := 0; iter < maxIterations; iter++ {
		// 调用 LLM（带工具）
		toolCalls, text, err := g.agent.InvokeWithTools(context.Background(), messages, params.Stream, func(chunk string) error {
			if params.Stream {
				return conn.WriteJSON(&protocol.Message{
					ID:     msg.ID,
					Result: protocol.AgentStreamChunk{Chunk: chunk},
				})
			}
			return nil
		})
		if err != nil {
			return &protocol.Message{
				ID: msg.ID,
				Error: &protocol.RPCError{
					Code:    -32000,
					Message: err.Error(),
				},
			}
		}

		// 累加文本
		if text != "" {
			messages = append(messages, agent.Message{Role: "assistant", Content: text})
			if !params.Stream {
				// 非流式直接返回
				return &protocol.Message{
					ID:     msg.ID,
					Result: protocol.AgentInvokeResult{Text: text},
				}
			}
		}

		// 如果没有工具调用，结束循环
		if len(toolCalls) == 0 {
			if text != "" {
				_ = conn.WriteJSON(&protocol.Message{
					ID:     msg.ID,
					Result: protocol.AgentInvokeResult{Text: text},
				})
			}
			return nil
		}

		// 执行工具调用
		for _, tc := range toolCalls {
			toolResult, err := g.tools.Execute(context.Background(), tc.Name, tc.Arguments)
			if err != nil {
				toolResult = fmt.Sprintf("Error: %v", err)
			}
			// 发送工具结果给客户端（可选）
			_ = conn.WriteJSON(&protocol.Message{
				ID: msg.ID,
				Result: protocol.ToolResultInfo{
					ToolCallID: tc.ID,
					Output:     toolResult,
				},
			})
			// 将工具结果添加到消息历史
			messages = append(messages, agent.Message{
				Role:      "tool",
				ToolCallID: tc.ID,
				Content:   toolResult,
			})
		}
	}

	// 达到最大迭代次数
	return &protocol.Message{
		ID: msg.ID,
		Error: &protocol.RPCError{
			Code:    -32001,
			Message: "max tool iterations exceeded",
		},
	}
}

func (g *Gateway) methodNotImplemented(msg *protocol.Message) *protocol.Message {
	return &protocol.Message{
		ID: msg.ID,
		Error: &protocol.RPCError{
			Code:    -32601,
			Message: "Method not implemented: " + msg.Method,
		},
	}
}

func (g *Gateway) sendError(conn *websocket.Conn, id interface{}, code int, message string) error {
	return conn.WriteJSON(&protocol.Message{
		ID: id,
		Error: &protocol.RPCError{
			Code:    code,
			Message: message,
		},
	})
}

// initChannels 初始化通道适配器
func (g *Gateway) initChannels(cfg *config.Config) {
	// Telegram
	if cfg.Telegram.Enabled {
		telegramCfg := &telegram.Config{
			Token:          cfg.Telegram.BotToken,
			WebhookURL:     cfg.Telegram.BotToken,
			AllowFrom:      cfg.Telegram.AllowFrom,
			AllowGroups:    nil, // 需要转换为 []string
			RequireMention: cfg.Telegram.RequireMention,
		}
		if adapter := telegram.New(telegramCfg); adapter != nil {
			adapter.OnMessage(func(msg channels.InboundMessage) error {
				slog.Info("telegram message received", "from", msg.Source.From, "content", msg.Content)
				return nil
			})
			g.channels.Register(adapter)
			slog.Info("telegram adapter registered")
		}
	}

	// Discord
	if cfg.Discord.Enabled {
		discordCfg := &discord.Config{
			Token:        cfg.Discord.BotToken,
			ApplicationID: cfg.Discord.AppID,
			PublicKey:    cfg.Discord.PublicKey,
			WebhookURL:   "",
		}
		if adapter := discord.New(discordCfg); adapter != nil {
			adapter.OnMessage(func(msg channels.InboundMessage) error {
				slog.Info("discord message received", "from", msg.Source.From, "content", msg.Content)
				return nil
			})
			g.channels.Register(adapter)
			slog.Info("discord adapter registered")
		}
	}

	// Slack
	if cfg.Slack.Enabled {
		slackCfg := &slack.Config{
			Token:          cfg.Slack.BotToken,
			SigningSecret:  cfg.Slack.SigningSecret,
			AppToken:       "",
			WebhookURL:     "",
		}
		if adapter := slack.New(slackCfg); adapter != nil {
			adapter.OnMessage(func(msg channels.InboundMessage) error {
				slog.Info("slack message received", "from", msg.Source.From, "content", msg.Content)
				return nil
			})
			g.channels.Register(adapter)
			slog.Info("slack adapter registered")
		}
	}

	// WhatsApp
	if cfg.WhatsApp.Enabled {
		whatsappCfg := &whatsapp.Config{
			AccountSID:  cfg.WhatsApp.AccountSID,
			AuthToken:   cfg.WhatsApp.AuthToken,
			PhoneNumber: cfg.WhatsApp.PhoneNumber,
		}
		if adapter := whatsapp.New(whatsappCfg); adapter != nil {
			adapter.OnMessage(func(msg channels.InboundMessage) error {
				slog.Info("whatsapp message received", "from", msg.Source.From, "content", msg.Content)
				return nil
			})
			g.channels.Register(adapter)
			slog.Info("whatsapp adapter registered")
		}
	}
}

// handleCronJob 处理 cron 任务执行
func (g *Gateway) handleCronJob(ctx context.Context, job *cron.CronJob) error {
	slog.Info("handling cron job", "id", job.ID, "name", job.Name, "payload", job.Payload.Kind)

	// 检查投递模式
	if job.Delivery != nil && job.Delivery.Mode == cron.DeliveryModeAnnounce {
		// 纯文本 announce，直接通过通道发送
		if job.Payload.Text != "" && job.Payload.Channel != "" {
			channelID := channels.ChannelID(job.Payload.Channel)
			adapter, ok := g.channels.Get(channelID)
			if ok && adapter != nil {
				msg := channels.OutboundMessage{
					To:      job.Payload.To,
					Content: job.Payload.Text,
				}
				if err := adapter.Send(ctx, msg); err != nil {
					slog.Error("cron announce send failed", "channel", job.Payload.Channel, "err", err)
					return err
				}
				slog.Info("cron announce sent", "channel", job.Payload.Channel, "to", job.Payload.To)
			}
		}
	} else if job.Delivery != nil && job.Delivery.Mode == cron.DeliveryModeWebhook {
		// webhook 投递模式
		// TODO: 实现 webhook 投递
	}

	// 处理其他类型的 payload
	switch job.Payload.Kind {
	case cron.PayloadKindAgentTurn:
		// agentTurn: 发送消息到 agent
		if job.Payload.Message != "" {
			// 查找或创建会话
			sessionID := job.Payload.Channel
			if sessionID == "" {
				sessionID = "cron-job-" + job.ID
			}
			// 调用 agent 处理
			slog.Info("cron agent turn", "session", sessionID, "message", job.Payload.Message)
		}

	case cron.PayloadKindSystemEvent:
		// systemEvent: 发送系统事件
		if job.Payload.Text != "" {
			// 发送 system-event
			slog.Info("cron system event", "text", job.Payload.Text)
		}
	}

	return nil
}
