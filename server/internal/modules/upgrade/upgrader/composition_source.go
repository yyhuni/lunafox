package upgrader

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/yyhuni/lunafox/contracts/ociartifact"
	"github.com/yyhuni/lunafox/contracts/releasemanifest"
)

const (
	runtimeCompositionSchemaVersion = 1
	// Fingerprint version 2 includes the Unix mode of every effective context
	// file. Without it, a chmod-only Docker COPY change could reuse an image
	// whose executable boundary no longer matches the release source.
	runtimeCompositionFingerprintSchemaVersion = 2
	runtimeCompositionKind                     = "lunafox.runtime-composition"
	runtimeCompositionAssetName                = "runtime-composition.json"
	maxRuntimeCompositionBytes                 = 8 << 20
)

var (
	ErrCompositionUnavailable = errors.New("runtime composition evidence is unavailable")
	ErrCompositionInvalid     = errors.New("runtime composition evidence is invalid")
	compositionReleaseTagRE   = regexp.MustCompile(`^v?\d+\.\d+\.\d+(?:-[0-9A-Za-z.-]+)?$`)
	compositionComponentIDRE  = regexp.MustCompile(`^(?:runtime\.(?:server|frontend|nginx|agent|bootstrap)|agent\.[a-z][a-z0-9_-]*|engine\.[a-z][a-z0-9_.-]*\.(?:runtime|package))$`)
	compositionNameRE         = regexp.MustCompile(`^[a-z][a-z0-9_.-]*$`)
	compositionPlatformRE     = regexp.MustCompile(`^[a-z0-9]+/[a-z0-9]+$`)
	compositionCommitRE       = regexp.MustCompile(`^[0-9a-f]{40}$`)
)

// CompositionCandidateDeploymentSource loads the release composition from the
// host's digest-addressed cache. The manifest remains the immutable desired
// state; the composition supplies the complete, provenance-bound inventory
// needed to distinguish a frontend-only change from a metadata-only digest
// difference.
type CompositionCandidateDeploymentSource struct {
	store *JournalStore
}

func NewCompositionCandidateDeploymentSource(store *JournalStore) *CompositionCandidateDeploymentSource {
	return &CompositionCandidateDeploymentSource{store: store}
}

// ValidateRuntimeCompositionAsset validates one release-owned composition
// asset against the already parsed manifest.  The Server-side release source
// uses this narrow boundary after downloading the fixed asset; canonical JSON
// normalization, component inventory checks, and digest binding remain owned
// by the upgrader package so the two control planes cannot drift.
func ValidateRuntimeCompositionAsset(raw []byte, manifest *releasemanifest.Manifest, manifestDigest, expectedDigest string) error {
	if manifest == nil {
		return fmt.Errorf("%w: release manifest is required", ErrCompositionInvalid)
	}
	if !manifest.HasRuntimeComposition() {
		return fmt.Errorf("%w: manifest has no runtimeComposition binding", ErrCompositionUnavailable)
	}
	if manifest.RuntimeComposition.SchemaVersion != runtimeCompositionSchemaVersion ||
		manifest.RuntimeComposition.Asset != runtimeCompositionAssetName ||
		!digestPattern.MatchString(manifest.RuntimeComposition.SHA256) {
		return fmt.Errorf("%w: manifest runtimeComposition binding is malformed", ErrCompositionInvalid)
	}
	if expectedDigest == "" {
		expectedDigest = manifest.RuntimeComposition.SHA256
	}
	if expectedDigest != manifest.RuntimeComposition.SHA256 {
		return fmt.Errorf("%w: expected composition digest does not match manifest binding", ErrCompositionInvalid)
	}
	if manifestDigest == "" || !digestPattern.MatchString(manifestDigest) {
		return fmt.Errorf("%w: manifest digest is invalid", ErrCompositionInvalid)
	}
	if manifest.Digest() != "" && manifest.Digest() != manifestDigest {
		return fmt.Errorf("%w: manifest digest does not match parsed bytes", ErrCompositionInvalid)
	}
	_, err := validateRuntimeComposition(raw, manifest, manifestDigest, expectedDigest)
	return err
}

