package application

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/yyhuni/lunafox/contracts/releasemanifest"
	"github.com/yyhuni/lunafox/server/internal/modules/upgrade/domain"
)

// ManifestLoadConfig describes a server-owned manifest location. The path is
// configuration, never request data; callers should construct this once at
// process startup and reuse the loader for checks and retries.
type ManifestLoadConfig struct {
	Path             string
	DeploymentDir    string
	ExpectedDigest   string
	ExpectedManifest string
}

// ManifestLoader reads the fixed release manifest and validates its identity.
type ManifestLoader struct {
	path           string
	deploymentDir  string
	expectedDigest string
	expectedID     string
}

func NewManifestLoader(config ManifestLoadConfig) (*ManifestLoader, error) {
	manifestPath := strings.TrimSpace(config.Path)
	if manifestPath == "" {
		return nil, fmt.Errorf("manifest path is required")
	}
	manifestPath = filepath.Clean(manifestPath)
	deploymentDir := strings.TrimSpace(config.DeploymentDir)
	if deploymentDir != "" {
		deploymentDir = filepath.Clean(deploymentDir)
		relative, err := filepath.Rel(deploymentDir, manifestPath)
		if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
			return nil, fmt.Errorf("manifest path must remain inside the configured deployment directory")
		}
	}
	if expected := strings.TrimSpace(config.ExpectedDigest); expected != "" && !isDigest(expected) {
		return nil, fmt.Errorf("expected manifest digest must be a sha256 digest")
	}
	return &ManifestLoader{
		path:           manifestPath,
		deploymentDir:  deploymentDir,
		expectedDigest: strings.TrimSpace(config.ExpectedDigest),
		expectedID:     strings.TrimSpace(config.ExpectedManifest),
	}, nil
}

// Load reads from the configured path only. No fallback manifest is attempted
// when reading or validation fails.
func (loader *ManifestLoader) Load() (*releasemanifest.Manifest, error) {
	if loader == nil || loader.path == "" {
		return nil, fmt.Errorf("manifest loader is not configured")
	}
	manifest, err := releasemanifest.Load(loader.path)
	if err != nil {
		// Deployment-owned loading may accept only the policy-pinned alpha.114
		// bytes or the exact alpha.164 bridge profile. Keep the normal parser
		// strict so absent modern evidence never becomes a general fallback.
		if legacy, legacyErr := releasemanifest.LoadLegacyCompatible(loader.path); legacyErr == nil {
			manifest = legacy
			err = nil
		}
	}
	if err != nil {
		if strings.Contains(err.Error(), "upgrade.databaseMigration") {
			if strings.Contains(err.Error(), "migrationType is not supported") {
				return nil, domain.NewMigrationUnsupported("migration type is not supported")
			}
			return nil, domain.WrapMigrationMetadataMissing(err)
		}
		return nil, domain.WrapManifestInvalid(err)
	}
	raw, err := os.ReadFile(loader.path)
	if err != nil {
		return nil, domain.WrapManifestInvalid(fmt.Errorf("read manifest for digest verification: %w", err))
	}
	hash := sha256.Sum256(raw)
	actualDigest := fmt.Sprintf("sha256:%x", hash[:])
	if manifest.Digest() == "" || manifest.Digest() != actualDigest {
		return nil, domain.WrapManifestInvalid(fmt.Errorf("parsed manifest digest does not match its source bytes"))
	}
	if loader.expectedDigest != "" && loader.expectedDigest != actualDigest {
		return nil, domain.NewManifestDigestMismatch(loader.expectedDigest, actualDigest)
	}
	expectedID := "lunafox-" + manifest.ReleaseVersion
	if manifest.Upgrade.ManifestID != expectedID {
		return nil, domain.NewManifestIdentityMismatch(expectedID, manifest.Upgrade.ManifestID)
	}
	if loader.expectedID != "" && loader.expectedID != manifest.Upgrade.ManifestID {
		return nil, domain.NewManifestIdentityMismatch(loader.expectedID, manifest.Upgrade.ManifestID)
	}
	if _, err := domain.TargetFromManifest(manifest); err != nil {
		return nil, err
	}
	return manifest, nil
}

// LoadReleaseManifest is a convenience for server wiring that has an
// expected digest from a previously persisted Operation. An empty expected
// digest means "compute and validate the digest from disk".
func LoadReleaseManifest(path, expectedDigest string) (*releasemanifest.Manifest, error) {
	loader, err := NewManifestLoader(ManifestLoadConfig{Path: path, ExpectedDigest: expectedDigest})
	if err != nil {
		return nil, domain.WrapManifestInvalid(err)
	}
	return loader.Load()
}

func isDigest(value string) bool {
	if len(value) != len("sha256:")+64 || !strings.HasPrefix(value, "sha256:") {
		return false
	}
	for _, char := range value[len("sha256:"):] {
		if (char < '0' || char > '9') && (char < 'a' || char > 'f') {
			return false
		}
	}
	return true
}
