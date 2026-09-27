package upgrader

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/yyhuni/lunafox/contracts/preheatmanifest"
	"github.com/yyhuni/lunafox/contracts/releasemanifest"
)

const (
	maxReleaseManifestSnapshotBytes    = 4 << 20
	maxRuntimeCompositionSnapshotBytes = 8 << 20
	maxPreheatManifestSnapshotBytes    = 8 << 20
	maxPublicComposeSnapshotBytes      = 16 << 20
	maxPublicPolicySnapshotBytes       = 2 << 20
	maxPublicEnvironmentBytes          = 1 << 20
	preheatManifestDigestKey           = "LUNAFOX_PREHEAT_MANIFEST_DIGEST"
)

var publicDeploymentSnapshotFiles = []string{
	publicComposeFile,
	publicEnvFile,
	defaultManifestName,
	publicRuntimeCompositionFile,
	publicThirdPartyPolicyFile,
	publicPreheatManifestFile,
}

const (
	// These files are private promotion evidence, not Compose inputs. They record
	// the active deployment bytes that were checked before the candidate was
	// staged, so a target release may legitimately contain different Compose or
	// policy bytes without weakening the concurrent-writer fence.
	deploymentSnapshotBaselineComposeFile = ".baseline-compose.yaml"
	deploymentSnapshotBaselinePolicyFile  = ".baseline-third-party-image-policy.json"
)

var publicDeploymentSnapshotBaselineFiles = []string{
	deploymentSnapshotBaselineComposeFile,
	deploymentSnapshotBaselinePolicyFile,
}

// publicDeploymentSnapshot is an immutable, fully checked candidate. The
// runtime cache is the handoff from the unprivileged Server to the socket-owning
// upgrader; root Compose and policy bytes are copied into the snapshot only
// after proving that the target preheat manifest binds them exactly.
type publicDeploymentSnapshot struct {
	compose         []byte
	environment     []byte
	manifest        []byte
	composition     []byte
	policy          []byte
	preheat         []byte
	baselineCompose []byte
	baselinePolicy  []byte
	releaseDigest   string
}

func (snapshot publicDeploymentSnapshot) file(name string) ([]byte, error) {
	switch name {
	case publicComposeFile:
		return snapshot.compose, nil
	case publicEnvFile:
		return snapshot.environment, nil
	case defaultManifestName:
		return snapshot.manifest, nil
	case publicRuntimeCompositionFile:
		return snapshot.composition, nil
	case publicThirdPartyPolicyFile:
		return snapshot.policy, nil
	case publicPreheatManifestFile:
		return snapshot.preheat, nil
	default:
		return nil, fmt.Errorf("unsupported deployment snapshot file %q", name)
	}
}

func (snapshot publicDeploymentSnapshot) baselineFile(name string) ([]byte, error) {
	switch name {
	case deploymentSnapshotBaselineComposeFile:
		return snapshot.baselineCompose, nil
	case deploymentSnapshotBaselinePolicyFile:
		return snapshot.baselinePolicy, nil
	default:
		return nil, fmt.Errorf("unsupported deployment snapshot baseline file %q", name)
	}
}

