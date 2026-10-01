package upgrader

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"

	"github.com/yyhuni/lunafox/contracts/releasemanifest"
)

func compositionSourceTestDigest(letter string) string {
	return "sha256:" + strings.Repeat(letter, 64)
}

func compositionSourceTestComponent(disposition, sourceTag string) map[string]any {
	digest := compositionSourceTestDigest("a")
	inputs := map[string]any{
		"schemaVersion": json.Number("2"),
		"componentId":   "runtime.frontend",
		"kind":          "runtime",
		"contextPath":   "frontend",
		"dockerfile":    "frontend/Dockerfile",
		"dockerignore":  "frontend/.dockerignore",
		"files": []any{
			map[string]any{"path": "frontend/Dockerfile", "digest": compositionSourceTestDigest("c"), "size": json.Number("1"), "mode": json.Number("420")},
			map[string]any{"path": "frontend/.dockerignore", "digest": compositionSourceTestDigest("d"), "size": json.Number("1"), "mode": json.Number("420")},
		},
		"namedContexts": map[string]any{},
		"buildArgs":     map[string]any{},
		"platforms":     []any{"linux/amd64", "linux/arm64"},
		"baseImages": []any{
			map[string]any{
				"ref":      "node:22-alpine",
				"identity": "node:22-alpine@" + compositionSourceTestDigest("e"),
				"resolved": true,
			},
		},
		"baseImagesResolved": true,
		"builderPolicy":      map[string]any{},
		"generatedInputs":    []any{},
	}
	canonical, err := canonicalCompositionJSON(inputs)
	if err != nil {
		panic(fmt.Sprintf("canonicalize composition fixture inputs: %v", err))
	}
	inputDigest := sha256.Sum256(canonical)
	return map[string]any{
		"id":   "runtime.frontend",
		"kind": "runtime",
		"name": "frontend",
		"inputFingerprint": map[string]any{
			"version":            json.Number("2"),
			"algorithm":          "sha256-canonical-json-v1",
			"digest":             "sha256:" + fmt.Sprintf("%x", inputDigest[:]),
			"baseImagesResolved": true,
			"inputs":             inputs,
		},
		"artifact": map[string]any{
			"ref":       "ghcr.io/yyhuni/lunafox-frontend@" + digest,
			"digest":    digest,
			"platforms": []any{"linux/amd64", "linux/arm64"},
		},
		"disposition": disposition,
		"sourceRelease": map[string]any{
			"tag": sourceTag,
		},
		"evidence": map[string]any{
			"image":      "runtime-evidence/frontend.json",
			"provenance": "runtime-evidence/frontend.json#provenance",
			"sbom":       "runtime-evidence/frontend.json#sbom",
			"signature":  "runtime-evidence/frontend.json#signature",
		},
	}
}

func TestNormalizeCompositionComponentRejectsBuiltEvidenceFromPriorRelease(t *testing.T) {
	_, err := normalizeCompositionComponent(compositionSourceTestComponent("built", "v1.0.0"), "v1.0.1", 0)
	if err == nil || !strings.Contains(err.Error(), "built source release must match current release") {
		t.Fatalf("built source release error = %v", err)
	}
}

func TestNormalizeCompositionComponentAcceptsBuiltEvidenceFromCurrentRelease(t *testing.T) {
	if _, err := normalizeCompositionComponent(compositionSourceTestComponent("built", "v1.0.1"), "v1.0.1", 0); err != nil {
		t.Fatalf("current built source release rejected: %v", err)
	}
}

func TestNormalizeCompositionComponentRequiresReusedCompositionBinding(t *testing.T) {
	_, err := normalizeCompositionComponent(compositionSourceTestComponent("reused", "v1.0.0"), "v1.0.1", 0)
	if err == nil || !strings.Contains(err.Error(), "reused source release composition digest is required") {
		t.Fatalf("reused composition binding error = %v", err)
	}
}

func TestNormalizeCompositionComponentRejectsMissingFingerprintClosure(t *testing.T) {
	component := compositionSourceTestComponent("built", "v1.0.1")
	delete(component["inputFingerprint"].(map[string]any), "inputs")
	_, err := normalizeCompositionComponent(component, "v1.0.1", 0)
	if err == nil || !strings.Contains(err.Error(), "inputs is required") {
		t.Fatalf("missing fingerprint inputs error = %v", err)
	}
}

