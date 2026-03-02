package handler

import "testing"

func TestNewAgentHandlerDoesNotFallbackRuntimeInternalURL(t *testing.T) {
	handler := NewAgentHandler(
		nil,
		nil,
		"v1.2.3",
		"https://public.example.com:8083",
		"",
		"docker.io/example/lunafox-agent:v1.2.3",
		"docker.io/example/lunafox-worker:v1.2.3",
		"1.2.3",
		"lunafox_data:/opt/lunafox",
		nil,
	)

	if handler.runtimeInternalURL != "" {
		t.Fatalf("expected empty runtimeInternalURL, got %q", handler.runtimeInternalURL)
	}
}
