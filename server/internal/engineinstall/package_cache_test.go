package engineinstall

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	enginepackagecatalog "github.com/yyhuni/lunafox/contracts/enginemanifest/packagecatalog"
	"github.com/yyhuni/lunafox/contracts/enginemanifest/packagemanifest"
	"github.com/yyhuni/lunafox/contracts/ociartifact"
)

func TestStageAndPromotePackagePublishesOnlyAfterValidation(t *testing.T) {
	cacheRoot := t.TempDir()
	t.Cleanup(func() { _ = removePackageCacheTree(cacheRoot) })
	archive := packageArchiveBytesForTest(t, validPackageCacheEntries(), time.Time{})
	digest, size, err := packagemanifest.ComputePackageDigest(bytes.NewReader(archive))
	if err != nil {
		t.Fatal(err)
	}
	installer := CacheInstaller{Root: cacheRoot, MaxArchiveBytes: 1 << 20}
	var validated atomic.Bool

	cached, err := installer.StageAndPromotePackage(
		context.Background(),
		bytes.NewReader(archive),
		digest,
		size,
		func(_ context.Context, layout enginepackagecatalog.EnginePackageLayout) error {
			if layout.Definition.EngineDefinition.EngineID != "engine.lunafox.port_scan" {
				t.Fatalf("validated engineId = %q", layout.Definition.EngineDefinition.EngineID)
			}
			archivePath := filepath.Join(cacheRoot, "archives", packageCacheKeyForTest(digest)+packagemanifest.EnginePackageArchiveSuffix)
			expandedPath := filepath.Join(cacheRoot, "expanded", packageCacheKeyForTest(digest))
			if _, err := os.Lstat(archivePath); !os.IsNotExist(err) {
				t.Fatalf("archive became canonical before validation completed: %v", err)
			}
			if _, err := os.Lstat(expandedPath); !os.IsNotExist(err) {
				t.Fatalf("expanded package became canonical before validation completed: %v", err)
			}
			validated.Store(true)
			return nil
		},
	)
	if err != nil {
		t.Fatalf("StageAndPromotePackage() error = %v", err)
	}
	if !validated.Load() {
		t.Fatal("pre-promotion validator was not called")
	}
	if cached.PackageDigest != digest || cached.Layout.Definition.PackageManifest.EngineVersion != "1.2.3" {
		t.Fatalf("unexpected cached package: %#v", cached)
	}
	assertExpandedPackageReadOnly(t, cached.RootPath)
	archiveInfo, err := os.Stat(filepath.Join(cacheRoot, "archives", packageCacheKeyForTest(digest)+packagemanifest.EnginePackageArchiveSuffix))
	if err != nil {
		t.Fatal(err)
	}
	if archiveInfo.Mode().Perm()&0o222 != 0 {
		t.Fatalf("canonical package archive is writable: mode=%#o", archiveInfo.Mode().Perm())
	}
}

func TestStageAndPromotePackageRejectsDescriptorSizeMismatchWithoutCanonicalCache(t *testing.T) {
	archive := packageArchiveBytesForTest(t, validPackageCacheEntries(), time.Time{})
	digest, size, err := packagemanifest.ComputePackageDigest(bytes.NewReader(archive))
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name         string
		expectedSize int64
	}{
		{name: "truncated descriptor", expectedSize: size + 1},
		{name: "extra archive byte", expectedSize: size - 1},
	} {
		t.Run(test.name, func(t *testing.T) {
			cacheRoot := t.TempDir()
			installer := CacheInstaller{Root: cacheRoot, MaxArchiveBytes: 1 << 20}
			_, err := installer.StageAndPromotePackage(
				context.Background(), bytes.NewReader(archive), digest, test.expectedSize,
				func(context.Context, enginepackagecatalog.EnginePackageLayout) error { return nil },
			)
			if err == nil || !strings.Contains(err.Error(), "archive size mismatch") {
				t.Fatalf("StageAndPromotePackage() error = %v, want exact size rejection", err)
			}
			assertPackageCanonicalCacheAbsent(t, cacheRoot, digest)
		})
	}
}

