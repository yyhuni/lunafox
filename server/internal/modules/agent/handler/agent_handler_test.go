package handler

import (
	"testing"

	agentdomain "github.com/yyhuni/lunafox/server/internal/modules/agent/domain"
)

type runtimeConfigPublisherStub struct{}

func (runtimeConfigPublisherStub) SendConfigUpdate(*agentdomain.Agent) {}

func TestNewAgentHandlerDoesNotFallbackRuntimeInternalURL(t *testing.T) {
	handler := NewAgentHandler(
		nil,
		runtimeConfigPublisherStub{},
		"v1.2.3",
		"https://public.example.com:8083",
		"",
		"docker.io/example/lunafox-agent:v1.2.3",
		"lunafox_data:/opt/lunafox",
		nil,
	)

	if handler.agentControlInternalURL != "" {
		t.Fatalf("expected empty agentControlInternalURL, got %q", handler.agentControlInternalURL)
	}
}
