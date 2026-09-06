package installedengines

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestServerInternalProductionDoesNotReadEngineDefinitionRoots(t *testing.T) {
	root := serverInternalRoot(t)
	forbidden := []string{
		"LoadEnginePackagesFromDefinitionRoot",
		"DefaultBuiltinEngineDefinitionRoot",
		"BUILTIN_ENGINE_CATALOG_SOURCE_ROOT",
		"BuildPrebuiltRuntimeBundle",
		"BuildBuiltinEnginePackage",
	}
	if err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		rel, _ := filepath.Rel(root, path)
		if strings.HasPrefix(rel, "installedengines/testsupport/") {
			return nil
		}
		payload, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		source := string(payload)
		for _, snippet := range forbidden {
			if strings.Contains(source, snippet) {
				t.Fatalf("server/internal production code must consume installed engine packages without definition roots or package builders: %s contains %q", rel, snippet)
			}
		}
		return nil
	}); err != nil {
		t.Fatalf("scan server/internal: %v", err)
	}
}

func TestServerCommandsDoNotOwnEngineContractVerifier(t *testing.T) {
	root := serverInternalRoot(t)
	serverRoot := filepath.Dir(root)
	legacyPath := filepath.Join(serverRoot, "cmd", "engine-contract-verify")
	if _, err := os.Stat(legacyPath); err == nil {
		t.Fatalf("engine contract verifier must live under tools/, not %s", legacyPath)
	} else if !os.IsNotExist(err) {
		t.Fatalf("stat legacy engine contract verifier path: %v", err)
	}
}

func serverInternalRoot(t *testing.T) string {
	t.Helper()
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime caller unavailable")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(filename), ".."))
}