func TestStageAndPromotePackageRejectsDigestAndValidationFailuresWithoutCanonicalCache(t *testing.T) {
	archive := packageArchiveBytesForTest(t, validPackageCacheEntries(), time.Time{})
	digest, size, err := packagemanifest.ComputePackageDigest(bytes.NewReader(archive))
	if err != nil {
		t.Fatal(err)
	}
	wrongDigest := ociartifact.PackageDigest("sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb")

	t.Run("digest", func(t *testing.T) {
		cacheRoot := t.TempDir()
		installer := CacheInstaller{Root: cacheRoot, MaxArchiveBytes: 1 << 20}
		_, err := installer.StageAndPromotePackage(
			context.Background(), bytes.NewReader(archive), wrongDigest, size,
			func(context.Context, enginepackagecatalog.EnginePackageLayout) error { return nil },
		)
		if err == nil || !strings.Contains(err.Error(), "archive digest mismatch") {
			t.Fatalf("StageAndPromotePackage() error = %v, want digest rejection", err)
		}
		assertPackageCanonicalCacheAbsent(t, cacheRoot, wrongDigest)
	})

	t.Run("pre-promotion validation", func(t *testing.T) {
		cacheRoot := t.TempDir()
		installer := CacheInstaller{Root: cacheRoot, MaxArchiveBytes: 1 << 20}
		_, err := installer.StageAndPromotePackage(
			context.Background(), bytes.NewReader(archive), digest, size,
			func(context.Context, enginepackagecatalog.EnginePackageLayout) error {
				return errors.New("Runtime Image index is invalid")
			},
		)
		if err == nil || !strings.Contains(err.Error(), "Runtime Image index is invalid") {
			t.Fatalf("StageAndPromotePackage() error = %v, want pre-promotion rejection", err)
		}
		assertPackageCanonicalCacheAbsent(t, cacheRoot, digest)
	})
}

func TestStageAndPromotePackageReusesSameDigestAfterVerifyingSecondLayer(t *testing.T) {
	cacheRoot := t.TempDir()
	t.Cleanup(func() { _ = removePackageCacheTree(cacheRoot) })
	archive := packageArchiveBytesForTest(t, validPackageCacheEntries(), time.Time{})
	digest, size, err := packagemanifest.ComputePackageDigest(bytes.NewReader(archive))
	if err != nil {
		t.Fatal(err)
	}
	installer := CacheInstaller{Root: cacheRoot, MaxArchiveBytes: 1 << 20}
	var validations atomic.Int32
	validator := func(context.Context, enginepackagecatalog.EnginePackageLayout) error {
		validations.Add(1)
		return nil
	}
	first, err := installer.StageAndPromotePackage(context.Background(), bytes.NewReader(archive), digest, size, validator)
	if err != nil {
		t.Fatal(err)
	}
	reader := &observedReader{reader: bytes.NewReader(archive)}
	second, err := installer.StageAndPromotePackage(context.Background(), reader, digest, size, validator)
	if err != nil {
		t.Fatalf("cache-hit StageAndPromotePackage() error = %v", err)
	}
	if !reader.read.Load() {
		t.Fatal("same-digest cache hit did not verify the current candidate layer stream")
	}
	if first.RootPath != second.RootPath || validations.Load() != 2 {
		t.Fatalf("cache reuse = %q/%q validations=%d", first.RootPath, second.RootPath, validations.Load())
	}
}

func TestStageAndPromotePackageSerializesConcurrentSameDigest(t *testing.T) {
	cacheRoot := t.TempDir()
	t.Cleanup(func() { _ = removePackageCacheTree(cacheRoot) })
	archive := packageArchiveBytesForTest(t, validPackageCacheEntries(), time.Time{})
	digest, size, err := packagemanifest.ComputePackageDigest(bytes.NewReader(archive))
	if err != nil {
		t.Fatal(err)
	}
	installer := CacheInstaller{Root: cacheRoot, MaxArchiveBytes: 1 << 20}
	start := make(chan struct{})
	results := make(chan CachedEnginePackage, 2)
	errors := make(chan error, 2)
	var wait sync.WaitGroup
	for range 2 {
		wait.Add(1)
		go func() {
			defer wait.Done()
			<-start
			cached, err := installer.StageAndPromotePackage(
				context.Background(), bytes.NewReader(archive), digest, size,
				func(context.Context, enginepackagecatalog.EnginePackageLayout) error { return nil },
			)
			if err != nil {
				errors <- err
				return
			}
			results <- cached
		}()
	}
	close(start)
	wait.Wait()
	close(results)
	close(errors)
	for err := range errors {
		t.Fatalf("concurrent StageAndPromotePackage() error = %v", err)
	}
	var roots []string
	for result := range results {
		roots = append(roots, result.RootPath)
	}
	if len(roots) != 2 || roots[0] != roots[1] {
		t.Fatalf("concurrent roots = %#v, want one canonical path", roots)
	}
}

