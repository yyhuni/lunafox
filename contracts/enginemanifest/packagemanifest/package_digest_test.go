package packagemanifest

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/opencontainers/go-digest"
	"github.com/yyhuni/lunafox/contracts/ociartifact"
)

func TestComputePackageDigestUsesExactArchiveBytes(t *testing.T) {
	payload := []byte("exact .lfengine.tar.gz bytes")

	got, size, err := ComputePackageDigest(bytes.NewReader(payload))
	if err != nil {
		t.Fatalf("ComputePackageDigest() error = %v", err)
	}
	want := ociartifact.PackageDigest(digest.FromBytes(payload).String())
	if got != want || size != int64(len(payload)) {
		t.Fatalf("digest/size = %q/%d, want %q/%d", got, size, want, len(payload))
	}
}

func TestComputePackageDigestDistinguishesArchiveSerialization(t *testing.T) {
	first := archiveWithGzipModTime(t, time.Unix(0, 0))
	second := archiveWithGzipModTime(t, time.Unix(1, 0))
	if bytes.Equal(first, second) {
		t.Fatal("fixture archives must differ at the byte level")
	}
	if !bytes.Equal(firstArchiveEntryPayload(t, first), firstArchiveEntryPayload(t, second)) {
		t.Fatal("fixture archives must expand to the same logical content")
	}

	firstDigest, _, err := ComputePackageDigest(bytes.NewReader(first))
	if err != nil {
		t.Fatal(err)
	}
	secondDigest, _, err := ComputePackageDigest(bytes.NewReader(second))
	if err != nil {
		t.Fatal(err)
	}
	if firstDigest == secondDigest {
		t.Fatal("different archive bytes must not share packageDigest")
	}
}

func firstArchiveEntryPayload(t *testing.T, archive []byte) []byte {
	t.Helper()
	gz, err := gzip.NewReader(bytes.NewReader(archive))
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if closeErr := gz.Close(); closeErr != nil {
			t.Errorf("close gzip reader: %v", closeErr)
		}
	}()
	tr := tar.NewReader(gz)
	if _, err := tr.Next(); err != nil {
		t.Fatal(err)
	}
	payload, err := io.ReadAll(tr)
	if err != nil {
		t.Fatal(err)
	}
	return payload
}

func TestParsePackageDigestRejectsNonCanonicalDigest(t *testing.T) {
	for _, value := range []string{"", " sha256:" + strings.Repeat("a", 64), "sha256:ABC", "sha256:abc"} {
		if _, err := ociartifact.ParsePackageDigest(value); err == nil {
			t.Fatalf("ParsePackageDigest(%q) succeeded, want rejection", value)
		}
	}
}

func archiveWithGzipModTime(t *testing.T, modTime time.Time) []byte {
	t.Helper()
	var out bytes.Buffer
	gz := gzip.NewWriter(&out)
	gz.ModTime = modTime
	tw := tar.NewWriter(gz)
	payload := []byte(`{"same":"expanded content"}`)
	if err := tw.WriteHeader(&tar.Header{Name: "engine.json", Mode: 0o644, Size: int64(len(payload))}); err != nil {
		t.Fatal(err)
	}
	if _, err := tw.Write(payload); err != nil {
		t.Fatal(err)
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
	return out.Bytes()
}
