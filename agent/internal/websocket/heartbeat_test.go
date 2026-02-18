package websocket

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/yyhuni/lunafox/agent/internal/health"
	"github.com/yyhuni/lunafox/agent/internal/metrics"
	"github.com/yyhuni/lunafox/agent/internal/protocol"
)

func TestHeartbeatSenderSendOnce(t *testing.T) {
	client := &Client{send: make(chan []byte, 1)}
	collector := metrics.NewCollector()
	healthManager := health.NewManager()
	healthManager.Set("paused", "maintenance", "waiting")

	sender := NewHeartbeatSender(client, collector, healthManager, "v1.0.0", "agent-host", func() int { return 3 })
	sender.sendOnce()

	select {
	case payload := <-client.send:
		var msg struct {
			Type      string                 `json:"type"`
			Payload   map[string]interface{} `json:"payload"`
			Timestamp time.Time              `json:"timestamp"`
		}
		if err := json.Unmarshal(payload, &msg); err != nil {
			t.Fatalf("unmarshal heartbeat: %v", err)
		}
		if msg.Type != protocol.MessageTypeHeartbeat {
			t.Fatalf("expected heartbeat type, got %s", msg.Type)
		}
		if msg.Timestamp.IsZero() {
			t.Fatalf("expected timestamp")
		}
		if msg.Payload["version"] != "v1.0.0" {
			t.Fatalf("expected version in payload")
		}
		if msg.Payload["hostname"] != "agent-host" {
			t.Fatalf("expected hostname in payload")
		}
		if tasks, ok := msg.Payload["tasks"].(float64); !ok || int(tasks) != 3 {
			t.Fatalf("expected tasks=3")
		}
		healthPayload, ok := msg.Payload["health"].(map[string]interface{})
		if !ok {
			t.Fatalf("expected health payload")
		}
		if healthPayload["state"] != "paused" {
			t.Fatalf("expected health state paused")
		}
	default:
		t.Fatalf("expected heartbeat message")
	}
}
