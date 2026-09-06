package enginepackagebuild

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"io"
	"reflect"
	"testing"
	"time"

	enginepackagecatalog "github.com/yyhuni/lunafox/contracts/enginemanifest/packagecatalog"
	"github.com/yyhuni/lunafox/contracts/enginemanifest/packagemanifest"
)

func TestWriteEnginePackageArchiveIsDeterministicAndClosed(t *testing.T) {
	entries := validPackageArchiveEntries()
	var first bytes.Buffer
	firstDigest, firstSize, err := WriteEnginePackageArchive(&first, entries)
	if err != nil {
		t.Fatalf("WriteEnginePackageArchive() error = %v", err)
	}
	reversed := append([]enginepackagecatalog.PackageLayoutEntry(nil), entries...)
	for left, right := 0, len(reversed)-1; left < right; left, right = left+1, right-1 {
		reversed[left], reversed[right] = reversed[right], reversed[left]
	}
	var second bytes.Buffer
	secondDigest, secondSize, err := WriteEnginePackageArchive(&second, reversed)
	if err != nil {
		t.Fatalf("WriteEnginePackageArchive() second error = %v", err)
	}
	if !bytes.Equal(first.Bytes(), second.Bytes()) || firstDigest != secondDigest || firstSize != secondSize {
		t.Fatal("identical normalized four-file input must produce identical archive bytes and packageDigest")
	}

	wantDigest, wantSize, err := packagemanifest.ComputePackageDigest(bytes.NewReader(first.Bytes()))
	if err != nil {
		t.Fatal(err)
	}
	if firstDigest != wantDigest || firstSize != wantSize {
		t.Fatalf("writer digest/size = %q/%d, exact-byte hash = %q/%d", firstDigest, firstSize, wantDigest, wantSize)
	}

	gz, err := gzip.NewReader(bytes.NewReader(first.Bytes()))
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if closeErr := gz.Close(); closeErr != nil {
			t.Errorf("close gzip reader: %v", closeErr)
		}
	}()
	if !gz.ModTime.IsZero() || gz.Name != "" || gz.Comment != "" || len(gz.Extra) != 0 || gz.OS != 255 {
		t.Fatalf("non-canonical gzip header: %+v", gz.Header)
	}

	tr := tar.NewReader(gz)
	var paths []string
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		paths = append(paths, header.Name)
		if header.Mode != 0o644 || header.Uid != 0 || header.Gid != 0 || header.Uname != "" || header.Gname != "" ||
			header.Typeflag != tar.TypeReg || header.Format != tar.FormatUSTAR || header.Linkname != "" ||
			!header.ModTime.Equal(time.Unix(0, 0)) || !header.AccessTime.IsZero() || !header.ChangeTime.IsZero() ||
			len(header.PAXRecords) != 0 || header.Devmajor != 0 || header.Devminor != 0 {
			t.Fatalf("non-canonical tar header for %q: %+v", header.Name, header)
		}
		if _, err := io.Copy(io.Discard, tr); err != nil {
			t.Fatal(err)
		}
	}
	if want := enginepackagecatalog.RequiredEnginePackagePaths(); !reflect.DeepEqual(paths, want) {
		t.Fatalf("archive paths = %#v, want canonical order %#v", paths, want)
	}
}

func validPackageArchiveEntries() []enginepackagecatalog.PackageLayoutEntry {
	digest := "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	locale := []byte(`{"engine":{"displayName":"Demo","description":"Demo"},"sections":{"scan":{"name":"Scan","description":"Scan settings","params":{"timeout":{"description":"Timeout"}}}}}`)
	return []enginepackagecatalog.PackageLayoutEntry{
		{Path: "package.json", Mode: 0o644, Payload: []byte(`{"packageFormatVersion":"lunafox.engine-package.v2","engineId":"engine.lunafox.port_scan","engineVersion":"1.2.3","runtimeImage":{"refs":["docker.io/lunafox/lunafox-engine-runtime-port-scan@` + digest + `"]}}`)},
		{Path: "engine.json", Mode: 0o644, Payload: []byte(`{"manifestVersion":"engine.v5","engineId":"engine.lunafox.port_scan","publisher":"lunafox","execution":{"engineApiMajor":2,"supportedTargetTypes":["domain","ip","cidr"],"configSections":[{"id":"scan","defaultEnabled":true,"params":[{"key":"timeout","type":"integer","default":30,"minimum":1}]}]}}`)},
		{Path: "locales/en.json", Mode: 0o644, Payload: append([]byte(nil), locale...)},
		{Path: "locales/zh.json", Mode: 0o644, Payload: append([]byte(nil), locale...)},
	}
}
