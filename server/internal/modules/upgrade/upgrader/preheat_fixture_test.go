package upgrader

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/yyhuni/lunafox/contracts/ociartifact"
	"github.com/yyhuni/lunafox/contracts/preheatmanifest"
	"github.com/yyhuni/lunafox/contracts/releasemanifest"
)

// installPublicPreheatFixture turns the compact release fixture used by the
// upgrader tests into a complete modern deployment snapshot. The checked-in
// fixture intentionally uses a placeholder composition digest; this helper
// replaces it with a deterministic, fully validated composition before writing
// the release-bound preheat manifest and private target cache.
func installPublicPreheatFixture(t *testing.T, root string, store *JournalStore, raw []byte) (*releasemanifest.Manifest, []byte) {
	t.Helper()
	manifest, err := releasemanifest.Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	core, compositionDigest, err := publicFixtureCompositionCore(manifest)
	if err != nil {
		t.Fatal(err)
	}
	updatedRaw := []byte(strings.Replace(string(raw), manifest.RuntimeComposition.SHA256, compositionDigest, 1))
	manifest, err = releasemanifest.Parse(updatedRaw)
	if err != nil {
		t.Fatal(err)
	}
	composition := map[string]any{
		"schemaVersion":     core["schemaVersion"],
		"kind":              core["kind"],
		"releaseTag":        core["releaseTag"],
		"components":        core["components"],
		"capabilities":      core["capabilities"],
		"compositionDigest": compositionDigest,
		"manifestBinding":   map[string]any{"manifestDigest": manifest.Digest()},
	}
	compositionBytes, err := json.Marshal(composition)
	if err != nil {
		t.Fatal(err)
	}
	composePath := filepath.Join(root, publicComposeFile)
	composeBytes, err := os.ReadFile(composePath)
	if err != nil {
		t.Fatal(err)
	}
	policyBytes := []byte(`{"schemaVersion":1,"profiles":{"embedded":{},"external":{}}}`)
	if err := os.WriteFile(filepath.Join(root, publicRuntimeCompositionFile), compositionBytes, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, publicThirdPartyPolicyFile), policyBytes, 0o600); err != nil {
		t.Fatal(err)
	}
	preheatBytes := publicFixturePreheatManifest(manifest, composeBytes, policyBytes)
	if err := os.WriteFile(filepath.Join(root, publicPreheatManifestFile), preheatBytes, 0o600); err != nil {
		t.Fatal(err)
	}
	preheatValue := map[string]any{}
	if err := json.Unmarshal(preheatBytes, &preheatValue); err != nil {
		t.Fatal(err)
	}
	envPath := filepath.Join(root, publicEnvFile)
	envBytes, err := os.ReadFile(envPath)
	if err != nil {
		t.Fatal(err)
	}
	envBytes = append(envBytes, []byte("LUNAFOX_PREHEAT_MANIFEST_DIGEST="+preheatValue["manifestDigest"].(string)+"\n")...)
	if err := os.WriteFile(envPath, envBytes, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, defaultManifestName), updatedRaw, 0o600); err != nil {
		t.Fatal(err)
	}
	manifestPath, err := store.ManifestPath(manifest.Digest())
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(manifestPath, updatedRaw, 0o600); err != nil {
		t.Fatal(err)
	}
	compositionPath, err := store.RuntimeCompositionPath(manifest.RuntimeComposition.SHA256)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(compositionPath, compositionBytes, 0o600); err != nil {
		t.Fatal(err)
	}
	preheatPath, err := store.PreheatManifestPath(manifest.Digest())
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(preheatPath, preheatBytes, 0o600); err != nil {
		t.Fatal(err)
	}
	assetRoot, err := store.DeploymentAssetsPath(manifest.Digest())
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(assetRoot, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(assetRoot, publicComposeFile), composeBytes, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(assetRoot, publicThirdPartyPolicyFile), policyBytes, 0o600); err != nil {
		t.Fatal(err)
	}
	return manifest, updatedRaw
}

