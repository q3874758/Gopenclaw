package tts

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"gopenclaw/internal/config"
)

// Provider TTS 提供商接口
type Provider interface {
	Name() string
	Convert(ctx context.Context, text string, voice string) ([]byte, error)
	ListVoices() []Voice
}

// Voice 语音
type Voice struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Lang    string `json:"lang"`
	Gender  string `json:"gender"`
	Engine  string `json:"engine,omitempty"`
}

// Result TTS 转换结果
type Result struct {
	Audio     []byte `json:"audio"`
	Format    string `json:"format"` // mp3, wav, ogg
	Duration  int    `json:"duration"` // ms
	Provider  string `json:"provider"`
	VocabSize int    `json:"vocabSize,omitempty"`
}

// Handler TTS 处理器
type Handler struct {
	mu       sync.RWMutex
	provider Provider
	enabled  bool
	voices   []Voice
	config   *config.TTSConfig
	client   *http.Client
}

// New 创建 TTS 处理器
func New(cfg *config.TTSConfig) *Handler {
	h := &Handler{
		config:  cfg,
		client:  &http.Client{Timeout: 60 * time.Second},
		enabled: false,
	}
	
	if cfg != nil && cfg.DefaultProvider != "" {
		h.SetProvider(cfg.DefaultProvider)
	}
	
	return h
}

// SetProvider 设置 TTS 提供商
func (h *Handler) SetProvider(name string) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	var provider Provider
	var err error

	switch strings.ToLower(name) {
	case "openai", "tts-1", "tts-1-hd":
		provider = NewOpenAIProvider(h.config.OpenAI)
	case "edge", "edge-tts":
		provider = NewEdgeProvider()
	case "google", "google-tts":
		provider = NewGoogleProvider()
	case "aws", "polly":
		provider = NewAWSProvider(h.config.AWS)
	case "azure", "azure-tts":
		provider = NewAzureProvider(h.config.Azure)
	default:
		// 默认为 OpenAI
		provider = NewOpenAIProvider(h.config.OpenAI)
	}

	h.provider = provider
	h.voices = provider.ListVoices()
	h.enabled = true

	slog.Info("tts provider set", "provider", name)
	return err
}

// Enable 启用 TTS
func (h *Handler) Enable() {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.enabled = true
}

// Disable 禁用 TTS
func (h *Handler) Disable() {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.enabled = false
}

// IsEnabled 检查是否启用
func (h *Handler) IsEnabled() bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.enabled
}

// GetProvider 获取当前提供商
func (h *Handler) GetProvider() string {
	h.mu.RLock()
	defer h.mu.RUnlock()
	if h.provider == nil {
		return ""
	}
	return h.provider.Name()
}

// Convert 转换文本到语音
func (h *Handler) Convert(ctx context.Context, text string, voice string) (*Result, error) {
	h.mu.RLock()
	if !h.enabled {
		h.mu.RUnlock()
		return nil, fmt.Errorf("tts is disabled")
	}
	if h.provider == nil {
		h.mu.RUnlock()
		return nil, fmt.Errorf("no tts provider configured")
	}
	provider := h.provider
	h.mu.RUnlock()

	audio, err := provider.Convert(ctx, text, voice)
	if err != nil {
		return nil, err
	}

	return &Result{
		Audio:    audio,
		Format:   "mp3",
		Provider: provider.Name(),
	}, nil
}

// ListVoices 列出可用语音
func (h *Handler) ListVoices() []Voice {
	h.mu.RLock()
	defer h.mu.RUnlock()
	
	if h.voices == nil {
		return []Voice{}
	}
	return h.voices
}

// ============ OpenAI TTS ============

// OpenAIProvider OpenAI TTS 提供商
type OpenAIProvider struct {
	model  string
	apiKey string
	voice  string
	client *http.Client
}

// NewOpenAIProvider 创建 OpenAI TTS 提供商
func NewOpenAIProvider(cfg *config.TTSOpenAIConfig) *OpenAIProvider {
	voice := "alloy"
	if cfg != nil && cfg.Voice != "" {
		voice = cfg.Voice
	}
	
	model := "tts-1"
	if cfg != nil && cfg.Model != "" {
		model = cfg.Model
	}

	return &OpenAIProvider{
		model:  model,
		apiKey: getEnv("OPENAI_API_KEY", ""),
		voice:  voice,
		client: &http.Client{Timeout: 60 * time.Second},
	}
}