func (source *CompositionCandidateDeploymentSource) LoadCandidateDeployment(_ context.Context, manifestDigest string) (CandidateDeployment, error) {
	if source == nil || source.store == nil {
		return CandidateDeployment{}, fmt.Errorf("composition candidate source is not configured")
	}
	manifestPath, err := source.store.ManifestPath(manifestDigest)
	if err != nil {
		return CandidateDeployment{}, err
	}
	rawManifest, err := readRegularPrivateFile(manifestPath, maxRuntimeCompositionBytes)
	if err != nil {
		return CandidateDeployment{}, fmt.Errorf("load candidate release manifest: %w", err)
	}
	manifest, err := releasemanifest.Parse(rawManifest)
	if err != nil {
		// Composition lookup is a deployment boundary. It accepts only the
		// policy-pinned alpha.114 bytes or the exact alpha.164 bridge profile;
		// every other malformed or incomplete Manifest remains a hard failure.
		manifest, err = releasemanifest.ParseLegacyCompatible(rawManifest)
		if err != nil {
			return CandidateDeployment{}, fmt.Errorf("validate candidate release manifest: %w", err)
		}
	}
	if manifest.Digest() != manifestDigest {
		return CandidateDeployment{}, ErrManifestMismatch
	}
	candidate, err := CandidateDeploymentFromManifest(manifest)
	if err != nil {
		return CandidateDeployment{}, err
	}
	if !manifest.HasRuntimeComposition() {
		return candidate, fmt.Errorf("%w: legacy-compatible manifest has no composition asset", ErrCompositionUnavailable)
	}
	binding := manifest.RuntimeComposition
	if binding.SchemaVersion != runtimeCompositionSchemaVersion || binding.Asset != runtimeCompositionAssetName || !digestPattern.MatchString(binding.SHA256) {
		return CandidateDeployment{}, fmt.Errorf("%w: manifest runtimeComposition binding is malformed", ErrCompositionInvalid)
	}
	compositionPath, err := source.store.RuntimeCompositionPath(binding.SHA256)
	if err != nil {
		return CandidateDeployment{}, fmt.Errorf("%w: resolve composition asset: %v", ErrCompositionInvalid, err)
	}
	compositionBytes, err := readRegularPrivateFile(compositionPath, maxRuntimeCompositionBytes)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return candidate, fmt.Errorf("%w: composition asset is not cached", ErrCompositionUnavailable)
		}
		return CandidateDeployment{}, fmt.Errorf("%w: read composition asset: %v", ErrCompositionInvalid, err)
	}
	composition, err := validateRuntimeComposition(compositionBytes, manifest, manifestDigest, binding.SHA256)
	if err != nil {
		return CandidateDeployment{}, err
	}
	return compositionCandidateDeployment(manifest, composition), nil
}

func readRegularPrivateFile(path string, maximum int64) ([]byte, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() || info.Mode().Perm() != 0o600 {
		return nil, fmt.Errorf("private evidence file must be a regular 0600 file")
	}
	if info.Size() <= 0 || info.Size() > maximum {
		return nil, fmt.Errorf("private evidence file size is invalid")
	}
	return os.ReadFile(path)
}

type normalizedRuntimeComposition struct {
	ReleaseTag        string
	CompositionDigest string
	Capabilities      map[string]any
	Components        []normalizedCompositionComponent
}

type normalizedCompositionComponent struct {
	ID     string
	Kind   string
	Name   string
	Digest string
	// The normalized JSON object is retained solely for canonical core hashing.
	JSON map[string]any
}

