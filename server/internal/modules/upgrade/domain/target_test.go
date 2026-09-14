package domain

import (
	"errors"
	"testing"

	"github.com/yyhuni/lunafox/contracts/releasemanifest"
)

const targetDigest = "sha256:" + "a000000000000000000000000000000000000000000000000000000000000000"

func testManifest() *releasemanifest.Manifest {
	return &releasemanifest.Manifest{
		ReleaseVersion: "1.2.3",
		Upgrade: releasemanifest.UpgradeMetadata{
			ManifestID:                "lunafox-1.2.3",
			DeploymentMode:            SupportedDeploymentMode,
			RequiresAdminConfirmation: true,
			DatabaseMigration: releasemanifest.DatabaseMigration{
				MigrationType: "none",
				PolicyVersion: 1,
			},
		},
	}
}

func TestTargetRequestRejectsClientImageReferences(t *testing.T) {
	manifest := testManifest()
	// A parsed Manifest carries the exact source digest; this test only needs
	// a valid server-produced target to exercise the request boundary.
	manifestDigest := targetDigest
	request := TargetRequest{ManifestID: manifest.Upgrade.ManifestID, ManifestDigest: manifestDigest, ImageRefs: []string{"docker.io/example/app:latest"}}
	// TargetFromManifest deliberately rejects a hand-built manifest without a
	// digest, so bind the expected identity through the same helper boundary.
	if _, err := TargetFromManifest(manifest); err == nil {
		t.Fatal("expected a hand-built manifest without digest to fail fast")
	}
	if err := ValidateTargetRequest(request, manifest); !errors.Is(err, ErrReleaseManifestTargetInvalid) {
		t.Fatalf("expected client image ref rejection, got %v", err)
	}
}

func TestEnsureSameTargetBindsRetries(t *testing.T) {
	original := Target{ManifestID: "lunafox-1.2.3", ManifestDigest: targetDigest}
	if err := EnsureSameTarget(original, original); err != nil {
		t.Fatalf("same target rejected: %v", err)
	}
	changed := original
	changed.ManifestDigest = "sha256:" + "b000000000000000000000000000000000000000000000000000000000000000"
	if err := EnsureSameTarget(original, changed); !errors.Is(err, ErrReleaseManifestTargetMismatch) || CodeOf(err) != ErrorCodeReleaseManifestTargetMismatch {
		t.Fatalf("expected retry target mismatch, got %v", err)
	}
}

func TestTargetValidationRejectsMalformedDigest(t *testing.T) {
	manifest, err := releasemanifest.Load("../testdata/release.manifest.yaml")
	if err != nil {
		t.Fatal(err)
	}
	manifestDigest := "sha256:NOT-A-DIGEST"
	request := TargetRequest{ManifestID: manifest.Upgrade.ManifestID, ManifestDigest: manifestDigest}
	if err := ValidateTargetRequest(request, manifest); !errors.Is(err, ErrReleaseManifestTargetInvalid) || CodeOf(err) != ErrorCodeReleaseManifestTargetInvalid {
		t.Fatalf("expected malformed request digest rejection, got %v (code=%q)", err, CodeOf(err))
	}

	original := Target{ManifestID: "lunafox-1.2.3", ManifestDigest: manifestDigest}
	if err := EnsureSameTarget(original, original); !errors.Is(err, ErrReleaseManifestTargetInvalid) || CodeOf(err) != ErrorCodeReleaseManifestTargetInvalid {
		t.Fatalf("expected malformed retry digest rejection, got %v (code=%q)", err, CodeOf(err))
	}
}
