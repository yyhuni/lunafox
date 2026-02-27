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
		PublicURL: "",
	}, nil, ui.NewPrinter(io.Discard, io.Discard))

	err := stepAgent{}.Run(context.Background(), installer)
	if err == nil {
		t.Fatalf("expected missing public url error")
	}
	if !strings.Contains(err.Error(), "PUBLIC_URL") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestStepAgentRunDoesNotRequireAgentServerURL(t *testing.T) {
	installer := NewInstaller(cli.Options{
		PublicURL: "https://example.com:8083",
	}, nil, ui.NewPrinter(io.Discard, io.Discard))

	err := stepAgent{}.Run(context.Background(), installer)
	if err == nil {
		t.Fatalf("expected TLS init error")
	}
	if !strings.Contains(err.Error(), "HTTPS 信任链未初始化") {
		t.Fatalf("unexpected error: %v", err)
	}
}
