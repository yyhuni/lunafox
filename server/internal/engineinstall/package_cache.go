package engineinstall

import (
	"archive/tar"
	"bufio"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	enginepackagecatalog "github.com/yyhuni/lunafox/contracts/enginemanifest/packagecatalog"
	"github.com/yyhuni/lunafox/contracts/enginemanifest/packagemanifest"
	"github.com/yyhuni/lunafox/contracts/ociartifact"
)

const (
	expandedPackageDirectoryMode fs.FileMode = 0o550
	expandedPackageFileMode      fs.FileMode = 0o440
	maxPackageTarOverhead                    = int64(64 << 10)
	maxInt64                                 = int64(^uint64(0) >> 1)
)

// CachedEnginePackage is a Server-local view of one verified packageDigest
// cache entry. RootPath is never persisted or sent to Agent.
type CachedEnginePackage struct {
	PackageDigest ociartifact.PackageDigest
	RootPath      string
	Layout        enginepackagecatalog.EnginePackageLayout
}

// PackagePromotionValidator completes package-dependent verification that
// must succeed before a newly downloaded archive becomes canonical. The
// installer uses it for expected Engine identity and remote Runtime Image
// descriptor/signature/platform checks.
type PackagePromotionValidator func(context.Context, enginepackagecatalog.EnginePackageLayout) error

// LoadOrRebuildPackage validates the exact archive bytes before consulting
// the expanded derivative. Any structural, permission, or semantic failure
// discards and atomically rebuilds the complete expanded entry.
func (installer CacheInstaller) LoadOrRebuildPackage(expectedDigest ociartifact.PackageDigest) (CachedEnginePackage, error) {
	parsedDigest, cacheKey, err := installer.validatePackageCacheRequest(expectedDigest)
	if err != nil {
		return CachedEnginePackage{}, err
	}
	if err := os.MkdirAll(installer.Root, 0o750); err != nil {
		return CachedEnginePackage{}, fmt.Errorf("create engine package cache root: %w", err)
	}

	lock := lockForDigest(cacheKey)
	lock.Lock()
	defer lock.Unlock()
	unlockProcess, err := acquireCacheProcessLock(installer.Root, cacheKey)
	if err != nil {
		return CachedEnginePackage{}, err
	}
	defer unlockProcess()
	return installer.loadOrRebuildPackageLocked(parsedDigest, cacheKey)
}