// preparePublicDeploymentSnapshot materializes a modern candidate into a
// private staging directory and validates every binding before returning it.
// Promotion is deliberately a separate step performed only after the target
// engine-preheater succeeds, so a failed preheat leaves the active deployment
// files untouched.
func (executor *ComposeExecutor) preparePublicDeploymentSnapshot(store *JournalStore, releaseDigest string, release *releasemanifest.Manifest) (string, publicDeploymentSnapshot, error) {
	if executor == nil || !executor.PublicLayout {
		return "", publicDeploymentSnapshot{}, nil
	}
	if store == nil || release == nil || !release.HasRuntimeComposition() {
		return "", publicDeploymentSnapshot{}, fmt.Errorf("modern public deployment snapshot requires release evidence")
	}
	if err := store.validatePrivateLayout(); err != nil {
		return "", publicDeploymentSnapshot{}, err
	}
	// If the live environment already names this exact preheat digest, its
	// release-bound files are the active candidate rather than an upgrade input.
	// Never repair such a candidate from the private cache: doing so could hide a
	// tampered public file and would violate the fail-closed boundary.
	targetPreheatPath, err := store.PreheatManifestPath(releaseDigest)
	if err != nil {
		return "", publicDeploymentSnapshot{}, err
	}
	targetPreheat, err := readDeploymentEvidence(targetPreheatPath, maxPreheatManifestSnapshotBytes, true)
	if err != nil {
		return "", publicDeploymentSnapshot{}, fmt.Errorf("read cached preheat manifest: %w", err)
	}
	parsedTargetPreheat, err := preheatmanifest.Parse(targetPreheat)
	if err != nil {
		return "", publicDeploymentSnapshot{}, fmt.Errorf("validate cached preheat manifest: %w", err)
	}
	environmentPath := filepath.Join(store.DeploymentRoot(), publicEnvFile)
	if environment, readErr := readDeploymentEvidence(environmentPath, maxPublicEnvironmentBytes, false); readErr == nil {
		if currentDigest, digestErr := preheatManifestDigestFromEnvironment(environment); digestErr == nil && currentDigest == parsedTargetPreheat.ManifestDigest {
			current, currentErr := publicDeploymentSnapshotFromRoot(store.DeploymentRoot(), releaseDigest)
			if currentErr != nil {
				return "", publicDeploymentSnapshot{}, currentErr
			}
			if !bytes.Equal(current.preheat, targetPreheat) {
				return "", publicDeploymentSnapshot{}, fmt.Errorf("active preheat manifest bytes do not match the verified cache")
			}
			if err := validatePublicDeploymentSnapshot(current, release); err != nil {
				return "", publicDeploymentSnapshot{}, err
			}
		}
	}
	snapshotPath, err := executor.stagePublicDeploymentSnapshot(store, releaseDigest, release)
	if err != nil {
		return "", publicDeploymentSnapshot{}, err
	}
	snapshot, err := loadPublicDeploymentSnapshot(snapshotPath, releaseDigest)
	if err != nil {
		return "", publicDeploymentSnapshot{}, err
	}
	if err := validatePublicDeploymentSnapshot(snapshot, release); err != nil {
		return "", publicDeploymentSnapshot{}, err
	}
	return snapshotPath, snapshot, nil
}

func publicDeploymentSnapshotFromRoot(root, releaseDigest string) (publicDeploymentSnapshot, error) {
	read := func(name string, limit int64) ([]byte, error) {
		data, err := readDeploymentEvidence(filepath.Join(root, name), limit, false)
		if err != nil {
			return nil, fmt.Errorf("read active deployment snapshot %s: %w", name, err)
		}
		return data, nil
	}
	manifest, err := read(defaultManifestName, maxReleaseManifestSnapshotBytes)
	if err != nil {
		return publicDeploymentSnapshot{}, err
	}
	composition, err := read(publicRuntimeCompositionFile, maxRuntimeCompositionSnapshotBytes)
	if err != nil {
		return publicDeploymentSnapshot{}, err
	}
	preheat, err := read(publicPreheatManifestFile, maxPreheatManifestSnapshotBytes)
	if err != nil {
		return publicDeploymentSnapshot{}, err
	}
	compose, err := read(publicComposeFile, maxPublicComposeSnapshotBytes)
	if err != nil {
		return publicDeploymentSnapshot{}, err
	}
	policy, err := read(publicThirdPartyPolicyFile, maxPublicPolicySnapshotBytes)
	if err != nil {
		return publicDeploymentSnapshot{}, err
	}
	environment, err := read(publicEnvFile, maxPublicEnvironmentBytes)
	if err != nil {
		return publicDeploymentSnapshot{}, err
	}
	return publicDeploymentSnapshot{
		compose:         compose,
		environment:     environment,
		manifest:        manifest,
		composition:     composition,
		policy:          policy,
		preheat:         preheat,
		baselineCompose: compose,
		baselinePolicy:  policy,
		releaseDigest:   releaseDigest,
	}, nil
}