func validateRuntimeComposition(raw []byte, manifest *releasemanifest.Manifest, manifestDigest, expectedDigest string) (normalizedRuntimeComposition, error) {
	value, err := decodeCompositionJSON(raw)
	if err != nil {
		return normalizedRuntimeComposition{}, fmt.Errorf("%w: %v", ErrCompositionInvalid, err)
	}
	root, ok := value.(map[string]any)
	if !ok {
		return normalizedRuntimeComposition{}, fmt.Errorf("%w: composition root must be an object", ErrCompositionInvalid)
	}
	if err := requireKeys(root, map[string]bool{"schemaVersion": true, "kind": true, "releaseTag": true, "sourceRevisionDigest": true, "publicMergeCommit": true, "components": true, "capabilities": true, "compositionDigest": true, "manifestBinding": true}); err != nil {
		return normalizedRuntimeComposition{}, fmt.Errorf("%w: %v", ErrCompositionInvalid, err)
	}
	if number, ok := root["schemaVersion"].(json.Number); !ok || number.String() != "1" {
		return normalizedRuntimeComposition{}, fmt.Errorf("%w: unsupported composition schemaVersion", ErrCompositionInvalid)
	}
	kind, ok := root["kind"].(string)
	if !ok || kind != runtimeCompositionKind {
		return normalizedRuntimeComposition{}, fmt.Errorf("%w: composition kind is invalid", ErrCompositionInvalid)
	}
	releaseTag, ok := root["releaseTag"].(string)
	if !ok || !compositionReleaseTagRE.MatchString(releaseTag) {
		return normalizedRuntimeComposition{}, fmt.Errorf("%w: composition releaseTag is invalid", ErrCompositionInvalid)
	}
	if manifest == nil || strings.TrimPrefix(releaseTag, "v") != manifest.ReleaseVersion {
		return normalizedRuntimeComposition{}, fmt.Errorf("%w: composition releaseTag does not match manifest", ErrCompositionInvalid)
	}
	sourceRevisionDigest, err := optionalDigest(root, "sourceRevisionDigest")
	if err != nil {
		return normalizedRuntimeComposition{}, fmt.Errorf("%w: %v", ErrCompositionInvalid, err)
	}
	publicMergeCommit, err := optionalString(root, "publicMergeCommit")
	if err != nil || (publicMergeCommit != "" && !compositionCommitRE.MatchString(publicMergeCommit)) {
		return normalizedRuntimeComposition{}, fmt.Errorf("%w: publicMergeCommit is invalid", ErrCompositionInvalid)
	}
	componentsValue, ok := root["components"].([]any)
	if !ok || len(componentsValue) == 0 {
		return normalizedRuntimeComposition{}, fmt.Errorf("%w: components are required", ErrCompositionInvalid)
	}
	components := make([]normalizedCompositionComponent, 0, len(componentsValue))
	seen := make(map[string]struct{}, len(componentsValue))
	for index, item := range componentsValue {
		component, err := normalizeCompositionComponent(item, releaseTag, index)
		if err != nil {
			return normalizedRuntimeComposition{}, fmt.Errorf("%w: %v", ErrCompositionInvalid, err)
		}
		if _, exists := seen[component.ID]; exists {
			return normalizedRuntimeComposition{}, fmt.Errorf("%w: duplicate component %s", ErrCompositionInvalid, component.ID)
		}
		seen[component.ID] = struct{}{}
		components = append(components, component)
	}
	sort.Slice(components, func(left, right int) bool { return components[left].ID < components[right].ID })
	capabilities, err := normalizeCompositionCapabilities(root["capabilities"])
	if err != nil {
		return normalizedRuntimeComposition{}, fmt.Errorf("%w: %v", ErrCompositionInvalid, err)
	}
	compositionDigest, ok := root["compositionDigest"].(string)
	if !ok || !digestPattern.MatchString(compositionDigest) || compositionDigest != expectedDigest {
		return normalizedRuntimeComposition{}, fmt.Errorf("%w: composition digest does not match the manifest binding", ErrCompositionInvalid)
	}
	binding, ok := root["manifestBinding"].(map[string]any)
	if !ok {
		return normalizedRuntimeComposition{}, fmt.Errorf("%w: manifestBinding is required", ErrCompositionInvalid)
	}
	if err := requireKeys(binding, map[string]bool{"manifestDigest": true}); err != nil {
		return normalizedRuntimeComposition{}, fmt.Errorf("%w: %v", ErrCompositionInvalid, err)
	}
	if bindingDigest, ok := binding["manifestDigest"].(string); !ok || bindingDigest != manifestDigest || !digestPattern.MatchString(bindingDigest) {
		return normalizedRuntimeComposition{}, fmt.Errorf("%w: manifest binding does not match the manifest", ErrCompositionInvalid)
	}
	core := map[string]any{
		"schemaVersion": 1,
		"kind":          kind,
		"releaseTag":    releaseTag,
		"components":    compositionComponentsJSON(components),
		"capabilities":  capabilities,
	}
	if sourceRevisionDigest != "" {
		core["sourceRevisionDigest"] = sourceRevisionDigest
	}
	if publicMergeCommit != "" {
		core["publicMergeCommit"] = publicMergeCommit
	}
	canonical, err := canonicalCompositionJSON(core)
	if err != nil {
		return normalizedRuntimeComposition{}, fmt.Errorf("%w: canonicalize composition: %v", ErrCompositionInvalid, err)
	}
	sum := sha256.Sum256(canonical)
	if got := "sha256:" + fmt.Sprintf("%x", sum[:]); got != compositionDigest {
		return normalizedRuntimeComposition{}, fmt.Errorf("%w: composition core digest does not match", ErrCompositionInvalid)
	}
	if err := validateCompositionInventory(manifest, components); err != nil {
		return normalizedRuntimeComposition{}, fmt.Errorf("%w: %v", ErrCompositionInvalid, err)
	}
	return normalizedRuntimeComposition{ReleaseTag: releaseTag, CompositionDigest: compositionDigest, Capabilities: capabilities, Components: components}, nil
}

func compositionCandidateDeployment(manifest *releasemanifest.Manifest, composition normalizedRuntimeComposition) CandidateDeployment {
	components := make([]DeploymentComponent, 0, len(composition.Components))
	for _, component := range composition.Components {
		components = append(components, DeploymentComponent{ID: component.ID, Digest: component.Digest})
	}
	sortDeploymentComponents(components)
	dynamic := false
	if value, ok := composition.Capabilities["dynamicFrontendUpstream"].(bool); ok {
		dynamic = value
	}
	return CandidateDeployment{
		ReleaseVersion:    manifest.ReleaseVersion,
		CompositionDigest: composition.CompositionDigest,
		Components:        components,
		Capabilities:      DeploymentCapabilities{DynamicFrontendUpstream: dynamic},
	}
}

