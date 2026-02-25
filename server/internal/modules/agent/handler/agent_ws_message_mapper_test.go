package handler

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/yyhuni/lunafox/server/internal/agentproto"
	agentapp "github.com/yyhuni/lunafox/server/internal/modules/agent/application"
)

func TestToRuntimeMessageInputHeartbeat(t *testing.T) {
	since := time.Date(2026, 2, 16, 8, 0, 0, 0, time.FixedZone("UTC+8", 8*60*60))
	raw, err := json.Marshal(agentproto.Message{
		Type: agentproto.MessageTypeHeartbeat,
		Payload: func() json.RawMessage {
			payload, _ := json.Marshal(agentproto.HeartbeatPayload{
				CPU:      1.2,
				Mem:      2.3,
				Disk:     3.4,
				Tasks:    5,
				Version:  "v1.2.3",
				Hostname: "node-1",
				Uptime:   42,
				Health: &agentproto.HealthStatus{
					State:   "healthy",
					Reason:  "ok",
					Message: "all good",
					Since:   &since,
				},
			})
			return payload
		}(),
		Timestamp: time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("marshal message: %v", err)
	}

	input, err := toRuntimeMessageInput(raw)
	if err != nil {
		t.Fatalf("toRuntimeMessageInput error: %v", err)
	}
	if input.Type != agentapp.RuntimeMessageTypeHeartbeat {
		t.Fatalf("expected heartbeat type, got %q", input.Type)
	}
	if input.Heartbeat == nil {
		t.Fatalf("expected heartbeat payload")
	}
	if input.Heartbeat.Health == nil || input.Heartbeat.Health.Since == nil {
		t.Fatalf("expected heartbeat health since")
	}
	if input.Heartbeat.Health.Since.Location() != time.UTC {
		t.Fatalf("expected health since normalized to UTC")
	}
}

func TestToRuntimeMessageInputUnknown(t *testing.T) {
	raw, err := json.Marshal(agentproto.Message{
		Type:      "unknown",
		Payload:   json.RawMessage(`{"x":1}`),
		Timestamp: time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("marshal message: %v", err)
	}

	input, err := toRuntimeMessageInput(raw)
	if err != nil {
		t.Fatalf("toRuntimeMessageInput error: %v", err)
	}
	if input.Type != "unknown" {
		t.Fatalf("expected unknown type passthrough, got %q", input.Type)
	}
	if input.Heartbeat != nil {
		t.Fatalf("expected no heartbeat payload for unknown type")
	}
}

func TestToRuntimeMessageInputInvalidPayload(t *testing.T) {
	raw := []byte(`{"type":"heartbeat","payload":{"version":}`)

	if _, err := toRuntimeMessageInput(raw); err == nil {
		t.Fatalf("expected decode error")
	}
}
