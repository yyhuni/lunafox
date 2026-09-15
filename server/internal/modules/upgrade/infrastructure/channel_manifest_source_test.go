package infrastructure

import (
	"crypto/sha256"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
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
	return []byte(strings.ReplaceAll(string(raw), "1.2.3", version))
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
