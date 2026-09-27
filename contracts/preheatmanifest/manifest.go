// Package preheatmanifest owns the release-bound Docker image closure consumed
// by the Compose engine-preheater. It deliberately validates only the
// immutable manifest and its input bindings; pulling and scheduling remain
// Agent responsibilities.
package preheatmanifest

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"regexp"
	"sort"
	"strings"

	"github.com/yyhuni/lunafox/contracts/ociartifact"
	"github.com/yyhuni/lunafox/contracts/releasemanifest"
)

const (
	SchemaVersion      = 1
	Kind               = "lunafox.preheat-manifest"
	CanonicalAlgorithm = "sha256-canonical-json-v1"
	CloudflareRegistry = "docker.lunafox.cc.cd"
	ProfileEmbedded    = "embedded"
	ProfileExternal    = "external"
	PlatformLinuxAMD64 = "linux/amd64"
	PlatformLinuxARM64 = "linux/arm64"
)

var (
	digestPattern          = regexp.MustCompile(`^sha256:[a-f0-9]{64}$`)
	releaseTagPattern      = regexp.MustCompile(`^v\d+\.\d+\.\d+(?:-[0-9A-Za-z.-]+)?$`)
	sourceNamePattern      = regexp.MustCompile(`^[a-z][a-z0-9_.-]*$`)
	firstPartyRepositoryRe = regexp.MustCompile(`^yyhuni/lunafox-[a-z0-9][a-z0-9._-]*$`)
)

// Manifest is the exact canonical JSON contract published beside a modern
// Compose release. ManifestDigest is a digest of the core payload rather than
// the file bytes so it cannot include itself in its own identity.
type Manifest struct {
	Entries         []Entry          `json:"entries"`
	Kind            string           `json:"kind"`
	ManifestDigest  string           `json:"manifestDigest"`
	ProfileClosures []ProfileClosure `json:"profileClosures"`
	Release         ReleaseBinding   `json:"release"`
	SchemaVersion   int              `json:"schemaVersion"`
}

type ReleaseBinding struct {
	ComposeDigest          string `json:"composeDigest"`
	CompositionDigest      string `json:"compositionDigest"`
	ManifestDigest         string `json:"manifestDigest"`
	Tag                    string `json:"tag"`
	ThirdPartyPolicyDigest string `json:"thirdPartyPolicyDigest"`
}

// Entry identifies one content-addressed image. Sources retain the logical
// service/Engine owners after equal repository and digest values are deduped.
type Entry struct {
	Candidates           []string        `json:"candidates"`
	CloudflareCandidates []string        `json:"cloudflareCandidates"`
	Digest               string          `json:"digest"`
	IdentityReference    string          `json:"identityReference"`
	Platforms            []string        `json:"platforms"`
	Profiles             []string        `json:"profiles"`
	Repository           string          `json:"repository"`
	Sources              []LogicalSource `json:"sources"`
	Trust                string          `json:"trust"`
}

type LogicalSource struct {
	Kind string `json:"kind"`
	Name string `json:"name"`
}

type ProfileClosure struct {
	Digest  string   `json:"digest"`
	Entries []string `json:"entries"`
	Profile string   `json:"profile"`
}

// BindingInputs are the read-only files mounted into the preheater container.
// No caller-provided image ref is accepted: the manifest binds the complete
// closure before any Docker API call is made.
type BindingInputs struct {
	ReleaseManifest      []byte
	RuntimeComposition   []byte
	Compose              []byte
	ThirdPartyPolicy     []byte
	ExpectedManifestHash string
	Profile              string
}

func sha256Digest(payload []byte) string {
	sum := sha256.Sum256(payload)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func canonicalJSON(value any) ([]byte, error) {
	return json.Marshal(value)
}

func decodeOne(data []byte, target any) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		if err == nil {
			return errors.New("multiple JSON values are not allowed")
		}
		return err
	}
	return nil
}

// rejectDuplicateKeys walks JSON tokens before regular decoding. encoding/json
// otherwise accepts duplicate keys and silently preserves only the last value,
// which is unsuitable for a signed/release-bound security contract.
func rejectDuplicateKeys(data []byte) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	if err := walkJSONValue(decoder); err != nil {
		return err
	}
	if _, err := decoder.Token(); !errors.Is(err, io.EOF) {
		if err == nil {
			return errors.New("multiple JSON values are not allowed")
		}
		return err
	}
	return nil
}

