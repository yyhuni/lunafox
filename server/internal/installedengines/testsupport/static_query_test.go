package testsupport

import (
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"sort"
	"testing"

	enginemanifest "github.com/yyhuni/lunafox/contracts/enginemanifest"
)

func TestLoadBuiltinSourcePackagesDiscoversEverySourceDefinition(t *testing.T) {
	repoRoot := testRepositoryRoot(t)
	packages, err := LoadBuiltinSourcePackages(repoRoot)
	if err != nil {
		t.Fatalf("LoadBuiltinSourcePackages() error = %v", err)
	}

	matches, err := filepath.Glob(filepath.Join(repoRoot, "extensions", "engines", "*", "engine.json"))
	if err != nil {
		t.Fatalf("glob builtin Engine definitions: %v", err)
	}
	want := make([]string, 0, len(matches))
	for _, path := range matches {
		payload, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		definition, err := enginemanifest.DecodeEngineDefinition(payload, path)
		if err != nil {
			t.Fatalf("decode %s: %v", path, err)
		}
		want = append(want, definition.EngineID)
	}
	sort.Strings(want)

	got := make([]string, 0, len(packages))
	for _, pkg := range packages {
		got = append(got, pkg.Registration.EngineID)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("discovered builtin Engines = %#v, want every source definition %#v", got, want)
	}
	if !containsString(got, "engine.lunafox.directory_scan") {
		t.Fatalf("Directory Scan source package was not discovered: %#v", got)
	}
}

func testRepositoryRoot(t *testing.T) string {
	t.Helper()
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime caller unavailable")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(filename), "..", "..", "..", ".."))
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
