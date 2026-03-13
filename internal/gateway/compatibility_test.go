package gateway

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"gopenclaw/internal/config"
	"gopenclaw/internal/protocol"

	"github.com/gorilla/websocket"
)

// 兼容性测试：与官方 OpenClaw Gateway 协议对齐的请求/响应格式

func TestCompatibility_WebSocketConnect(t *testing.T) {
	cfg := config.Default()
	g := New(cfg)
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		_ = g.ServeListener(ctx, ln)
	}()
	time.Sleep(50 * time.Millisecond)

	addr := "ws://" + ln.Addr().String() + "/ws"
	conn, _, err := websocket.DefaultDialer.Dial(addr, nil)
	if err != nil {
		t.Fatalf("ws connect: %v", err)
	}
	defer conn.Close()

	// 发送有效 JSON 请求
	req := protocol.Message{Method: "config.get", Params: map[string]interface{}{}}
	if err := conn.WriteJSON(&req); err != nil {
		t.Fatalf("write: %v", err)
	}

	var resp protocol.Message
	if err := conn.ReadJSON(&resp); err != nil {
		t.Fatalf("read: %v", err)
	}
	if resp.Error != nil {
		t.Fatalf("unexpected error: %v", resp.Error)
	}
	if resp.Result == nil {
		t.Fatal("expected result")
	}
}

func TestCompatibility_InvalidJSONReturnsError(t *testing.T) {
	cfg := config.Default()
	g := New(cfg)
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		_ = g.ServeListener(ctx, ln)
	}()
	time.Sleep(50 * time.Millisecond)

	addr := "ws://" + ln.Addr().String() + "/ws"
	conn, _, err := websocket.DefaultDialer.Dial(addr, nil)
	if err != nil {
		t.Fatalf("ws connect: %v", err)
	}
	defer conn.Close()

	// 发送无效 JSON
	if err := conn.WriteMessage(websocket.TextMessage, []byte("not json")); err != nil {
		t.Fatalf("write: %v", err)
	}

	var resp protocol.Message
	if err := conn.ReadJSON(&resp); err != nil {
		t.Fatalf("read: %v", err)
	}
	if resp.Error == nil {
		t.Fatal("expected error for invalid JSON")
	}
	if resp.Error.Code != -32700 {
		t.Errorf("expected parse error code -32700, got %d", resp.Error.Code)
	}
}

func TestCompatibility_ConfigGet(t *testing.T) {
	cfg := config.Default()
	g := New(cfg)
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		_ = g.ServeListener(ctx, ln)
	}()
	time.Sleep(50 * time.Millisecond)

	addr := "ws://" + ln.Addr().String() + "/ws"
	conn, _, err := websocket.DefaultDialer.Dial(addr, nil)
	if err != nil {
		t.Fatalf("ws connect: %v", err)
	}
	defer conn.Close()

	req := protocol.Message{ID: "1", Method: "config.get", Params: map[string]interface{}{}}
	if err := conn.WriteJSON(&req); err != nil {
		t.Fatal(err)
	}

	var resp protocol.Message
	if err := conn.ReadJSON(&resp); err != nil {
		t.Fatal(err)
	}
	if resp.Error != nil {
		t.Fatalf("config.get error: %v", resp.Error)
	}
	if resp.Result == nil {
		t.Fatal("config.get: expected result")
	}
	// result 应为 { config: object }
	m, ok := resp.Result.(map[string]interface{})
	if !ok {
		t.Fatalf("config.get result not map: %T", resp.Result)
	}
	if _, has := m["config"]; !has {
		t.Errorf("config.get result missing 'config' key: %v", m)
	}
}

func TestCompatibility_SessionsList(t *testing.T) {
	cfg := config.Default()
	g := New(cfg)
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		_ = g.ServeListener(ctx, ln)
	}()
	time.Sleep(50 * time.Millisecond)

	addr := "ws://" + ln.Addr().String() + "/ws"
	conn, _, err := websocket.DefaultDialer.Dial(addr, nil)
	if err != nil {
		t.Fatalf("ws connect: %v", err)
	}
	defer conn.Close()

	req := protocol.Message{ID: "2", Method: "sessions.list", Params: map[string]interface{}{}}
	if err := conn.WriteJSON(&req); err != nil {
		t.Fatal(err)
	}

	var resp protocol.Message
	if err := conn.ReadJSON(&resp); err != nil {
		t.Fatal(err)
	}
	if resp.Error != nil {
		t.Fatalf("sessions.list error: %v", resp.Error)
	}
	if resp.Result == nil {
		t.Fatal("sessions.list: expected result")
	}
	m, ok := resp.Result.(map[string]interface{})
	if !ok {
		t.Fatalf("sessions.list result not map: %T", resp.Result)
	}
	if _, has := m["sessions"]; !has {
		t.Errorf("sessions.list result missing 'sessions' key: %v", m)
	}
}

