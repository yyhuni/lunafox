package bootstrap

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReadDefaultWordlistImportsRequiresFrozenRegularResources(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "wordlists")
	if err := os.MkdirAll(source, 0o755); err != nil {
		t.Fatalf("create source directory: %v", err)
	}
	if err := os.WriteFile(filepath.Join(source, "default.txt"), []byte("one\ntwo\n"), 0o644); err != nil {
		t.Fatalf("write default wordlist: %v", err)
	}
	manifest := filepath.Join(source, "manifest.json")
	if err := os.WriteFile(manifest, []byte(`{"wordlists":[{"fileName":"default.txt","description":"Default","tags":["default"],"required":true}]}`), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	imports, err := readDefaultWordlistImports(manifest, source)
	if err != nil {
		t.Fatalf("read imports: %v", err)
	}
	if len(imports) != 1 || imports[0].entry.FileName != "default.txt" || imports[0].sourcePath != filepath.Join(source, "default.txt") {
		t.Fatalf("unexpected imports: %+v", imports)
	}
}

func TestReadDefaultWordlistImportsRejectsOptionalAndSymlinkedResources(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "wordlists")
	if err := os.MkdirAll(source, 0o755); err != nil {
		t.Fatalf("create source directory: %v", err)
	}
	if err := os.WriteFile(filepath.Join(source, "default.txt"), []byte("one\n"), 0o644); err != nil {
		t.Fatalf("write wordlist: %v", err)
	}
	manifest := filepath.Join(source, "manifest.json")
	if err := os.WriteFile(manifest, []byte(`{"wordlists":[{"fileName":"default.txt","required":false}]}`), 0o644); err != nil {
		t.Fatalf("write optional manifest: %v", err)
	}
	if _, err := readDefaultWordlistImports(manifest, source); err == nil || !strings.Contains(err.Error(), "optional") {
		t.Fatalf("optional resource error = %v", err)
	}

	if err := os.WriteFile(manifest, []byte(`{"wordlists":[{"fileName":"linked.txt","required":true}]}`), 0o644); err != nil {
		t.Fatalf("write symlink manifest: %v", err)
	}
	if err := os.Symlink(filepath.Join(source, "default.txt"), filepath.Join(source, "linked.txt")); err != nil {
		t.Fatalf("create symlink: %v", err)
	}
	if _, err := readDefaultWordlistImports(manifest, source); err == nil || !strings.Contains(err.Error(), "regular non-symlink") {
		t.Fatalf("symlink resource error = %v", err)
	}
}