func (executor *ComposeExecutor) stagePublicDeploymentSnapshot(store *JournalStore, releaseDigest string, release *releasemanifest.Manifest) (string, error) {
	snapshotPath, err := store.DeploymentSnapshotPath(releaseDigest)
	if err != nil {
		return "", err
	}
	if info, statErr := os.Lstat(snapshotPath); statErr == nil {
		if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() || info.Mode().Perm() != 0o700 {
			return "", fmt.Errorf("staged deployment snapshot must be a regular 0700 directory")
		}
		return snapshotPath, nil
	} else if !os.IsNotExist(statErr) {
		return "", statErr
	}

	snapshot, err := buildPublicDeploymentSnapshot(store, releaseDigest, release)
	if err != nil {
		return "", err
	}
	if err := validatePublicDeploymentSnapshot(snapshot, release); err != nil {
		return "", err
	}

	parent := filepath.Dir(snapshotPath)
	staging, err := os.MkdirTemp(parent, ".staging-")
	if err != nil {
		return "", err
	}
	removeStaging := true
	defer func() {
		if removeStaging {
			_ = os.RemoveAll(staging)
		}
	}()
	if err := os.Chmod(staging, 0o700); err != nil {
		return "", err
	}
	for _, name := range publicDeploymentSnapshotFiles {
		data, err := snapshot.file(name)
		if err != nil {
			return "", err
		}
		if err := atomicWrite(filepath.Join(staging, name), data, privateUpgradeFileMode); err != nil {
			return "", fmt.Errorf("stage deployment snapshot %s: %w", name, err)
		}
	}
	for _, name := range publicDeploymentSnapshotBaselineFiles {
		data, err := snapshot.baselineFile(name)
		if err != nil {
			return "", err
		}
		if err := atomicWrite(filepath.Join(staging, name), data, privateUpgradeFileMode); err != nil {
			return "", fmt.Errorf("stage deployment snapshot baseline %s: %w", name, err)
		}
	}
	if err := os.Rename(staging, snapshotPath); err != nil {
		if !os.IsExist(err) {
			return "", err
		}
		// A same-digest snapshot appearing during staging can only be accepted
		// after it passes the normal load/validation path below.
	} else {
		removeStaging = false
		if err := syncDirectory(parent); err != nil {
			return "", err
		}
		return snapshotPath, nil
	}

	if info, statErr := os.Lstat(snapshotPath); statErr != nil || info.Mode()&os.ModeSymlink != 0 || !info.IsDir() || info.Mode().Perm() != 0o700 {
		return "", fmt.Errorf("staged deployment snapshot appeared with an unsafe shape")
	}
	return snapshotPath, nil
}

