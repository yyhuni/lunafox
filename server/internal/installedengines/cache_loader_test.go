package installedengines

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	enginemanifest "github.com/yyhuni/lunafox/contracts/enginemanifest"
	"github.com/yyhuni/lunafox/contracts/enginemanifest/enginepackagebuild"
	enginepackagecatalog "github.com/yyhuni/lunafox/contracts/enginemanifest/packagecatalog"
	"github.com/yyhuni/lunafox/contracts/enginemanifest/packagemanifest"
	"github.com/yyhuni/lunafox/contracts/enginemanifest/repositoryname"
	"github.com/yyhuni/lunafox/server/internal/engineinstall"
)

func TestCacheInstallerExactPackageLoaderRequiresCacheConfiguration(t *testing.T) {
	for _, cache := range []engineinstall.CacheInstaller{
		{},
		{Root: t.TempDir()},
		{MaxArchiveBytes: 1 << 20},
	} {
		if _, err := NewCacheInstallerExactPackageLoader(cache); err == nil {
			t.Fatalf("NewCacheInstallerExactPackageLoader() accepted invalid cache %#v", cache)
		}
	}
}

func TestCacheInstallerExactPackageLoaderLoadsVerifiedArchiveWithoutExposingPath(t *testing.T) {
	fixture := newInstalledEnginePackageFixture(
		t,
		repositoryname.FirstPartyEngineIDPortScan,
		"1.2.3",
		'a',
		'b',
		'c',
	)
	entries := packageLayoutEntriesTest(t, fixture.layout)
	var archive bytes.Buffer
	digest, _, err := enginepackagebuild.WriteEnginePackageArchive(&archive, entries)
	if err != nil {
		t.Fatal(err)
	}
	cacheRoot := t.TempDir()
	t.Cleanup(func() {
		_ = filepath.Walk(cacheRoot, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil
			}
			if info.IsDir() {
				return os.Chmod(path, 0o700)
			}
			return os.Chmod(path, 0o600)
		})
		_ = os.RemoveAll(cacheRoot)
	})
	cacheKey := "sha256-" + strings.TrimPrefix(string(digest), "sha256:")
	archiveRoot := filepath.Join(cacheRoot, "archives")
	if err := os.MkdirAll(archiveRoot, 0o750); err != nil {
		t.Fatal(err)
	}
	archivePath := filepath.Join(archiveRoot, cacheKey+packagemanifest.EnginePackageArchiveSuffix)
	if err := os.WriteFile(archivePath, archive.Bytes(), 0o440); err != nil {
		t.Fatal(err)
	}

	loader, err := NewCacheInstallerExactPackageLoader(engineinstall.CacheInstaller{
		Root:            cacheRoot,
		MaxArchiveBytes: 1 << 20,
	})
	if err != nil {
		t.Fatal(err)
	}
	loaded, err := loader.LoadExactPackage(digest)
	if err != nil {
		t.Fatalf("LoadExactPackage() error = %v", err)
	}
	if loaded.PackageDigest != digest ||
		loaded.Layout.Definition.EngineDefinition.EngineID != fixture.record.EngineID {
		t.Fatalf("loaded exact cache entry = %#v", loaded)
	}
	entryType := reflect.TypeOf(loaded)
	if entryType.NumField() != 2 || entryType.Field(0).Name != "PackageDigest" || entryType.Field(1).Name != "Layout" {
		t.Fatalf("exact cache entry exposes non-contract fields: %v", entryType)
	}
	if _, err := os.Stat(filepath.Join(cacheRoot, "expanded", cacheKey, "engine.json")); err != nil {
		t.Fatalf("verified archive was not rebuilt into its exact expanded derivative: %v", err)
	}
}

