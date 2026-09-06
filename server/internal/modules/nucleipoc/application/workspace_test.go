package application

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/uuid"
)

func TestLocalWorkspaceCreatesPrivateIsolatedChildrenAndCleansOnlyUUIDs(t *testing.T) {
	root := filepath.Join(t.TempDir(), "workspaces")
	workspace, err := NewLocalWorkspace(root)
	if err != nil {
		t.Fatalf("new workspace: %v", err)
	}
	path, err := workspace.Create(context.Background(), uuid.New())
	if err != nil {
		t.Fatalf("create workspace: %v", err)
	}
	info, err := os.Stat(path)
	if err != nil || info.Mode().Perm() != 0o700 {
		t.Fatalf("workspace mode=%v err=%v", info.Mode().Perm(), err)
	}
	if err := os.WriteFile(filepath.Join(path, "marker"), []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(root, "keep-me"), 0o700); err != nil {
		t.Fatal(err)
	}
	removed, err := workspace.RemoveResiduals(context.Background())
	if err != nil || removed != 1 {
		t.Fatalf("removed=%d err=%v", removed, err)
	}
	if _, err := os.Stat(filepath.Join(root, "keep-me")); err != nil {
		t.Fatalf("non-UUID directory should remain: %v", err)
	}
}

func TestLocalWorkspaceRejectsEscapedAndSymlinkRoots(t *testing.T) {
	root := filepath.Join(t.TempDir(), "workspaces")
	workspace, err := NewLocalWorkspace(root)
	if err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(t.TempDir(), "outside")
	if err := os.Mkdir(outside, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := workspace.Remove(context.Background(), outside); err == nil {
		t.Fatal("expected outside path to be rejected")
	}
	if err := os.Remove(root); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, root); err != nil {
		t.Fatal(err)
	}
	if _, err := workspace.Create(context.Background(), uuid.New()); err == nil {
		t.Fatal("expected replaced workspace root to fail closed")
	}
}
