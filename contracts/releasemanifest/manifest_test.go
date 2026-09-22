package releasemanifest

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
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
	if got := manifest.RuntimeComposition; got.SchemaVersion != 1 || got.Asset != "runtime-composition.json" || got.SHA256 != testDigest {
		t.Fatalf("runtime composition = %#v, want schema v1 bound to %q", got, testDigest)
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

func TestParseValidatesReleaseNotesBinding(t *testing.T) {
	tests := []struct {
		name        string
		manifest    string
		wantMessage string
	}{
		{
			name: "missing production notes",
			manifest: strings.Replace(validManifest(), `releaseNotes:
  digest: "sha256:4406112ce062dd05feacce5f519b8cb7250fd01c7237c43e0f7335da912e8188"
  body: |
    ## English

    - Test release notes.

    ## 简体中文

    - 测试发布说明。
`, "", 1),
			wantMessage: "releaseNotes is required",
		},
		{
			name:        "digest mismatch",
			manifest:    strings.Replace(validManifest(), "sha256:4406112ce062dd05feacce5f519b8cb7250fd01c7237c43e0f7335da912e8188", testDigest, 1),
			wantMessage: "releaseNotes.digest does not match",
		},
		{
			name: "missing language section",
			manifest: func() string {
				body := "## English\n\n- Test release notes.\n"
				digest := sha256.Sum256([]byte(body))
				return strings.Replace(validManifest(), `  digest: "sha256:4406112ce062dd05feacce5f519b8cb7250fd01c7237c43e0f7335da912e8188"
  body: |
    ## English

    - Test release notes.

    ## 简体中文

    - 测试发布说明。
`, fmt.Sprintf("  digest: \"sha256:%x\"\n  body: |\n    ## English\n\n    - Test release notes.\n", digest), 1)
			}(),
			wantMessage: "exactly one English and one 简体中文",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := Parse([]byte(test.manifest)); err == nil || !strings.Contains(err.Error(), test.wantMessage) {
				t.Fatalf("Parse() error = %v, want %q", err, test.wantMessage)
			}
		})
	}
}

func TestParseRejectsMissingRuntimeComposition(t *testing.T) {
	manifest := strings.Replace(validManifest(), `runtimeComposition:
  schemaVersion: 1
  asset: "runtime-composition.json"
  sha256: "`+testDigest+`"
`, "", 1)
	if _, err := Parse([]byte(manifest)); err == nil || !strings.Contains(err.Error(), "runtimeComposition.schemaVersion") {
		t.Fatalf("expected missing runtime composition rejection, got %v", err)
	}
}

func TestParseRejectsInvalidRuntimeComposition(t *testing.T) {
	tests := []struct {
		name        string
		replacement string
		wantMessage string
	}{
		{name: "unsupported schema", replacement: "schemaVersion: 2", wantMessage: "runtimeComposition.schemaVersion must be 1"},
		{name: "path asset", replacement: `asset: "../runtime-composition.json"`, wantMessage: "runtimeComposition.asset must be runtime-composition.json"},
		{name: "url asset", replacement: `asset: "https://example.invalid/runtime-composition.json"`, wantMessage: "runtimeComposition.asset must be runtime-composition.json"},
		{name: "invalid digest", replacement: `sha256: "sha256:not-a-digest"`, wantMessage: "runtimeComposition.sha256 must be a sha256 digest"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			manifest := strings.Replace(validManifest(), "schemaVersion: 1", test.replacement, 1)
			if strings.HasPrefix(test.replacement, "asset:") {
				manifest = strings.Replace(validManifest(), `asset: "runtime-composition.json"`, test.replacement, 1)
			}
			if strings.HasPrefix(test.replacement, "sha256:") {
				manifest = strings.Replace(validManifest(), `sha256: "`+testDigest+`"`, test.replacement, 1)
			}
			_, err := Parse([]byte(manifest))
			if err == nil || !strings.Contains(err.Error(), test.wantMessage) {
				t.Fatalf("expected %q rejection, got %v", test.wantMessage, err)
			}
		})
	}
}

func TestParseAllowsNotesToBeOmittedForDevelopmentManifest(t *testing.T) {
	manifest := strings.Replace(validManifest(), "releaseVersion: 1.2.3\nreleaseNotes:", "releaseVersion: 0.0.0-dev\nreleaseNotes:", 1)
	manifest = strings.Replace(manifest, `releaseNotes:
  digest: "sha256:4406112ce062dd05feacce5f519b8cb7250fd01c7237c43e0f7335da912e8188"
  body: |
    ## English

    - Test release notes.

    ## 简体中文

    - 测试发布说明。
`, "", 1)
	if _, err := Parse([]byte(manifest)); err != nil {
		t.Fatalf("development manifest without notes rejected: %v", err)
	}
}

func TestParseRejectsUnknownRuntimeCompositionField(t *testing.T) {
	manifest := strings.Replace(validManifest(), `  sha256: "`+testDigest+`"
`, `  sha256: "`+testDigest+`"
  unexpected: true
`, 1)
	if _, err := Parse([]byte(manifest)); err == nil || !strings.Contains(err.Error(), "unexpected") {
		t.Fatalf("expected unknown runtime composition field rejection, got %v", err)
	}
}

func TestParseLegacyAlpha114AcceptsOnlyThePinnedManifest(t *testing.T) {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller() failed")
	}
	path := filepath.Join(filepath.Dir(filename), "..", "..", "scripts", "ci", "fixtures", "legacy-alpha114-release.manifest.yaml")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Parse(raw); err == nil {
		t.Fatal("strict Parse accepted the legacy manifest without composition")
	}
	manifest, err := ParseLegacyAlpha114(raw)
	if err != nil {
		t.Fatalf("ParseLegacyAlpha114() error = %v", err)
	}
	if manifest.ReleaseVersion != LegacyAlpha114ReleaseVersion || manifest.Digest() != LegacyAlpha114ManifestDigest {
		t.Fatalf("legacy identity = %s %s", manifest.ReleaseVersion, manifest.Digest())
	}

	tampered := append([]byte(nil), raw...)
	tampered = []byte(strings.Replace(string(tampered), "maintenanceWindowMinutes: 15", "maintenanceWindowMinutes: 16", 1))
	if _, err := ParseLegacyAlpha114(tampered); err == nil {
		t.Fatal("ParseLegacyAlpha114 accepted tampered legacy bytes")
	}
}

func TestLoadLegacyAlpha114UsesTheExplicitException(t *testing.T) {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller() failed")
	}
	path := filepath.Join(filepath.Dir(filename), "..", "..", "scripts", "ci", "fixtures", "legacy-alpha114-release.manifest.yaml")
	manifest, err := LoadLegacyAlpha114(path)
	if err != nil {
		t.Fatalf("LoadLegacyAlpha114() error = %v", err)
	}
	if manifest.Digest() != LegacyAlpha114ManifestDigest {
		t.Fatalf("LoadLegacyAlpha114() digest = %q", manifest.Digest())
	}
}

func validManifest() string {
	return `releaseVersion: 1.2.3
releaseNotes:
  digest: "sha256:4406112ce062dd05feacce5f519b8cb7250fd01c7237c43e0f7335da912e8188"
  body: |
    ## English

    - Test release notes.

    ## 简体中文

    - 测试发布说明。
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
runtimeComposition:
  schemaVersion: 1
  asset: "runtime-composition.json"
  sha256: "` + testDigest + `"
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
