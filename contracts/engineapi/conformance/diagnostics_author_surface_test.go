package conformance

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFirstPartyHandwrittenEnginesCannotUseDiagnosticProtocol(t *testing.T) {
	engineRoot := filepath.Clean(filepath.Join("..", "..", "..", "extensions", "engines"))
	forbidden := []string{
		"EngineExecutionDiagnostics",
		"ReportTerminalDiagnostics",
		"GeneratedDiagnostics",
		"GeneratedFailure",
		"ResultTypeWatermark",
		"failed_stage",
		"error_type",
	}
	err := filepath.WalkDir(engineRoot, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		// container is repository conformance/release tooling rather than a
		// handwritten Engine handler, so it must construct protocol fixtures.
		if entry.IsDir() || strings.Contains(filepath.ToSlash(path), "/container/") || filepath.Ext(path) != ".go" || filepath.Base(path) == "execution_run_generated.go" {
			return nil
		}
		source, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		for _, value := range forbidden {
			if strings.Contains(string(source), value) {
				t.Errorf("handwritten Engine source %s reaches generated diagnostic surface %q", path, value)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk first-party Engine source: %v", err)
	}
}