// StageAndPromotePackage consumes one package-layer attempt. It verifies the
// exact descriptor size and packageDigest while streaming to private staging,
// validates the closed package layout, and runs all package-dependent remote
// checks before publishing either canonical cache path.
func (installer CacheInstaller) StageAndPromotePackage(
	ctx context.Context,
	reader io.Reader,
	expectedDigest ociartifact.PackageDigest,
	expectedSize int64,
	validateBeforePromotion PackagePromotionValidator,
) (CachedEnginePackage, error) {
	if ctx == nil {
		return CachedEnginePackage{}, fmt.Errorf("engine package v2 staging context is required")
	}
	if reader == nil {
		return CachedEnginePackage{}, fmt.Errorf("engine package v2 archive reader is required")
	}
	if validateBeforePromotion == nil {
		return CachedEnginePackage{}, fmt.Errorf("engine package v2 pre-promotion validator is required")
	}
	parsedDigest, cacheKey, err := installer.validatePackageCacheRequest(expectedDigest)
	if err != nil {
		return CachedEnginePackage{}, err
	}
	if expectedSize <= 0 {
		return CachedEnginePackage{}, fmt.Errorf("engine package v2 descriptor size must be positive")
	}
	if expectedSize > installer.MaxArchiveBytes {
		return CachedEnginePackage{}, fmt.Errorf(
			"engine package v2 descriptor size %d exceeds maximum size %d",
			expectedSize,
			installer.MaxArchiveBytes,
		)
	}
	if err := ctx.Err(); err != nil {
		return CachedEnginePackage{}, err
	}
	if err := os.MkdirAll(installer.Root, 0o750); err != nil {
		return CachedEnginePackage{}, fmt.Errorf("create engine package cache root: %w", err)
	}

	lock := lockForDigest(cacheKey)
	lock.Lock()
	defer lock.Unlock()
	unlockProcess, err := acquireCacheProcessLock(installer.Root, cacheKey)
	if err != nil {
		return CachedEnginePackage{}, err
	}
	defer unlockProcess()

	if cached, cacheErr := installer.loadOrRebuildPackageLocked(parsedDigest, cacheKey); cacheErr == nil {
		if err := consumeAndVerifyPackageArchive(ctx, reader, io.Discard, parsedDigest, expectedSize); err != nil {
			return CachedEnginePackage{}, err
		}
		if err := validateBeforePromotion(ctx, cached.Layout); err != nil {
			return CachedEnginePackage{}, fmt.Errorf("validate cached engine package v2 before reuse: %w", err)
		}
		return cached, nil
	}
	if err := ctx.Err(); err != nil {
		return CachedEnginePackage{}, err
	}
	archiveDestination, expandedDestination := installer.packageCachePaths(cacheKey)
	if err := os.MkdirAll(filepath.Dir(archiveDestination), 0o750); err != nil {
		return CachedEnginePackage{}, fmt.Errorf("create engine package v2 archive cache root: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(expandedDestination), 0o750); err != nil {
		return CachedEnginePackage{}, fmt.Errorf("create expanded package v2 cache root: %w", err)
	}

	stagingDir, err := os.MkdirTemp(installer.Root, ".engine-install-v2-")
	if err != nil {
		return CachedEnginePackage{}, fmt.Errorf("create engine package v2 staging directory: %w", err)
	}
	defer func() { _ = removePackageCacheTree(stagingDir) }()

	stagedArchive := filepath.Join(stagingDir, "package"+packagemanifest.EnginePackageArchiveSuffix)
	if err := copyAndVerifyPackageArchive(ctx, reader, stagedArchive, parsedDigest, expectedSize); err != nil {
		return CachedEnginePackage{}, err
	}
	entries, err := readPackageArchive(stagedArchive, installer.MaxArchiveBytes)
	if err != nil {
		return CachedEnginePackage{}, err
	}
	// macOS rejects moving a read-only directory across parent directories.
	// Reserve the private expanded entry beside its canonical destination so
	// the final same-parent rename preserves the already read-only tree.
	stagedExpanded, err := os.MkdirTemp(filepath.Dir(expandedDestination), ".staged-package-v2-")
	if err != nil {
		return CachedEnginePackage{}, fmt.Errorf("reserve expanded package v2 staging path: %w", err)
	}
	if err := os.Remove(stagedExpanded); err != nil {
		return CachedEnginePackage{}, fmt.Errorf("prepare expanded package v2 staging path: %w", err)
	}
	defer func() { _ = removePackageCacheTree(stagedExpanded) }()
	if err := writeReadOnlyExpandedPackage(stagedExpanded, entries); err != nil {
		return CachedEnginePackage{}, err
	}
	layout, err := loadReadOnlyExpandedPackage(stagedExpanded, installer.MaxArchiveBytes)
	if err != nil {
		return CachedEnginePackage{}, fmt.Errorf("validate staged expanded package v2 cache: %w", err)
	}
	if err := validateBeforePromotion(ctx, layout); err != nil {
		return CachedEnginePackage{}, fmt.Errorf("validate engine package v2 before cache promotion: %w", err)
	}
	if err := ctx.Err(); err != nil {
		return CachedEnginePackage{}, err
	}

	if err := removePackageArchive(archiveDestination); err != nil {
		return CachedEnginePackage{}, fmt.Errorf("discard invalid engine package v2 archive cache: %w", err)
	}
	if err := discardExpandedPackage(expandedDestination); err != nil {
		return CachedEnginePackage{}, fmt.Errorf("discard invalid expanded package v2 cache: %w", err)
	}
	if err := os.Chmod(stagedArchive, expandedPackageFileMode); err != nil {
		return CachedEnginePackage{}, fmt.Errorf("make staged engine package v2 archive read-only: %w", err)
	}
	if err := promotePath(stagedArchive, archiveDestination); err != nil {
		return CachedEnginePackage{}, fmt.Errorf("promote engine package v2 archive cache: %w", err)
	}
	if err := promotePath(stagedExpanded, expandedDestination); err != nil {
		_ = removePackageArchive(archiveDestination)
		return CachedEnginePackage{}, fmt.Errorf("promote expanded package v2 cache: %w", err)
	}

	cached, err := installer.loadOrRebuildPackageLocked(parsedDigest, cacheKey)
	if err != nil {
		_ = removePackageArchive(archiveDestination)
		_ = discardExpandedPackage(expandedDestination)
		return CachedEnginePackage{}, fmt.Errorf("validate promoted engine package v2 cache: %w", err)
	}
	return cached, nil
}

func (installer CacheInstaller) validatePackageCacheRequest(expectedDigest ociartifact.PackageDigest) (ociartifact.PackageDigest, string, error) {
	if strings.TrimSpace(installer.Root) == "" ||
		installer.MaxArchiveBytes <= 0 ||
		installer.MaxArchiveBytes > maxInt64-maxPackageTarOverhead-1 {
		return "", "", fmt.Errorf("engine package cache root and size limit are required")
	}
	parsedDigest, err := ociartifact.ParsePackageDigest(string(expectedDigest))
	if err != nil {
		return "", "", err
	}
	return parsedDigest, "sha256-" + strings.TrimPrefix(string(parsedDigest), "sha256:"), nil
}

func (installer CacheInstaller) packageCachePaths(cacheKey string) (string, string) {
	return filepath.Join(installer.Root, "archives", cacheKey+packagemanifest.EnginePackageArchiveSuffix),
		filepath.Join(installer.Root, "expanded", cacheKey)
}

func (installer CacheInstaller) loadOrRebuildPackageLocked(parsedDigest ociartifact.PackageDigest, cacheKey string) (CachedEnginePackage, error) {
	archivePath, expandedPath := installer.packageCachePaths(cacheKey)

	if err := verifyPackageArchive(archivePath, parsedDigest, installer.MaxArchiveBytes); err != nil {
		return CachedEnginePackage{}, err
	}
	entries, err := readPackageArchive(archivePath, installer.MaxArchiveBytes)
	if err != nil {
		return CachedEnginePackage{}, err
	}
	if layout, err := loadReadOnlyExpandedPackage(expandedPath, installer.MaxArchiveBytes); err == nil {
		matchesArchive, compareErr := expandedPackageMatchesArchive(expandedPath, entries)
		if compareErr == nil && matchesArchive {
			return CachedEnginePackage{PackageDigest: parsedDigest, RootPath: expandedPath, Layout: layout}, nil
		}
	}

	// The expanded tree is one derivative cache entry. Never preserve or patch
	// individual files after any validation failure.
	if err := discardExpandedPackage(expandedPath); err != nil {
		return CachedEnginePackage{}, fmt.Errorf("discard invalid expanded package v2 cache %q: %w", expandedPath, err)
	}
	if err := os.MkdirAll(filepath.Dir(expandedPath), 0o750); err != nil {
		return CachedEnginePackage{}, fmt.Errorf("create expanded package v2 cache root: %w", err)
	}
	stagedExpandedPath, err := os.MkdirTemp(filepath.Dir(expandedPath), ".staged-package-v2-")
	if err != nil {
		return CachedEnginePackage{}, fmt.Errorf("reserve package v2 rebuild staging path: %w", err)
	}
	if err := os.Remove(stagedExpandedPath); err != nil {
		return CachedEnginePackage{}, fmt.Errorf("prepare package v2 rebuild staging path: %w", err)
	}
	defer func() { _ = removePackageCacheTree(stagedExpandedPath) }()

	if err := writeReadOnlyExpandedPackage(stagedExpandedPath, entries); err != nil {
		return CachedEnginePackage{}, err
	}
	if _, err := loadReadOnlyExpandedPackage(stagedExpandedPath, installer.MaxArchiveBytes); err != nil {
		return CachedEnginePackage{}, fmt.Errorf("validate rebuilt expanded package v2 cache: %w", err)
	}
	if err := promotePath(stagedExpandedPath, expandedPath); err != nil {
		return CachedEnginePackage{}, fmt.Errorf("promote expanded package v2 cache: %w", err)
	}
	layout, err := loadReadOnlyExpandedPackage(expandedPath, installer.MaxArchiveBytes)
	if err != nil {
		_ = discardExpandedPackage(expandedPath)
		return CachedEnginePackage{}, fmt.Errorf("validate promoted expanded package v2 cache: %w", err)
	}
	return CachedEnginePackage{PackageDigest: parsedDigest, RootPath: expandedPath, Layout: layout}, nil
}

func expandedPackageMatchesArchive(root string, entries []enginepackagecatalog.PackageLayoutEntry) (bool, error) {
	payloads := make(map[string][]byte, len(entries))
	for _, entry := range entries {
		payloads[entry.Path] = entry.Payload
	}
	for _, relativePath := range enginepackagecatalog.RequiredEnginePackagePaths() {
		expected, exists := payloads[relativePath]
		if !exists {
			return false, fmt.Errorf("archive is missing required package v2 file %q", relativePath)
		}
		path := filepath.Join(root, filepath.FromSlash(relativePath))
		file, err := os.Open(path)
		if err != nil {
			return false, fmt.Errorf("open expanded package v2 file %q for archive comparison: %w", relativePath, err)
		}
		actual, readErr := io.ReadAll(io.LimitReader(file, int64(len(expected))+1))
		closeErr := file.Close()
		if readErr != nil {
			return false, fmt.Errorf("read expanded package v2 file %q for archive comparison: %w", relativePath, readErr)
		}
		if closeErr != nil {
			return false, fmt.Errorf("close expanded package v2 file %q after archive comparison: %w", relativePath, closeErr)
		}
		if !bytes.Equal(actual, expected) {
			return false, nil
		}
	}
	return true, nil
}

func copyAndVerifyPackageArchive(
	ctx context.Context,
	reader io.Reader,
	destination string,
	expectedDigest ociartifact.PackageDigest,
	expectedSize int64,
) (returnErr error) {
	file, err := os.OpenFile(destination, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return fmt.Errorf("create staged engine package v2 archive: %w", err)
	}
	defer func() {
		if err := file.Close(); returnErr == nil && err != nil {
			returnErr = fmt.Errorf("close staged engine package v2 archive: %w", err)
		}
	}()

	if err := consumeAndVerifyPackageArchive(ctx, reader, file, expectedDigest, expectedSize); err != nil {
		return err
	}
	if err := file.Sync(); err != nil {
		return fmt.Errorf("sync staged engine package v2 archive: %w", err)
	}
	return nil
}

func consumeAndVerifyPackageArchive(
	ctx context.Context,
	reader io.Reader,
	destination io.Writer,
	expectedDigest ociartifact.PackageDigest,
	expectedSize int64,
) error {
	hash := sha256.New()
	limited := &io.LimitedReader{R: &contextReader{ctx: ctx, reader: reader}, N: expectedSize + 1}
	written, err := io.Copy(io.MultiWriter(destination, hash), limited)
	if err != nil {
		return fmt.Errorf("stream engine package v2 archive: %w", err)
	}
	if written != expectedSize {
		return fmt.Errorf("engine package v2 archive size mismatch: got %d want %d", written, expectedSize)
	}
	actualDigest := ociartifact.PackageDigest("sha256:" + hex.EncodeToString(hash.Sum(nil)))
	if actualDigest != expectedDigest {
		return fmt.Errorf("engine package v2 archive digest mismatch: got %s want %s", actualDigest, expectedDigest)
	}
	return nil
}

type contextReader struct {
	ctx    context.Context
	reader io.Reader
}

func (reader *contextReader) Read(payload []byte) (int, error) {
	if err := reader.ctx.Err(); err != nil {
		return 0, err
	}
	read, err := reader.reader.Read(payload)
	if err == nil {
		if contextErr := reader.ctx.Err(); contextErr != nil {
			return read, contextErr
		}
	}
	return read, err
}

func removePackageArchive(path string) error {
	info, err := os.Lstat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return os.Remove(path)
	}
	if info.IsDir() && info.Mode()&os.ModeSymlink == 0 {
		return removePackageCacheTree(path)
	}
	if info.Mode().Perm()&0o200 == 0 {
		if err := os.Chmod(path, 0o600); err != nil {
			return err
		}
	}
	return os.Remove(path)
}

