package scanwiring

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestScanWiringDoesNotOwnCreateCatalogReaders(t *testing.T) {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime caller unavailable")
	}
	dir := filepath.Dir(filename)
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read scan wiring dir: %v", err)
	}

	forbidden := []string{
		"enginepackagecatalog.LoadEnginePackagesFromDefinitionRoot",
		"DefaultBuiltinEngineDefinitionRoot",
		"workflowmanifest.GetManifest",
		"workflowToScanCreateWorkflowManifest",
		"func cloneMap",
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") || strings.HasSuffix(entry.Name(), "_test.go") {
			continue
		}
		payload, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			t.Fatalf("read wiring source %s: %v", entry.Name(), err)
		}
		source := string(payload)
		for _, snippet := range forbidden {
			if strings.Contains(source, snippet) {
				t.Fatalf("scan wiring must only assemble create catalog readers; %s contains %q", entry.Name(), snippet)
			}
		}
	}
}
