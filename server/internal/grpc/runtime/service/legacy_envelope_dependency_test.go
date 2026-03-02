package service

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRuntimeServicePackageDoesNotUseLegacyAgentProtoEnvelope(t *testing.T) {
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatalf("glob runtime service files failed: %v", err)
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
		if strings.Contains(source, "agentproto.Message") {
			t.Fatalf("runtime service must not depend on legacy agentproto.Message envelope: %s", file)
		}
	}
}
