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
	"unicode/utf8"

	"github.com/blang/semver"
	"github.com/yyhuni/lunafox/contracts/ociartifact"
	"github.com/yyhuni/lunafox/contracts/ocidistribution"
	"go.yaml.in/yaml/v3"
)

// Manifest is a strict, immutable deployment inventory. Its digest is the
// SHA-256 of the exact YAML bytes accepted by Parse or Load.
type Manifest struct {
	ReleaseVersion     string             `yaml:"releaseVersion"`
	ReleaseNotes       *ReleaseNotes      `yaml:"releaseNotes,omitempty"`
	RuntimeImages      []RuntimeImage     `yaml:"runtimeImages"`
	EnginePackages     []EnginePackage    `yaml:"enginePackages"`
	RuntimeComposition RuntimeComposition `yaml:"runtimeComposition"`
	Upgrade            UpgradeMetadata    `yaml:"upgrade"`

	digest string
}

// ReleaseNotes is the public, user-facing explanation bound to a release.
// Digest is calculated over Body's canonical UTF-8 bytes, including its final
// newline. Keeping both values in the signed manifest prevents a consumer from
// silently substituting notes from another tag.
type ReleaseNotes struct {
	Body   string `yaml:"body"`
	Digest string `yaml:"digest"`
}

type RuntimeImage struct {
	Name string   `yaml:"name"`
	Refs []string `yaml:"refs"`
}

type EnginePackage struct {
	Refs []string `yaml:"refs"`
}

// RuntimeComposition identifies the canonical, provenance-bound component
// composition used to produce this release manifest. SHA256 is the digest of
// the composition's canonical core payload; the complete JSON asset digest is
// bound separately by release provenance to avoid a manifest/composition hash
// cycle.
type RuntimeComposition struct {
	SchemaVersion int    `yaml:"schemaVersion"`
	Asset         string `yaml:"asset"`
	SHA256        string `yaml:"sha256"`
}

// LegacyAlpha114ReleaseVersion and LegacyAlpha114ManifestDigest identify the
// single public bootstrap release that predates runtime-composition evidence.
// Keep this exception pinned to the reviewed bytes; ordinary Parse/Load calls
// must continue to require a runtimeComposition binding.
const (
	LegacyAlpha114ReleaseVersion = "0.0.1-alpha.114"
	LegacyAlpha114ManifestDigest = "sha256:e0e742054888daf0fb6482be721c82bd8e162d795143badbf2089cbd978c5e8b"
)

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
var releaseNotesHeadingPattern = regexp.MustCompile(`^##[ \t]+[^\n]+$`)

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

// LoadLegacyAlpha114 is the file-oriented counterpart to ParseLegacyAlpha114.
// It is intended only for the host's fixed legacy deployment path; callers
// handling ordinary release-channel candidates must continue to use Load.
func LoadLegacyAlpha114(filePath string) (*Manifest, error) {
	raw, err := os.ReadFile(strings.TrimSpace(filePath))
	if err != nil {
		return nil, fmt.Errorf("failed to read legacy release manifest: %w", err)
	}
	return ParseLegacyAlpha114(raw)
}

// Parse validates exactly one manifest document and retains its content hash.
func Parse(raw []byte) (*Manifest, error) {
	return parseManifest(raw, false)
}

// ParseLegacyAlpha114 parses only the policy-pinned v1 bootstrap manifest.
// This is intentionally separate from Parse so a missing composition cannot
// silently become valid for ordinary releases.
func ParseLegacyAlpha114(raw []byte) (*Manifest, error) {
	manifest, err := parseManifest(raw, true)
	if err != nil {
		return nil, err
	}
	if manifest.ReleaseVersion != LegacyAlpha114ReleaseVersion || manifest.Digest() != LegacyAlpha114ManifestDigest {
		return nil, fmt.Errorf("legacy alpha.114 manifest identity is not policy-pinned")
	}
	return manifest, nil
}

