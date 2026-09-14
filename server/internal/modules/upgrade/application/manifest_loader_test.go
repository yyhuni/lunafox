package application

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/yyhuni/lunafox/server/internal/modules/upgrade/domain"
)

func fixturePath(name string) string {
	return filepath.Join("..", "testdata", name)
}

func TestManifestLoaderValidatesSourceDigestAndIdentity(t *testing.T) {
	path := fixturePath("release.manifest.yaml")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	hash := sha256.Sum256(raw)
	wantDigest := fmt.Sprintf("sha256:%x", hash[:])
	loader, err := NewManifestLoader(ManifestLoadConfig{
		Path:             path,
		DeploymentDir:    filepath.Dir(path),
		ExpectedDigest:   wantDigest,
		ExpectedManifest: "lunafox-1.2.3",
	})
	if err != nil {
		t.Fatal(err)
	}
	manifest, err := loader.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if manifest.Digest() != wantDigest {
		t.Fatalf("Digest() = %q, want %q", manifest.Digest(), wantDigest)
	}

	loader, err = NewManifestLoader(ManifestLoadConfig{Path: path, ExpectedDigest: "sha256:" + strings.Repeat("b", 64)})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := loader.Load(); !errors.Is(err, domain.ErrReleaseManifestDigestMismatch) || domain.CodeOf(err) != domain.ErrorCodeReleaseManifestDigestMismatch {
		t.Fatalf("expected stable digest mismatch, got %v (code=%q)", err, domain.CodeOf(err))
	}
}

func TestManifestLoaderRejectsMutableReferencesAndIdentityDrift(t *testing.T) {
	path := filepath.Join(t.TempDir(), "release.manifest.yaml")
	raw, err := os.ReadFile(fixturePath("release.manifest.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	mutable := strings.Replace(string(raw), "@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", ":latest", 1)
	if err := os.WriteFile(path, []byte(mutable), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadReleaseManifest(path, ""); !errors.Is(err, domain.ErrReleaseManifestInvalid) {
		t.Fatalf("expected mutable reference rejection, got %v", err)
	}

	identity := strings.Replace(string(raw), `manifestId: "lunafox-1.2.3"`, `manifestId: "lunafox-9.9.9"`, 1)
	if err := os.WriteFile(path, []byte(identity), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadReleaseManifest(path, ""); !errors.Is(err, domain.ErrReleaseManifestIdentityMismatch) {
		t.Fatalf("expected identity mismatch, got %v", err)
	}
}

func TestManifestLoaderRejectsManifestOutsideDeploymentDirectory(t *testing.T) {
	_, err := NewManifestLoader(ManifestLoadConfig{
		Path:          filepath.Join(t.TempDir(), "outside.yaml"),
		DeploymentDir: filepath.Join(t.TempDir(), "deployment"),
	})
	if err == nil || !strings.Contains(err.Error(), "inside the configured deployment directory") {
		t.Fatalf("expected deployment directory rejection, got %v", err)
	}
}
