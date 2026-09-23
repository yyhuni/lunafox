package infrastructure

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"testing"

	"github.com/yyhuni/lunafox/contracts/ociartifact"
	"github.com/yyhuni/lunafox/contracts/releasemanifest"
)

func TestChannelManifestSourceFetchesValidCandidateAndCachesDigest(t *testing.T) {
	manifestBytes := releaseManifestBytes(t, "1.2.3")
	record := channelRecordBytes("v1.2.3", manifestBytes)
	server := newReleaseMetadataServer(t, func() ([]byte, []byte) { return record, manifestBytes })
	source := newChannelSourceForTest(t, server.URL)

	manifest, err := source.Load()
	if err != nil {
		t.Fatal(err)
	}
	if manifest.ReleaseVersion != "1.2.3" {
		t.Fatalf("release version = %q, want 1.2.3", manifest.ReleaseVersion)
	}
	cached, err := os.ReadFile(filepath.Join(source.cacheDir, strings.TrimPrefix(manifest.Digest(), "sha256:")+".yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if string(cached) != string(manifestBytes) {
		t.Fatal("cached manifest bytes differ from the validated response")
	}
	compositionPath := filepath.Join(source.compositionCache, strings.TrimPrefix(manifest.RuntimeComposition.SHA256, "sha256:")+".json")
	compositionInfo, err := os.Stat(compositionPath)
	if err != nil {
		t.Fatal(err)
	}
	if compositionInfo.Mode().Perm() != 0o600 {
		t.Fatalf("composition cache mode = %o, want 600", compositionInfo.Mode().Perm())
	}
}

func TestChannelManifestSourceAcceptsPinnedLegacyAlpha114WithoutCompositionFetch(t *testing.T) {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller() failed")
	}
	manifestPath := filepath.Join(filepath.Dir(filename), "..", "..", "..", "..", "..", "scripts", "ci", "fixtures", "legacy-alpha114-release.manifest.yaml")
	manifestBytes, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	var compositionRequested bool
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/channels/canary.env":
			_, _ = fmt.Fprintf(writer, "SCHEMA_VERSION=3\nVERSION=v%s\nRELEASE_MANIFEST=manifests/v%s.yaml\nRELEASE_MANIFEST_SHA256=%s\n", releasemanifest.LegacyAlpha114ReleaseVersion, releasemanifest.LegacyAlpha114ReleaseVersion, strings.TrimPrefix(releasemanifest.LegacyAlpha114ManifestDigest, "sha256:"))
		case "/manifests/v0.0.1-alpha.114.yaml":
			_, _ = writer.Write(manifestBytes)
		default:
			if strings.HasSuffix(request.URL.Path, "/runtime-composition.json") {
				compositionRequested = true
			}
			http.NotFound(writer, request)
		}
	}))
	t.Cleanup(server.Close)
	source := newChannelSourceForTest(t, server.URL)
	manifest, err := source.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if manifest.ReleaseVersion != releasemanifest.LegacyAlpha114ReleaseVersion || manifest.Digest() != releasemanifest.LegacyAlpha114ManifestDigest {
		t.Fatalf("legacy manifest identity = %s %s", manifest.ReleaseVersion, manifest.Digest())
	}
	if compositionRequested {
		t.Fatal("legacy channel load fetched a composition asset")
	}
	cachedPath := filepath.Join(source.cacheDir, strings.TrimPrefix(manifest.Digest(), "sha256:")+".yaml")
	if _, err := os.Stat(cachedPath); err != nil {
		t.Fatalf("legacy manifest was not cached: %v", err)
	}
	entries, err := os.ReadDir(source.compositionCache)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("legacy load unexpectedly cached composition evidence: %v", entries)
	}
}

