package cron

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// ScheduleType cron 调度类型
type ScheduleType string

const (
	ScheduleTypeAt    ScheduleType = "at"
	ScheduleTypeEvery ScheduleType = "every"
	ScheduleTypeCron  ScheduleType = "cron"
)

// SessionTarget 会话目标
type SessionTarget string

const (
	SessionTargetMain     SessionTarget = "main"
	SessionTargetIsolated SessionTarget = "isolated"
)

// WakeMode 唤醒模式
type WakeMode string

const (
	WakeModeNextHeartbeat WakeMode = "next-heartbeat"
	WakeModeNow           WakeMode = "now"
)

// DeliveryMode 投递模式
type DeliveryMode string

const (
	DeliveryModeNone    DeliveryMode = "none"
	DeliveryModeAnnounce DeliveryMode = "announce"
	DeliveryModeWebhook  DeliveryMode = "webhook"
)

// PayloadKind payload 类型
type PayloadKind string

const (
	PayloadKindSystemEvent PayloadKind = "systemEvent"
	PayloadKindAgentTurn   PayloadKind = "agentTurn"
)

// Schedule 调度配置
type Schedule struct {
	Kind      ScheduleType `json:"kind"` // "at" | "every" | "cron"
	At        string       `json:"at,omitempty"`
	EveryMs   int64        `json:"everyMs,omitempty"`
	Expr      string       `json:"expr,omitempty"`      // cron 表达式
	Tz        string       `json:"tz,omitempty"`       // 时区
	StaggerMs int64        `json:"staggerMs,omitempty"` // 随机偏移窗口
}

// Payload 任务 payload
type Payload struct {
	Kind                     PayloadKind `json:"kind"` // "systemEvent" | "agentTurn"
	Text                     string      `json:"text,omitempty"`          // systemEvent 时使用
	Message                  string      `json:"message,omitempty"`       // agentTurn 时使用
	Model                    string      `json:"model,omitempty"`         // 模型覆盖
	Thinking                 string      `json:"thinking,omitempty"`       // thinking 级别
	TimeoutSeconds           int         `json:"timeoutSeconds,omitempty"` // 超时秒数
	AllowUnsafeExternalContent bool      `json:"allowUnsafeExternalContent,omitempty"`
	LightContext             bool        `json:"lightContext,omitempty"`
	Deliver                  *bool       `json:"deliver,omitempty"`
	Channel                  string      `json:"channel,omitempty"` // 投递频道
	To                       string      `json:"to,omitempty"`     // 投递目标
}

// Delivery 投递配置
type Delivery struct {
	Mode          DeliveryMode `json:"mode"` // "none" | "announce" | "webhook"
	Channel       string       `json:"channel,omitempty"`
	To            string       `json:"to,omitempty"`
	AccountID     string       `json:"accountId,omitempty"`
	BestEffort    bool         `json:"bestEffort,omitempty"`
	FailureDestination *FailureDestination `json:"failureDestination,omitempty"`
}

// FailureDestination 失败通知目标
type FailureDestination struct {
	Channel  string `json:"channel,omitempty"`
	To       string `json:"to,omitempty"`
	AccountID string `json:"accountId,omitempty"`
	Mode     string `json:"mode,omitempty"` // "announce" | "webhook"
}

// JobState 任务状态
type JobState struct {
	NextRunAtMs          *int64   `json:"nextRunAtMs,omitempty"`
	RunningAtMs          *int64   `json:"runningAtMs,omitempty"`
	LastRunAtMs          *int64   `json:"lastRunAtMs,omitempty"`
	LastRunStatus        string   `json:"lastRunStatus,omitempty"` // "ok" | "error" | "skipped"
	LastError            string   `json:"lastError,omitempty"`
	LastDurationMs       int64    `json:"lastDurationMs,omitempty"`
	ConsecutiveErrors    int      `json:"consecutiveErrors,omitempty"`
	LastFailureAlertAtMs *int64   `json:"lastFailureAlertAtMs,omitempty"`
	ScheduleErrorCount   int      `json:"scheduleErrorCount,omitempty"`
	LastDeliveryStatus   string   `json:"lastDeliveryStatus,omitempty"` // "delivered" | "not-delivered" | "unknown" | "not-requested"
	LastDeliveryError    string   `json:"lastDeliveryError,omitempty"`
	LastDelivered        bool     `json:"lastDelivered,omitempty"`
}

// CronJob cron 任务
type CronJob struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Enabled     bool                   `json:"enabled"`
	Schedule    Schedule               `json:"schedule"`
	SessionTarget SessionTarget        `json:"sessionTarget"` // "main" | "isolated"
	WakeMode    WakeMode               `json:"wakeMode"`      // "next-heartbeat" | "now"
	Payload     Payload                `json:"payload"`
	Delivery    *Delivery              `json:"delivery,omitempty"`
	FailureAlert *FailureAlert        `json:"failureAlert,omitempty"`
	CreatedAtMs int64                 `json:"createdAtMs"`
	UpdatedAtMs int64                 `json:"updatedAtMs"`
	State       JobState               `json:"state"`
}