func validateCompositionInventory(manifest *releasemanifest.Manifest, components []normalizedCompositionComponent) error {
	if manifest == nil {
		return fmt.Errorf("manifest is required")
	}
	byID := make(map[string]normalizedCompositionComponent, len(components))
	for _, component := range components {
		byID[component.ID] = component
	}
	for _, name := range []string{"server", "frontend", "nginx", "agent", "bootstrap"} {
		component, ok := byID["runtime."+name]
		if !ok {
			return fmt.Errorf("composition is missing runtime.%s", name)
		}
		want, err := manifest.RuntimeImageDigest(name)
		if err != nil || component.Digest != want {
			return fmt.Errorf("composition runtime.%s artifact does not match manifest", name)
		}
	}
	packageDigests := make(map[string]string)
	runtimePairs := make(map[string]bool)
	for _, component := range components {
		if strings.HasPrefix(component.ID, "engine.") {
			parts := strings.Split(component.ID, ".")
			if len(parts) < 3 {
				return fmt.Errorf("engine component id %q is invalid", component.ID)
			}
			pair := strings.Join(parts[:len(parts)-1], ".")
			kind := parts[len(parts)-1]
			if kind == "package" {
				packageDigests[component.Digest] = component.ID
			} else if kind == "runtime" {
				runtimePairs[pair] = true
			}
		}
	}
	if len(packageDigests) != len(manifest.EnginePackages) {
		return fmt.Errorf("composition engine package inventory does not match manifest")
	}
	for index, enginePackage := range manifest.EnginePackages {
		if len(enginePackage.Refs) == 0 {
			return fmt.Errorf("manifest engine package %d is empty", index)
		}
		ref, err := ociartifact.ParseDigestReference(enginePackage.Refs[0])
		if err != nil {
			return fmt.Errorf("parse manifest engine package %d: %v", index, err)
		}
		componentID, ok := packageDigests[ref.Digest]
		if !ok {
			return fmt.Errorf("composition is missing manifest engine package %s", ref.Digest)
		}
		pair := strings.TrimSuffix(componentID, ".package")
		if !runtimePairs[pair] {
			return fmt.Errorf("composition is missing engine runtime pair %s", pair)
		}
	}
	return nil
}

func normalizeCompositionComponent(value any, releaseTag string, index int) (normalizedCompositionComponent, error) {
	object, ok := value.(map[string]any)
	if !ok {
		return normalizedCompositionComponent{}, fmt.Errorf("components[%d] must be an object", index)
	}
	if err := requireKeys(object, map[string]bool{"id": true, "kind": true, "name": true, "inputFingerprint": true, "artifact": true, "disposition": true, "sourceRelease": true, "evidence": true}); err != nil {
		return normalizedCompositionComponent{}, err
	}
	id, ok := object["id"].(string)
	if !ok || !compositionComponentIDRE.MatchString(id) {
		return normalizedCompositionComponent{}, fmt.Errorf("components[%d].id is invalid", index)
	}
	kind, ok := object["kind"].(string)
	if !ok || kind != strings.SplitN(id, ".", 2)[0] {
		return normalizedCompositionComponent{}, fmt.Errorf("components[%d].kind does not match id", index)
	}
	name, ok := object["name"].(string)
	if !ok || !compositionNameRE.MatchString(name) {
		return normalizedCompositionComponent{}, fmt.Errorf("components[%d].name is invalid", index)
	}
	disposition, ok := object["disposition"].(string)
	if !ok || (disposition != "built" && disposition != "reused") {
		return normalizedCompositionComponent{}, fmt.Errorf("components[%d].disposition is invalid", index)
	}
	fingerprint, err := normalizeCompositionFingerprint(object["inputFingerprint"])
	if err != nil {
		return normalizedCompositionComponent{}, fmt.Errorf("components[%d].inputFingerprint: %v", index, err)
	}
	artifact, digest, err := normalizeCompositionArtifact(object["artifact"])
	if err != nil {
		return normalizedCompositionComponent{}, fmt.Errorf("components[%d].artifact: %v", index, err)
	}
	sourceRelease, err := normalizeCompositionSourceRelease(object["sourceRelease"], releaseTag)
	if err != nil {
		return normalizedCompositionComponent{}, fmt.Errorf("components[%d].sourceRelease: %v", index, err)
	}
	evidence, err := normalizeCompositionEvidence(object["evidence"])
	if err != nil {
		return normalizedCompositionComponent{}, fmt.Errorf("components[%d].evidence: %v", index, err)
	}
	if disposition == "reused" && sourceRelease["tag"] == releaseTag {
		return normalizedCompositionComponent{}, fmt.Errorf("components[%d].reused source release points at current release", index)
	}
	if disposition == "reused" {
		if _, ok := sourceRelease["compositionDigest"]; !ok {
			return normalizedCompositionComponent{}, fmt.Errorf("components[%d].reused source release composition digest is required", index)
		}
	}
	if disposition == "built" && sourceRelease["tag"] != releaseTag {
		return normalizedCompositionComponent{}, fmt.Errorf("components[%d].built source release must match current release", index)
	}
	return normalizedCompositionComponent{
		ID: id, Kind: kind, Name: name, Digest: digest,
		JSON: map[string]any{"id": id, "kind": kind, "name": name, "inputFingerprint": fingerprint, "artifact": artifact, "disposition": disposition, "sourceRelease": sourceRelease, "evidence": evidence},
	}, nil
}