func TestCacheInstallerExactPackageLoaderRejectsLegacyRuntimeManifestArchive(t *testing.T) {
	fixture := newInstalledEnginePackageFixture(
		t,
		repositoryname.FirstPartyEngineIDPortScan,
		"1.2.3",
		'a',
		'b',
		'c',
	)
	entries := packageLayoutEntriesTest(t, fixture.layout)
	entries[len(entries)-1] = enginepackagecatalog.PackageLayoutEntry{
		Path: "artifacts/runtimes/port_scan.runtime.json",
		Mode: 0o440,
		Payload: []byte(`{
  "manifestVersion":"runtime.v1",
  "runtimeRef":"runtime.port_scan",
  "runtimeSource":"runtime.port_scan",
  "engineApiMajor":1,
  "install":{"command":"engine"},
  "capabilities":[]
}`),
	}
	archive := packageLayoutArchiveTest(t, entries)
	digest, _, err := packagemanifest.ComputePackageDigest(bytes.NewReader(archive))
	if err != nil {
		t.Fatal(err)
	}
	cacheRoot := t.TempDir()
	archiveRoot := filepath.Join(cacheRoot, "archives")
	if err := os.MkdirAll(archiveRoot, 0o750); err != nil {
		t.Fatal(err)
	}
	cacheKey := "sha256-" + strings.TrimPrefix(string(digest), "sha256:")
	archivePath := filepath.Join(archiveRoot, cacheKey+packagemanifest.EnginePackageArchiveSuffix)
	if err := os.WriteFile(archivePath, archive, 0o440); err != nil {
		t.Fatal(err)
	}

	loader, err := NewCacheInstallerExactPackageLoader(engineinstall.CacheInstaller{
		Root:            cacheRoot,
		MaxArchiveBytes: 1 << 20,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := loader.LoadExactPackage(digest); err == nil || !strings.Contains(err.Error(), "unexpected package v2 file") {
		t.Fatalf("LoadExactPackage() error = %v, want legacy runtime manifest rejection", err)
	}
	if _, err := os.Lstat(filepath.Join(cacheRoot, "expanded", cacheKey)); !os.IsNotExist(err) {
		t.Fatalf("legacy runtime manifest archive left an expanded cache entry: %v", err)
	}
}

func TestCacheInstallerExactPackageLoaderRejectsStaleEngineAPIMajorArchivesWithoutExpandedCache(t *testing.T) {
	for _, major := range []uint32{1, 3} {
		t.Run(fmt.Sprintf("major-%d", major), func(t *testing.T) {
			fixture := newInstalledEnginePackageFixture(
				t,
				repositoryname.FirstPartyEngineIDPortScan,
				"1.2.3",
				'a',
				'b',
				'c',
			)
			fixture.layout.Definition.EngineDefinition.Execution.EngineAPIMajor = major
			archive := packageLayoutArchiveTest(t, packageLayoutEntriesTest(t, fixture.layout))
			digest, _, err := packagemanifest.ComputePackageDigest(bytes.NewReader(archive))
			if err != nil {
				t.Fatal(err)
			}
			cacheRoot := t.TempDir()
			cacheKey := "sha256-" + strings.TrimPrefix(string(digest), "sha256:")
			archiveRoot := filepath.Join(cacheRoot, "archives")
			if err := os.MkdirAll(archiveRoot, 0o750); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(archiveRoot, cacheKey+packagemanifest.EnginePackageArchiveSuffix), archive, 0o440); err != nil {
				t.Fatal(err)
			}

			loader, err := NewCacheInstallerExactPackageLoader(engineinstall.CacheInstaller{
				Root:            cacheRoot,
				MaxArchiveBytes: 1 << 20,
			})
			if err != nil {
				t.Fatal(err)
			}
			if _, err := loader.LoadExactPackage(digest); err == nil || !strings.Contains(err.Error(), "unsupported Engine API major") {
				t.Fatalf("LoadExactPackage() error = %v, want stale Engine API major rejection", err)
			}
			if _, err := os.Lstat(filepath.Join(cacheRoot, "expanded", cacheKey)); !os.IsNotExist(err) {
				t.Fatalf("stale Engine API archive left an expanded cache entry: %v", err)
			}
		})
	}
}

func packageLayoutEntriesTest(
	t *testing.T,
	layout enginepackagecatalog.EnginePackageLayout,
) []enginepackagecatalog.PackageLayoutEntry {
	t.Helper()
	payloads := map[string][]byte{
		enginepackagecatalog.PackageManifestPath: mustMarshalJSONTest(t, layout.Definition.PackageManifest),
		enginepackagecatalog.EngineManifestPath:  mustMarshalJSONTest(t, layout.Definition.EngineDefinition),
	}
	for _, locale := range enginemanifest.RequiredLocales() {
		localePath, err := enginemanifest.LocaleResourcePath(locale)
		if err != nil {
			t.Fatal(err)
		}
		payload, err := json.Marshal(layout.LocaleResources[locale])
		if err != nil {
			t.Fatal(err)
		}
		payloads[localePath] = payload
	}
	entries := make([]enginepackagecatalog.PackageLayoutEntry, 0, len(payloads))
	for _, path := range enginepackagecatalog.RequiredEnginePackagePaths() {
		entries = append(entries, enginepackagecatalog.PackageLayoutEntry{
			Path:    path,
			Mode:    0o440,
			Payload: payloads[path],
		})
	}
	return entries
}

func packageLayoutArchiveTest(t *testing.T, entries []enginepackagecatalog.PackageLayoutEntry) []byte {
	t.Helper()
	var archive bytes.Buffer
	gzipWriter := gzip.NewWriter(&archive)
	tarWriter := tar.NewWriter(gzipWriter)
	for _, entry := range entries {
		header := &tar.Header{
			Name: entry.Path,
			Mode: int64(entry.Mode.Perm()),
			Size: int64(len(entry.Payload)),
		}
		if err := tarWriter.WriteHeader(header); err != nil {
			t.Fatal(err)
		}
		if _, err := tarWriter.Write(entry.Payload); err != nil {
			t.Fatal(err)
		}
	}
	if err := tarWriter.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gzipWriter.Close(); err != nil {
		t.Fatal(err)
	}
	return append([]byte(nil), archive.Bytes()...)
}