func (p *OpenAIProvider) Name() string { return "openai" }

func (p *OpenAIProvider) Convert(ctx context.Context, text string, voice string) ([]byte, error) {
	if voice == "" {
		voice = p.voice
	}

	apiURL := "https://api.openai.com/v1/audio/speech"
	
	data := map[string]interface{}{
		"model": p.model,
		"input": text,
		"voice": voice,
		"response_format": "mp3",
	}

	jsonData, _ := json.Marshal(data)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, strings.NewReader(string(jsonData)))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.apiKey)

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("openai tts error: %d %s", resp.StatusCode, string(body))
	}

	return io.ReadAll(resp.Body)
}

func (p *OpenAIProvider) ListVoices() []Voice {
	return []Voice{
		{ID: "alloy", Name: "Alloy", Lang: "en", Gender: "neutral"},
		{ID: "echo", Name: "Echo", Lang: "en", Gender: "male"},
		{ID: "fable", Name: "Fable", Lang: "en", Gender: "male"},
		{ID: "onyx", Name: "Onyx", Lang: "en", Gender: "male"},
		{ID: "nova", Name: "Nova", Lang: "en", Gender: "female"},
		{ID: "shimmer", Name: "Shimmer", Lang: "en", Gender: "female"},
	}
}

// ============ Edge TTS ============

// EdgeProvider Edge TTS 提供商
type EdgeProvider struct {
	client *http.Client
}

// NewEdgeProvider 创建 Edge TTS 提供商
func NewEdgeProvider() *EdgeProvider {
	return &EdgeProvider{
		client: &http.Client{Timeout: 60 * time.Second},
	}
}

func (p *EdgeProvider) Name() string { return "edge" }

func (p *EdgeProvider) Convert(ctx context.Context, text string, voice string) ([]byte, error) {
	// Edge TTS 使用 websockets，简化实现返回错误
	return nil, fmt.Errorf("edge tts requires websockets, use openai provider for now")
}

func (p *EdgeProvider) ListVoices() []Voice {
	return []Voice{
		{ID: "en-US-AriaNeural", Name: "Aria", Lang: "en-US", Gender: "female"},
		{ID: "en-US-GuyNeural", Name: "Guy", Lang: "en-US", Gender: "male"},
		{ID: "zh-CN-XiaoxiaoNeural", Name: "Xiaoxiao", Lang: "zh-CN", Gender: "female"},
		{ID: "zh-CN-YunxiNeural", Name: "Yunxi", Lang: "zh-CN", Gender: "male"},
	}
}

// ============ Google TTS ============

// GoogleProvider Google TTS 提供商
type GoogleProvider struct {
	client *http.Client
}

// NewGoogleProvider 创建 Google TTS 提供商
func NewGoogleProvider() *GoogleProvider {
	return &GoogleProvider{
		client: &http.Client{Timeout: 60 * time.Second},
	}
}

func (p *GoogleProvider) Name() string { return "google" }