func normalizeCompositionFingerprint(value any) (map[string]any, error) {
	object, ok := value.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("must be an object")
	}
	if err := requireKeys(object, map[string]bool{"version": true, "algorithm": true, "digest": true, "baseImagesResolved": true, "inputs": true}); err != nil {
		return nil, err
	}
	if err := requireFields(object, "version", "algorithm", "digest", "baseImagesResolved", "inputs"); err != nil {
		return nil, err
	}
	version, ok := object["version"].(json.Number)
	if !ok || version.String() != strconv.Itoa(runtimeCompositionFingerprintSchemaVersion) {
		return nil, fmt.Errorf("version is unsupported")
	}
	algorithm, ok := object["algorithm"].(string)
	if !ok || algorithm != "sha256-canonical-json-v1" {
		return nil, fmt.Errorf("algorithm is unsupported")
	}
	digest, ok := object["digest"].(string)
	if !ok || !digestPattern.MatchString(digest) {
		return nil, fmt.Errorf("digest is invalid")
	}
	resolved, ok := object["baseImagesResolved"].(bool)
	if !ok {
		return nil, fmt.Errorf("baseImagesResolved is required")
	}
	inputs, err := normalizeCompositionInputs(object["inputs"])
	if err != nil {
		return nil, fmt.Errorf("inputs: %v", err)
	}
	inputsResolved, ok := inputs["baseImagesResolved"].(bool)
	if !ok || inputsResolved != resolved {
		return nil, fmt.Errorf("baseImagesResolved does not match inputs")
	}
	canonical, err := canonicalCompositionJSON(inputs)
	if err != nil {
		return nil, fmt.Errorf("inputs are not canonical JSON: %v", err)
	}
	sum := sha256.Sum256(canonical)
	if got := "sha256:" + fmt.Sprintf("%x", sum[:]); got != digest {
		return nil, fmt.Errorf("digest does not match inputs")
	}
	result := map[string]any{"version": runtimeCompositionFingerprintSchemaVersion, "algorithm": algorithm, "digest": digest, "baseImagesResolved": resolved, "inputs": inputs}
	return result, nil
}

