package application

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	installedenginetest "github.com/yyhuni/lunafox/server/internal/installedengines/testsupport"
	workflowmanifest "github.com/yyhuni/lunafox/server/internal/scanworkflow/manifest"
)

func TestMain(m *testing.M) {
	root := repoRootForTest()
	packages, err := installedenginetest.LoadBuiltinSourcePackages(root)
	if err != nil {
		panic(err)
	}
	if err := installedenginetest.ConfigurePackages(packages); err != nil {
		panic(err)
	}
	_ = workflowmanifest.ConfigureWorkflowDefinitionsRoot(filepath.Join(root, "extensions", "workflows"))
	os.Exit(m.Run())
}

func repoRootForTest() string {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		return "."
	}
	return filepath.Clean(filepath.Join(filepath.Dir(filename), "..", "..", "..", "..", ".."))
}