func buildPublicDeploymentSnapshot(store *JournalStore, releaseDigest string, release *releasemanifest.Manifest) (publicDeploymentSnapshot, error) {
	if store == nil || release == nil {
		return publicDeploymentSnapshot{}, fmt.Errorf("deployment snapshot requires a journal store and release manifest")
	}
	manifestPath, err := store.ManifestPath(releaseDigest)
	if err != nil {
		return publicDeploymentSnapshot{}, err
	}
	manifest, err := readDeploymentEvidence(manifestPath, maxReleaseManifestSnapshotBytes, true)
	if err != nil {
		return publicDeploymentSnapshot{}, fmt.Errorf("read cached release manifest: %w", err)
	}
	compositionPath, err := store.RuntimeCompositionPath(release.RuntimeComposition.SHA256)
	if err != nil {
		return publicDeploymentSnapshot{}, err
	}
	composition, err := readDeploymentEvidence(compositionPath, maxRuntimeCompositionSnapshotBytes, true)
	if err != nil {
		return publicDeploymentSnapshot{}, fmt.Errorf("read cached runtime composition: %w", err)
	}
	preheatPath, err := store.PreheatManifestPath(releaseDigest)
	if err != nil {
		return publicDeploymentSnapshot{}, err
	}
	preheat, err := readDeploymentEvidence(preheatPath, maxPreheatManifestSnapshotBytes, true)
	if err != nil {
		return publicDeploymentSnapshot{}, fmt.Errorf("read cached preheat manifest: %w", err)
	}
	assetRoot, err := store.DeploymentAssetsPath(releaseDigest)
	if err != nil {
		return publicDeploymentSnapshot{}, err
	}
	compose, err := readDeploymentEvidence(filepath.Join(assetRoot, publicComposeFile), maxPublicComposeSnapshotBytes, true)
	if err != nil {
		return publicDeploymentSnapshot{}, fmt.Errorf("read cached target Compose snapshot: %w", err)
	}
	policy, err := readDeploymentEvidence(filepath.Join(assetRoot, publicThirdPartyPolicyFile), maxPublicPolicySnapshotBytes, true)
	if err != nil {
		return publicDeploymentSnapshot{}, fmt.Errorf("read cached target third-party policy snapshot: %w", err)
	}
	root := store.DeploymentRoot()
	baselineCompose, err := readDeploymentEvidence(filepath.Join(root, publicComposeFile), maxPublicComposeSnapshotBytes, false)
	if err != nil {
		return publicDeploymentSnapshot{}, fmt.Errorf("read current deployment Compose baseline: %w", err)
	}
	baselinePolicy, err := readDeploymentEvidence(filepath.Join(root, publicThirdPartyPolicyFile), maxPublicPolicySnapshotBytes, false)
	if err != nil {
		return publicDeploymentSnapshot{}, fmt.Errorf("read current third-party policy baseline: %w", err)
	}
	environment, err := readDeploymentEvidence(filepath.Join(root, publicEnvFile), maxPublicEnvironmentBytes, false)
	if err != nil {
		return publicDeploymentSnapshot{}, fmt.Errorf("read current deployment environment: %w", err)
	}
	parsedPreheat, err := preheatmanifest.Parse(preheat)
	if err != nil {
		return publicDeploymentSnapshot{}, err
	}
	environment, err = replacePreheatManifestDigest(environment, parsedPreheat.ManifestDigest)
	if err != nil {
		return publicDeploymentSnapshot{}, err
	}
	return publicDeploymentSnapshot{
		compose:         compose,
		environment:     environment,
		manifest:        manifest,
		composition:     composition,
		policy:          policy,
		preheat:         preheat,
		baselineCompose: baselineCompose,
		baselinePolicy:  baselinePolicy,
		releaseDigest:   releaseDigest,
	}, nil
}

func loadPublicDeploymentSnapshot(directory, releaseDigest string) (publicDeploymentSnapshot, error) {
	if strings.TrimSpace(directory) == "" {
		return publicDeploymentSnapshot{}, fmt.Errorf("deployment snapshot directory is required")
	}
	info, err := os.Lstat(directory)
	if err != nil || info.Mode()&os.ModeSymlink != 0 || !info.IsDir() || info.Mode().Perm() != 0o700 {
		return publicDeploymentSnapshot{}, fmt.Errorf("staged deployment snapshot is unavailable")
	}
	files := make(map[string][]byte, len(publicDeploymentSnapshotFiles))
	for _, name := range publicDeploymentSnapshotFiles {
		limit := int64(maxPublicComposeSnapshotBytes)
		switch name {
		case publicEnvFile:
			limit = maxPublicEnvironmentBytes
		case defaultManifestName:
			limit = maxReleaseManifestSnapshotBytes
		case publicRuntimeCompositionFile:
			limit = maxRuntimeCompositionSnapshotBytes
		case publicPreheatManifestFile:
			limit = maxPreheatManifestSnapshotBytes
		case publicThirdPartyPolicyFile:
			limit = maxPublicPolicySnapshotBytes
		}
		data, err := readDeploymentEvidence(filepath.Join(directory, name), limit, true)
		if err != nil {
			return publicDeploymentSnapshot{}, fmt.Errorf("read staged deployment snapshot %s: %w", name, err)
		}
		files[name] = data
	}
	baselineCompose, err := readDeploymentEvidence(filepath.Join(directory, deploymentSnapshotBaselineComposeFile), maxPublicComposeSnapshotBytes, true)
	if err != nil {
		return publicDeploymentSnapshot{}, fmt.Errorf("read staged deployment snapshot baseline %s: %w", deploymentSnapshotBaselineComposeFile, err)
	}
	baselinePolicy, err := readDeploymentEvidence(filepath.Join(directory, deploymentSnapshotBaselinePolicyFile), maxPublicPolicySnapshotBytes, true)
	if err != nil {
		return publicDeploymentSnapshot{}, fmt.Errorf("read staged deployment snapshot baseline %s: %w", deploymentSnapshotBaselinePolicyFile, err)
	}
	return publicDeploymentSnapshot{
		compose:         files[publicComposeFile],
		environment:     files[publicEnvFile],
		manifest:        files[defaultManifestName],
		composition:     files[publicRuntimeCompositionFile],
		policy:          files[publicThirdPartyPolicyFile],
		preheat:         files[publicPreheatManifestFile],
		baselineCompose: baselineCompose,
		baselinePolicy:  baselinePolicy,
		releaseDigest:   releaseDigest,
	}, nil
}