func TestChannelManifestSourceAcceptsRegisteredAlpha164BridgeWithoutCompositionFetch(t *testing.T) {
	manifestBytes, err := os.ReadFile(filepath.Join("..", "testdata", "alpha164-bridge.release.manifest.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	const bridgeTag = "v0.0.1-alpha.183"
	var compositionRequested bool
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/channels/canary.env":
			_, _ = writer.Write(channelRecordBytes(bridgeTag, manifestBytes))
		case "/manifests/" + bridgeTag + ".yaml":
			_, _ = writer.Write(manifestBytes)
		default:
			if strings.HasSuffix(request.URL.Path, "/runtime-composition.json") {
				compositionRequested = true
			}
			http.NotFound(writer, request)
		}
	}))
	t.Cleanup(server.Close)
	source := newChannelSourceForTest(t, server.URL)

	manifest, err := source.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if manifest.ReleaseVersion != "0.0.1-alpha.183" || manifest.HasRuntimeComposition() {
		t.Fatalf("bridge manifest = version:%q composition:%t", manifest.ReleaseVersion, manifest.HasRuntimeComposition())
	}
	if compositionRequested {
		t.Fatal("bridge channel load fetched a composition asset")
	}
	entries, err := os.ReadDir(source.compositionCache)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("bridge load unexpectedly cached composition evidence: %v", entries)
	}

	reloaded, err := source.LoadTarget(manifest.Digest())
	if err != nil {
		t.Fatalf("LoadTarget() error = %v", err)
	}
	if reloaded.Digest() != manifest.Digest() || reloaded.HasRuntimeComposition() {
		t.Fatalf("reloaded bridge target = digest:%q composition:%t", reloaded.Digest(), reloaded.HasRuntimeComposition())
	}
}

func TestChannelManifestSourceRejectsDigestMismatch(t *testing.T) {
	manifestBytes := releaseManifestBytes(t, "1.2.3")
	record := []byte("SCHEMA_VERSION=3\nVERSION=v1.2.3\nRELEASE_MANIFEST=manifests/v1.2.3.yaml\nRELEASE_MANIFEST_SHA256=" + strings.Repeat("0", 64) + "\n")
	server := newReleaseMetadataServer(t, func() ([]byte, []byte) { return record, manifestBytes })
	source := newChannelSourceForTest(t, server.URL)

	if _, err := source.Load(); err == nil {
		t.Fatal("digest mismatch was accepted")
	}
}

func TestParseChannelRecordRejectsNonCanonicalInput(t *testing.T) {
	validDigest := strings.Repeat("a", 64)
	tests := map[string]string{
		"unknown field":   "SCHEMA_VERSION=3\nVERSION=v1.2.3\nRELEASE_MANIFEST=manifests/v1.2.3.yaml\nRELEASE_MANIFEST_SHA256=" + validDigest + "\nEXTRA=value\n",
		"duplicate field": "SCHEMA_VERSION=3\nVERSION=v1.2.3\nVERSION=v1.2.4\nRELEASE_MANIFEST=manifests/v1.2.3.yaml\nRELEASE_MANIFEST_SHA256=" + validDigest + "\n",
		"CRLF":            "SCHEMA_VERSION=3\r\nVERSION=v1.2.3\r\nRELEASE_MANIFEST=manifests/v1.2.3.yaml\r\nRELEASE_MANIFEST_SHA256=" + validDigest + "\r\n",
		"unsafe path":     "SCHEMA_VERSION=3\nVERSION=v1.2.3\nRELEASE_MANIFEST=../v1.2.3.yaml\nRELEASE_MANIFEST_SHA256=" + validDigest + "\n",
	}
	for name, raw := range tests {
		t.Run(name, func(t *testing.T) {
			if _, err := parseChannelRecord([]byte(raw)); err == nil {
				t.Fatal("non-canonical channel record was accepted")
			}
		})
	}
}

func TestChannelManifestSourceRejectsCrossOriginRedirect(t *testing.T) {
	target := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(target.Close)
	sourceServer := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		http.Redirect(writer, &http.Request{}, target.URL, http.StatusFound)
	}))
	t.Cleanup(sourceServer.Close)
	source := newChannelSourceForTest(t, sourceServer.URL)

	if _, err := source.Load(); err == nil || !strings.Contains(err.Error(), "changed origin") {
		t.Fatalf("cross-origin redirect error = %v", err)
	}
}

func TestChannelManifestSourceRejectsOversizedResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		_, _ = writer.Write([]byte(strings.Repeat("x", maxChannelRecordBytes+1)))
	}))
	t.Cleanup(server.Close)
	source := newChannelSourceForTest(t, server.URL)

	if _, err := source.Load(); err == nil || !strings.Contains(err.Error(), "exceeds") {
		t.Fatalf("oversized response error = %v", err)
	}
}

func TestChannelManifestSourceLoadTargetSurvivesChannelAdvancement(t *testing.T) {
	versions := []string{"1.2.3", "1.2.4"}
	var mu sync.Mutex
	index := 0
	server := newReleaseMetadataServer(t, func() ([]byte, []byte) {
		mu.Lock()
		defer mu.Unlock()
		manifestBytes := releaseManifestBytes(t, versions[index])
		return channelRecordBytes("v"+versions[index], manifestBytes), manifestBytes
	})
	source := newChannelSourceForTest(t, server.URL)

	first, err := source.Load()
	if err != nil {
		t.Fatal(err)
	}
	mu.Lock()
	index = 1
	mu.Unlock()
	second, err := source.Load()
	if err != nil {
		t.Fatal(err)
	}
	if first.Digest() == second.Digest() {
		t.Fatal("channel advancement did not change the manifest digest")
	}
	reloaded, err := source.LoadTarget(first.Digest())
	if err != nil {
		t.Fatal(err)
	}
	if reloaded.ReleaseVersion != first.ReleaseVersion || reloaded.Digest() != first.Digest() {
		t.Fatalf("reloaded target = %s %s, want %s %s", reloaded.ReleaseVersion, reloaded.Digest(), first.ReleaseVersion, first.Digest())
	}
}

func TestChannelManifestSourceRejectsExistingCacheSymlink(t *testing.T) {
	manifestBytes := releaseManifestBytes(t, "1.2.3")
	record := channelRecordBytes("v1.2.3", manifestBytes)
	server := newReleaseMetadataServer(t, func() ([]byte, []byte) { return record, manifestBytes })
	source := newChannelSourceForTest(t, server.URL)

	manifest, err := source.Load()
	if err != nil {
		t.Fatal(err)
	}
	cachePath := filepath.Join(source.cacheDir, strings.TrimPrefix(manifest.Digest(), "sha256:")+".yaml")
	if err := os.Remove(cachePath); err != nil {
		t.Fatal(err)
	}
	outsidePath := filepath.Join(t.TempDir(), "outside.yaml")
	if err := os.WriteFile(outsidePath, manifestBytes, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outsidePath, cachePath); err != nil {
		t.Skipf("symlink is unavailable on this host: %v", err)
	}

	if _, err := source.Load(); err == nil || !strings.Contains(err.Error(), "regular file") {
		t.Fatalf("existing cache symlink error = %v", err)
	}
}

func TestChannelManifestSourceRejectsOversizedExistingCacheFile(t *testing.T) {
	manifestBytes := releaseManifestBytes(t, "1.2.3")
	record := channelRecordBytes("v1.2.3", manifestBytes)
	server := newReleaseMetadataServer(t, func() ([]byte, []byte) { return record, manifestBytes })
	source := newChannelSourceForTest(t, server.URL)

	manifest, err := source.Load()
	if err != nil {
		t.Fatal(err)
	}
	cachePath := filepath.Join(source.cacheDir, strings.TrimPrefix(manifest.Digest(), "sha256:")+".yaml")
	if err := os.WriteFile(cachePath, []byte(strings.Repeat("x", maxReleaseManifestBytes+1)), 0o600); err != nil {
		t.Fatal(err)
	}

	if _, err := source.Load(); err == nil || !strings.Contains(err.Error(), "exceeds") {
		t.Fatalf("oversized existing cache error = %v", err)
	}
}

func newChannelSourceForTest(t *testing.T, baseURL string) *ChannelManifestSource {
	t.Helper()
	source, err := NewChannelManifestSource(ChannelManifestSourceConfig{
		MetadataBaseURL: baseURL,
		Channel:         "canary",
		DeploymentRoot:  t.TempDir(),
	})
	if err != nil {
		t.Fatal(err)
	}
	return source
}