type observedReader struct {
	reader *bytes.Reader
	read   atomic.Bool
}

func (reader *observedReader) Read(payload []byte) (int, error) {
	reader.read.Store(true)
	return reader.reader.Read(payload)
}

func assertPackageCanonicalCacheAbsent(t *testing.T, cacheRoot string, digest ociartifact.PackageDigest) {
	t.Helper()
	archivePath := filepath.Join(cacheRoot, "archives", packageCacheKeyForTest(digest)+packagemanifest.EnginePackageArchiveSuffix)
	expandedPath := filepath.Join(cacheRoot, "expanded", packageCacheKeyForTest(digest))
	if _, err := os.Lstat(archivePath); !os.IsNotExist(err) {
		t.Fatalf("failed install left canonical archive cache: %v", err)
	}
	if _, err := os.Lstat(expandedPath); !os.IsNotExist(err) {
		t.Fatalf("failed install left canonical expanded cache: %v", err)
	}
}

func TestLoadOrRebuildPackageCreatesReadOnlyDigestKeyedExpandedCache(t *testing.T) {
	cacheRoot := t.TempDir()
	entries := validPackageCacheEntries()
	digest := stagePackageArchiveForTest(t, cacheRoot, entries, time.Time{})
	installer := CacheInstaller{Root: cacheRoot, MaxArchiveBytes: 1 << 20}

	loaded, err := installer.LoadOrRebuildPackage(digest)
	if err != nil {
		t.Fatalf("LoadOrRebuildPackage() error = %v", err)
	}
	wantRoot := filepath.Join(cacheRoot, "expanded", packageCacheKeyForTest(digest))
	if loaded.PackageDigest != digest || loaded.RootPath != wantRoot {
		t.Fatalf("loaded cache identity = %q/%q, want %q/%q", loaded.PackageDigest, loaded.RootPath, digest, wantRoot)
	}
	if loaded.Layout.Definition.EngineDefinition.EngineID != "engine.lunafox.port_scan" {
		t.Fatalf("loaded engineId = %q", loaded.Layout.Definition.EngineDefinition.EngineID)
	}
	assertExpandedPackageReadOnly(t, loaded.RootPath)
	firstInfo, err := os.Stat(loaded.RootPath)
	if err != nil {
		t.Fatal(err)
	}
	loadedAgain, err := installer.LoadOrRebuildPackage(digest)
	if err != nil {
		t.Fatalf("cache-hit LoadOrRebuildPackage() error = %v", err)
	}
	secondInfo, err := os.Stat(loadedAgain.RootPath)
	if err != nil {
		t.Fatal(err)
	}
	if !os.SameFile(firstInfo, secondInfo) {
		t.Fatal("valid expanded package v2 cache hit must not rebuild the entry")
	}
}

