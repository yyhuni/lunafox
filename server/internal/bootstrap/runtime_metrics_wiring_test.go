package bootstrap

import (
	"os"
	"strings"
	"testing"
)

func TestWireSystemModuleUsesSharedDataRootForRuntimeMetricsDisk(t *testing.T) {
	source, err := os.ReadFile("wiring.go")
	if err != nil {
		t.Fatalf("read wiring.go: %v", err)
	}

	content := string(source)
	start := strings.Index(content, "func wireSystemModule")
	if start < 0 {
		t.Fatalf("wireSystemModule not found")
	}
	end := strings.Index(content[start:], "func wireSnapshotModule")
	if end < 0 {
		t.Fatalf("wireSnapshotModule not found after wireSystemModule")
	}
	block := content[start : start+end]

	if strings.Contains(block, "WordlistsBasePath") {
		t.Fatalf("runtime metrics disk path must not depend on wordlist storage config")
	}
	if !strings.Contains(block, "sharedstorage.SharedDataRoot") {
		t.Fatalf("runtime metrics disk path must use sharedstorage.SharedDataRoot")
	}
}