func TestCompositionSourceTreatsPinnedLegacyAlpha114AsUnavailable(t *testing.T) {
	root := t.TempDir()
	store, err := NewPublicJournalStore(root)
	if err != nil {
		t.Fatal(err)
	}
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller() failed")
	}
	fixture := filepath.Join(filepath.Dir(filename), "..", "..", "..", "..", "..", "scripts", "ci", "fixtures", "legacy-alpha114-release.manifest.yaml")
	raw, err := os.ReadFile(fixture)
	if err != nil {
		t.Fatal(err)
	}
	manifestPath, err := store.ManifestPath(releasemanifest.LegacyAlpha114ManifestDigest)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(manifestPath, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	source := NewCompositionCandidateDeploymentSource(store)
	candidate, err := source.LoadCandidateDeployment(context.Background(), releasemanifest.LegacyAlpha114ManifestDigest)
	if !errors.Is(err, ErrCompositionUnavailable) {
		t.Fatalf("legacy composition load error = %v, want ErrCompositionUnavailable", err)
	}
	if candidate.CompositionDigest != "" {
		t.Fatalf("legacy candidate retained composition digest %q", candidate.CompositionDigest)
	}
}

func TestCompositionSourceTreatsRegisteredAlpha164BridgeAsUnavailable(t *testing.T) {
	root := t.TempDir()
	store, err := NewPublicJournalStore(root)
	if err != nil {
		t.Fatal(err)
	}
	raw := alpha164BridgeManifestFixtureBytes(t)
	manifest, err := releasemanifest.ParseLegacyCompatible(raw)
	if err != nil {
		t.Fatal(err)
	}
	manifestPath, err := store.ManifestPath(manifest.Digest())
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(manifestPath, raw, 0o600); err != nil {
		t.Fatal(err)
	}

	source := NewCompositionCandidateDeploymentSource(store)
	candidate, err := source.LoadCandidateDeployment(context.Background(), manifest.Digest())
	if !errors.Is(err, ErrCompositionUnavailable) {
		t.Fatalf("bridge composition load error = %v, want ErrCompositionUnavailable", err)
	}
	if candidate.ReleaseVersion != manifest.ReleaseVersion || candidate.CompositionDigest != "" {
		t.Fatalf("bridge candidate = version:%q composition:%q", candidate.ReleaseVersion, candidate.CompositionDigest)
	}
}