func TestLoadOrRebuildPackageRebuildsWholeInvalidExpandedEntry(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*testing.T, string)
	}{
		{
			name: "structure",
			mutate: func(t *testing.T, root string) {
				t.Helper()
				if err := os.Chmod(root, 0o750); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(root, "partial-repair-marker"), []byte("must disappear"), 0o440); err != nil {
					t.Fatal(err)
				}
				if err := os.Chmod(root, expandedPackageDirectoryMode); err != nil {
					t.Fatal(err)
				}
			},
		},
		{
			name: "permission",
			mutate: func(t *testing.T, root string) {
				t.Helper()
				if err := os.Chmod(filepath.Join(root, "package.json"), 0o640); err != nil {
					t.Fatal(err)
				}
			},
		},
		{
			name: "semantics",
			mutate: func(t *testing.T, root string) {
				t.Helper()
				localePath := filepath.Join(root, "locales", "zh.json")
				if err := os.Chmod(localePath, 0o640); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(localePath, []byte(`{"engine":{"displayName":"broken"}}`), 0o440); err != nil {
					t.Fatal(err)
				}
				if err := os.Chmod(localePath, expandedPackageFileMode); err != nil {
					t.Fatal(err)
				}
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cacheRoot := t.TempDir()
			entries := validPackageCacheEntries()
			digest := stagePackageArchiveForTest(t, cacheRoot, entries, time.Time{})
			installer := CacheInstaller{Root: cacheRoot, MaxArchiveBytes: 1 << 20}
			first, err := installer.LoadOrRebuildPackage(digest)
			if err != nil {
				t.Fatalf("initial LoadOrRebuildPackage() error = %v", err)
			}
			test.mutate(t, first.RootPath)

			rebuilt, err := installer.LoadOrRebuildPackage(digest)
			if err != nil {
				t.Fatalf("rebuilt LoadOrRebuildPackage() error = %v", err)
			}
			if rebuilt.RootPath != first.RootPath {
				t.Fatalf("rebuilt root = %q, want canonical digest root %q", rebuilt.RootPath, first.RootPath)
			}
			if _, err := os.Lstat(filepath.Join(rebuilt.RootPath, "partial-repair-marker")); !os.IsNotExist(err) {
				t.Fatalf("whole-entry rebuild retained invalid sibling: %v", err)
			}
			wantLocale := packageCacheEntryPayload(t, entries, "locales/zh.json")
			gotLocale, err := os.ReadFile(filepath.Join(rebuilt.RootPath, "locales", "zh.json"))
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(gotLocale, wantLocale) {
				t.Fatal("whole-entry rebuild did not restore archive-derived locale bytes")
			}
			assertExpandedPackageReadOnly(t, rebuilt.RootPath)
		})
	}
}

func TestLoadOrRebuildPackageRebuildsSemanticallyValidExpandedEntryFromAnotherArchive(t *testing.T) {
	cacheRoot := t.TempDir()
	canonicalEntries := validPackageCacheEntries()
	canonicalDigest := stagePackageArchiveForTest(t, cacheRoot, canonicalEntries, time.Time{})
	otherEntries := validPackageCacheEntries()
	for index := range otherEntries {
		if otherEntries[index].Path == "package.json" {
			otherEntries[index].Payload = bytes.ReplaceAll(otherEntries[index].Payload, []byte(`"engineVersion":"1.2.3"`), []byte(`"engineVersion":"9.9.9"`))
		}
	}
	otherDigest := stagePackageArchiveForTest(t, cacheRoot, otherEntries, time.Time{})
	if canonicalDigest == otherDigest {
		t.Fatal("different semantic package contents must produce different packageDigest values")
	}

	installer := CacheInstaller{Root: cacheRoot, MaxArchiveBytes: 1 << 20}
	canonical, err := installer.LoadOrRebuildPackage(canonicalDigest)
	if err != nil {
		t.Fatalf("load canonical package: %v", err)
	}
	other, err := installer.LoadOrRebuildPackage(otherDigest)
	if err != nil {
		t.Fatalf("load alternate package: %v", err)
	}
	if _, err := loadReadOnlyExpandedPackage(other.RootPath, installer.MaxArchiveBytes); err != nil {
		t.Fatalf("alternate archive fixture must be independently valid: %v", err)
	}

	if err := discardExpandedPackage(canonical.RootPath); err != nil {
		t.Fatalf("replace canonical expanded fixture: %v", err)
	}
	if err := writeReadOnlyExpandedPackage(canonical.RootPath, otherEntries); err != nil {
		t.Fatalf("install alternate valid expanded fixture: %v", err)
	}
	if _, err := loadReadOnlyExpandedPackage(canonical.RootPath, installer.MaxArchiveBytes); err != nil {
		t.Fatalf("alternate contents at canonical path must remain semantically valid: %v", err)
	}
	rebuilt, err := installer.LoadOrRebuildPackage(canonicalDigest)
	if err != nil {
		t.Fatalf("LoadOrRebuildPackage() error = %v", err)
	}
	// Directory identities may be reused by the filesystem after replacement.
	// The archive-byte checks below assert the observable semantic rebuild.
	for _, relativePath := range enginepackagecatalog.RequiredEnginePackagePaths() {
		actual, err := os.ReadFile(filepath.Join(rebuilt.RootPath, filepath.FromSlash(relativePath)))
		if err != nil {
			t.Fatal(err)
		}
		expected := packageCacheEntryPayload(t, canonicalEntries, relativePath)
		if !bytes.Equal(actual, expected) {
			t.Fatalf("rebuilt expanded file %q does not match canonical archive bytes", relativePath)
		}
	}
	assertExpandedPackageReadOnly(t, rebuilt.RootPath)
}

