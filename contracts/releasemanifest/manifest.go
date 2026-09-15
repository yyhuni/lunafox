// Package releasemanifest owns the strict, immutable LunaFox release
// inventory shared by installation and upgrade control planes.
package releasemanifest

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path"
	"regexp"
	"strings"

	"github.com/blang/semver"
	"github.com/yyhuni/lunafox/contracts/ociartifact"
	"github.com/yyhuni/lunafox/contracts/ocidistribution"
	"go.yaml.in/yaml/v3"
)

// Manifest is a strict, immutable deployment inventory. Its digest is the
// SHA-256 of the exact YAML bytes accepted by Parse or Load.
type Manifest struct {
	ReleaseVersion string          `yaml:"releaseVersion"`
	RuntimeImages  []RuntimeImage  `yaml:"runtimeImages"`
	EnginePackages []EnginePackage `yaml:"enginePackages"`
	Upgrade        UpgradeMetadata `yaml:"upgrade"`

	digest string
}

type RuntimeImage struct {
	Name string   `yaml:"name"`
	Refs []string `yaml:"refs"`
}

type EnginePackage struct {
	Refs []string `yaml:"refs"`
}

// UpgradeMetadata is intentionally part of the signed/reviewed release
// inventory. It never contains a host path, command, credential, or mutable
// image reference.
type UpgradeMetadata struct {
	ManifestID                string            `yaml:"manifestId"`
	DeploymentMode            string            `yaml:"deploymentMode"`
	CompatibilityRange        string            `yaml:"compatibilityRange"`
	MaintenanceWindowMinutes  int               `yaml:"maintenanceWindowMinutes"`
	RequiresAdminConfirmation bool              `yaml:"requiresAdminConfirmation"`
	DatabaseMigration         DatabaseMigration `yaml:"databaseMigration"`
}

// DatabaseMigration describes a release-level migration gate. A migration
// checksum identifies the reviewed migration payload; it is not a database
// backup or a request to run a down migration.
type DatabaseMigration struct {
	HasDatabaseMigration bool   `yaml:"hasDatabaseMigration"`
	MigrationType        string `yaml:"migrationType"`
	MigrationID          string `yaml:"migrationId"`
	Checksum             string `yaml:"checksum"`
	PolicyVersion        int    `yaml:"policyVersion"`
}

var semVerPattern = regexp.MustCompile(`^\d+\.\d+\.\d+([\-+][0-9A-Za-z.+-]+)?$`)
var manifestIDPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9._/-]*$`)
var migrationIDPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._/-]*$`)
var checksumPattern = regexp.MustCompile(`^sha256:[a-f0-9]{64}$`)

var requiredRuntimeImages = map[string]struct{}{
	"server": {}, "frontend": {}, "nginx": {}, "agent": {}, "bootstrap": {},
}

// Load reads and validates a release manifest from disk.
func Load(filePath string) (*Manifest, error) {
	raw, err := os.ReadFile(strings.TrimSpace(filePath))
	if err != nil {
		return nil, fmt.Errorf("failed to read release manifest: %w", err)
	}
	return Parse(raw)
}

// Parse validates exactly one manifest document and retains its content hash.
func Parse(raw []byte) (*Manifest, error) {
	manifest, err := parse(raw)
	if err != nil {
		return nil, fmt.Errorf("failed to parse release manifest: %w", err)
	}
	if err := manifest.normalizeAndValidate(); err != nil {
		return nil, err
	}
	digest := sha256.Sum256(raw)
	manifest.digest = "sha256:" + fmt.Sprintf("%x", digest[:])
	return manifest, nil
}

// Digest returns the digest of the exact accepted manifest bytes.
func (manifest *Manifest) Digest() string {
	if manifest == nil {
		return ""
	}
	return manifest.digest
}

func (manifest *Manifest) RuntimeImageRefs(name string) ([]string, error) {
	if manifest == nil {
		return nil, fmt.Errorf("release manifest is required")
	}
	for _, image := range manifest.RuntimeImages {
		if image.Name == name {
			return append([]string(nil), image.Refs...), nil
		}
	}
	return nil, fmt.Errorf("required runtime image %q is missing", name)
}

// RuntimeImageDigest returns the canonical digest shared by Docker Hub and
// GHCR candidates for one product component.
func (manifest *Manifest) RuntimeImageDigest(name string) (string, error) {
	refs, err := manifest.RuntimeImageRefs(name)
	if err != nil {
		return "", err
	}
	if len(refs) == 0 {
		return "", fmt.Errorf("runtime image %q has no candidates", name)
	}
	ref, err := ociartifact.ParseDigestReference(refs[0])
	if err != nil {
		return "", err
	}
	return ref.Digest, nil
}