func walkJSONValue(decoder *json.Decoder) error {
	token, err := decoder.Token()
	if err != nil {
		return err
	}
	delimiter, ok := token.(json.Delim)
	if !ok {
		return nil
	}
	switch delimiter {
	case '{':
		seen := map[string]struct{}{}
		for decoder.More() {
			keyToken, err := decoder.Token()
			if err != nil {
				return err
			}
			key, ok := keyToken.(string)
			if !ok {
				return errors.New("JSON object key is not a string")
			}
			if _, exists := seen[key]; exists {
				return fmt.Errorf("JSON object contains duplicate key %q", key)
			}
			seen[key] = struct{}{}
			if err := walkJSONValue(decoder); err != nil {
				return err
			}
		}
		end, err := decoder.Token()
		if err != nil || end != json.Delim('}') {
			return errors.New("JSON object is not terminated")
		}
	case '[':
		for decoder.More() {
			if err := walkJSONValue(decoder); err != nil {
				return err
			}
		}
		end, err := decoder.Token()
		if err != nil || end != json.Delim(']') {
			return errors.New("JSON array is not terminated")
		}
	default:
		return fmt.Errorf("unexpected JSON delimiter %q", delimiter)
	}
	return nil
}

func exactKeys(value map[string]any, expected ...string) error {
	if len(value) != len(expected) {
		return errors.New("object has unknown or missing fields")
	}
	for _, key := range expected {
		if _, ok := value[key]; !ok {
			return fmt.Errorf("object is missing field %q", key)
		}
	}
	return nil
}

