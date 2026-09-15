package releasemanifest

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const testDigest = "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"

func TestParseRetainsManifestDigestAndUpgradeMetadata(t *testing.T) {
	raw := []byte(validManifest())
	manifest, err := Parse(raw)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	wantHash := sha256.Sum256(raw)
	if got, want := manifest.Digest(), "sha256:"+fmt.Sprintf("%x", wantHash[:]); got != want {
		t.Fatalf("Digest() = %q, want %q", got, want)
	}
	if manifest.Upgrade.ManifestID != "lunafox-1.2.3" || manifest.Upgrade.DatabaseMigration.MigrationType != "none" {
		t.Fatalf("upgrade metadata = %#v, want generated no-migration metadata", manifest.Upgrade)
	}
}

func TestLoadValidatesTheSameManifestFromDisk(t *testing.T) {
	path := filepath.Join(t.TempDir(), "release.manifest.yaml")
	if err := os.WriteFile(path, []byte(validManifest()), 0o600); err != nil {
		t.Fatal(err)
	}
	manifest, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if got, err := manifest.RuntimeImageDigest("server"); err != nil || got != testDigest {
		t.Fatalf("RuntimeImageDigest(server) = %q, %v; want %q", got, err, testDigest)
	}
}

func TestParseRejectsManifestWithoutUpgradeMetadata(t *testing.T) {
	_, err := Parse([]byte(strings.TrimSuffix(validManifest(), "upgrade:\n  manifestId: lunafox-1.2.3\n  deploymentMode: single-node-compose\n  compatibilityRange: \">=1.0.0 <2.0.0\"\n  maintenanceWindowMinutes: 15\n  requiresAdminConfirmation: true\n  databaseMigration:\n    hasDatabaseMigration: false\n    migrationType: none\n    migrationId: \"\"\n    checksum: \"\"\n    policyVersion: 1\n")))
	if err == nil || !strings.Contains(err.Error(), "upgrade.manifestId") {
		t.Fatalf("expected missing upgrade metadata rejection, got %v", err)
	}
}

func TestParseValidatesDatabaseMigrationMetadata(t *testing.T) {
	compatibleMigration := strings.Replace(validManifest(), `    hasDatabaseMigration: false
    migrationType: none
    migrationId: ""
    checksum: ""`, `    hasDatabaseMigration: true
    migrationType: compatible
    migrationId: 000002_add_upgrade_operations
    checksum: sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb`, 1)
	if _, err := Parse([]byte(compatibleMigration)); err != nil {
		t.Fatalf("Parse() compatible migration error = %v", err)
	}

	tests := []struct {
		name        string
		manifest    string
		wantMessage string
	}{
		{
			name:        "unknown migration type",
			manifest:    strings.Replace(compatibleMigration, "migrationType: compatible", "migrationType: unknown", 1),
			wantMessage: "migrationType is not supported",
		},
		{
			name:        "invalid checksum",
			manifest:    strings.Replace(compatibleMigration, "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", "sha256:invalid", 1),
			wantMessage: "checksum must be a sha256 digest",
		},
		{
			name:        "missing policy version",
			manifest:    strings.Replace(compatibleMigration, "policyVersion: 1", "policyVersion: 0", 1),
			wantMessage: "policyVersion must be positive",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := Parse([]byte(test.manifest))
			if err == nil || !strings.Contains(err.Error(), test.wantMessage) {
				t.Fatalf("expected %q rejection, got %v", test.wantMessage, err)
			}
		})
	}
}

func TestParseRejectsMalformedCompatibilityRange(t *testing.T) {
	manifest := strings.Replace(validManifest(), `compatibilityRange: ">=1.0.0 <2.0.0"`, `compatibilityRange: "not-a-range"`, 1)
	if _, err := Parse([]byte(manifest)); err == nil || !strings.Contains(err.Error(), "valid semantic version range") {
		t.Fatalf("expected malformed compatibility range rejection, got %v", err)
	}
}

func validManifest() string {
	return `releaseVersion: 1.2.3
runtimeImages:
  - name: server
    refs:
      - docker.io/yyhuni/lunafox-server@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa
      - ghcr.io/yyhuni/lunafox-server@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa
  - name: frontend
    refs:
      - docker.io/yyhuni/lunafox-frontend@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa
      - ghcr.io/yyhuni/lunafox-frontend@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa
  - name: nginx
    refs:
      - docker.io/yyhuni/lunafox-nginx@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa
      - ghcr.io/yyhuni/lunafox-nginx@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa
  - name: agent
    refs:
      - docker.io/yyhuni/lunafox-agent@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa
      - ghcr.io/yyhuni/lunafox-agent@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa
  - name: bootstrap
    refs:
      - docker.io/yyhuni/lunafox-bootstrap@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa
      - ghcr.io/yyhuni/lunafox-bootstrap@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa
enginePackages:
  - refs:
      - docker.io/yyhuni/lunafox-engine-runtime-port-scan@sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc
      - ghcr.io/yyhuni/lunafox-engine-runtime-port-scan@sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc
upgrade:
  manifestId: lunafox-1.2.3
  deploymentMode: single-node-compose
  compatibilityRange: ">=1.0.0 <2.0.0"
  maintenanceWindowMinutes: 15
  requiresAdminConfirmation: true
  databaseMigration:
    hasDatabaseMigration: false
    migrationType: none
    migrationId: ""
    checksum: ""
    policyVersion: 1
`
}