func TestLoadOrRebuildPackageNeverUsesExpandedEntryWhenArchiveDigestFails(t *testing.T) {
	cacheRoot := t.TempDir()
	digest := stagePackageArchiveForTest(t, cacheRoot, validPackageCacheEntries(), time.Time{})
	installer := CacheInstaller{Root: cacheRoot, MaxArchiveBytes: 1 << 20}
	loaded, err := installer.LoadOrRebuildPackage(digest)
	if err != nil {
		t.Fatal(err)
	}
	archivePath := filepath.Join(cacheRoot, "archives", packageCacheKeyForTest(digest)+packagemanifest.EnginePackageArchiveSuffix)
	if err := os.WriteFile(archivePath, []byte("corrupt"), 0o600); err != nil {
		t.Fatal(err)
	}

	if _, err := installer.LoadOrRebuildPackage(digest); err == nil || !strings.Contains(err.Error(), "digest mismatch") {
		t.Fatalf("LoadOrRebuildPackage() error = %v, want archive digest rejection", err)
	}
	if _, err := os.Stat(loaded.RootPath); err != nil {
		t.Fatalf("archive failure should not mutate the expanded derivative before failing: %v", err)
	}
}

func TestLoadOrRebuildPackageDoesNotReuseCacheAcrossArchiveSerializations(t *testing.T) {
	cacheRoot := t.TempDir()
	entries := validPackageCacheEntries()
	firstDigest := stagePackageArchiveForTest(t, cacheRoot, entries, time.Time{})
	secondDigest := stagePackageArchiveForTest(t, cacheRoot, entries, time.Unix(1, 0))
	if firstDigest == secondDigest {
		t.Fatal("different gzip serialization must produce a different packageDigest")
	}
	installer := CacheInstaller{Root: cacheRoot, MaxArchiveBytes: 1 << 20}
	first, err := installer.LoadOrRebuildPackage(firstDigest)
	if err != nil {
		t.Fatal(err)
	}
	second, err := installer.LoadOrRebuildPackage(secondDigest)
	if err != nil {
		t.Fatal(err)
	}
	if first.RootPath == second.RootPath {
		t.Fatalf("different archive-byte identities reused expanded cache root %q", first.RootPath)
	}
	for _, relativePath := range enginepackagecatalog.RequiredEnginePackagePaths() {
		firstPayload, err := os.ReadFile(filepath.Join(first.RootPath, filepath.FromSlash(relativePath)))
		if err != nil {
			t.Fatal(err)
		}
		secondPayload, err := os.ReadFile(filepath.Join(second.RootPath, filepath.FromSlash(relativePath)))
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(firstPayload, secondPayload) {
			t.Fatalf("fixture expanded content differs at %q", relativePath)
		}
	}
}

func TestLoadOrRebuildPackageRejectsExecutableArchiveEntry(t *testing.T) {
	cacheRoot := t.TempDir()
	entries := validPackageCacheEntries()
	entries[1].Mode = 0o755
	digest := stagePackageArchiveForTest(t, cacheRoot, entries, time.Time{})
	installer := CacheInstaller{Root: cacheRoot, MaxArchiveBytes: 1 << 20}

	if _, err := installer.LoadOrRebuildPackage(digest); err == nil || !strings.Contains(err.Error(), "must not be executable") {
		t.Fatalf("LoadOrRebuildPackage() error = %v, want executable payload rejection", err)
	}
	canonicalRoot := filepath.Join(cacheRoot, "expanded", packageCacheKeyForTest(digest))
	if _, err := os.Lstat(canonicalRoot); !os.IsNotExist(err) {
		t.Fatalf("failed rebuild left a canonical partial entry: %v", err)
	}
}

func TestLoadOrRebuildPackageRejectsTrailingGzipMember(t *testing.T) {
	cacheRoot := t.TempDir()
	archive := packageArchiveBytesForTest(t, validPackageCacheEntries(), time.Time{})
	var trailing bytes.Buffer
	gz := gzip.NewWriter(&trailing)
	if _, err := gz.Write([]byte("hidden second member")); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
	archive = append(archive, trailing.Bytes()...)
	digest := stageRawPackageArchiveForTest(t, cacheRoot, archive)
	installer := CacheInstaller{Root: cacheRoot, MaxArchiveBytes: 1 << 20}

	if _, err := installer.LoadOrRebuildPackage(digest); err == nil || !strings.Contains(err.Error(), "exactly one gzip member") {
		t.Fatalf("LoadOrRebuildPackage() error = %v, want trailing gzip member rejection", err)
	}
}