func normalizeCompositionInputs(value any) (map[string]any, error) {
	object, ok := value.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("must be an object")
	}
	allowed := map[string]bool{
		"schemaVersion": true, "componentId": true, "kind": true, "contextPath": true, "dockerfile": true,
		"dockerignore": true, "files": true, "namedContexts": true, "buildArgs": true, "platforms": true,
		"baseImages": true, "baseImagesResolved": true, "builderPolicy": true, "generatedInputs": true,
	}
	if err := requireKeys(object, allowed); err != nil {
		return nil, err
	}
	if err := requireFields(object, "schemaVersion", "componentId", "kind", "contextPath", "dockerfile", "dockerignore", "files", "namedContexts", "buildArgs", "platforms", "baseImages", "baseImagesResolved", "builderPolicy", "generatedInputs"); err != nil {
		return nil, err
	}
	version, ok := object["schemaVersion"].(json.Number)
	if !ok || version.String() != strconv.Itoa(runtimeCompositionFingerprintSchemaVersion) {
		return nil, fmt.Errorf("schemaVersion is unsupported")
	}
	componentID, ok := object["componentId"].(string)
	if !ok || !compositionComponentIDRE.MatchString(componentID) {
		return nil, fmt.Errorf("componentId is invalid")
	}
	kind, ok := object["kind"].(string)
	if !ok || (kind != "runtime" && kind != "agent" && kind != "engine") || !strings.HasPrefix(componentID, kind+".") {
		return nil, fmt.Errorf("kind does not match componentId")
	}
	for _, key := range []string{"contextPath", "dockerfile", "dockerignore"} {
		text, ok := object[key].(string)
		if !ok || key != "dockerignore" && text == "" {
			return nil, fmt.Errorf("%s is invalid", key)
		}
	}
	files, ok := object["files"].([]any)
	if !ok {
		return nil, fmt.Errorf("files must be an array")
	}
	for index, value := range files {
		file, ok := value.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("files[%d] must be an object", index)
		}
		if err := requireKeys(file, map[string]bool{"path": true, "digest": true, "size": true, "mode": true, "symlink": true}); err != nil {
			return nil, fmt.Errorf("files[%d]: %v", index, err)
		}
		if err := requireFields(file, "path", "digest", "size", "mode"); err != nil {
			return nil, fmt.Errorf("files[%d]: %v", index, err)
		}
		pathValue, pathOK := file["path"].(string)
		digestValue, digestOK := file["digest"].(string)
		if !pathOK || pathValue == "" || !digestOK || !digestPattern.MatchString(digestValue) {
			return nil, fmt.Errorf("files[%d] path or digest is invalid", index)
		}
		if err := requireNonNegativeInteger(file["size"]); err != nil {
			return nil, fmt.Errorf("files[%d].size: %v", index, err)
		}
		if err := requireNonNegativeInteger(file["mode"]); err != nil {
			return nil, fmt.Errorf("files[%d].mode: %v", index, err)
		}
		mode, _ := strconv.ParseInt(file["mode"].(json.Number).String(), 10, 64)
		if mode > 0o7777 {
			return nil, fmt.Errorf("files[%d].mode is invalid", index)
		}
		if symlink, present := file["symlink"]; present {
			if _, ok := symlink.(string); !ok {
				return nil, fmt.Errorf("files[%d].symlink is invalid", index)
			}
		}
	}
	contexts, ok := object["namedContexts"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("namedContexts must be an object")
	}
	for name, value := range contexts {
		context, ok := value.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("namedContexts.%s must be an object", name)
		}
		if err := requireKeys(context, map[string]bool{"contextPath": true, "dockerignore": true}); err != nil {
			return nil, fmt.Errorf("namedContexts.%s: %v", name, err)
		}
		if err := requireFields(context, "contextPath", "dockerignore"); err != nil {
			return nil, fmt.Errorf("namedContexts.%s: %v", name, err)
		}
		contextPath, pathOK := context["contextPath"].(string)
		_, ignoreOK := context["dockerignore"].(string)
		if !pathOK || contextPath == "" || !ignoreOK {
			return nil, fmt.Errorf("namedContexts.%s path fields are invalid", name)
		}
	}
	buildArgs, ok := object["buildArgs"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("buildArgs must be an object")
	}
	for key, value := range buildArgs {
		if _, ok := value.(string); !ok {
			return nil, fmt.Errorf("buildArgs.%s must be a string", key)
		}
	}
	platforms, err := normalizeCompositionStringArray(object["platforms"], compositionPlatformRE, "platforms")
	if err != nil {
		return nil, err
	}
	if !equalStringArrayValue(object["platforms"], platforms) {
		return nil, fmt.Errorf("platforms must be unique and sorted")
	}
	baseImages, ok := object["baseImages"].([]any)
	if !ok {
		return nil, fmt.Errorf("baseImages must be an array")
	}
	allResolved := true
	for index, value := range baseImages {
		image, ok := value.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("baseImages[%d] must be an object", index)
		}
		if err := requireKeys(image, map[string]bool{"ref": true, "identity": true, "resolved": true}); err != nil {
			return nil, fmt.Errorf("baseImages[%d]: %v", index, err)
		}
		if err := requireFields(image, "ref", "identity", "resolved"); err != nil {
			return nil, fmt.Errorf("baseImages[%d]: %v", index, err)
		}
		ref, refOK := image["ref"].(string)
		_, identityOK := image["identity"].(string)
		resolved, resolvedOK := image["resolved"].(bool)
		if !refOK || ref == "" || !identityOK || !resolvedOK {
			return nil, fmt.Errorf("baseImages[%d] fields are invalid", index)
		}
		allResolved = allResolved && resolved
	}
	resolved, ok := object["baseImagesResolved"].(bool)
	if !ok || resolved != allResolved {
		return nil, fmt.Errorf("baseImagesResolved does not match baseImages")
	}
	if _, ok := object["builderPolicy"].(map[string]any); !ok {
		return nil, fmt.Errorf("builderPolicy must be an object")
	}
	generated, ok := object["generatedInputs"].([]any)
	if !ok {
		return nil, fmt.Errorf("generatedInputs must be an array")
	}
	for index, value := range generated {
		input, ok := value.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("generatedInputs[%d] must be an object", index)
		}
		if err := requireKeys(input, map[string]bool{"name": true, "digest": true}); err != nil {
			return nil, fmt.Errorf("generatedInputs[%d]: %v", index, err)
		}
		if err := requireFields(input, "name", "digest"); err != nil {
			return nil, fmt.Errorf("generatedInputs[%d]: %v", index, err)
		}
		name, nameOK := input["name"].(string)
		digest, digestOK := input["digest"].(string)
		if !nameOK || name == "" || !digestOK || !digestPattern.MatchString(digest) {
			return nil, fmt.Errorf("generatedInputs[%d] fields are invalid", index)
		}
	}
	return object, nil
}