func TestCompatibility_NodesList(t *testing.T) {
	cfg := config.Default()
	g := New(cfg)
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		_ = g.ServeListener(ctx, ln)
	}()
	time.Sleep(50 * time.Millisecond)

	addr := "ws://" + ln.Addr().String() + "/ws"
	conn, _, err := websocket.DefaultDialer.Dial(addr, nil)
	if err != nil {
		t.Fatalf("ws connect: %v", err)
	}
	defer conn.Close()

	req := protocol.Message{ID: "3", Method: "nodes.list", Params: map[string]interface{}{}}
	if err := conn.WriteJSON(&req); err != nil {
		t.Fatal(err)
	}

	var resp protocol.Message
	if err := conn.ReadJSON(&resp); err != nil {
		t.Fatal(err)
	}
	if resp.Error != nil {
		t.Fatalf("nodes.list error: %v", resp.Error)
	}
	if resp.Result == nil {
		t.Fatal("nodes.list: expected result")
	}
	m, ok := resp.Result.(map[string]interface{})
	if !ok {
		t.Fatalf("nodes.list result not map: %T", resp.Result)
	}
	if _, has := m["nodes"]; !has {
		t.Errorf("nodes.list result missing 'nodes' key: %v", m)
	}
}

func TestCompatibility_AgentInvokeMissingMessageReturnsError(t *testing.T) {
	cfg := config.Default()
	g := New(cfg)
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		_ = g.ServeListener(ctx, ln)
	}()
	time.Sleep(50 * time.Millisecond)

	addr := "ws://" + ln.Addr().String() + "/ws"
	conn, _, err := websocket.DefaultDialer.Dial(addr, nil)
	if err != nil {
		t.Fatalf("ws connect: %v", err)
	}
	defer conn.Close()

	req := protocol.Message{
		ID:     "4",
		Method: "agent.invoke",
		Params: map[string]interface{}{"sessionId": "test"},
	}
	if err := conn.WriteJSON(&req); err != nil {
		t.Fatal(err)
	}

	var resp protocol.Message
	if err := conn.ReadJSON(&resp); err != nil {
		t.Fatal(err)
	}
	if resp.Error == nil {
		t.Fatal("agent.invoke without message should return error")
	}
	if resp.Error.Code != -32602 {
		t.Errorf("expected -32602, got %d", resp.Error.Code)
	}
}

func TestCompatibility_HealthHTTP(t *testing.T) {
	cfg := config.Default()
	g := New(cfg)
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		_ = g.ServeListener(ctx, ln)
	}()
	time.Sleep(50 * time.Millisecond)

	// HTTP GET /health
	client := &http.Client{Timeout: 2 * time.Second}
	req, _ := http.NewRequest("GET", "http://"+ln.Addr().String()+"/health", nil)
	res, err := client.Do(req)
	if err != nil {
		t.Fatalf("health request: %v", err)
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		t.Errorf("health status: %d", res.StatusCode)
	}
	var body map[string]interface{}
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if v, ok := body["ok"]; !ok || v != true {
		t.Errorf("health body: %v", body)
	}
}