func publicFixturePreheatManifest(manifest *releasemanifest.Manifest, compose, policy []byte) []byte {
	digest := func(letter string) string { return "sha256:" + strings.Repeat(letter, 64) }
	repository := "yyhuni/lunafox-engine-runtime-port-scan"
	entryDigest := digest("c")
	entry := preheatmanifest.Entry{
		Candidates:           []string{"docker.io/" + repository + "@" + entryDigest, "ghcr.io/" + repository + "@" + entryDigest},
		CloudflareCandidates: []string{"docker.lunafox.cc.cd/" + repository + "@" + entryDigest, "docker.io/" + repository + "@" + entryDigest, "ghcr.io/" + repository + "@" + entryDigest},
		Digest:               entryDigest,
		IdentityReference:    "ghcr.io/" + repository + "@" + entryDigest,
		Platforms:            []string{preheatmanifest.PlatformLinuxAMD64, preheatmanifest.PlatformLinuxARM64},
		Profiles:             []string{preheatmanifest.ProfileEmbedded, preheatmanifest.ProfileExternal},
		Repository:           repository,
		Sources:              []preheatmanifest.LogicalSource{{Kind: "engine-runtime", Name: "engine.lunafox.port_scan"}},
		Trust:                "first-party",
	}
	identity := repository + "@" + entryDigest
	closures := make([]preheatmanifest.ProfileClosure, 0, 2)
	for _, profile := range []string{preheatmanifest.ProfileEmbedded, preheatmanifest.ProfileExternal} {
		closurePayload, _ := json.Marshal(map[string]any{"entries": []string{identity}, "profile": profile})
		closures = append(closures, preheatmanifest.ProfileClosure{Profile: profile, Entries: []string{identity}, Digest: sha256BytesForFixture(closurePayload)})
	}
	value := map[string]any{
		"entries":         []preheatmanifest.Entry{entry},
		"kind":            preheatmanifest.Kind,
		"profileClosures": closures,
		"release": preheatmanifest.ReleaseBinding{
			ComposeDigest:          sha256BytesForFixture(compose),
			CompositionDigest:      manifest.RuntimeComposition.SHA256,
			ManifestDigest:         manifest.Digest(),
			Tag:                    "v" + manifest.ReleaseVersion,
			ThirdPartyPolicyDigest: sha256BytesForFixture(policy),
		},
		"schemaVersion": preheatmanifest.SchemaVersion,
	}
	core, _ := json.Marshal(value)
	object := map[string]any{}
	_ = json.Unmarshal(core, &object)
	object["manifestDigest"] = sha256BytesForFixture(core)
	result, _ := json.Marshal(object)
	return result
}

func publicFixtureCompositionCore(manifest *releasemanifest.Manifest) (map[string]any, string, error) {
	if manifest == nil {
		return nil, "", fmt.Errorf("fixture manifest is nil")
	}
	digest := func(letter string) string { return "sha256:" + strings.Repeat(letter, 64) }
	components := make([]map[string]any, 0, 7)
	for _, name := range []string{"server", "frontend", "nginx", "agent", "bootstrap"} {
		componentDigest, err := manifest.RuntimeImageDigest(name)
		if err != nil {
			return nil, "", err
		}
		components = append(components, publicFixtureComponent("runtime."+name, "runtime", name, componentDigest, manifest.ReleaseVersion))
	}
	if len(manifest.EnginePackages) == 0 {
		return nil, "", fmt.Errorf("fixture has no engine package")
	}
	packageRef, err := ociartifact.ParseDigestReference(manifest.EnginePackages[0].Refs[0])
	if err != nil {
		return nil, "", err
	}
	components = append(components,
		publicFixtureComponent("engine.port-scan.package", "engine", "package", packageRef.Digest, manifest.ReleaseVersion),
		publicFixtureComponent("engine.port-scan.runtime", "engine", "runtime", digest("d"), manifest.ReleaseVersion),
	)
	sort.Slice(components, func(left, right int) bool { return components[left]["id"].(string) < components[right]["id"].(string) })
	core := map[string]any{
		"schemaVersion": 1,
		"kind":          "lunafox.runtime-composition",
		"releaseTag":    "v" + manifest.ReleaseVersion,
		"components":    components,
		"capabilities":  map[string]any{"dynamicFrontendUpstream": true},
	}
	coreBytes, err := json.Marshal(core)
	if err != nil {
		return nil, "", err
	}
	return core, sha256BytesForFixture(coreBytes), nil
}

func publicFixtureComponent(id, kind, name, componentDigest, releaseVersion string) map[string]any {
	inputs := map[string]any{
		"schemaVersion":      2,
		"componentId":        id,
		"kind":               kind,
		"contextPath":        ".",
		"dockerfile":         id + "/Dockerfile",
		"dockerignore":       "",
		"files":              []any{},
		"namedContexts":      map[string]any{},
		"buildArgs":          map[string]any{},
		"platforms":          []any{"linux/amd64", "linux/arm64"},
		"baseImages":         []any{},
		"baseImagesResolved": true,
		"builderPolicy":      map[string]any{},
		"generatedInputs":    []any{},
	}
	inputBytes, _ := json.Marshal(inputs)
	return map[string]any{
		"id":   id,
		"kind": kind,
		"name": name,
		"inputFingerprint": map[string]any{
			"version":            2,
			"algorithm":          "sha256-canonical-json-v1",
			"digest":             sha256BytesForFixture(inputBytes),
			"baseImagesResolved": true,
			"inputs":             inputs,
		},
		"artifact": map[string]any{
			"ref":    "ghcr.io/yyhuni/lunafox-" + name + "@" + componentDigest,
			"digest": componentDigest,
		},
		"disposition":   "built",
		"sourceRelease": map[string]any{"tag": "v" + releaseVersion},
		"evidence": map[string]any{
			"image":      "image.json",
			"provenance": "provenance.json",
			"sbom":       "sbom.json",
			"signature":  "signature.json",
		},
	}
}

func sha256BytesForFixture(value []byte) string {
	sum := sha256.Sum256(value)
	return "sha256:" + fmt.Sprintf("%x", sum[:])
}
