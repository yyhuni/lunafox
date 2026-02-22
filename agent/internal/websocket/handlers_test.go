package websocket

import (
	"fmt"
	"testing"

	"github.com/yyhuni/lunafox/agent/internal/protocol"
)

func TestHandlersTaskAvailable(t *testing.T) {
	h := NewHandler()
	called := 0
	h.OnTaskAvailable(func() { called++ })

	message := fmt.Sprintf(`{"type":"%s","payload":{},"timestamp":"2026-01-01T00:00:00Z"}`, protocol.MessageTypeTaskAvailable)
	h.Handle([]byte(message))
	if called != 1 {
		t.Fatalf("expected callback to be called")
	}
}

func TestHandlersTaskCancel(t *testing.T) {
	h := NewHandler()
	var got int
	h.OnTaskCancel(func(id int) { got = id })

	message := fmt.Sprintf(`{"type":"%s","payload":{"taskId":123},"timestamp":"2026-01-01T00:00:00Z"}`, protocol.MessageTypeTaskCancel)
	h.Handle([]byte(message))
	if got != 123 {
		t.Fatalf("expected taskId 123")
	}
}

func TestHandlersConfigUpdate(t *testing.T) {
	h := NewHandler()
	var maxTasks int
	h.OnConfigUpdate(func(payload protocol.ConfigUpdatePayload) {
		if payload.MaxTasks != nil {
			maxTasks = *payload.MaxTasks
		}
	})

	message := fmt.Sprintf(`{"type":"%s","payload":{"maxTasks":8},"timestamp":"2026-01-01T00:00:00Z"}`, protocol.MessageTypeConfigUpdate)
	h.Handle([]byte(message))
	if maxTasks != 8 {
		t.Fatalf("expected maxTasks 8")
	}
}

func TestHandlersUpdateRequired(t *testing.T) {
	h := NewHandler()
	var (
		version  string
		imageRef string
	)
	h.OnUpdateRequired(func(payload protocol.UpdateRequiredPayload) {
		version = payload.Version
		imageRef = payload.ImageRef
	})

	message := fmt.Sprintf(`{"type":"%s","payload":{"version":"v1.0.1","imageRef":"yyhuni/lunafox-agent@sha256:abc"},"timestamp":"2026-01-01T00:00:00Z"}`, protocol.MessageTypeUpdateRequired)
	h.Handle([]byte(message))
	if version != "v1.0.1" {
		t.Fatalf("expected version")
	}
	if imageRef != "yyhuni/lunafox-agent@sha256:abc" {
		t.Fatalf("expected imageRef")
	}
}

func TestHandlersIgnoreInvalidJSON(t *testing.T) {
	h := NewHandler()
	called := 0
	h.OnTaskAvailable(func() { called++ })

	h.Handle([]byte("{bad json"))
	if called != 0 {
		t.Fatalf("expected no callbacks on invalid json")
	}
}

func TestHandlersUpdateRequiredMissingFields(t *testing.T) {
	h := NewHandler()
	called := 0
	h.OnUpdateRequired(func(payload protocol.UpdateRequiredPayload) { called++ })

	message := fmt.Sprintf(`{"type":"%s","payload":{"version":"","imageRef":"yyhuni/lunafox-agent@sha256:abc"}}`, protocol.MessageTypeUpdateRequired)
	h.Handle([]byte(message))
	message = fmt.Sprintf(`{"type":"%s","payload":{"version":"v1.2.3","imageRef":""}}`, protocol.MessageTypeUpdateRequired)
	h.Handle([]byte(message))
	if called != 0 {
		t.Fatalf("expected no callbacks for invalid payload")
	}
}

func TestHandlersLogOpen(t *testing.T) {
	h := NewHandler()
	var (
		requestID string
		container string
	)
	h.OnLogOpen(func(payload protocol.LogOpenPayload) {
		requestID = payload.RequestID
		container = payload.Container
	})

	message := fmt.Sprintf(`{"type":"%s","payload":{"requestId":"req-1","container":"lunafox-agent","tail":200,"follow":true,"timestamps":true}}`, protocol.MessageTypeLogOpen)
	h.Handle([]byte(message))
	if requestID != "req-1" {
		t.Fatalf("expected requestId req-1")
	}
	if container != "lunafox-agent" {
		t.Fatalf("expected container lunafox-agent")
	}
}

func TestHandlersLogCancel(t *testing.T) {
	h := NewHandler()
	var requestID string
	h.OnLogCancel(func(payload protocol.LogCancelPayload) {
		requestID = payload.RequestID
	})

	message := fmt.Sprintf(`{"type":"%s","payload":{"requestId":"req-2"}}`, protocol.MessageTypeLogCancel)
	h.Handle([]byte(message))
	if requestID != "req-2" {
		t.Fatalf("expected requestId req-2")
	}
}