// RuntimeImageDigests projects only verified component identities. Callers do
// not need to parse image references or accept a caller-provided image ref.
func (manifest *Manifest) RuntimeImageDigests() (map[string]string, error) {
	if manifest == nil {
		return nil, fmt.Errorf("release manifest is required")
	}
	result := make(map[string]string, len(manifest.RuntimeImages))
	for _, image := range manifest.RuntimeImages {
		digest, err := manifest.RuntimeImageDigest(image.Name)
		if err != nil {
			return nil, err
		}
		result[image.Name] = digest
	}
	return result, nil
}

// WriteEngineInventory projects only immutable engine refs needed by bootstrap.
func (manifest *Manifest) WriteEngineInventory(filePath string) error {
	return manifest.writeEngineInventory(filePath, false)
}

// WriteCloudflareAcceleratedEngineInventory preserves release identity while
// adding the approved transport candidate ahead of the release candidates.
func (manifest *Manifest) WriteCloudflareAcceleratedEngineInventory(filePath string) error {
	return manifest.writeEngineInventory(filePath, true)
}

func (manifest *Manifest) writeEngineInventory(filePath string, cloudflareAcceleration bool) error {
	if manifest == nil || len(manifest.EnginePackages) == 0 {
		return fmt.Errorf("release engine inventory is required")
	}
	enginePackages := make([]EnginePackage, len(manifest.EnginePackages))
	for index, enginePackage := range manifest.EnginePackages {
		refs := append([]string(nil), enginePackage.Refs...)
		if cloudflareAcceleration {
			acceleration, err := ocidistribution.BuildCloudflareAcceleration(refs)
			if err != nil {
				return fmt.Errorf("map enginePackages[%d] for Cloudflare acceleration: %w", index, err)
			}
			refs = acceleration.DownloadReferenceStrings()
		}
		enginePackages[index] = EnginePackage{Refs: refs}
	}
	payload, err := yaml.Marshal(struct {
		EnginePackages []EnginePackage `yaml:"enginePackages"`
	}{EnginePackages: enginePackages})
	if err != nil {
		return fmt.Errorf("marshal engine inventory: %w", err)
	}
	if err := os.WriteFile(filePath, payload, 0o600); err != nil {
		return fmt.Errorf("write engine inventory: %w", err)
	}
	return nil
}

func (manifest *Manifest) normalizeAndValidate() error {
	if manifest == nil {
		return fmt.Errorf("release manifest cannot be empty")
	}
	if manifest.ReleaseVersion == "" || manifest.ReleaseVersion != strings.TrimSpace(manifest.ReleaseVersion) {
		return fmt.Errorf("releaseVersion must be non-empty and canonical")
	}
	if !semVerPattern.MatchString(manifest.ReleaseVersion) {
		return fmt.Errorf("releaseVersion must match MAJOR.MINOR.PATCH(-suffix or +suffix)")
	}

	seenRuntimeNames := make(map[string]struct{}, len(manifest.RuntimeImages))
	for index := range manifest.RuntimeImages {
		image := &manifest.RuntimeImages[index]
		if image.Name == "" || image.Name != strings.TrimSpace(image.Name) {
			return fmt.Errorf("runtimeImages[%d].name must be non-empty and canonical", index)
		}
		if _, required := requiredRuntimeImages[image.Name]; !required {
			return fmt.Errorf("runtimeImages[%d].name %q is not supported", index, image.Name)
		}
		if _, duplicate := seenRuntimeNames[image.Name]; duplicate {
			return fmt.Errorf("runtimeImages contains duplicate name %q", image.Name)
		}
		seenRuntimeNames[image.Name] = struct{}{}
		if err := validateProductionCandidateRefs(image.Refs, "runtimeImages["+image.Name+"].refs"); err != nil {
			return err
		}
		expectedRepository := "lunafox-" + image.Name
		for _, rawRef := range image.Refs {
			ref, err := ociartifact.ParseDigestReference(rawRef)
			if err != nil {
				return err
			}
			if path.Base(ref.Repository) != expectedRepository {
				return fmt.Errorf("runtimeImages[%s] ref %q must use repository %q", image.Name, ref.String(), expectedRepository)
			}
		}
	}
	for name := range requiredRuntimeImages {
		if _, found := seenRuntimeNames[name]; !found {
			return fmt.Errorf("runtimeImages is missing required image %q", name)
		}
	}

	if len(manifest.EnginePackages) == 0 {
		return fmt.Errorf("enginePackages cannot be empty")
	}
	for index := range manifest.EnginePackages {
		if err := validateProductionCandidateRefs(manifest.EnginePackages[index].Refs, fmt.Sprintf("enginePackages[%d].refs", index)); err != nil {
			return err
		}
	}
	return validateUpgradeMetadata(manifest.Upgrade)
}

