package application

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestServerRuntimeImageIncludesGitForNucleiSync(t *testing.T) {
	root := repositoryRootForRuntimeImageContract(t)
	dockerfile, err := os.ReadFile(filepath.Join(root, "server", "Dockerfile"))
	if err != nil {
		t.Fatalf("read Server Dockerfile: %v", err)
	}
	installBlock := string(dockerfile)
	if !strings.Contains(installBlock, "    git \\") {
		t.Fatal("Server runtime image must install git for Nuclei POC synchronization")
	}
}

func repositoryRootForRuntimeImageContract(t *testing.T) string {
	t.Helper()
	directory, err := os.Getwd()
	if err != nil {
		t.Fatalf("get working directory: %v", err)
	}
	for {
		if _, err := os.Stat(filepath.Join(directory, "server", "Dockerfile")); err == nil {
			return directory
		}
		parent := filepath.Dir(directory)
		if parent == directory {
			t.Fatal("could not locate repository root")
		}
		directory = parent
	}
}