// FailureAlert 失败告警
type FailureAlert struct {
	After     *int64  `json:"after,omitempty"`
	Channel   string  `json:"channel,omitempty"`
	To        string  `json:"to,omitempty"`
	CooldownMs *int64 `json:"cooldownMs,omitempty"`
	Mode      string  `json:"mode,omitempty"` // "announce" | "webhook"
	AccountID string  `json:"accountId,omitempty"`
}

// RunHandler 任务执行处理器
type RunHandler func(ctx context.Context, job *CronJob) error

// Scheduler cron 调度器
type Scheduler struct {
	mu         sync.RWMutex
	jobs       map[string]*CronJob
	storePath  string
	ctx        context.Context
	cancel     context.CancelFunc
	wg         sync.WaitGroup
	running    bool
	handler    RunHandler
	tzLocation *time.Location
}

// Options 调度器选项
type Options struct {
	StorePath string
	TimeZone  string
	Handler   RunHandler
}

// New 创建调度器
func New(opts Options) (*Scheduler, error) {
	ctx, cancel := context.WithCancel(context.Background())

	// 解析时区
	loc := time.Local
	if opts.TimeZone != "" {
		var err error
		loc, err = time.LoadLocation(opts.TimeZone)
		if err != nil {
			slog.Warn("invalid timezone, using local", "tz", opts.TimeZone, "err", err)
			loc = time.Local
		}
	}

	s := &Scheduler{
		jobs:       make(map[string]*CronJob),
		storePath:  opts.StorePath,
		ctx:        ctx,
		cancel:     cancel,
		handler:    opts.Handler,
		tzLocation: loc,
	}

	// 确保存储目录存在
	if s.storePath != "" {
		dir := filepath.Dir(s.storePath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("create store dir: %w", err)
		}
		// 加载已有任务
		if err := s.load(); err != nil {
			slog.Warn("load cron jobs failed", "err", err)
		}
	}

	return s, nil
}

// Start 启动调度器
func (s *Scheduler) Start() {
	s.mu.Lock()
	s.running = true
	s.mu.Unlock()

	s.wg.Add(1)
	go s.runLoop()
	slog.Info("cron scheduler started", "jobs", len(s.jobs))
}

// Stop 停止调度器
func (s *Scheduler) Stop() {
	s.cancel()
	s.wg.Wait()
	s.mu.Lock()
	s.running = false
	s.mu.Unlock()
	slog.Info("cron scheduler stopped")
}

// Add 添加任务
func (s *Scheduler) Add(job *CronJob) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 设置时间
	now := time.Now().UnixMilli()
	if job.CreatedAtMs == 0 {
		job.CreatedAtMs = now
	}
	job.UpdatedAtMs = now

	// 计算下次运行时间
	if job.Enabled {
		next, err := s.computeNextRun(job)
		if err != nil {
			return err
		}
		job.State.NextRunAtMs = &next
	}

	s.jobs[job.ID] = job

	// 保存到磁盘
	go s.save()

	return nil
}

// Update 更新任务
func (s *Scheduler) Update(id string, patch func(*CronJob) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	job, ok := s.jobs[id]
	if !ok {
		return fmt.Errorf("job %q not found", id)
	}

	if err := patch(job); err != nil {
		return err
	}

	job.UpdatedAtMs = time.Now().UnixMilli()

	// 重新计算下次运行时间
	if job.Enabled {
		next, err := s.computeNextRun(job)
		if err != nil {
			return err
		}
		job.State.NextRunAtMs = &next
	} else {
		job.State.NextRunAtMs = nil
	}

	go s.save()
	return nil
}

// Remove 删除任务
func (s *Scheduler) Remove(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.jobs[id]; !ok {
		return fmt.Errorf("job %q not found", id)
	}

	delete(s.jobs, id)
	go s.save()
	return nil
}

// List 列出所有任务
func (s *Scheduler) List() []CronJob {
	s.mu.RLock()
	defer s.mu.RUnlock()

	list := make([]CronJob, 0, len(s.jobs))
	for _, j := range s.jobs {
		list = append(list, *j)
	}
	return list
}

// Get 获取任务
func (s *Scheduler) Get(id string) (*CronJob, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	j, ok := s.jobs[id]
	if ok {
		return j, true
	}
	return nil, false
}

