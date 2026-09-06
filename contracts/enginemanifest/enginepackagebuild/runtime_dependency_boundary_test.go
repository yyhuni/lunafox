package enginepackagebuild

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestRuntimeCodeDoesNotImportEnginePackageBuild(t *testing.T) {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime caller unavailable")
	}
	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(filename), "..", "..", ".."))
	forbiddenImport := `"github.com/yyhuni/lunafox/contracts/enginemanifest/enginepackagebuild"`
	runtimeRoots := []string{
		filepath.Join(repoRoot, "agent", "internal"),
		filepath.Join(repoRoot, "server", "internal"),
	}

	for _, root := range runtimeRoots {
		if _, err := os.Stat(root); err != nil {
			if os.IsNotExist(err) && isPublicProjection(repoRoot) && strings.HasSuffix(filepath.ToSlash(root), "/agent/internal") {
				continue
			}
			t.Fatalf("stat runtime root %s: %v", root, err)
		}
		if err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if entry.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return nil
			}
			payload, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			if strings.Contains(string(payload), forbiddenImport) {
				rel, _ := filepath.Rel(repoRoot, path)
				t.Fatalf("runtime production code must import package manifest contracts, not engine package build helpers: %s", rel)
			}
			return nil
		}); err != nil {
			t.Fatalf("scan runtime root %s: %v", root, err)
		}
	}
}

func isPublicProjection(repoRoot string) bool {
	_, err := os.Stat(filepath.Join(repoRoot, "PUBLIC_PROVENANCE.json"))
	return err == nil
}
