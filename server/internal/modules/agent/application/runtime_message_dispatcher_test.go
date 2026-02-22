package application

import (
	"testing"
)

func TestNewRuntimeMessageDispatcherIncludesHeartbeat(t *testing.T) {
	handlers := newRuntimeMessageDispatcher()
	if handlers[RuntimeMessageTypeHeartbeat] == nil {
		t.Fatalf("expected heartbeat handler to be registered")
	}
	if handlers[RuntimeMessageTypeLogStarted] == nil {
		t.Fatalf("expected log_started handler to be registered")
	}
	if handlers[RuntimeMessageTypeLogChunk] == nil {
		t.Fatalf("expected log_chunk handler to be registered")
	}
	if handlers[RuntimeMessageTypeLogEnd] == nil {
		t.Fatalf("expected log_end handler to be registered")
	}
	if handlers[RuntimeMessageTypeLogError] == nil {
		t.Fatalf("expected log_error handler to be registered")
	}
}