// Run 手动运行任务
func (s *Scheduler) Run(id string, mode string) error {
	s.mu.RLock()
	job, ok := s.jobs[id]
	s.mu.RUnlock()

	if !ok {
		return fmt.Errorf("job %q not found", id)
	}

	if s.handler != nil {
		return s.handler(s.ctx, job)
	}
	return nil
}

// computeNextRun 计算下次运行时间
func (s *Scheduler) computeNextRun(job *CronJob) (int64, error) {
	now := time.Now().In(s.tzLocation)

	switch job.Schedule.Kind {
	case ScheduleTypeAt:
		t, err := time.ParseInLocation(time.RFC3339, job.Schedule.At, s.tzLocation)
		if err != nil {
			return 0, fmt.Errorf("parse at: %w", err)
		}
		return t.UnixMilli(), nil

	case ScheduleTypeEvery:
		// 从 createdAtMs 或 now 开始
		start := job.CreatedAtMs
		if start == 0 {
			start = time.Now().UnixMilli()
		}
		// 计算下一个周期点
		elapsed := time.Now().UnixMilli() - start
		period := job.Schedule.EveryMs
		next := start + ((elapsed/period + 1) * period)
		return next, nil

	case ScheduleTypeCron:
		return computeCronNext(job.Schedule.Expr, now, s.tzLocation)

	default:
		return 0, fmt.Errorf("unknown schedule kind: %s", job.Schedule.Kind)
	}
}

// runLoop 主循环
func (s *Scheduler) runLoop() {
	defer s.wg.Done()

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			s.checkAndRun()
		}
	}
}

// checkAndRun 检查并执行到期的任务
func (s *Scheduler) checkAndRun() {
	nowMs := time.Now().UnixMilli()

	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, job := range s.jobs {
		if !job.Enabled {
			continue
		}

		nextRun := job.State.NextRunAtMs
		if nextRun == nil {
			continue
		}

		if nowMs >= *nextRun {
			// 标记为运行中
			now := nowMs
			job.State.RunningAtMs = &now

			slog.Info("running cron job", "id", job.ID, "name", job.Name)

			// 异步执行
			go func(j *CronJob) {
				start := time.Now().UnixMilli()

				var err error
				if s.handler != nil {
					err = s.handler(s.ctx, j)
				}

				end := time.Now().UnixMilli()
				duration := end - start

				// 更新状态
				s.mu.Lock()
				j.State.LastRunAtMs = &end
				j.State.RunningAtMs = nil
				j.State.LastDurationMs = duration

				if err != nil {
					j.State.LastRunStatus = "error"
					j.State.LastError = err.Error()
					j.State.ConsecutiveErrors++
				} else {
					j.State.LastRunStatus = "ok"
					j.State.LastError = ""
					j.State.ConsecutiveErrors = 0
				}

				// 计算下次运行时间
				if j.Enabled {
					next, err := s.computeNextRun(j)
					if err != nil {
						j.State.ScheduleErrorCount++
						slog.Error("compute next run failed", "id", j.ID, "err", err)
					} else {
						j.State.NextRunAtMs = &next
						j.State.ScheduleErrorCount = 0
					}
				}
				s.mu.Unlock()

				go s.save()
			}(job)
		}
	}
}

// load 从磁盘加载任务
func (s *Scheduler) load() error {
	if s.storePath == "" {
		return nil
	}

	data, err := os.ReadFile(s.storePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	var store struct {
		Version int       `json:"version"`
		Jobs    []CronJob `json:"jobs"`
	}

	if err := json.Unmarshal(data, &store); err != nil {
		return err
	}

	for i := range store.Jobs {
		s.jobs[store.Jobs[i].ID] = &store.Jobs[i]
	}

	return nil
}

// save 保存任务到磁盘
func (s *Scheduler) save() {
	if s.storePath == "" {
		return
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	store := struct {
		Version int       `json:"version"`
		Jobs    []CronJob `json:"jobs"`
	}{
		Version: 1,
		Jobs:    make([]CronJob, 0, len(s.jobs)),
	}

	for _, j := range s.jobs {
		store.Jobs = append(store.Jobs, *j)
	}

	data, err := json.MarshalIndent(store, "", "  ")
	if err != nil {
		slog.Error("marshal cron store failed", "err", err)
		return
	}

	if err := os.WriteFile(s.storePath, data, 0644); err != nil {
		slog.Error("write cron store failed", "err", err)
	}
}

// computeCronNext 计算 cron 表达式的下次运行时间（简化版）
// 完整实现应使用 robfig/cron 库
func computeCronNext(expr string, now time.Time, loc *time.Location) (int64, error) {
	// 简化实现：支持标准 5 段 cron
	// 格式: 分 时 日 月 周
	// 完整实现需要解析并计算下一个匹配时间点
	
	// 这里先用占位实现，返回 1 分钟后
	next := now.Add(1 * time.Minute)
	return next.UnixMilli(), nil
}