func parseManifest(raw []byte, allowLegacyAlpha114 bool) (*Manifest, error) {
	manifest, err := parse(raw)
	if err != nil {
		return nil, fmt.Errorf("failed to parse release manifest: %w", err)
	}
	if err := manifest.normalizeAndValidate(allowLegacyAlpha114); err != nil {
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

func (manifest *Manifest) normalizeAndValidate(allowLegacyAlpha114 bool) error {
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
	if err := validateUpgradeMetadata(manifest.Upgrade); err != nil {
		return err
	}
	if manifest.RuntimeComposition.SchemaVersion == 0 && manifest.RuntimeComposition.Asset == "" && manifest.RuntimeComposition.SHA256 == "" {
		if !allowLegacyAlpha114 || manifest.ReleaseVersion != LegacyAlpha114ReleaseVersion {
			return fmt.Errorf("runtimeComposition.schemaVersion must be 1")
		}
	} else if err := validateRuntimeComposition(manifest.RuntimeComposition); err != nil {
		return err
	}
	if manifest.ReleaseNotes == nil && allowLegacyAlpha114 && manifest.ReleaseVersion == LegacyAlpha114ReleaseVersion {
		// The pinned alpha.114 bootstrap predates both public release notes and
		// runtime-composition evidence; ParseLegacyAlpha114 verifies its exact
		// raw digest immediately after this compatibility check.
		return nil
	}
	return validateReleaseNotes(manifest.ReleaseVersion, manifest.ReleaseNotes)
}

func validateRuntimeComposition(composition RuntimeComposition) error {
	if composition.SchemaVersion != 1 {
		return fmt.Errorf("runtimeComposition.schemaVersion must be 1")
	}
	if composition.Asset != "runtime-composition.json" {
		return fmt.Errorf("runtimeComposition.asset must be runtime-composition.json")
	}
	if !checksumPattern.MatchString(composition.SHA256) {
		return fmt.Errorf("runtimeComposition.sha256 must be a sha256 digest")
	}
	return nil
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

func validateReleaseNotes(version string, notes *ReleaseNotes) error {
	if notes == nil {
		// The repository's development manifest intentionally has no public
		// release note. Published versions must always bind one.
		if version == "0.0.0-dev" {
			return nil
		}
		return fmt.Errorf("releaseNotes is required for releaseVersion %s", version)
	}
	body := notes.Body
	if !utf8.ValidString(body) {
		return fmt.Errorf("releaseNotes.body must be valid UTF-8")
	}
	if len([]byte(body)) == 0 || strings.TrimSpace(body) == "" {
		return fmt.Errorf("releaseNotes.body must be non-empty")
	}
	if strings.ContainsAny(body, "\r\x00\uFFFD") || strings.ContainsAny(body, "\u0001\u0002\u0003\u0004\u0005\u0006\u0007\u0008\u000B\u000C\u000E\u000F\u0010\u0011\u0012\u0013\u0014\u0015\u0016\u0017\u0018\u0019\u001A\u001B\u001C\u001D\u001E\u001F\u007F") {
		return fmt.Errorf("releaseNotes.body must use canonical UTF-8 LF text")
	}
	if len([]byte(body)) > 64*1024 {
		return fmt.Errorf("releaseNotes.body must not exceed 65536 bytes")
	}
	if !strings.HasSuffix(body, "\n") {
		return fmt.Errorf("releaseNotes.body must end with a newline")
	}
	lines := strings.Split(body, "\n")
	sections := make(map[string]struct{}, 2)
	for index, line := range lines {
		heading := strings.TrimRight(line, " \t")
		if heading != "## English" && heading != "## 简体中文" {
			continue
		}
		name := heading[3:]
		if _, duplicate := sections[name]; duplicate {
			return fmt.Errorf("releaseNotes.body must contain exactly one English and one 简体中文 section")
		}
		sections[name] = struct{}{}
		end := len(lines) - 1
		for next := index + 1; next < len(lines); next++ {
			if releaseNotesHeadingPattern.MatchString(lines[next]) {
				end = next
				break
			}
		}
		if strings.TrimSpace(strings.Join(lines[index+1:end], "\n")) == "" {
			return fmt.Errorf("releaseNotes sections must be non-empty")
		}
	}
	if len(sections) != 2 {
		return fmt.Errorf("releaseNotes.body must contain exactly one English and one 简体中文 section")
	}
	if !checksumPattern.MatchString(notes.Digest) {
		return fmt.Errorf("releaseNotes.digest must be a sha256 digest")
	}
	hash := sha256.Sum256([]byte(body))
	want := "sha256:" + fmt.Sprintf("%x", hash[:])
	if notes.Digest != want {
		return fmt.Errorf("releaseNotes.digest does not match releaseNotes.body")
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
