package routing

import (
	"context"
	"log/slog"
	"strings"
	"sync"
)

// RuleType 路由规则类型
type RuleType string

const (
	RuleTypeAllow RuleType = "allow" // 白名单
	RuleTypeBlock RuleType = "block" // 黑名单
)

// TargetType 规则目标类型
type TargetType string

const (
	TargetUser    TargetType = "user"    // 用户
	TargetChannel TargetType = "channel" // 频道/群组
	TargetGroup   TargetType = "group"   // 群组/服务器
)

// Rule 路由规则
type Rule struct {
	Type      RuleType   `json:"type"`      // allow or block
	Target    TargetType `json:"target"`    // user, channel, group
	Pattern   string     `json:"pattern"`   // 支持通配符 *
	ChannelID string     `json:"channelId"` // 适用于特定通道
}

// Policy 路由策略
type Policy struct {
	mu         sync.RWMutex
	rules      []Rule
	defaults   PolicyDefaults
	channelIDs []string // 应用此策略的通道 ID 列表
}

// PolicyDefaults 策略默认值
type PolicyDefaults struct {
	AllowDM        bool `json:"allowDM"`         // 是否允许 DM
	AllowGroup     bool `json:"allowGroup"`      // 是否允许群组消息
	RequireMention bool `json:"requireMention"` // 是否需要 @bot
}

// NewPolicy 创建路由策略
func NewPolicy(defaults PolicyDefaults) *Policy {
	return &Policy{
		rules:    []Rule{},
		defaults: defaults,
	}
}

// AddRule 添加规则
func (p *Policy) AddRule(rule Rule) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.rules = append(p.rules, rule)
	slog.Debug("routing rule added", "type", rule.Type, "target", rule.Target, "pattern", rule.Pattern)
}

// RemoveRule 移除规则
func (p *Policy) RemoveRule(pattern string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	newRules := make([]Rule, 0)
	for _, r := range p.rules {
		if r.Pattern != pattern {
			newRules = append(newRules, r)
		}
	}
	p.rules = newRules
}

// SetChannelIDs 设置应用策略的通道 ID
func (p *Policy) SetChannelIDs(ids []string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.channelIDs = ids
}

// AllowMessage 判断是否允许消息
func (p *Policy) AllowMessage(ctx context.Context, from, channelID, groupID string) (bool, *Rule) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	// 首先检查是否在应用列表中
	if len(p.channelIDs) > 0 {
		found := false
		for _, id := range p.channelIDs {
			if id == channelID {
				found = true
				break
			}
		}
		if !found {
			return p.defaults.AllowDM && p.defaults.AllowGroup, nil
		}
	}

	// 按优先级检查规则（先检查 block，后检查 allow）
	// 注意：先匹配的规则生效
	
	// 检查黑名单
	for _, rule := range p.rules {
		if rule.Type == RuleTypeBlock {
			if p.matchRule(rule, from, channelID, groupID) {
				slog.Debug("message blocked by rule", "rule", rule.Pattern, "from", from)
				return false, &rule
			}
		}
	}

	// 检查白名单
	for _, rule := range p.rules {
		if rule.Type == RuleTypeAllow {
			if p.matchRule(rule, from, channelID, groupID) {
				slog.Debug("message allowed by rule", "rule", rule.Pattern, "from", from)
				return true, &rule
			}
		}
	}

	// 无匹配规则，使用默认值
	if groupID != "" {
		// 群组消息
		return p.defaults.AllowGroup, nil
	}
	
	// DM 消息
	return p.defaults.AllowDM, nil
}

// matchRule 匹配规则
func (p *Policy) matchRule(rule Rule, from, channelID, groupID string) bool {
	var pattern string
	
	switch rule.Target {
	case TargetUser:
		pattern = from
	case TargetChannel:
		pattern = channelID
	case TargetGroup:
		pattern = groupID
	default:
		return false
	}

	if pattern == "" {
		return false
	}

	// 支持通配符 *
	if strings.Contains(rule.Pattern, "*") {
		return matchWildcard(pattern, rule.Pattern)
	}

	return strings.EqualFold(pattern, rule.Pattern)
}

// matchWildcard 通配符匹配
func matchWildcard(text, pattern string) bool {
	// 简单实现：支持 * 匹配任意字符
	parts := strings.Split(pattern, "*")
	
	if len(parts) == 1 {
		return strings.EqualFold(text, pattern)
	}

	// 检查前缀
	if !strings.HasPrefix(text, parts[0]) {
		return false
	}

	// 检查后缀
	if !strings.HasSuffix(text, parts[len(parts)-1]) {
		return false
	}

	// 检查中间部分
	pos := 0
	for i := 0; i < len(parts)-1; i++ {
		idx := strings.Index(text[pos:], parts[i])
		if idx == -1 {
			return false
		}
		pos += idx + len(parts[i])
	}

	return true
}

// ListRules 列出所有规则
func (p *Policy) ListRules() []Rule {
	p.mu.RLock()
	defer p.mu.RUnlock()
	
	result := make([]Rule, len(p.rules))
	copy(result, p.rules)
	return result
}

// ClearRules 清除所有规则
func (p *Policy) ClearRules() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.rules = []Rule{}
}

// GetDefaults 获取默认策略
func (p *Policy) GetDefaults() PolicyDefaults {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.defaults
}

// SetDefaults 设置默认策略
func (p *Policy) SetDefaults(defaults PolicyDefaults) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.defaults = defaults
}

// Router 消息路由器
type Router struct {
	mu          sync.RWMutex
	policies    map[string]*Policy // key: channelID
	defaultPolicy *Policy
}

// NewRouter 创建路由器
func NewRouter() *Router {
	return &Router{
		policies: make(map[string]*Policy),
		defaultPolicy: NewPolicy(PolicyDefaults{
			AllowDM:    true,
			AllowGroup: true,
		}),
	}
}

// SetPolicy 设置通道策略
func (r *Router) SetPolicy(channelID string, policy *Policy) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.policies[channelID] = policy
}

// GetPolicy 获取通道策略
func (r *Router) GetPolicy(channelID string) *Policy {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	if policy, ok := r.policies[channelID]; ok {
		return policy
	}
	return r.defaultPolicy
}

// RemovePolicy 移除通道策略
func (r *Router) RemovePolicy(channelID string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.policies, channelID)
}

// ListPolicies 列出所有策略
func (r *Router) ListPolicies() map[string]*Policy {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	result := make(map[string]*Policy)
	for k, v := range r.policies {
		result[k] = v
	}
	return result
}

// AllowMessage 判断消息是否允许通过
func (r *Router) AllowMessage(ctx context.Context, channelID, from, groupID string) (bool, *Rule) {
	policy := r.GetPolicy(channelID)
	return policy.AllowMessage(ctx, from, channelID, groupID)
}