func discardExpandedPackage(root string) error {
	parent := filepath.Dir(root)
	if err := os.MkdirAll(parent, 0o750); err != nil {
		return err
	}
	if _, err := os.Lstat(root); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	discardedPath, err := os.MkdirTemp(parent, ".discarded-package-v2-")
	if err != nil {
		return err
	}
	if err := os.Remove(discardedPath); err != nil {
		return err
	}
	if err := os.Rename(root, discardedPath); err != nil {
		return err
	}
	return removePackageCacheTree(discardedPath)
}

func removePackageCacheTree(root string) error {
	if err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return os.Chmod(path, 0o700)
		}
		return nil
	}); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	return os.RemoveAll(root)
}

func verifyPackageArchive(path string, expectedDigest ociartifact.PackageDigest, maxBytes int64) error {
	info, err := os.Lstat(path)
	if err != nil {
		return fmt.Errorf("inspect engine package v2 archive %q: %w", path, err)
	}
	if !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("engine package v2 archive %q must be a regular file", path)
	}
	if info.Size() > maxBytes {
		return fmt.Errorf("engine package v2 archive exceeds maximum size %d", maxBytes)
	}
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open engine package v2 archive %q: %w", path, err)
	}
	defer file.Close()

	digest, size, err := packagemanifest.ComputePackageDigest(io.LimitReader(file, maxBytes+1))
	if err != nil {
		return err
	}
	if size > maxBytes {
		return fmt.Errorf("engine package v2 archive exceeds maximum size %d", maxBytes)
	}
	if digest != expectedDigest {
		return fmt.Errorf("engine package v2 archive digest mismatch: got %s want %s", digest, expectedDigest)
	}
	return nil
}