func validatePublicDeploymentSnapshot(snapshot publicDeploymentSnapshot, expected *releasemanifest.Manifest) error {
	if expected == nil || !expected.HasRuntimeComposition() {
		return fmt.Errorf("deployment snapshot requires a modern release manifest")
	}
	if len(snapshot.baselineCompose) == 0 || len(snapshot.baselinePolicy) == 0 {
		return fmt.Errorf("deployment snapshot is missing promotion baseline evidence")
	}
	manifest, err := releasemanifest.Parse(snapshot.manifest)
	if err != nil {
		return fmt.Errorf("validate staged release manifest: %w", err)
	}
	if manifest.Digest() != snapshot.releaseDigest || manifest.Digest() != expected.Digest() {
		return fmt.Errorf("staged release manifest digest does not match target")
	}
	if err := ValidateRuntimeCompositionAsset(snapshot.composition, manifest, snapshot.releaseDigest, manifest.RuntimeComposition.SHA256); err != nil {
		return fmt.Errorf("validate staged runtime composition: %w", err)
	}
	preheat, err := preheatmanifest.Parse(snapshot.preheat)
	if err != nil {
		return fmt.Errorf("validate staged preheat manifest: %w", err)
	}
	if preheat.Release.ManifestDigest != snapshot.releaseDigest || preheat.Release.Tag != "v"+manifest.ReleaseVersion || preheat.Release.CompositionDigest != manifest.RuntimeComposition.SHA256 {
		return fmt.Errorf("staged preheat manifest release binding does not match target")
	}
	profile, err := deploymentPreheatProfileFromEnvironment(snapshot.environment)
	if err != nil {
		return err
	}
	if !profileClosurePresent(preheat, profile) {
		return fmt.Errorf("staged preheat manifest has no closure for profile %q", profile)
	}
	if err := preheat.ValidateBindings(preheatmanifest.BindingInputs{
		ReleaseManifest:      snapshot.manifest,
		RuntimeComposition:   snapshot.composition,
		Compose:              snapshot.compose,
		ThirdPartyPolicy:     snapshot.policy,
		ExpectedManifestHash: preheat.ManifestDigest,
		Profile:              profile,
	}); err != nil {
		return fmt.Errorf("validate staged preheat bindings: %w", err)
	}
	actualDigest, err := preheatManifestDigestFromEnvironment(snapshot.environment)
	if err != nil {
		return err
	}
	if actualDigest != preheat.ManifestDigest {
		return fmt.Errorf("staged deployment environment does not bind the preheat manifest")
	}
	return nil
}

