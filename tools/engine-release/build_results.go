package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"regexp"
	"sort"
	"strconv"

	"github.com/yyhuni/lunafox/contracts/enginemanifest/runtimeimage"
	"github.com/yyhuni/lunafox/contracts/ociartifact"
)

const (
	buildResultsSchemaVersion = "lunafox.engine-runtime-image-build-results.v1"
	buildModeDevelopment      = "development"
	buildModeProduction       = "production"
	ociImageIndexMediaType    = "application/vnd.oci.image.index.v1+json"
)

// RuntimeImageBuildResults is an ephemeral publisher receipt. It proves the
// inputs consumed by package generation came from one image build and one
// verified OCI index. It is not an Engine manifest, package file, or runtime
// identity source and must never be copied into an archive.
type RuntimeImageBuildResults struct {
	SchemaVersion string                    `json:"schemaVersion"`
	Mode          string                    `json:"mode"`
	Engines       []RuntimeImageBuildResult `json:"engines"`
}

type RuntimeImageBuildResult struct {
	EngineID       string   `json:"engineId"`
	Dockerfile     string   `json:"dockerfile"`
	BuildContext   string   `json:"buildContext"`
	Repository     string   `json:"repository"`
	BuildCount     int      `json:"buildCount"`
	IndexDigest    string   `json:"indexDigest"`
	IndexMediaType string   `json:"indexMediaType"`
	Platforms      []string `json:"platforms"`
	Refs           []string `json:"refs"`
	SourceRef      string   `json:"sourceRef"`
	CopiedRefs     []string `json:"copiedRefs"`
}

var sha256DigestPattern = regexp.MustCompile(`^sha256:[a-f0-9]{64}$`)

func decodeBuildResults(payload []byte, source string) (RuntimeImageBuildResults, error) {
	if err := rejectDuplicateJSONFields(bytes.NewReader(payload)); err != nil {
		return RuntimeImageBuildResults{}, fmt.Errorf("decode image build results %q: %w", source, err)
	}
	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.DisallowUnknownFields()
	var results RuntimeImageBuildResults
	if err := decoder.Decode(&results); err != nil {
		return RuntimeImageBuildResults{}, fmt.Errorf("decode image build results %q: %w", source, err)
	}
	if err := consumeJSONEOF(decoder); err != nil {
		return RuntimeImageBuildResults{}, fmt.Errorf("decode image build results %q: %w", source, err)
	}
	return results, nil
}

func validateBuildResults(discovery Discovery, results RuntimeImageBuildResults, expectedMode string) error {
	if results.SchemaVersion != buildResultsSchemaVersion {
		return fmt.Errorf("unsupported image build results schemaVersion %q", results.SchemaVersion)
	}
	if results.Mode != buildModeDevelopment && results.Mode != buildModeProduction {
		return fmt.Errorf("unsupported image build results mode %q", results.Mode)
	}
	if expectedMode != "" && results.Mode != expectedMode {
		return fmt.Errorf("image build results mode %q does not match expected mode %q", results.Mode, expectedMode)
	}
	if len(results.Engines) == 0 {
		return fmt.Errorf("image build results engines cannot be empty")
	}
	if len(results.Engines) != len(discovery.Engines) {
		return fmt.Errorf("image build results engine count %d does not match discovery count %d", len(results.Engines), len(discovery.Engines))
	}
	byID := make(map[string]RuntimeImageBuildResult, len(results.Engines))
	lastEngineID := ""
	for index, result := range results.Engines {
		if result.EngineID == "" {
			return fmt.Errorf("image build results engines[%d].engineId is required", index)
		}
		if _, duplicate := byID[result.EngineID]; duplicate {
			return fmt.Errorf("image build results contains duplicate engineId %q", result.EngineID)
		}
		if lastEngineID != "" && result.EngineID <= lastEngineID {
			return fmt.Errorf("image build results engines must be ordered by canonical engineId")
		}
		lastEngineID = result.EngineID
		byID[result.EngineID] = result
	}
	for _, source := range discovery.Engines {
		result, ok := byID[source.EngineID]
		if !ok {
			return fmt.Errorf("image build results is missing discovered engine %q", source.EngineID)
		}
		if err := validateOneBuildResult(source, result, results.Mode); err != nil {
			return err
		}
	}
	return nil
}

