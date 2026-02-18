package steps

import (
	"context"
	"io"
	"strings"
	"testing"

	"github.com/yyhuni/lunafox/tools/installer/internal/cli"
	"github.com/yyhuni/lunafox/tools/installer/internal/ui"
)

func TestStepAgentRunRequiresPublicURL(t *testing.T) {
	installer := NewInstaller(cli.Options{
		PublicURL:      "",
		AgentServerURL: "http://server:8080",
	}, nil, ui.NewPrinter(io.Discard, io.Discard))

	err := stepAgent{}.Run(context.Background(), installer)
	if err == nil {
		t.Fatalf("expected missing public url error")
	}
	if !strings.Contains(err.Error(), "PUBLIC_URL") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestStepAgentRunRequiresAgentServerURL(t *testing.T) {
	installer := NewInstaller(cli.Options{
		PublicURL:      "https://example.com:8083",
		AgentServerURL: "",
	}, nil, ui.NewPrinter(io.Discard, io.Discard))

	err := stepAgent{}.Run(context.Background(), installer)
	if err == nil {
		t.Fatalf("expected missing agent server url error")
	}
	if !strings.Contains(err.Error(), "LUNAFOX_AGENT_SERVER_URL") {
		t.Fatalf("unexpected error: %v", err)
	}
}
