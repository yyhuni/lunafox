package release

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadManifestSuccess(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "release.manifest.yaml")
	content := `releaseVersion: v1.2.3
agentVersion: 1.2.3
workerVersion: 1.2.3
agentImageRef: docker.io/yyhuni/lunafox-agent@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa
workerImageRef: docker.io/yyhuni/lunafox-worker@sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	manifest, err := LoadManifest(path)
	if err != nil {
		t.Fatalf("load manifest: %v", err)
	}
	if manifest.ReleaseVersion != "1.2.3" || manifest.AgentVersion != "1.2.3" || manifest.WorkerVersion != "1.2.3" {
		t.Fatalf("unexpected normalized versions: %+v", manifest)
	}
}

func TestLoadManifestRejectsNonDigestImageRef(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "release.manifest.yaml")
	content := `releaseVersion: 1.2.3
agentVersion: 1.2.3
workerVersion: 1.2.3
agentImageRef: docker.io/yyhuni/lunafox-agent:v1.2.3
workerImageRef: docker.io/yyhuni/lunafox-worker@sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	if _, err := LoadManifest(path); err == nil {
		t.Fatalf("expected non-digest image ref to be rejected")
	}
}

func TestLoadManifestRejectsMismatchedVersions(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "release.manifest.yaml")
	content := `releaseVersion: 1.2.3
agentVersion: 1.2.4
workerVersion: 1.2.3
agentImageRef: docker.io/yyhuni/lunafox-agent@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa
workerImageRef: docker.io/yyhuni/lunafox-worker@sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	if _, err := LoadManifest(path); err == nil {
		t.Fatalf("expected version mismatch to be rejected")
	}
}