func canonicalRawObject(raw []byte) (map[string]any, error) {
	if len(raw) == 0 {
		return nil, errors.New("preheat manifest is empty")
	}
	if err := rejectDuplicateKeys(raw); err != nil {
		return nil, fmt.Errorf("decode preheat manifest: %w", err)
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	var value any
	if err := decoder.Decode(&value); err != nil {
		return nil, fmt.Errorf("decode preheat manifest: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return nil, errors.New("preheat manifest contains multiple JSON values")
	}
	object, ok := value.(map[string]any)
	if !ok {
		return nil, errors.New("preheat manifest must be a JSON object")
	}
	canonical, err := canonicalJSON(object)
	if err != nil {
		return nil, fmt.Errorf("canonicalize preheat manifest: %w", err)
	}
	if !bytes.Equal(raw, canonical) {
		return nil, errors.New("preheat manifest must use canonical JSON bytes")
	}
	return object, nil
}

// Parse decodes a single canonical manifest and validates every structural,
// identity, candidate, and closure invariant before returning it.
func Parse(raw []byte) (Manifest, error) {
	object, err := canonicalRawObject(raw)
	if err != nil {
		return Manifest{}, err
	}
	if err := exactKeys(object, "entries", "kind", "manifestDigest", "profileClosures", "release", "schemaVersion"); err != nil {
		return Manifest{}, fmt.Errorf("preheat manifest: %w", err)
	}
	var manifest Manifest
	if err := decodeOne(raw, &manifest); err != nil {
		return Manifest{}, fmt.Errorf("decode preheat manifest: %w", err)
	}
	if err := manifest.Validate(); err != nil {
		return Manifest{}, err
	}
	return manifest, nil
}

func validDigest(value string) bool { return digestPattern.MatchString(value) }

func validProfile(profile string) bool {
	return profile == ProfileEmbedded || profile == ProfileExternal
}

func validPlatform(platform string) bool {
	return platform == PlatformLinuxAMD64 || platform == PlatformLinuxARM64
}

func canonicalStringSet(values []string) bool {
	if len(values) == 0 {
		return false
	}
	for index, value := range values {
		if value == "" || (index > 0 && values[index-1] >= value) {
			return false
		}
	}
	return true
}

func parseCandidate(raw, repository, digest, label string) (ociartifact.DigestReference, error) {
	reference, err := ociartifact.ParseDigestReference(raw)
	if err != nil || reference.String() != raw {
		if err == nil {
			err = errors.New("reference is not canonical")
		}
		return ociartifact.DigestReference{}, fmt.Errorf("%s: %w", label, err)
	}
	if reference.Repository != repository || reference.Digest != digest {
		return ociartifact.DigestReference{}, fmt.Errorf("%s does not preserve repository and digest identity", label)
	}
	return reference, nil
}

func entryIdentity(entry Entry) string {
	return entry.Repository + "@" + entry.Digest
}

func entryCore(entry Entry) map[string]any {
	return map[string]any{
		"candidates":           entry.Candidates,
		"cloudflareCandidates": entry.CloudflareCandidates,
		"digest":               entry.Digest,
		"identityReference":    entry.IdentityReference,
		"platforms":            entry.Platforms,
		"profiles":             entry.Profiles,
		"repository":           entry.Repository,
		"sources":              entry.Sources,
		"trust":                entry.Trust,
	}
}

func (entry Entry) validate(index int) error {
	label := fmt.Sprintf("entries[%d]", index)
	if entry.Repository == "" || strings.TrimSpace(entry.Repository) != entry.Repository {
		return fmt.Errorf("%s.repository is invalid", label)
	}
	if !validDigest(entry.Digest) {
		return fmt.Errorf("%s.digest must be a sha256 digest", label)
	}
	if entry.Trust != "first-party" && entry.Trust != "third-party" {
		return fmt.Errorf("%s.trust is unsupported", label)
	}
	if !canonicalStringSet(entry.Platforms) {
		return fmt.Errorf("%s.platforms must be a non-empty canonical set", label)
	}
	for _, platform := range entry.Platforms {
		if !validPlatform(platform) {
			return fmt.Errorf("%s.platforms contains unsupported platform %q", label, platform)
		}
	}
	if !canonicalStringSet(entry.Profiles) {
		return fmt.Errorf("%s.profiles must be a non-empty canonical set", label)
	}
	for _, profile := range entry.Profiles {
		if !validProfile(profile) {
			return fmt.Errorf("%s.profiles contains unsupported profile %q", label, profile)
		}
	}
	if len(entry.Sources) == 0 {
		return fmt.Errorf("%s.sources are required", label)
	}
	previousSource := ""
	for sourceIndex, source := range entry.Sources {
		if source.Kind != "compose-service" && source.Kind != "engine-runtime" {
			return fmt.Errorf("%s.sources[%d].kind is unsupported", label, sourceIndex)
		}
		if !sourceNamePattern.MatchString(source.Name) {
			return fmt.Errorf("%s.sources[%d].name is invalid", label, sourceIndex)
		}
		current := source.Kind + ":" + source.Name
		if previousSource >= current {
			return fmt.Errorf("%s.sources must use canonical order without duplicates", label)
		}
		previousSource = current
	}
	if len(entry.Candidates) == 0 || len(entry.CloudflareCandidates) == 0 {
		return fmt.Errorf("%s candidate lists are required", label)
	}
	seenCandidates := map[string]struct{}{}
	parsedCandidates := make([]ociartifact.DigestReference, len(entry.Candidates))
	for candidateIndex, raw := range entry.Candidates {
		reference, err := parseCandidate(raw, entry.Repository, entry.Digest, fmt.Sprintf("%s.candidates[%d]", label, candidateIndex))
		if err != nil {
			return err
		}
		if _, exists := seenCandidates[raw]; exists {
			return fmt.Errorf("%s.candidates contains duplicate candidate", label)
		}
		seenCandidates[raw] = struct{}{}
		parsedCandidates[candidateIndex] = reference
	}
	parsedCloudflare := make([]ociartifact.DigestReference, len(entry.CloudflareCandidates))
	seenCloudflare := map[string]struct{}{}
	for candidateIndex, raw := range entry.CloudflareCandidates {
		reference, err := parseCandidate(raw, entry.Repository, entry.Digest, fmt.Sprintf("%s.cloudflareCandidates[%d]", label, candidateIndex))
		if err != nil {
			return err
		}
		if _, exists := seenCloudflare[raw]; exists {
			return fmt.Errorf("%s.cloudflareCandidates contains duplicate candidate", label)
		}
		seenCloudflare[raw] = struct{}{}
		parsedCloudflare[candidateIndex] = reference
	}
	identityReference, err := parseCandidate(entry.IdentityReference, entry.Repository, entry.Digest, label+".identityReference")
	if err != nil {
		return err
	}
	switch entry.Trust {
	case "first-party":
		if !firstPartyRepositoryRe.MatchString(entry.Repository) {
			return fmt.Errorf("%s.repository is not first-party", label)
		}
		if len(parsedCandidates) != 2 || parsedCandidates[0].Registry != "docker.io" || parsedCandidates[1].Registry != "ghcr.io" {
			return fmt.Errorf("%s.candidates must be Docker Hub then GHCR", label)
		}
		if identityReference != parsedCandidates[1] {
			return fmt.Errorf("%s.identityReference must be the GHCR candidate", label)
		}
		if len(parsedCloudflare) != 3 || parsedCloudflare[0].Registry != CloudflareRegistry || parsedCloudflare[1] != parsedCandidates[0] || parsedCloudflare[2] != parsedCandidates[1] {
			return fmt.Errorf("%s.cloudflareCandidates must be CF, Docker Hub, GHCR", label)
		}
	case "third-party":
		if len(parsedCandidates) != 1 || identityReference != parsedCandidates[0] {
			return fmt.Errorf("%s third-party identity must use its only origin candidate", label)
		}
		if len(parsedCloudflare) != 2 || parsedCloudflare[0].Registry != CloudflareRegistry || parsedCloudflare[1] != parsedCandidates[0] {
			return fmt.Errorf("%s.cloudflareCandidates must be CF then origin", label)
		}
	}
	return nil
}

func closureDigest(profile string, entries []string) (string, error) {
	payload, err := canonicalJSON(map[string]any{"entries": entries, "profile": profile})
	if err != nil {
		return "", err
	}
	return sha256Digest(payload), nil
}

func (manifest Manifest) corePayload() map[string]any {
	entries := make([]map[string]any, len(manifest.Entries))
	for index, entry := range manifest.Entries {
		entries[index] = entryCore(entry)
	}
	return map[string]any{
		"entries":         entries,
		"kind":            manifest.Kind,
		"profileClosures": manifest.ProfileClosures,
		"release":         manifest.Release,
		"schemaVersion":   manifest.SchemaVersion,
	}
}

// Validate confirms the manifest's intrinsic contract. It intentionally does
// not access the filesystem or Docker; ValidateBindings covers mounted inputs.
func (manifest Manifest) Validate() error {
	if manifest.SchemaVersion != SchemaVersion || manifest.Kind != Kind {
		return errors.New("unsupported preheat manifest schema")
	}
	if !releaseTagPattern.MatchString(manifest.Release.Tag) {
		return errors.New("preheat manifest release tag is invalid")
	}
	for _, value := range []string{
		manifest.Release.ManifestDigest,
		manifest.Release.CompositionDigest,
		manifest.Release.ComposeDigest,
		manifest.Release.ThirdPartyPolicyDigest,
		manifest.ManifestDigest,
	} {
		if !validDigest(value) {
			return errors.New("preheat manifest digest is invalid")
		}
	}
	if len(manifest.Entries) == 0 {
		return errors.New("preheat manifest has no entries")
	}
	previousIdentity := ""
	for index, entry := range manifest.Entries {
		if err := entry.validate(index); err != nil {
			return err
		}
		currentIdentity := entryIdentity(entry)
		if previousIdentity >= currentIdentity {
			return errors.New("preheat manifest entries must use canonical identity order without duplicates")
		}
		previousIdentity = currentIdentity
	}
	if len(manifest.ProfileClosures) != 2 {
		return errors.New("preheat manifest must contain exactly two profile closures")
	}
	for closureIndex, closure := range manifest.ProfileClosures {
		if !validProfile(closure.Profile) || !canonicalStringSet(closure.Entries) || !validDigest(closure.Digest) {
			return fmt.Errorf("preheat manifest profileClosures[%d] is invalid", closureIndex)
		}
		if closureIndex > 0 && manifest.ProfileClosures[closureIndex-1].Profile >= closure.Profile {
			return errors.New("preheat manifest profile closures must use canonical order without duplicates")
		}
		expected := make([]string, 0, len(manifest.Entries))
		for _, entry := range manifest.Entries {
			if contains(entry.Profiles, closure.Profile) {
				expected = append(expected, entryIdentity(entry))
			}
		}
		sort.Strings(expected)
		if !equalStrings(expected, closure.Entries) {
			return fmt.Errorf("preheat manifest profile closure %q does not match entries", closure.Profile)
		}
		expectedDigest, err := closureDigest(closure.Profile, closure.Entries)
		if err != nil || expectedDigest != closure.Digest {
			return fmt.Errorf("preheat manifest profile closure %q digest does not match entries", closure.Profile)
		}
	}
	core, err := canonicalJSON(manifest.corePayload())
	if err != nil {
		return fmt.Errorf("canonicalize preheat manifest core: %w", err)
	}
	if sha256Digest(core) != manifest.ManifestDigest {
		return errors.New("preheat manifest digest does not match canonical payload")
	}
	return nil
}

func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func equalStrings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

type runtimeCompositionBinding struct {
	CompositionDigest string
	ManifestDigest    string
	ReleaseTag        string
}

func decodeRuntimeCompositionBinding(raw []byte) (runtimeCompositionBinding, error) {
	if len(raw) == 0 {
		return runtimeCompositionBinding{}, errors.New("runtime composition is empty")
	}
	if err := rejectDuplicateKeys(raw); err != nil {
		return runtimeCompositionBinding{}, fmt.Errorf("decode runtime composition: %w", err)
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	var value any
	if err := decoder.Decode(&value); err != nil {
		return runtimeCompositionBinding{}, fmt.Errorf("decode runtime composition: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		if err == nil {
			return runtimeCompositionBinding{}, errors.New("runtime composition contains multiple JSON values")
		}
		return runtimeCompositionBinding{}, fmt.Errorf("decode runtime composition: %w", err)
	}
	composition, ok := value.(map[string]any)
	if !ok {
		return runtimeCompositionBinding{}, errors.New("runtime composition must be a JSON object")
	}
	allowed := map[string]struct{}{
		"capabilities":         {},
		"components":           {},
		"compositionDigest":    {},
		"kind":                 {},
		"manifestBinding":      {},
		"publicMergeCommit":    {},
		"releaseTag":           {},
		"schemaVersion":        {},
		"sourceRevisionDigest": {},
	}
	for key := range composition {
		if _, ok := allowed[key]; !ok {
			return runtimeCompositionBinding{}, fmt.Errorf("runtime composition contains unknown field %q", key)
		}
	}
	for _, key := range []string{"schemaVersion", "kind", "releaseTag", "components", "capabilities", "compositionDigest", "manifestBinding"} {
		if _, ok := composition[key]; !ok {
			return runtimeCompositionBinding{}, fmt.Errorf("runtime composition is missing field %q", key)
		}
	}
	if schemaVersion, ok := composition["schemaVersion"].(json.Number); !ok || schemaVersion.String() != "1" {
		return runtimeCompositionBinding{}, errors.New("runtime composition schemaVersion is unsupported")
	}
	if kind, ok := composition["kind"].(string); !ok || kind != "lunafox.runtime-composition" {
		return runtimeCompositionBinding{}, errors.New("runtime composition kind is invalid")
	}
	releaseTag, ok := composition["releaseTag"].(string)
	if !ok || !releaseTagPattern.MatchString(releaseTag) {
		return runtimeCompositionBinding{}, errors.New("runtime composition releaseTag is invalid")
	}
	components, ok := composition["components"].([]any)
	if !ok || len(components) == 0 {
		return runtimeCompositionBinding{}, errors.New("runtime composition components are invalid")
	}
	if _, ok := composition["capabilities"].(map[string]any); !ok {
		return runtimeCompositionBinding{}, errors.New("runtime composition capabilities are invalid")
	}
	compositionDigest, ok := composition["compositionDigest"].(string)
	if !ok || !validDigest(compositionDigest) {
		return runtimeCompositionBinding{}, errors.New("runtime composition compositionDigest is invalid")
	}
	binding, ok := composition["manifestBinding"].(map[string]any)
	if !ok {
		return runtimeCompositionBinding{}, errors.New("runtime composition manifestBinding is invalid")
	}
	if err := exactKeys(binding, "manifestDigest"); err != nil {
		return runtimeCompositionBinding{}, fmt.Errorf("runtime composition manifestBinding: %w", err)
	}
	manifestDigest, ok := binding["manifestDigest"].(string)
	if !ok || !validDigest(manifestDigest) {
		return runtimeCompositionBinding{}, errors.New("runtime composition manifestBinding.manifestDigest is invalid")
	}
	if sourceRevision, exists := composition["sourceRevisionDigest"]; exists {
		value, ok := sourceRevision.(string)
		if !ok || !validDigest(value) {
			return runtimeCompositionBinding{}, errors.New("runtime composition sourceRevisionDigest is invalid")
		}
	}
	if publicMergeCommit, exists := composition["publicMergeCommit"]; exists {
		value, ok := publicMergeCommit.(string)
		if !ok || !regexp.MustCompile(`^[0-9a-f]{40}$`).MatchString(value) {
			return runtimeCompositionBinding{}, errors.New("runtime composition publicMergeCommit is invalid")
		}
	}
	return runtimeCompositionBinding{
		CompositionDigest: compositionDigest,
		ManifestDigest:    manifestDigest,
		ReleaseTag:        releaseTag,
	}, nil
}

// EntriesForPlatform returns the selected profile closure for the Docker
// daemon platform. A modern manifest with no Engine Runtime for that platform
// is invalid rather than silently becoming a Compose-only installation.
func (manifest Manifest) EntriesForPlatform(profile, platform string) ([]Entry, error) {
	if err := manifest.Validate(); err != nil {
		return nil, err
	}
	if !validProfile(profile) || !validPlatform(platform) {
		return nil, errors.New("preheat profile or Docker platform is unsupported")
	}
	entries := make([]Entry, 0, len(manifest.Entries))
	engineRuntimeCount := 0
	for _, entry := range manifest.Entries {
		if !contains(entry.Profiles, profile) || !contains(entry.Platforms, platform) {
			continue
		}
		entries = append(entries, entry)
		for _, source := range entry.Sources {
			if source.Kind == "engine-runtime" {
				engineRuntimeCount++
				break
			}
		}
	}
	if len(entries) == 0 || engineRuntimeCount == 0 {
		return nil, fmt.Errorf("preheat manifest has no supported Engine Runtime for %s on %s", profile, platform)
	}
	return entries, nil
}

// ValidateBindings verifies the files mounted into the preheater before it
// opens a Docker client. This makes a substituted manifest/configuration fail
// closed before any image pull or application service start.
func (manifest Manifest) ValidateBindings(inputs BindingInputs) error {
	if err := manifest.Validate(); err != nil {
		return err
	}
	if inputs.ExpectedManifestHash != manifest.ManifestDigest {
		return errors.New("preheat manifest digest does not match the Compose configuration fingerprint")
	}
	if !validProfile(inputs.Profile) {
		return errors.New("preheat profile is invalid")
	}
	if sha256Digest(inputs.ReleaseManifest) != manifest.Release.ManifestDigest ||
		sha256Digest(inputs.Compose) != manifest.Release.ComposeDigest ||
		sha256Digest(inputs.ThirdPartyPolicy) != manifest.Release.ThirdPartyPolicyDigest {
		return errors.New("preheat manifest release binding does not match mounted files")
	}
	// A preheat manifest is a modern-release capability. Using the legacy
	// compatibility parser here would reject every release that carries the
	// runtimeComposition binding this contract is required to verify.
	release, err := releasemanifest.Parse(inputs.ReleaseManifest)
	if err != nil {
		return fmt.Errorf("parse release manifest binding: %w", err)
	}
	if release.Digest() != manifest.Release.ManifestDigest || "v"+release.ReleaseVersion != manifest.Release.Tag {
		return errors.New("preheat manifest release identity does not match mounted release manifest")
	}
	composition, err := decodeRuntimeCompositionBinding(inputs.RuntimeComposition)
	if err != nil {
		return fmt.Errorf("parse runtime composition binding: %w", err)
	}
	if composition.CompositionDigest != manifest.Release.CompositionDigest ||
		composition.ReleaseTag != manifest.Release.Tag ||
		composition.ManifestDigest != manifest.Release.ManifestDigest {
		return errors.New("preheat manifest runtime composition binding does not match mounted files")
	}
	if !release.HasRuntimeComposition() || release.RuntimeComposition.SHA256 != manifest.Release.CompositionDigest {
		return errors.New("preheat manifest runtime composition digest does not match release manifest")
	}
	return nil
}