func readPackageArchive(path string, maxExpandedBytes int64) ([]enginepackagecatalog.PackageLayoutEntry, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open engine package v2 archive %q: %w", path, err)
	}
	defer file.Close()
	buffered := bufio.NewReader(file)
	gz, err := gzip.NewReader(buffered)
	if err != nil {
		return nil, fmt.Errorf("open engine package v2 gzip stream: %w", err)
	}
	defer gz.Close()
	gz.Multistream(false)

	if maxExpandedBytes <= 0 || maxExpandedBytes > maxInt64-maxPackageTarOverhead-1 {
		return nil, fmt.Errorf("engine package v2 expanded payload size limit is invalid")
	}
	// Bound all decompressed tar bytes, including headers, PAX/GNU metadata,
	// padding, and malicious content after the tar end markers. The fixed
	// overhead budget is intentionally independent of package payload size.
	decompressed := &io.LimitedReader{R: gz, N: maxExpandedBytes + maxPackageTarOverhead + 1}
	tw := tar.NewReader(decompressed)
	requiredCount := len(enginepackagecatalog.RequiredEnginePackagePaths())
	entries := make([]enginepackagecatalog.PackageLayoutEntry, 0, requiredCount)
	var totalSize int64
	for {
		header, err := tw.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("read engine package v2 tar stream: %w", err)
		}
		if header.Typeflag != tar.TypeReg && header.Typeflag != tar.TypeRegA {
			return nil, fmt.Errorf("engine package v2 archive entry %q must be a regular file", header.Name)
		}
		if len(entries) >= requiredCount {
			return nil, fmt.Errorf("engine package v2 archive contains more than %d files", requiredCount)
		}
		if header.Size < 0 || header.Size > maxExpandedBytes-totalSize {
			return nil, fmt.Errorf("engine package v2 expanded payload exceeds maximum size %d", maxExpandedBytes)
		}
		entry := enginepackagecatalog.PackageLayoutEntry{
			Path: header.Name,
			Mode: fs.FileMode(header.Mode),
		}
		if err := enginepackagecatalog.ValidatePackageLayoutEntry(entry); err != nil {
			return nil, err
		}
		entry.Payload, err = io.ReadAll(tw)
		if err != nil {
			return nil, fmt.Errorf("read engine package v2 archive entry %q: %w", header.Name, err)
		}
		totalSize += int64(len(entry.Payload))
		entries = append(entries, entry)
	}
	var trailing [1]byte
	trailingBytes, err := decompressed.Read(trailing[:])
	if err != nil && err != io.EOF {
		return nil, fmt.Errorf("finish engine package v2 gzip stream: %w", err)
	}
	if trailingBytes != 0 {
		return nil, fmt.Errorf("engine package v2 tar stream has trailing uncompressed bytes")
	}
	if err == nil || decompressed.N == 0 {
		return nil, fmt.Errorf("engine package v2 decompressed tar stream exceeds maximum size %d", maxExpandedBytes+maxPackageTarOverhead)
	}
	if _, err := buffered.Peek(1); err == nil {
		return nil, fmt.Errorf("engine package v2 archive must contain exactly one gzip member and no trailing bytes")
	} else if err != io.EOF {
		return nil, fmt.Errorf("inspect engine package v2 archive trailer: %w", err)
	}
	if _, err := enginepackagecatalog.DecodeEnginePackageLayout(entries, path); err != nil {
		return nil, err
	}
	return entries, nil
}

