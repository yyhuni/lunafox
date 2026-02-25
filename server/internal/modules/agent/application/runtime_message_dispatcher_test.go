package application

import (
	"testing"
)

func TestNewRuntimeMessageDispatcherIncludesHeartbeat(t *testing.T) {
	handlers := newRuntimeMessageDispatcher()
	if handlers[RuntimeMessageTypeHeartbeat] == nil {
		t.Fatalf("expected heartbeat handler to be registered")
	}
}