func promotePublicDeploymentSnapshotFiles(root string, snapshot publicDeploymentSnapshot) error {
	if strings.TrimSpace(root) == "" {
		return fmt.Errorf("deployment root is required")
	}
	// Protect the staging proof from an out-of-band edit immediately before the
	// first destination replacement. The lifecycle/upgrade lock covers supported
	// writers; this extra comparison makes a manual concurrent replacement fail
	// closed as well.
	for _, baseline := range []struct {
		name     string
		contents []byte
		limit    int64
	}{
		{name: publicComposeFile, contents: snapshot.baselineCompose, limit: maxPublicComposeSnapshotBytes},
		{name: publicThirdPartyPolicyFile, contents: snapshot.baselinePolicy, limit: maxPublicPolicySnapshotBytes},
	} {
		current, err := readDeploymentEvidence(filepath.Join(root, baseline.name), baseline.limit, false)
		if err != nil {
			return fmt.Errorf("read current deployment snapshot %s: %w", baseline.name, err)
		}
		if !bytes.Equal(current, baseline.contents) {
			return fmt.Errorf("current deployment %s changed after snapshot validation", baseline.name)
		}
	}
	// The evidence files are individually atomic writes. The environment
	// fingerprint is intentionally last: before it changes, Compose still names
	// the old preheat manifest and cannot run a mixed candidate after a crash.
	order := []string{
		defaultManifestName,
		publicRuntimeCompositionFile,
		publicPreheatManifestFile,
		publicComposeFile,
		publicThirdPartyPolicyFile,
		publicEnvFile,
	}
	for _, name := range order {
		path := filepath.Join(root, name)
		info, err := os.Lstat(path)
		if err != nil || info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
			return fmt.Errorf("deployment snapshot destination %s is not a regular file", name)
		}
		data, err := snapshot.file(name)
		if err != nil {
			return err
		}
		if err := atomicWrite(path, data, info.Mode().Perm()); err != nil {
			return fmt.Errorf("promote deployment snapshot %s: %w", name, err)
		}
	}
	return nil
}

func replacePreheatManifestDigest(environment []byte, digest string) ([]byte, error) {
	if !digestPattern.MatchString(digest) {
		return nil, fmt.Errorf("preheat manifest digest is invalid")
	}
	lines := strings.Split(string(environment), "\n")
	found := false
	for index, line := range lines {
		key, value, hasValue := strings.Cut(line, "=")
		if key != preheatManifestDigestKey {
			continue
		}
		if !hasValue || found || strings.TrimSpace(key) != key || strings.TrimSpace(value) != value || !digestPattern.MatchString(strings.Trim(value, "\"'")) {
			return nil, fmt.Errorf("deployment environment has an invalid %s entry", preheatManifestDigestKey)
		}
		lines[index] = preheatManifestDigestKey + "=" + digest
		found = true
	}
	if !found {
		return nil, fmt.Errorf("deployment environment is missing %s", preheatManifestDigestKey)
	}
	return []byte(strings.Join(lines, "\n")), nil
}

func preheatManifestDigestFromEnvironment(environment []byte) (string, error) {
	values, err := parseDeploymentEnvironment(environment)
	if err != nil {
		return "", err
	}
	digest := values[preheatManifestDigestKey]
	if !digestPattern.MatchString(digest) {
		return "", fmt.Errorf("deployment environment has an invalid %s entry", preheatManifestDigestKey)
	}
	return digest, nil
}

func deploymentPreheatProfileFromEnvironment(environment []byte) (string, error) {
	values, err := parseDeploymentEnvironment(environment)
	if err != nil {
		return "", err
	}
	profile := values["DATABASE_MODE"]
	if profile == "" {
		profile = preheatmanifest.ProfileEmbedded
	}
	if profile != preheatmanifest.ProfileEmbedded && profile != preheatmanifest.ProfileExternal {
		return "", fmt.Errorf("DATABASE_MODE must be embedded or external")
	}
	composeProfiles := values["COMPOSE_PROFILES"]
	if composeProfiles == "" {
		composeProfiles = profile
	}
	if composeProfiles != profile {
		return "", fmt.Errorf("COMPOSE_PROFILES must match DATABASE_MODE")
	}
	return profile, nil
}

func parseDeploymentEnvironment(environment []byte) (map[string]string, error) {
	values := make(map[string]string)
	for lineNumber, raw := range strings.Split(string(environment), "\n") {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, found := strings.Cut(line, "=")
		if !found || strings.TrimSpace(key) != key || key == "" {
			return nil, fmt.Errorf("deployment environment line %d is invalid", lineNumber+1)
		}
		if _, exists := values[key]; exists {
			return nil, fmt.Errorf("deployment environment contains duplicate %s", key)
		}
		value = strings.TrimSpace(value)
		if len(value) >= 2 && ((value[0] == '\'' && value[len(value)-1] == '\'') || (value[0] == '"' && value[len(value)-1] == '"')) {
			value = value[1 : len(value)-1]
		}
		if strings.ContainsAny(value, "\r\n") {
			return nil, fmt.Errorf("deployment environment value %s contains a control character", key)
		}
		values[key] = value
	}
	return values, nil
}