func newReleaseMetadataServer(t *testing.T, current func() ([]byte, []byte)) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		record, manifest := current()
		switch request.URL.Path {
		case "/channels/canary.env":
			_, _ = writer.Write(record)
		case "/manifests/v1.2.3.yaml", "/manifests/v1.2.4.yaml":
			_, _ = writer.Write(manifest)
		case "/manifests/v1.2.3/runtime-composition.json", "/manifests/v1.2.4/runtime-composition.json":
			composition, err := runtimeCompositionBytes(t, manifest)
			if err != nil {
				http.Error(writer, err.Error(), http.StatusInternalServerError)
				return
			}
			_, _ = writer.Write(composition)
		default:
			http.NotFound(writer, request)
		}
	}))
	t.Cleanup(server.Close)
	return server
}

func releaseManifestBytes(t *testing.T, version string) []byte {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("..", "testdata", "release.manifest.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	// The fixture carries a placeholder composition digest. Replace it with a
	// digest calculated from the deterministic asset served by the test server.
	material := []byte(strings.ReplaceAll(string(raw), "1.2.3", version))
	manifest, err := releasemanifest.Parse(material)
	if err != nil {
		t.Fatal(err)
	}
	_, compositionDigest, err := compositionCoreForManifest(manifest)
	if err != nil {
		t.Fatal(err)
	}
	return []byte(strings.Replace(string(material), "sha256:"+strings.Repeat("b", 64), compositionDigest, 1))
}

func runtimeCompositionBytes(t *testing.T, manifestBytes []byte) ([]byte, error) {
	manifest, err := releasemanifest.Parse(manifestBytes)
	if err != nil {
		return nil, err
	}
	core, compositionDigest, err := compositionCoreForManifest(manifest)
	if err != nil {
		return nil, err
	}
	if manifest.RuntimeComposition.SHA256 != compositionDigest {
		return nil, fmt.Errorf("fixture manifest composition digest %s does not match generated %s", manifest.RuntimeComposition.SHA256, compositionDigest)
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
	return json.MarshalIndent(composition, "", "  ")
}

func compositionCoreForManifest(manifest *releasemanifest.Manifest) (map[string]any, string, error) {
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
		components = append(components, compositionComponent("runtime."+name, "runtime", name, componentDigest, manifest.ReleaseVersion))
	}
	if len(manifest.EnginePackages) == 0 {
		return nil, "", fmt.Errorf("fixture has no engine package")
	}
	packageRef, err := ociartifact.ParseDigestReference(manifest.EnginePackages[0].Refs[0])
	if err != nil {
		return nil, "", err
	}
	components = append(components,
		compositionComponent("engine.port-scan.package", "engine", "package", packageRef.Digest, manifest.ReleaseVersion),
		compositionComponent("engine.port-scan.runtime", "engine", "runtime", digest("d"), manifest.ReleaseVersion),
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
	coreSum := sha256.Sum256(coreBytes)
	compositionDigest := "sha256:" + fmt.Sprintf("%x", coreSum[:])
	return core, compositionDigest, nil
}

func compositionComponent(id, kind, name, componentDigest, releaseVersion string) map[string]any {
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
	inputBytes, err := json.Marshal(inputs)
	if err != nil {
		panic(fmt.Sprintf("marshal composition fixture inputs: %v", err))
	}
	inputDigest := sha256.Sum256(inputBytes)
	return map[string]any{
		"id":   id,
		"kind": kind,
		"name": name,
		"inputFingerprint": map[string]any{
			"version":            2,
			"algorithm":          "sha256-canonical-json-v1",
			"digest":             "sha256:" + fmt.Sprintf("%x", inputDigest[:]),
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

func channelRecordBytes(version string, manifest []byte) []byte {
	digest := sha256.Sum256(manifest)
	return []byte(fmt.Sprintf(
		"SCHEMA_VERSION=3\nVERSION=%s\nRELEASE_MANIFEST=manifests/%s.yaml\nRELEASE_MANIFEST_SHA256=%x\n",
		version,
		version,
		digest,
	))
}