func TestLoadOrRebuildPackageRejectsTrailingTarPayloadInSingleGzipMember(t *testing.T) {
	cacheRoot := t.TempDir()
	archive := packageArchiveBytesWithTrailingPayloadForTest(
		t,
		validPackageCacheEntries(),
		time.Time{},
		[]byte("hidden payload after tar end markers"),
	)
	digest := stageRawPackageArchiveForTest(t, cacheRoot, archive)
	installer := CacheInstaller{Root: cacheRoot, MaxArchiveBytes: 1 << 20}

	if _, err := installer.LoadOrRebuildPackage(digest); err == nil || !strings.Contains(err.Error(), "trailing uncompressed bytes") {
		t.Fatalf("LoadOrRebuildPackage() error = %v, want trailing tar payload rejection", err)
	}
}

func TestLoadOrRebuildPackageBoundsCompressedTarMetadata(t *testing.T) {
	cacheRoot := t.TempDir()
	archive := packageArchiveWithPAXMetadataForTest(t, validPackageCacheEntries(), 256<<10)
	if len(archive) >= 256<<10 {
		t.Fatalf("metadata bomb fixture did not compress: archive size = %d", len(archive))
	}
	digest := stageRawPackageArchiveForTest(t, cacheRoot, archive)
	installer := CacheInstaller{Root: cacheRoot, MaxArchiveBytes: int64(len(archive) + 1024)}

	if _, err := installer.LoadOrRebuildPackage(digest); err == nil {
		t.Fatal("LoadOrRebuildPackage() accepted tar metadata beyond the decompressed stream budget")
	}
	canonicalRoot := filepath.Join(cacheRoot, "expanded", packageCacheKeyForTest(digest))
	if _, err := os.Lstat(canonicalRoot); !os.IsNotExist(err) {
		t.Fatalf("metadata limit failure left a canonical partial entry: %v", err)
	}
}

func assertExpandedPackageReadOnly(t *testing.T, root string) {
	t.Helper()
	paths := append([]string{"", "locales"}, enginepackagecatalog.RequiredEnginePackagePaths()...)
	for _, relativePath := range paths {
		info, err := os.Lstat(filepath.Join(root, filepath.FromSlash(relativePath)))
		if err != nil {
			t.Fatalf("inspect expanded package v2 path %q: %v", relativePath, err)
		}
		if info.Mode().Perm()&0o222 != 0 {
			t.Fatalf("expanded package v2 path %q is writable: mode=%#o", relativePath, info.Mode().Perm())
		}
	}
}

func stagePackageArchiveForTest(t *testing.T, cacheRoot string, entries []enginepackagecatalog.PackageLayoutEntry, modTime time.Time) ociartifact.PackageDigest {
	t.Helper()
	return stageRawPackageArchiveForTest(t, cacheRoot, packageArchiveBytesForTest(t, entries, modTime))
}

func packageArchiveBytesForTest(t *testing.T, entries []enginepackagecatalog.PackageLayoutEntry, modTime time.Time) []byte {
	t.Helper()
	return packageArchiveBytesWithTrailingPayloadForTest(t, entries, modTime, nil)
}

