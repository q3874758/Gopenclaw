package cli

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/spf13/cobra"

	"gopenclaw/internal/config"
	"gopenclaw/internal/protocol"
)

func agentCmd() *cobra.Command {
	var message string
	var port int
	var stream bool
	cmd := &cobra.Command{
		Use:   "agent",
		Short: "向 Agent 发送一条消息（通过 WS 连接 Gateway）",
		RunE: func(cmd *cobra.Command, args []string) error {
			if message == "" {
				_, _ = cmd.OutOrStdout().Write([]byte("usage: gopenclaw agent --message \"...\" [--stream]\n"))
				return nil
			}
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			if port == 0 {
				port = cfg.Gateway.Port
				if port == 0 {
					port = 11999
				}
			}
			addr := fmt.Sprintf("ws://127.0.0.1:%d/ws", port)
			conn, _, err := websocket.DefaultDialer.Dial(addr, http.Header{})
			if err != nil {
				return fmt.Errorf("connect to gateway: %w", err)
			}
			defer conn.Close()

			req := protocol.Message{
				ID:     1,
				Method: "agent.invoke",
				Params: protocol.AgentInvokeParams{Message: message, Stream: stream},
			}
			if err := conn.WriteJSON(req); err != nil {
				return err
			}
			out := cmd.OutOrStdout()
			for {
				_, data, err := conn.ReadMessage()
				if err != nil {
					return err
				}
				var resp protocol.Message
				if err := json.Unmarshal(data, &resp); err != nil {
					return err
				}
				if resp.Error != nil {
					_, _ = cmd.OutOrStderr().Write([]byte(fmt.Sprintf("error: %s\n", resp.Error.Message)))
					return nil
				}
				raw, _ := json.Marshal(resp.Result)
				if m, ok := resp.Result.(map[string]interface{}); ok {
					if c, ok := m["chunk"].(string); ok {
						_, _ = out.Write([]byte(c))
						continue
					}
					if t, ok := m["text"].(string); ok {
						_, _ = out.Write([]byte(t))
						if !stream {
							_, _ = out.Write([]byte("\n"))
						}
						return nil
					}
				}
				var result protocol.AgentInvokeResult
				if json.Unmarshal(raw, &result) == nil && result.Text != "" {
					_, _ = out.Write([]byte(result.Text))
					if !stream {
						_, _ = out.Write([]byte("\n"))
					}
					return nil
				}
			}
		},
	}
	cmd.Flags().StringVar(&message, "message", "", "要发送的消息")
	cmd.Flags().IntVar(&port, "port", 0, "Gateway 端口（0=使用配置）")
	cmd.Flags().BoolVar(&stream, "stream", true, "流式输出（默认 true）")
	return cmd
}
