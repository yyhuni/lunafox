package main

import (
	"path/filepath"
	"strings"
	"testing"
)

func smokeConfigArgs(root string) []string {
	return []string{
		"--listen", "127.0.0.1:1",
		"--addr-file", filepath.Join(root, "ready"),
		"--done-file", filepath.Join(root, "done"),
		"--evidence-file", filepath.Join(root, "evidence.json"),
		"--package-build-results", filepath.Join(root, "packages.json"),
		"--package-cache-root", filepath.Join(root, "cache"),
		"--fixture-ip", "192.0.2.10",
		"--fixture-port", "18080",
		"--authentication-token", "token",
	}
}

func TestParseSmokeConfigRequiresFixtureRefsAsPair(t *testing.T) {
	root := t.TempDir()
	args := append(smokeConfigArgs(root), "--nuclei-fixture-source-ref", "localhost:5000/nuclei@sha256:"+strings.Repeat("a", 64))
	if _, err := parseSmokeConfig(args); err == nil || !strings.Contains(err.Error(), "supplied together") {
		t.Fatalf("parseSmokeConfig() error = %v, want paired fixture refs", err)
	}
}

func TestParseSmokeConfigAcceptsCanonicalFixtureRefPair(t *testing.T) {
	root := t.TempDir()
	source := "localhost:5000/nuclei@sha256:" + strings.Repeat("a", 64)
	fixture := "localhost:5000/nuclei-fixture@sha256:" + strings.Repeat("b", 64)
	args := append(smokeConfigArgs(root), "--nuclei-fixture-source-ref", source, "--nuclei-fixture-image-ref", fixture)
	cfg, err := parseSmokeConfig(args)
	if err != nil {
		t.Fatalf("parseSmokeConfig() error = %v", err)
	}
	if cfg.nucleiFixtureSourceRef != source || cfg.nucleiFixtureImageRef != fixture {
		t.Fatalf("fixture refs = %q / %q, want supplied canonical refs", cfg.nucleiFixtureSourceRef, cfg.nucleiFixtureImageRef)
	}
}