func (p *GoogleProvider) Convert(ctx context.Context, text string, voice string) ([]byte, error) {
	apiKey := getEnv("GOOGLE_TTS_API_KEY", "")
	if apiKey == "" {
		return nil, fmt.Errorf("GOOGLE_TTS_API_KEY not set")
	}

	voiceName := "en-US-Standard-A"
	if voice != "" {
		voiceName = voice
	}

	apiURL := fmt.Sprintf("https://texttospeech.googleapis.com/v1/text:synthesize?key=%s", apiKey)

	data := map[string]interface{}{
		"input": map[string]string{
			"text": text,
		},
		"voice": map[string]string{
			"languageCode": "en-US",
			"name":        voiceName,
		},
		"audioConfig": map[string]interface{}{
			"audioEncoding": "MP3",
		},
	}

	jsonData, _ := json.Marshal(data)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, strings.NewReader(string(jsonData)))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("google tts error: %d", resp.StatusCode)
	}

	var result struct {
		AudioContent string `json:"audioContent"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return base64.StdEncoding.DecodeString(result.AudioContent)
}

func (p *GoogleProvider) ListVoices() []Voice {
	return []Voice{
		{ID: "en-US-Standard-A", Name: "Standard A", Lang: "en-US", Gender: "female"},
		{ID: "en-US-Standard-B", Name: "Standard B", Lang: "en-US", Gender: "male"},
		{ID: "en-US-Wavenet-A", Name: "Wavenet A", Lang: "en-US", Gender: "female"},
		{ID: "zh-CN-Standard-A", Name: "Standard A", Lang: "zh-CN", Gender: "female"},
	}
}

// ============ AWS Polly ============

// AWSProvider AWS Polly TTS 提供商
type AWSProvider struct {
	region string
	client *http.Client
}

// NewAWSProvider 创建 AWS Polly TTS 提供商
func NewAWSProvider(cfg *config.TTSAWSConfig) *AWSProvider {
	region := "us-east-1"
	if cfg != nil && cfg.Region != "" {
		region = cfg.Region
	}
	return &AWSProvider{
		region: region,
		client: &http.Client{Timeout: 60 * time.Second},
	}
}

func (p *AWSProvider) Name() string { return "aws" }

func (p *AWSProvider) Convert(ctx context.Context, text string, voice string) ([]byte, error) {
	// AWS Polly 需要 AWS SDK，简化实现返回错误
	return nil, fmt.Errorf("aws polly requires aws-sdk-go, use openai provider for now")
}

func (p *AWSProvider) ListVoices() []Voice {
	return []Voice{
		{ID: "Joanna", Name: "Joanna", Lang: "en-US", Gender: "female", Engine: "neural"},
		{ID: "Matthew", Name: "Matthew", Lang: "en-US", Gender: "male", Engine: "neural"},
		{ID: "Amy", Name: "Amy", Lang: "en-GB", Gender: "female", Engine: "neural"},
		{ID: "Zhiyu", Name: "Zhiyu", Lang: "zh-CN", Gender: "female", Engine: "neural"},
	}
}

// ============ Azure TTS ============

// AzureProvider Azure TTS 提供商
type AzureProvider struct {
	key    string
	region string
	client *http.Client
}

// NewAzureProvider 创建 Azure TTS 提供商
func NewAzureProvider(cfg *config.TTSAzureConfig) *AzureProvider {
	key := getEnv("AZURE_SPEECH_KEY", "")
	region := "eastus"
	if cfg != nil {
		if cfg.Key != "" {
			key = cfg.Key
		}
		if cfg.Region != "" {
			region = cfg.Region
		}
	}
	return &AzureProvider{
		key:    key,
		region: region,
		client: &http.Client{Timeout: 60 * time.Second},
	}
}

func (p *AzureProvider) Name() string { return "azure" }

func (p *AzureProvider) Convert(ctx context.Context, text string, voice string) ([]byte, error) {
	if p.key == "" {
		return nil, fmt.Errorf("AZURE_SPEECH_KEY not set")
	}

	apiURL := fmt.Sprintf("https://%s.tts.speech.microsoft.com/cognitiveservices/v1", p.region)

	voiceName := "en-US-AriaNeural"
	if voice != "" {
		voiceName = voice
	}

	ssml := fmt.Sprintf(`<speak version='1.0' xml:lang='en-US'><voice xml:lang='en-US' name='%s'>%s</voice></speak>`, voiceName, text)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, strings.NewReader(ssml))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/ssml+xml")
	req.Header.Set("Ocp-Apim-Subscription-Key", p.key)
	req.Header.Set("X-Microsoft-OutputFormat", "audio-24khz-48kbitrate-mono-mp3")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("azure tts error: %d %s", resp.StatusCode, string(body))
	}

	return io.ReadAll(resp.Body)
}

func (p *AzureProvider) ListVoices() []Voice {
	return []Voice{
		{ID: "en-US-AriaNeural", Name: "Aria", Lang: "en-US", Gender: "female"},
		{ID: "en-US-GuyNeural", Name: "Guy", Lang: "en-US", Gender: "male"},
		{ID: "zh-CN-XiaoxiaoNeural", Name: "Xiaoxiao", Lang: "zh-CN", Gender: "female"},
		{ID: "zh-CN-YunxiNeural", Name: "Yunxi", Lang: "zh-CN", Gender: "male"},
	}
}

// ============ 辅助函数 ============

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
