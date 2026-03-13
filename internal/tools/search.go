package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
)

// SearchProvider Web 搜索 provider
type SearchProvider interface {
	Name() string
	Search(ctx context.Context, query string, opts ...SearchOption) ([]SearchResult, error)
}

// SearchOption 搜索选项
type SearchOption func(*searchOptions)

type searchOptions struct {
	num        int
	lang       string
	region     string
	timeRange  string
}

func WithNumResults(n int) SearchOption {
	return func(o *searchOptions) { o.num = n }
}

func WithLanguage(lang string) SearchOption {
	return func(o *searchOptions) { o.lang = lang }
}

func WithRegion(region string) SearchOption {
	return func(o *searchOptions) { o.region = region }
}

func WithTimeRange(tr string) SearchOption {
	return func(o *searchOptions) { o.timeRange = tr }
}

// SearchResult 搜索结果
type SearchResult struct {
	Title   string `json:"title"`
	URL     string `json:"url"`
	Content string `json:"content"`
}

// SearchExecutor Web 搜索执行器
type SearchExecutor struct {
	BaseExecutor
	provider SearchProvider
}

// NewSearchExecutor 创建搜索执行器
func NewSearchExecutor(provider SearchProvider) *SearchExecutor {
	return &SearchExecutor{
		BaseExecutor: BaseExecutor{
			name:        "web_search",
			description: "Search the web for information",
		},
		provider: provider,
	}
}

// Execute 执行搜索
func (e *SearchExecutor) Execute(ctx context.Context, args map[string]any) (string, error) {
	query, _ := args["query"].(string)
	if query == "" {
		return "", fmt.Errorf("query is required")
	}

	num := 10 // 默认返回 10 条结果
	if n, ok := args["num"].(float64); ok {
		num = int(n)
	}

	// 提取过滤器参数
	lang, _ := args["lang"].(string)
	region, _ := args["region"].(string)
	timeRange, _ := args["time_range"].(string)

	// 创建搜索选项
	opts := []SearchOption{
		WithNumResults(num),
	}
	if lang != "" {
		opts = append(opts, WithLanguage(lang))
	}
	if region != "" {
		opts = append(opts, WithRegion(region))
	}
	if timeRange != "" {
		opts = append(opts, WithTimeRange(timeRange))
	}

	results, err := e.provider.Search(ctx, query, opts...)
	if err != nil {
		return "", err
	}

	data, _ := json.Marshal(results)
	return string(data), nil
}

// GetSchema 返回工具 schema
func (e *SearchExecutor) GetSchema() map[string]any {
	return map[string]any{
		"type": "function",
		"function": map[string]any{
			"name":        "web_search",
			"description": "Search the web for information. Returns a list of search results with title, URL, and content.",
			"parameters": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"query": map[string]any{
						"type":        "string",
						"description": "The search query",
					},
					"num": map[string]any{
						"type":        "number",
						"description": "Number of results to return (default 10)",
					},
					"lang": map[string]any{
						"type":        "string",
						"description": "Language code (e.g., 'en', 'zh-cn', 'es')",
					},
					"region": map[string]any{
						"type":        "string",
						"description": "Region code (e.g., 'us', 'cn', 'gb')",
					},
					"time_range": map[string]any{
						"type":        "string",
						"description": "Time range (e.g., 'day', 'week', 'month', 'year')",
					},
				},
				"required": []string{"query"},
			},
		},
	}
}

// DuckDuckGoProvider DuckDuckGo 搜索 provider
type DuckDuckGoProvider struct{}

// NewDuckDuckGoProvider 创建 DuckDuckGo provider
func NewDuckDuckGoProvider() *DuckDuckGoProvider {
	return &DuckDuckGoProvider{}
}

// Name 返回 provider 名称
func (p *DuckDuckGoProvider) Name() string {
	return "duckduckgo"
}