func normalizeCompositionArtifact(value any) (map[string]any, string, error) {
	object, ok := value.(map[string]any)
	if !ok {
		return nil, "", fmt.Errorf("must be an object")
	}
	if err := requireKeys(object, map[string]bool{"ref": true, "digest": true, "platforms": true}); err != nil {
		return nil, "", err
	}
	ref, ok := object["ref"].(string)
	if !ok {
		return nil, "", fmt.Errorf("ref is required")
	}
	parsed, err := ociartifact.ParseDigestReference(ref)
	if err != nil {
		return nil, "", err
	}
	digest, ok := object["digest"].(string)
	if !ok || !digestPattern.MatchString(digest) || parsed.Digest != digest {
		return nil, "", fmt.Errorf("digest does not match immutable ref")
	}
	result := map[string]any{"ref": ref, "digest": digest}
	if platforms, present := object["platforms"]; present {
		values, err := normalizeCompositionStringArray(platforms, compositionPlatformRE, "platforms")
		if err != nil {
			return nil, "", err
		}
		result["platforms"] = values
	}
	return result, digest, nil
}

func normalizeCompositionSourceRelease(value any, releaseTag string) (map[string]any, error) {
	object, ok := value.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("must be an object")
	}
	if err := requireKeys(object, map[string]bool{"tag": true, "publicMergeCommit": true, "workflowRun": true, "compositionDigest": true}); err != nil {
		return nil, err
	}
	tag, ok := object["tag"].(string)
	if !ok || !compositionReleaseTagRE.MatchString(tag) {
		return nil, fmt.Errorf("tag is invalid")
	}
	result := map[string]any{"tag": tag}
	for _, key := range []string{"publicMergeCommit", "workflowRun"} {
		if value, present := object[key]; present {
			text, ok := value.(string)
			if !ok || text == "" {
				return nil, fmt.Errorf("%s is invalid", key)
			}
			result[key] = text
		}
	}
	if value, present := object["compositionDigest"]; present {
		digest, ok := value.(string)
		if !ok || !digestPattern.MatchString(digest) {
			return nil, fmt.Errorf("compositionDigest is invalid")
		}
		result["compositionDigest"] = digest
	}
	return result, nil
}

func normalizeCompositionEvidence(value any) (map[string]any, error) {
	object, ok := value.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("must be an object")
	}
	if err := requireKeys(object, map[string]bool{"image": true, "provenance": true, "sbom": true, "signature": true}); err != nil {
		return nil, err
	}
	result := make(map[string]any, 4)
	for _, key := range []string{"image", "provenance", "sbom", "signature"} {
		value, present := object[key]
		if !present {
			return nil, fmt.Errorf("%s is required", key)
		}
		switch item := value.(type) {
		case string:
			if strings.TrimSpace(item) == "" {
				return nil, fmt.Errorf("%s is empty", key)
			}
			result[key] = strings.TrimSpace(item)
		case []any:
			values := make([]string, 0, len(item))
			for _, entry := range item {
				text, ok := entry.(string)
				if !ok || strings.TrimSpace(text) == "" {
					return nil, fmt.Errorf("%s contains an invalid reference", key)
				}
				values = append(values, strings.TrimSpace(text))
			}
			values = uniqueSortedStrings(values)
			if len(values) == 0 {
				return nil, fmt.Errorf("%s is empty", key)
			}
			result[key] = values
		default:
			return nil, fmt.Errorf("%s must be a string or string array", key)
		}
	}
	return result, nil
}

func normalizeCompositionCapabilities(value any) (map[string]any, error) {
	if value == nil {
		return map[string]any{}, nil
	}
	object, ok := value.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("capabilities must be an object")
	}
	result := make(map[string]any, len(object))
	for key, value := range object {
		switch value.(type) {
		case bool, string, json.Number:
			result[key] = value
		default:
			return nil, fmt.Errorf("capability %s must be scalar", key)
		}
	}
	return result, nil
}

func compositionComponentsJSON(components []normalizedCompositionComponent) []any {
	result := make([]any, 0, len(components))
	for _, component := range components {
		result = append(result, component.JSON)
	}
	return result
}

