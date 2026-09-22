package upgrader

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
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