func validateOneBuildResult(source EngineSource, result RuntimeImageBuildResult, mode string) error {
	if result.Dockerfile != source.Dockerfile {
		return fmt.Errorf("image build result %q Dockerfile %q does not match discovered %q", result.EngineID, result.Dockerfile, source.Dockerfile)
	}
	if result.BuildContext != source.BuildContext {
		return fmt.Errorf("image build result %q buildContext %q does not match discovered %q", result.EngineID, result.BuildContext, source.BuildContext)
	}
	if result.Repository != source.Repository {
		return fmt.Errorf("image build result %q repository %q does not match engineId-derived %q", result.EngineID, result.Repository, source.Repository)
	}
	if result.BuildCount != 1 {
		return fmt.Errorf("image build result %q must record exactly one image build, got %d", result.EngineID, result.BuildCount)
	}
	if !sha256DigestPattern.MatchString(result.IndexDigest) {
		return fmt.Errorf("image build result %q indexDigest must be a canonical sha256 digest", result.EngineID)
	}
	if result.IndexMediaType != ociImageIndexMediaType {
		return fmt.Errorf("image build result %q indexMediaType %q is not an OCI image index", result.EngineID, result.IndexMediaType)
	}
	if err := validatePlatforms(result.Platforms, mode); err != nil {
		return fmt.Errorf("image build result %q platforms: %w", result.EngineID, err)
	}
	if result.SourceRef == "" || len(result.Refs) == 0 {
		return fmt.Errorf("image build result %q must include sourceRef and refs", result.EngineID)
	}
	candidates, err := runtimeimage.ParseFirstPartyCandidates(result.EngineID, result.Refs)
	if err != nil {
		return fmt.Errorf("image build result %q refs: %w", result.EngineID, err)
	}
	if candidates.RuntimeImageDigest != runtimeimage.RuntimeImageDigest(result.IndexDigest) {
		return fmt.Errorf("image build result %q refs digest %q does not match verified index digest %q", result.EngineID, candidates.RuntimeImageDigest, result.IndexDigest)
	}
	if result.SourceRef != result.Refs[0] {
		return fmt.Errorf("image build result %q sourceRef must be the first ordered candidate", result.EngineID)
	}
	if mode == buildModeDevelopment {
		if len(result.Refs) != 1 || len(result.CopiedRefs) != 0 {
			return fmt.Errorf("development image build result %q must contain exactly one uncopied candidate", result.EngineID)
		}
		if err := validateDevelopmentRegistry(candidates.References[0].Registry); err != nil {
			return fmt.Errorf("development image build result %q must use a host-reachable localhost Registry candidate", result.EngineID)
		}
		return nil
	}
	if len(result.Refs) != 2 || len(result.CopiedRefs) != 1 {
		return fmt.Errorf("production image build result %q must contain two candidates and one copied candidate", result.EngineID)
	}
	if candidates.References[0].Registry != "docker.io" || candidates.References[1].Registry != "ghcr.io" {
		return fmt.Errorf("production image build result %q must order docker.io then ghcr.io", result.EngineID)
	}
	if result.CopiedRefs[0] != result.Refs[1] {
		return fmt.Errorf("production image build result %q copiedRefs must identify the copied GHCR candidate", result.EngineID)
	}
	return nil
}

func validateDevelopmentRegistry(registry string) error {
	host, portText, err := net.SplitHostPort(registry)
	if err != nil || host != "localhost" {
		return fmt.Errorf("development Registry must use localhost with an explicit port")
	}
	port, err := strconv.Atoi(portText)
	if err != nil || port < 1 || port > 65535 {
		return fmt.Errorf("development Registry port is invalid")
	}
	return nil
}

func validatePlatforms(platforms []string, mode string) error {
	if len(platforms) == 0 || len(platforms) > 2 {
		return fmt.Errorf("must contain one or two supported Linux platforms")
	}
	want := []string{"linux/amd64", "linux/arm64"}
	actual := append([]string(nil), platforms...)
	sort.Strings(actual)
	for index, platform := range actual {
		if platform != want[0] && platform != want[1] {
			return fmt.Errorf("must contain only linux/amd64 and linux/arm64")
		}
		if index > 0 && platform == actual[index-1] {
			return fmt.Errorf("must not repeat a platform")
		}
	}
	if mode == buildModeProduction && (len(actual) != 2 || actual[0] != want[0] || actual[1] != want[1]) {
		return fmt.Errorf("production must contain linux/amd64 and linux/arm64 exactly once")
	}
	return nil
}

func rejectDuplicateJSONFields(reader io.Reader) error {
	decoder := json.NewDecoder(reader)
	return walkJSONValue(decoder)
}

func walkJSONValue(decoder *json.Decoder) error {
	token, err := decoder.Token()
	if err != nil {
		return err
	}
	delim, ok := token.(json.Delim)
	if !ok {
		return nil
	}
	switch delim {
	case '{':
		seen := map[string]struct{}{}
		for decoder.More() {
			field, err := decoder.Token()
			if err != nil {
				return err
			}
			name, ok := field.(string)
			if !ok {
				return fmt.Errorf("JSON object field name must be a string")
			}
			if _, exists := seen[name]; exists {
				return fmt.Errorf("duplicate JSON field %q", name)
			}
			seen[name] = struct{}{}
			if err := walkJSONValue(decoder); err != nil {
				return err
			}
		}
		closing, err := decoder.Token()
		if err != nil {
			return err
		}
		if closing != json.Delim('}') {
			return fmt.Errorf("unexpected JSON delimiter %q", closing)
		}
	case '[':
		for decoder.More() {
			if err := walkJSONValue(decoder); err != nil {
				return err
			}
		}
		closing, err := decoder.Token()
		if err != nil {
			return err
		}
		if closing != json.Delim(']') {
			return fmt.Errorf("unexpected JSON delimiter %q", closing)
		}
	default:
		return fmt.Errorf("unexpected JSON delimiter %q", delim)
	}
	return nil
}

func consumeJSONEOF(decoder *json.Decoder) error {
	if _, err := decoder.Token(); err != nil {
		if err == io.EOF {
			return nil
		}
		return err
	}
	return fmt.Errorf("unexpected trailing JSON content")
}

func parseBuildResultRefs(result RuntimeImageBuildResult) (ociartifact.DigestReference, error) {
	if len(result.Refs) == 0 {
		return ociartifact.DigestReference{}, fmt.Errorf("runtime image refs are required")
	}
	return ociartifact.ParseDigestReference(result.Refs[0])
}
