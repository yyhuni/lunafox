package bootstrap

import (
	"os"
	"strings"
	"testing"
)

func TestAgentClusterSummaryWiringIsReadOnlyAndClaimIsolated(t *testing.T) {
	source, err := os.ReadFile("wiring.go")
	if err != nil {
		t.Fatalf("read wiring.go: %v", err)
	}
	wiring := string(source)
	start := strings.Index(wiring, "func wireAgentModule(")
	end := strings.Index(wiring, "func wireSystemModule(")
	if start < 0 || end <= start {
		t.Fatal("wireAgentModule block not found")
	}
	block := wiring[start:end]
	if strings.Count(block, "NewAgentClusterSummaryService(repos.agentRepo, agentClock)") != 1 {
		t.Fatal("Agent cluster summary must be constructed once from its read store and application clock")
	}
	if strings.Count(block, "clusterSummaryService") != 2 {
		t.Fatal("Agent cluster summary service must only be constructed and passed to its HTTP handler")
	}
	if strings.Count(block, "NewAgentTaskService(scanTaskBridge)") != 1 {
		t.Fatal("Agent claim service must remain wired only to the Scan task bridge")
	}
}
