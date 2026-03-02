package runtime

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRuntimePackageDoesNotImportLegacyProtocolEnvelope(t *testing.T) {
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatalf("glob runtime files failed: %v", err)
	}
	for _, file := range files {
		if strings.HasSuffix(file, "_test.go") {
			continue
		}
		data, err := os.ReadFile(file)
		if err != nil {
			t.Fatalf("read %s failed: %v", file, err)
		}
		source := string(data)
		if strings.Contains(source, "/agent/internal/protocol") {
			t.Fatalf("runtime package must not import legacy protocol envelope: %s", file)
		}
	}
}