// Search 执行搜索
func (p *DuckDuckGoProvider) Search(ctx context.Context, query string, opts ...SearchOption) ([]SearchResult, error) {
	// 应用搜索选项
	options := &searchOptions{num: 10}
	for _, opt := range opts {
		opt(options)
	}

	// 构建 URL（支持语言和时间范围参数）
	url := fmt.Sprintf("https://html.duckduckgo.com/html/?q=%s", strings.ReplaceAll(query, " ", "+"))
	
	// 添加语言参数
	if options.lang != "" {
		url += "&kl=" + options.lang
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("search failed with status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// 简单解析 HTML 结果
	// 注意：这是一个简化实现，生产环境建议使用官方 API
	results := p.parseResults(string(body), options.num)
	return results, nil
}

// parseResults 解析 HTML 结果
func (p *DuckDuckGoProvider) parseResults(html string, num int) []SearchResult {
	// 简化解析：提取标题和链接
	var results []SearchResult

	// 查找结果块 (简化实现)
	parts := strings.Split(html, `<a rel="nofollow" class="result__a" href="`)
	for i := 1; i < len(parts) && len(results) < num; i++ {
		part := parts[i]
		
		// 提取 URL
		urlEnd := strings.Index(part, `"`)
		if urlEnd == -1 {
			continue
		}
		url := part[:urlEnd]
		
		// 跳过 DuckDuckGo 内部链接
		if !strings.HasPrefix(url, "http") {
			continue
		}
		
		// 提取标题
		titleStart := strings.Index(part, ">")
		if titleStart == -1 {
			continue
		}
		titleEnd := strings.Index(part[titleStart:], "</a>")
		if titleEnd == -1 {
			continue
		}
		title := part[titleStart+1 : titleStart+titleEnd]
		
		results = append(results, SearchResult{
			Title:   title,
			URL:     url,
			Content: "",
		})
	}

	return results
}

// GoogleProvider Google 搜索 provider (需要 API Key)
type GoogleProvider struct {
	apiKey string
}

// NewGoogleProvider 创建 Google provider
func NewGoogleProvider(apiKey string) *GoogleProvider {
	return &GoogleProvider{apiKey: apiKey}
}

// Name 返回 provider 名称
func (p *GoogleProvider) Name() string {
	return "google"
}

// Search 执行搜索
func (p *GoogleProvider) Search(ctx context.Context, query string, opts ...SearchOption) ([]SearchResult, error) {
	if p.apiKey == "" {
		return nil, fmt.Errorf("Google API key required")
	}

	// 应用搜索选项
	options := &searchOptions{num: 10}
	for _, opt := range opts {
		opt(options)
	}

	url := fmt.Sprintf("https://www.googleapis.com/customsearch/v1?key=%s&q=%s&num=%d", 
		p.apiKey, strings.ReplaceAll(query, " ", "+"), options.num)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		slog.Error("google search failed", "status", resp.StatusCode, "body", string(body))
		return nil, fmt.Errorf("search failed with status %d", resp.StatusCode)
	}

	var result struct {
		Items []struct {
			Title string `json:"title"`
			Link  string `json:"link"`
			Snippet string `json:"snippet"`
		} `json:"items"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	results := make([]SearchResult, len(result.Items))
	for i, item := range result.Items {
		results[i] = SearchResult{
			Title:   item.Title,
			URL:     item.Link,
			Content: item.Snippet,
		}
	}

	return results, nil
}

// SearchWithFallback 带 fallback 的搜索
type SearchWithFallback struct {
	providers []SearchProvider
}

// NewSearchWithFallback 创建带 fallback 的搜索
func NewSearchWithFallback(providers ...SearchProvider) *SearchWithFallback {
	return &SearchWithFallback{providers: providers}
}

// Name 返回搜索器名称
func (s *SearchWithFallback) Name() string {
	return "search_with_fallback"
}

// Search 执行搜索，尝试每个 provider
func (s *SearchWithFallback) Search(ctx context.Context, query string, opts ...SearchOption) ([]SearchResult, error) {
	var lastErr error
	for _, p := range s.providers {
		results, err := p.Search(ctx, query, opts...)
		if err != nil {
			lastErr = err
			slog.Warn("search provider failed, trying next", "provider", p.Name(), "err", err)
			continue
		}
		return results, nil
	}
	if lastErr != nil {
		return nil, fmt.Errorf("all search providers failed: %w", lastErr)
	}
	return nil, fmt.Errorf("no search providers available")
}

// GrokProvider Grok (xAI) 搜索 provider
type GrokProvider struct {
	apiKey string
}

// NewGrokProvider 创建 Grok provider
func NewGrokProvider(apiKey string) *GrokProvider {
	return &GrokProvider{apiKey: apiKey}
}

// Name 返回 provider 名称
func (p *GrokProvider) Name() string {
	return "grok"
}

// Search 执行搜索 (使用 Grok 的 web search 能力)
func (p *GrokProvider) Search(ctx context.Context, query string, opts ...SearchOption) ([]SearchResult, error) {
	if p.apiKey == "" {
		return nil, fmt.Errorf("Grok API key required (xAI_API_KEY)")
	}

	// 应用搜索选项
	options := &searchOptions{num: 10}
	for _, opt := range opts {
		opt(options)
	}

	url := "https://api.x.ai/v1/chat/completions"
	
	reqBody := map[string]interface{}{
		"model": "grok-2-search",
		"messages": []map[string]string{
			{"role": "user", "content": "Search for: " + query},
		},
		"max_tokens": 4096,
	}
	
	body, _ := json.Marshal(reqBody)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, strings.NewReader(string(body)))
	if err != nil {
		return nil, err
	}
	
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.apiKey)
	
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Grok API failed with status %d", resp.StatusCode)
	}
	
	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	
	if len(result.Choices) == 0 {
		return nil, fmt.Errorf("no search results from Grok")
	}
	
	content := result.Choices[0].Message.Content
	results := []SearchResult{
		{
			Title:   query,
			URL:     "https://x.ai",
			Content: content,
		},
	}
	
	return results, nil
}

// KimiProvider Kimi (月之暗面) 搜索 provider
type KimiProvider struct {
	apiKey string
}

// NewKimiProvider 创建 Kimi provider
func NewKimiProvider(apiKey string) *KimiProvider {
	return &KimiProvider{apiKey: apiKey}
}

// Name 返回 provider 名称
func (p *KimiProvider) Name() string {
	return "kimi"
}

// Search 执行搜索 (使用 Kimi 的 web search 能力)
func (p *KimiProvider) Search(ctx context.Context, query string, opts ...SearchOption) ([]SearchResult, error) {
	if p.apiKey == "" {
		return nil, fmt.Errorf("Kimi API key required (MOONSHOT_API_KEY)")
	}

	// 应用搜索选项
	_ = opts // opts 保留用于未来扩展

	url := "https://api.moonshot.cn/v1/chat/completions"
	
	reqBody := map[string]interface{}{
		"model": "kimi-k2.5-preview",
		"messages": []map[string]string{
			{"role": "user", "content": "Search for: " + query},
		},
		"max_tokens": 4096,
	}
	
	body, _ := json.Marshal(reqBody)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, strings.NewReader(string(body)))
	if err != nil {
		return nil, err
	}
	
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.apiKey)
	
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Kimi API failed with status %d", resp.StatusCode)
	}
	
	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	
	if len(result.Choices) == 0 {
		return nil, fmt.Errorf("no search results from Kimi")
	}
	
	content := result.Choices[0].Message.Content
	results := []SearchResult{
		{
			Title:   query,
			URL:     "https://www.moonshot.cn",
			Content: content,
		},
	}
	
	return results, nil
}