func normalizeCompositionStringArray(value any, pattern *regexp.Regexp, label string) ([]string, error) {
	items, ok := value.([]any)
	if !ok || len(items) == 0 {
		return nil, fmt.Errorf("%s must be a non-empty array", label)
	}
	values := make([]string, 0, len(items))
	for _, item := range items {
		text, ok := item.(string)
		if !ok || !pattern.MatchString(text) {
			return nil, fmt.Errorf("%s contains an invalid value", label)
		}
		values = append(values, text)
	}
	return uniqueSortedStrings(values), nil
}

func uniqueSortedStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}

func requireKeys(object map[string]any, allowed map[string]bool) error {
	for key := range object {
		if !allowed[key] {
			return fmt.Errorf("unknown field %q", key)
		}
	}
	return nil
}

func requireFields(object map[string]any, fields ...string) error {
	for _, field := range fields {
		if _, present := object[field]; !present {
			return fmt.Errorf("%s is required", field)
		}
	}
	return nil
}

func requireNonNegativeInteger(value any) error {
	number, ok := value.(json.Number)
	if !ok {
		return fmt.Errorf("must be an integer")
	}
	parsed, err := strconv.ParseInt(number.String(), 10, 64)
	if err != nil || parsed < 0 {
		return fmt.Errorf("must be a non-negative integer")
	}
	return nil
}

func equalStringArrayValue(value any, expected []string) bool {
	items, ok := value.([]any)
	if !ok || len(items) != len(expected) {
		return false
	}
	for index, item := range items {
		text, ok := item.(string)
		if !ok || text != expected[index] {
			return false
		}
	}
	return true
}

func optionalDigest(object map[string]any, key string) (string, error) {
	value, present := object[key]
	if !present {
		return "", nil
	}
	digest, ok := value.(string)
	if !ok || !digestPattern.MatchString(digest) {
		return "", fmt.Errorf("%s is invalid", key)
	}
	return digest, nil
}

func optionalString(object map[string]any, key string) (string, error) {
	value, present := object[key]
	if !present {
		return "", nil
	}
	text, ok := value.(string)
	if !ok {
		return "", fmt.Errorf("%s is invalid", key)
	}
	return text, nil
}

func decodeCompositionJSON(raw []byte) (any, error) {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	var value any
	if err := decoder.Decode(&value); err != nil {
		return nil, err
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		if err == nil {
			return nil, fmt.Errorf("trailing JSON value")
		}
		return nil, err
	}
	return value, nil
}

// canonicalCompositionJSON mirrors the release resolver's sorted-key canonical
// JSON. Release assets are generated by Node, so preserving JSON numbers as
// json.Number avoids introducing a Go float-rounding change into the digest.
func canonicalCompositionJSON(value any) ([]byte, error) {
	var buffer bytes.Buffer
	if err := writeCanonicalCompositionJSON(&buffer, value); err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

func writeCanonicalCompositionJSON(buffer *bytes.Buffer, value any) error {
	switch item := value.(type) {
	case nil:
		buffer.WriteString("null")
	case bool:
		if item {
			buffer.WriteString("true")
		} else {
			buffer.WriteString("false")
		}
	case string:
		encoded, err := json.Marshal(item)
		if err != nil {
			return err
		}
		// JSON generated by the release resolver is ASCII for all identity fields;
		// retain encoding/json's safe escaping for arbitrary evidence text.
		buffer.Write(encoded)
	case json.Number:
		if _, err := strconv.ParseFloat(item.String(), 64); err != nil {
			return err
		}
		buffer.WriteString(item.String())
	case int:
		buffer.WriteString(strconv.Itoa(item))
	case int64:
		buffer.WriteString(strconv.FormatInt(item, 10))
	case uint:
		buffer.WriteString(strconv.FormatUint(uint64(item), 10))
	case uint64:
		buffer.WriteString(strconv.FormatUint(item, 10))
	case []any:
		buffer.WriteByte('[')
		for index, child := range item {
			if index > 0 {
				buffer.WriteByte(',')
			}
			if err := writeCanonicalCompositionJSON(buffer, child); err != nil {
				return err
			}
		}
		buffer.WriteByte(']')
	case map[string]any:
		keys := make([]string, 0, len(item))
		for key := range item {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		buffer.WriteByte('{')
		for index, key := range keys {
			if index > 0 {
				buffer.WriteByte(',')
			}
			encoded, err := json.Marshal(key)
			if err != nil {
				return err
			}
			buffer.Write(encoded)
			buffer.WriteByte(':')
			if err := writeCanonicalCompositionJSON(buffer, item[key]); err != nil {
				return err
			}
		}
		buffer.WriteByte('}')
	default:
		return fmt.Errorf("unsupported canonical JSON value %T", value)
	}
	return nil
}