// TestCompatibility_AgentInvoke_NonStream E2E：mock LLM 后 agent.invoke 非流式请求/响应格式
func TestCompatibility_AgentInvoke_NonStream(t *testing.T) {
	mockContent := "Hello from mock LLM"
	mockSrv := http.Server{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/v1/chat/completions" {
				http.NotFound(w, r)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"choices": []map[string]interface{}{
					{"message": map[string]interface{}{"role": "assistant", "content": mockContent}},
				},
			})
		}),
	}
	lnMock, _ := net.Listen("tcp", "127.0.0.1:0")
	go func() { _ = mockSrv.Serve(lnMock) }()
	defer mockSrv.Close()

	baseURL := "http://" + lnMock.Addr().String() + "/v1"
	origBase, origKey := os.Getenv("OPENAI_BASE_URL"), os.Getenv("OPENAI_API_KEY")
	_ = os.Setenv("OPENAI_BASE_URL", baseURL)
	_ = os.Setenv("OPENAI_API_KEY", "test-key")
	defer func() {
		_ = os.Setenv("OPENAI_BASE_URL", origBase)
		_ = os.Setenv("OPENAI_API_KEY", origKey)
	}()

	cfg := config.Default()
	g := New(cfg)
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		_ = g.ServeListener(ctx, ln)
	}()
	time.Sleep(80 * time.Millisecond)

	conn, _, err := websocket.DefaultDialer.Dial("ws://"+ln.Addr().String()+"/ws", nil)
	if err != nil {
		t.Fatalf("ws connect: %v", err)
	}
	defer conn.Close()

	req := protocol.Message{
		ID:     "e2e-1",
		Method: "agent.invoke",
		Params: map[string]interface{}{"message": "hi", "stream": false},
	}
	if err := conn.WriteJSON(&req); err != nil {
		t.Fatal(err)
	}

	var resp protocol.Message
	if err := conn.ReadJSON(&resp); err != nil {
		t.Fatal(err)
	}
	if resp.Error != nil {
		t.Fatalf("agent.invoke error: %v", resp.Error)
	}
	if resp.Result == nil {
		t.Fatal("expected result")
	}
	m, ok := resp.Result.(map[string]interface{})
	if !ok {
		t.Fatalf("result not map: %T", resp.Result)
	}
	text, _ := m["text"].(string)
	if text != mockContent {
		t.Errorf("result.text = %q, want %q", text, mockContent)
	}
}

// TestCompatibility_AgentInvoke_Stream E2E：mock LLM 后 agent.invoke 流式 chunk 格式
func TestCompatibility_AgentInvoke_Stream(t *testing.T) {
	chunks := []string{"Hello ", "from ", "stream"}
	mockSrv := http.Server{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/v1/chat/completions" {
				http.NotFound(w, r)
				return
			}
			w.Header().Set("Content-Type", "text/event-stream")
			for _, c := range chunks {
				_, _ = w.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":\"" + c + "\"}}]}\n\n"))
				if flusher, ok := w.(http.Flusher); ok {
					flusher.Flush()
				}
			}
			_, _ = w.Write([]byte("data: [DONE]\n\n"))
		}),
	}
	lnMock, _ := net.Listen("tcp", "127.0.0.1:0")
	go func() { _ = mockSrv.Serve(lnMock) }()
	defer mockSrv.Close()

	baseURL := "http://" + lnMock.Addr().String() + "/v1"
	origBase, origKey := os.Getenv("OPENAI_BASE_URL"), os.Getenv("OPENAI_API_KEY")
	_ = os.Setenv("OPENAI_BASE_URL", baseURL)
	_ = os.Setenv("OPENAI_API_KEY", "test-key")
	defer func() {
		_ = os.Setenv("OPENAI_BASE_URL", origBase)
		_ = os.Setenv("OPENAI_API_KEY", origKey)
	}()

	cfg := config.Default()
	g := New(cfg)
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		_ = g.ServeListener(ctx, ln)
	}()
	time.Sleep(80 * time.Millisecond)

	conn, _, err := websocket.DefaultDialer.Dial("ws://"+ln.Addr().String()+"/ws", nil)
	if err != nil {
		t.Fatalf("ws connect: %v", err)
	}
	defer conn.Close()

	req := protocol.Message{
		ID:     "e2e-2",
		Method: "agent.invoke",
		Params: map[string]interface{}{"message": "hi", "stream": true},
	}
	if err := conn.WriteJSON(&req); err != nil {
		t.Fatal(err)
	}

	var gotChunks []string
	for i := 0; i < 10; i++ {
		var resp protocol.Message
		if err := conn.SetReadDeadline(time.Now().Add(2 * time.Second)); err != nil {
			t.Fatal(err)
		}
		if err := conn.ReadJSON(&resp); err != nil {
			break
		}
		if resp.Error != nil {
			t.Fatalf("agent.invoke stream error: %v", resp.Error)
		}
		if resp.Result != nil {
			if m, ok := resp.Result.(map[string]interface{}); ok {
				if ch, ok := m["chunk"].(string); ok && ch != "" {
					gotChunks = append(gotChunks, ch)
				}
				if _, hasText := m["text"]; hasText {
					break
				}
			}
		}
	}
	got := strings.Join(gotChunks, "")
	want := strings.Join(chunks, "")
	if got != want {
		t.Errorf("stream chunks = %q, want %q", got, want)
	}
	if len(gotChunks) == 0 {
		t.Error("expected at least one result.chunk")
	}
}
