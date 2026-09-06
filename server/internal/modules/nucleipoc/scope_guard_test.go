package nucleipoc_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// TestFirstPhaseScopeGuard keeps the catalog-management change from becoming
// an accidental execution integration. Agent/Worker/Scan consumption belongs
// to a separately reviewed change with its own runtime contract.
func TestFirstPhaseScopeGuard(t *testing.T) {
	root := moduleRoot(t)
	forbiddenImports := []string{
		"/modules/agent",
		"/modules/scan",
		"/worker",
		"/execution",
		"/template/runtime",
	}
	forbiddenIdentifiers := map[string]struct{}{
		"ExecuteTemplate": {},
		"RunTemplate":     {},
		"AgentRuntime":    {},
		"WorkerRuntime":   {},
		"ScanRuntime":     {},
		"EngineRunner":    {},
	}

	moduleDir := filepath.Join(root, "server/internal/modules/nucleipoc")
	err := filepath.WalkDir(moduleDir, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || filepath.Ext(path) != ".go" || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		file, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
		if err != nil {
			return err
		}
		for _, imported := range file.Imports {
			importPath := strings.Trim(imported.Path.Value, `"`)
			for _, fragment := range forbiddenImports {
				if strings.Contains(importPath, fragment) {
					t.Fatalf("Nuclei POC production file %s imports excluded runtime boundary %q", path, importPath)
				}
			}
		}
		ast.Inspect(file, func(node ast.Node) bool {
			identifier, ok := node.(*ast.Ident)
			if ok {
				if _, forbidden := forbiddenIdentifiers[identifier.Name]; forbidden {
					t.Fatalf("Nuclei POC production file %s references excluded runtime identifier %q", path, identifier.Name)
				}
			}
			return true
		})
		return nil
	})
	if err != nil {
		t.Fatalf("walk Nuclei POC production files: %v", err)
	}

	routerPath := filepath.Join(moduleDir, "router/routes.go")
	router, err := os.ReadFile(routerPath)
	if err != nil {
		t.Fatalf("read Nuclei POC router: %v", err)
	}
	routerSource := string(router)
	for _, forbidden := range []string{"agent", "worker", "scan", ":run", ":execute"} {
		if strings.Contains(strings.ToLower(routerSource), forbidden) {
			t.Fatalf("Nuclei POC router exposes excluded execution surface %q", forbidden)
		}
	}
	for _, route := range []string{
		"/nucleiPocSources:sync",
		"/nucleiPocSources/current",
		"/nucleiPocSyncTasks/:task",
		"/nucleiPocs",
		"/nucleiPocs/filterOptions",
		"/nucleiPocs:setActivation",
		"/nucleiPocs/:nucleiPoc",
	} {
		if !strings.Contains(routerSource, route) {
			t.Fatalf("Nuclei POC router lost expected management route %q", route)
		}
	}
}

func moduleRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve Nuclei POC scope guard path")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "../../../.."))
}