func writeReadOnlyExpandedPackage(root string, entries []enginepackagecatalog.PackageLayoutEntry) error {
	if err := os.Mkdir(root, 0o750); err != nil {
		return fmt.Errorf("create staged expanded package v2 root: %w", err)
	}
	payloads := make(map[string][]byte, len(entries))
	for _, entry := range entries {
		payloads[entry.Path] = entry.Payload
	}
	for _, relativePath := range enginepackagecatalog.RequiredEnginePackagePaths() {
		destination := filepath.Join(root, filepath.FromSlash(relativePath))
		if err := os.MkdirAll(filepath.Dir(destination), 0o750); err != nil {
			return fmt.Errorf("create expanded package v2 directory: %w", err)
		}
		file, err := os.OpenFile(destination, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
		if err != nil {
			return fmt.Errorf("create expanded package v2 file %q: %w", relativePath, err)
		}
		_, writeErr := file.Write(payloads[relativePath])
		closeErr := file.Close()
		if writeErr != nil {
			return fmt.Errorf("write expanded package v2 file %q: %w", relativePath, writeErr)
		}
		if closeErr != nil {
			return fmt.Errorf("close expanded package v2 file %q: %w", relativePath, closeErr)
		}
		if err := os.Chmod(destination, expandedPackageFileMode); err != nil {
			return fmt.Errorf("make expanded package v2 file %q read-only: %w", relativePath, err)
		}
	}
	if err := os.Chmod(filepath.Join(root, "locales"), expandedPackageDirectoryMode); err != nil {
		return fmt.Errorf("make expanded package v2 locales directory read-only: %w", err)
	}
	if err := os.Chmod(root, expandedPackageDirectoryMode); err != nil {
		return fmt.Errorf("make expanded package v2 root read-only: %w", err)
	}
	return nil
}

func loadReadOnlyExpandedPackage(root string, maxPayloadBytes int64) (enginepackagecatalog.EnginePackageLayout, error) {
	for _, directory := range []string{root, filepath.Join(root, "locales")} {
		info, err := os.Lstat(directory)
		if err != nil {
			return enginepackagecatalog.EnginePackageLayout{}, err
		}
		if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 || info.Mode().Perm()&0o222 != 0 {
			return enginepackagecatalog.EnginePackageLayout{}, fmt.Errorf("expanded package v2 directory %q must be a read-only real directory", directory)
		}
	}
	for _, relativePath := range enginepackagecatalog.RequiredEnginePackagePaths() {
		path := filepath.Join(root, filepath.FromSlash(relativePath))
		info, err := os.Lstat(path)
		if err != nil {
			return enginepackagecatalog.EnginePackageLayout{}, err
		}
		if !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 || info.Mode().Perm()&0o222 != 0 {
			return enginepackagecatalog.EnginePackageLayout{}, fmt.Errorf("expanded package v2 file %q must be a read-only regular file", path)
		}
	}
	layout, err := enginepackagecatalog.LoadEnginePackageLayoutFromRoot(root, maxPayloadBytes)
	if err != nil {
		return enginepackagecatalog.EnginePackageLayout{}, err
	}
	return layout, nil
}