func validateUpgradeMetadata(metadata UpgradeMetadata) error {
	if metadata.ManifestID == "" || metadata.ManifestID != strings.TrimSpace(metadata.ManifestID) || !manifestIDPattern.MatchString(metadata.ManifestID) {
		return fmt.Errorf("upgrade.manifestId must be a canonical release identity")
	}
	if metadata.DeploymentMode != "single-node-compose" {
		return fmt.Errorf("upgrade.deploymentMode must be single-node-compose")
	}
	if metadata.CompatibilityRange == "" || metadata.CompatibilityRange != strings.TrimSpace(metadata.CompatibilityRange) || len(metadata.CompatibilityRange) > 128 {
		return fmt.Errorf("upgrade.compatibilityRange must be non-empty and canonical")
	}
	compatibilityRange, err := semver.ParseRange(metadata.CompatibilityRange)
	if err != nil || compatibilityRange == nil {
		return fmt.Errorf("upgrade.compatibilityRange must be a valid semantic version range")
	}
	if metadata.MaintenanceWindowMinutes < 1 || metadata.MaintenanceWindowMinutes > 1440 {
		return fmt.Errorf("upgrade.maintenanceWindowMinutes must be between 1 and 1440")
	}
	if !metadata.RequiresAdminConfirmation {
		return fmt.Errorf("upgrade.requiresAdminConfirmation must be true")
	}
	migration := metadata.DatabaseMigration
	if migration.PolicyVersion <= 0 {
		return fmt.Errorf("upgrade.databaseMigration.policyVersion must be positive")
	}
	if !migration.HasDatabaseMigration {
		if migration.MigrationType != "none" || migration.MigrationID != "" || migration.Checksum != "" {
			return fmt.Errorf("upgrade.databaseMigration without a migration must use type none and no identity or checksum")
		}
		return nil
	}
	if migration.MigrationType != "compatible" && migration.MigrationType != "preserve-data" && migration.MigrationType != "destructive" {
		return fmt.Errorf("upgrade.databaseMigration.migrationType is not supported")
	}
	if !migrationIDPattern.MatchString(migration.MigrationID) {
		return fmt.Errorf("upgrade.databaseMigration.migrationId must be canonical")
	}
	if !checksumPattern.MatchString(migration.Checksum) {
		return fmt.Errorf("upgrade.databaseMigration.checksum must be a sha256 digest")
	}
	return nil
}

func validateProductionCandidateRefs(refs []string, field string) error {
	if len(refs) != 2 {
		return fmt.Errorf("%s must contain exactly Docker Hub and GHCR candidates", field)
	}
	seenRefs := make(map[string]struct{}, len(refs))
	var digest string
	expectedRegistries := [...]string{"docker.io", "ghcr.io"}
	for index, rawRef := range refs {
		ref, err := ociartifact.ParseDigestReference(rawRef)
		if err != nil {
			return fmt.Errorf("%s[%d]: %w", field, index, err)
		}
		if _, duplicate := seenRefs[ref.String()]; duplicate {
			return fmt.Errorf("%s contains duplicate ref %q", field, ref.String())
		}
		seenRefs[ref.String()] = struct{}{}
		if ref.Registry != expectedRegistries[index] {
			return fmt.Errorf("%s must order docker.io then ghcr.io candidates", field)
		}
		if digest == "" {
			digest = ref.Digest
		} else if digest != ref.Digest {
			return fmt.Errorf("%s candidates must use the same manifest digest", field)
		}
	}
	return nil
}

func parse(raw []byte) (*Manifest, error) {
	decoder := yaml.NewDecoder(strings.NewReader(string(raw)))
	decoder.KnownFields(true)
	var result Manifest
	if err := decoder.Decode(&result); err != nil {
		return nil, err
	}
	var trailing yaml.Node
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err != nil {
			return nil, err
		}
		return nil, fmt.Errorf("release manifest must contain exactly one YAML document")
	}
	return &result, nil
}
