package application

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/yyhuni/lunafox/contracts/releasemanifest"
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

func TestManifestLoaderAcceptsOnlyPinnedLegacyAlpha114(t *testing.T) {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller() failed")
	}
	path := filepath.Join(filepath.Dir(filename), "..", "..", "..", "..", "..", "scripts", "ci", "fixtures", "legacy-alpha114-release.manifest.yaml")
	manifest, err := LoadReleaseManifest(path, releasemanifest.LegacyAlpha114ManifestDigest)
	if err != nil {
		t.Fatalf("LoadReleaseManifest() error = %v", err)
	}
	if manifest.ReleaseVersion != releasemanifest.LegacyAlpha114ReleaseVersion || manifest.Digest() != releasemanifest.LegacyAlpha114ManifestDigest {
		t.Fatalf("legacy manifest identity = %s %s", manifest.ReleaseVersion, manifest.Digest())
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	tamperedPath := filepath.Join(t.TempDir(), "release.manifest.yaml")
	tampered := strings.Replace(string(raw), "maintenanceWindowMinutes: 15", "maintenanceWindowMinutes: 16", 1)
	if err := os.WriteFile(tamperedPath, []byte(tampered), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadReleaseManifest(tamperedPath, ""); !errors.Is(err, domain.ErrReleaseManifestInvalid) {
		t.Fatalf("tampered legacy manifest error = %v", err)
	}
}

func TestManifestLoaderAcceptsRegisteredAlpha164Bridge(t *testing.T) {
	path := fixturePath("alpha164-bridge.release.manifest.yaml")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(raw)
	wantDigest := fmt.Sprintf("sha256:%x", digest[:])

	loader, err := NewManifestLoader(ManifestLoadConfig{
		Path:             path,
		DeploymentDir:    filepath.Dir(path),
		ExpectedDigest:   wantDigest,
		ExpectedManifest: "lunafox-0.0.1-alpha.183",
	})
	if err != nil {
		t.Fatal(err)
	}
	manifest, err := loader.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if manifest.HasReleaseNotes() || manifest.HasRuntimeComposition() {
		t.Fatalf("bridge metadata presence = notes:%t composition:%t, want both false", manifest.HasReleaseNotes(), manifest.HasRuntimeComposition())
	}

	tamperedPath := filepath.Join(t.TempDir(), "release.manifest.yaml")
	tampered := strings.Replace(string(raw), "0.0.1-alpha.183", "0.0.1-alpha.184", 1)
	if err := os.WriteFile(tamperedPath, []byte(tampered), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadReleaseManifest(tamperedPath, ""); !errors.Is(err, domain.ErrReleaseManifestInvalid) {
		t.Fatalf("unregistered bridge manifest error = %v", err)
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