func packageArchiveBytesWithTrailingPayloadForTest(t *testing.T, entries []enginepackagecatalog.PackageLayoutEntry, modTime time.Time, trailingPayload []byte) []byte {
	t.Helper()
	var archive bytes.Buffer
	gz := gzip.NewWriter(&archive)
	gz.ModTime = modTime
	tw := tar.NewWriter(gz)
	for _, relativePath := range enginepackagecatalog.RequiredEnginePackagePaths() {
		entry := packageCacheEntryForPath(t, entries, relativePath)
		header := &tar.Header{Name: entry.Path, Mode: int64(entry.Mode.Perm()), Size: int64(len(entry.Payload))}
		if err := tw.WriteHeader(header); err != nil {
			t.Fatal(err)
		}
		if _, err := tw.Write(entry.Payload); err != nil {
			t.Fatal(err)
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if len(trailingPayload) > 0 {
		if _, err := gz.Write(trailingPayload); err != nil {
			t.Fatal(err)
		}
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
	return append([]byte(nil), archive.Bytes()...)
}

func packageArchiveWithPAXMetadataForTest(t *testing.T, entries []enginepackagecatalog.PackageLayoutEntry, metadataBytes int) []byte {
	t.Helper()
	var archive bytes.Buffer
	gz := gzip.NewWriter(&archive)
	tw := tar.NewWriter(gz)
	for index, relativePath := range enginepackagecatalog.RequiredEnginePackagePaths() {
		entry := packageCacheEntryForPath(t, entries, relativePath)
		header := &tar.Header{
			Name:   entry.Path,
			Mode:   int64(entry.Mode.Perm()),
			Size:   int64(len(entry.Payload)),
			Format: tar.FormatPAX,
		}
		if index == 0 {
			header.PAXRecords = map[string]string{"comment": strings.Repeat("a", metadataBytes)}
		}
		if err := tw.WriteHeader(header); err != nil {
			t.Fatal(err)
		}
		if _, err := tw.Write(entry.Payload); err != nil {
			t.Fatal(err)
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
	return append([]byte(nil), archive.Bytes()...)
}

func stageRawPackageArchiveForTest(t *testing.T, cacheRoot string, archive []byte) ociartifact.PackageDigest {
	t.Helper()
	t.Cleanup(func() { _ = removePackageCacheTree(cacheRoot) })
	digest, _, err := packagemanifest.ComputePackageDigest(bytes.NewReader(archive))
	if err != nil {
		t.Fatal(err)
	}
	archiveRoot := filepath.Join(cacheRoot, "archives")
	if err := os.MkdirAll(archiveRoot, 0o750); err != nil {
		t.Fatal(err)
	}
	archivePath := filepath.Join(archiveRoot, packageCacheKeyForTest(digest)+packagemanifest.EnginePackageArchiveSuffix)
	if err := os.WriteFile(archivePath, archive, 0o600); err != nil {
		t.Fatal(err)
	}
	return digest
}

func packageCacheKeyForTest(digest ociartifact.PackageDigest) string {
	return "sha256-" + strings.TrimPrefix(string(digest), "sha256:")
}

func packageCacheEntryForPath(t *testing.T, entries []enginepackagecatalog.PackageLayoutEntry, path string) enginepackagecatalog.PackageLayoutEntry {
	t.Helper()
	for _, entry := range entries {
		if entry.Path == path {
			return entry
		}
	}
	t.Fatalf("missing package v2 test entry %q", path)
	return enginepackagecatalog.PackageLayoutEntry{}
}

func packageCacheEntryPayload(t *testing.T, entries []enginepackagecatalog.PackageLayoutEntry, path string) []byte {
	t.Helper()
	return packageCacheEntryForPath(t, entries, path).Payload
}

func validPackageCacheEntries() []enginepackagecatalog.PackageLayoutEntry {
	digest := "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	locale := []byte(`{"engine":{"displayName":"Port scan","description":"Port scan engine"},"sections":{"scan":{"name":"Scan","description":"Scan settings","params":{"timeout":{"description":"Timeout"}}}}}`)
	return []enginepackagecatalog.PackageLayoutEntry{
		{Path: "package.json", Mode: 0o644, Payload: []byte(`{"packageFormatVersion":"lunafox.engine-package.v2","engineId":"engine.lunafox.port_scan","engineVersion":"1.2.3","runtimeImage":{"refs":["docker.io/lunafox/lunafox-engine-runtime-port-scan@` + digest + `"]}}`)},
		{Path: "engine.json", Mode: 0o644, Payload: []byte(`{"manifestVersion":"engine.v5","engineId":"engine.lunafox.port_scan","publisher":"lunafox","execution":{"engineApiMajor":2,"supportedTargetTypes":["domain","ip","cidr"],"configSections":[{"id":"scan","defaultEnabled":true,"params":[{"key":"timeout","type":"integer","default":30,"minimum":1}]}]}}`)},
		{Path: "locales/en.json", Mode: 0o644, Payload: append([]byte(nil), locale...)},
		{Path: "locales/zh.json", Mode: 0o644, Payload: append([]byte(nil), locale...)},
	}
}