func alpha164BridgeManifestFixtureBytes(t *testing.T) []byte {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("..", "testdata", "alpha164-bridge.release.manifest.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func TestNormalizeCompositionComponentRejectsIncompleteOrMismatchedFingerprintClosure(t *testing.T) {
	component := compositionSourceTestComponent("built", "v1.0.1")
	inputs := component["inputFingerprint"].(map[string]any)["inputs"].(map[string]any)
	delete(inputs, "generatedInputs")
	canonical, err := canonicalCompositionJSON(inputs)
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(canonical)
	component["inputFingerprint"].(map[string]any)["digest"] = "sha256:" + fmt.Sprintf("%x", sum[:])
	_, err = normalizeCompositionComponent(component, "v1.0.1", 0)
	if err == nil || !strings.Contains(err.Error(), "generatedInputs is required") {
		t.Fatalf("incomplete fingerprint closure error = %v", err)
	}

	component = compositionSourceTestComponent("built", "v1.0.1")
	inputs = component["inputFingerprint"].(map[string]any)["inputs"].(map[string]any)
	inputs["files"] = append(inputs["files"].([]any), map[string]any{"path": "frontend/app", "digest": compositionSourceTestDigest("f"), "size": json.Number("1"), "mode": json.Number("420")})
	_, err = normalizeCompositionComponent(component, "v1.0.1", 0)
	if err == nil || !strings.Contains(err.Error(), "digest does not match inputs") {
		t.Fatalf("mismatched fingerprint digest error = %v", err)
	}
}

func TestCanonicalCompositionJSONSerializesStringSlicesLikeAnySlices(t *testing.T) {
	anyForm := map[string]any{
		"platforms": []any{"linux/amd64", "linux/arm64"},
		"evidence":  []any{"runtime-evidence/server.json#sbom", "runtime-evidence/server.json"},
	}
	stringForm := map[string]any{
		"platforms": []string{"linux/amd64", "linux/arm64"},
		"evidence":  []string{"runtime-evidence/server.json#sbom", "runtime-evidence/server.json"},
	}
	anyBytes, err := canonicalCompositionJSON(anyForm)
	if err != nil {
		t.Fatalf("canonicalize []any form: %v", err)
	}
	stringBytes, err := canonicalCompositionJSON(stringForm)
	if err != nil {
		t.Fatalf("canonicalize []string form: %v", err)
	}
	if !bytes.Equal(anyBytes, stringBytes) {
		t.Fatalf("canonical bytes diverge between slice forms:\n[]any:   %s\n[]string: %s", anyBytes, stringBytes)
	}
}

// compositionEndToEndComponent builds a fingerprint-closed component document
// for the full ValidateRuntimeCompositionAsset path. artifactDigest binds the
// component to the manifest inventory; evidence exercises the string and
// string-array forms the release resolver emits.
func compositionEndToEndComponent(id, artifactDigest string, evidence map[string]any) map[string]any {
	kind := strings.SplitN(id, ".", 2)[0]
	name := id[strings.Index(id, ".")+1:]
	inputs := map[string]any{
		"schemaVersion":      json.Number("2"),
		"componentId":        id,
		"kind":               kind,
		"contextPath":        "contexts/" + name,
		"dockerfile":         "contexts/" + name + "/Dockerfile",
		"dockerignore":       "contexts/" + name + "/.dockerignore",
		"files":              []any{map[string]any{"path": "contexts/" + name + "/Dockerfile", "digest": compositionSourceTestDigest("c"), "size": json.Number("1"), "mode": json.Number("420")}},
		"namedContexts":      map[string]any{},
		"buildArgs":          map[string]any{},
		"platforms":          []any{"linux/amd64", "linux/arm64"},
		"baseImages":         []any{map[string]any{"ref": "debian:bookworm-slim", "identity": "debian:bookworm-slim@" + compositionSourceTestDigest("e"), "resolved": true}},
		"baseImagesResolved": true,
		"builderPolicy":      map[string]any{},
		"generatedInputs":    []any{},
	}
	canonical, err := canonicalCompositionJSON(inputs)
	if err != nil {
		panic(fmt.Sprintf("canonicalize composition end-to-end inputs: %v", err))
	}
	inputDigest := sha256.Sum256(canonical)
	return map[string]any{
		"id":   id,
		"kind": kind,
		"name": name,
		"inputFingerprint": map[string]any{
			"version":            json.Number("2"),
			"algorithm":          "sha256-canonical-json-v1",
			"digest":             "sha256:" + fmt.Sprintf("%x", inputDigest[:]),
			"baseImagesResolved": true,
			"inputs":             inputs,
		},
		"artifact": map[string]any{
			"ref":       "ghcr.io/yyhuni/lunafox-" + strings.ReplaceAll(name, ".", "-") + "@" + artifactDigest,
			"digest":    artifactDigest,
			"platforms": []any{"linux/amd64", "linux/arm64"},
		},
		"disposition": "built",
		"sourceRelease": map[string]any{
			"tag": "v1.2.3",
		},
		"evidence": evidence,
	}
}

func TestValidateRuntimeCompositionAssetAcceptsStringSliceEvidenceAndPlatforms(t *testing.T) {
	runtimeDigest := compositionSourceTestDigest("a")
	enginePackageDigest := compositionSourceTestDigest("c")
	engineRuntimeDigest := compositionSourceTestDigest("d")
	components := []any{
		compositionEndToEndComponent("runtime.server", runtimeDigest, map[string]any{
			"image":      []any{"runtime-evidence/server.json"},
			"provenance": []any{"runtime-evidence/server.json#provenance"},
			"sbom":       []any{"runtime-evidence/server.json#sbom"},
			"signature":  []any{"runtime-evidence/server.json#signature"},
		}),
		compositionEndToEndComponent("runtime.frontend", runtimeDigest, map[string]any{
			"image":      "runtime-evidence/frontend.json",
			"provenance": "runtime-evidence/frontend.json#provenance",
			"sbom":       []any{"runtime-evidence/frontend.json#sbom"},
			"signature":  []any{"runtime-evidence/frontend.json#signature"},
		}),
		compositionEndToEndComponent("runtime.nginx", runtimeDigest, map[string]any{
			"image": "runtime-evidence/nginx.json", "provenance": "runtime-evidence/nginx.json#provenance",
			"sbom": "runtime-evidence/nginx.json#sbom", "signature": "runtime-evidence/nginx.json#signature",
		}),
		compositionEndToEndComponent("runtime.agent", runtimeDigest, map[string]any{
			"image": "runtime-evidence/agent.json", "provenance": "runtime-evidence/agent.json#provenance",
			"sbom": "runtime-evidence/agent.json#sbom", "signature": "runtime-evidence/agent.json#signature",
		}),
		compositionEndToEndComponent("runtime.bootstrap", runtimeDigest, map[string]any{
			"image": "runtime-evidence/bootstrap.json", "provenance": "runtime-evidence/bootstrap.json#provenance",
			"sbom": "runtime-evidence/bootstrap.json#sbom", "signature": "runtime-evidence/bootstrap.json#signature",
		}),
		compositionEndToEndComponent("engine.port-scan.runtime", engineRuntimeDigest, map[string]any{
			"image": "runtime-evidence/port-scan-runtime.json", "provenance": "runtime-evidence/port-scan-runtime.json#provenance",
			"sbom": "runtime-evidence/port-scan-runtime.json#sbom", "signature": "runtime-evidence/port-scan-runtime.json#signature",
		}),
		compositionEndToEndComponent("engine.port-scan.package", enginePackageDigest, map[string]any{
			"image": "runtime-evidence/port-scan-package.json", "provenance": "runtime-evidence/port-scan-package.json#provenance",
			"sbom": "runtime-evidence/port-scan-package.json#sbom", "signature": "runtime-evidence/port-scan-package.json#signature",
		}),
	}

	normalized := make([]normalizedCompositionComponent, 0, len(components))
	for index, item := range components {
		component, err := normalizeCompositionComponent(item, "v1.2.3", index)
		if err != nil {
			t.Fatalf("normalizeCompositionComponent(%d) error = %v", index, err)
		}
		normalized = append(normalized, component)
	}
	sort.Slice(normalized, func(left, right int) bool { return normalized[left].ID < normalized[right].ID })
	compositionDigest, err := canonicalCompositionCoreDigest(runtimeCompositionKind, "v1.2.3", "", "", normalized, map[string]any{})
	if err != nil {
		t.Fatalf("canonicalCompositionCoreDigest() error = %v", err)
	}

	manifestRaw, err := os.ReadFile(filepath.Join("..", "testdata", "release.manifest.yaml"))
	if err != nil {
		t.Fatalf("read manifest fixture: %v", err)
	}
	manifestRaw = bytes.ReplaceAll(manifestRaw,
		[]byte(compositionSourceTestDigest("b")),
		[]byte(compositionDigest))
	manifest, err := releasemanifest.Parse(manifestRaw)
	if err != nil {
		t.Fatalf("parse manifest fixture: %v", err)
	}

	document := map[string]any{
		"schemaVersion":     json.Number("1"),
		"kind":              runtimeCompositionKind,
		"releaseTag":        "v1.2.3",
		"components":        components,
		"capabilities":      map[string]any{},
		"compositionDigest": compositionDigest,
		"manifestBinding":   map[string]any{"manifestDigest": manifest.Digest()},
	}
	raw, err := json.Marshal(document)
	if err != nil {
		t.Fatalf("marshal composition document: %v", err)
	}
	if err := ValidateRuntimeCompositionAsset(raw, manifest, manifest.Digest(), ""); err != nil {
		t.Fatalf("ValidateRuntimeCompositionAsset() error = %v", err)
	}
}
