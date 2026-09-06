package main

import (
	"archive/tar"
	"bufio"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"

	enginepackagebuild "github.com/yyhuni/lunafox/contracts/enginemanifest/enginepackagebuild"
	enginepackagecatalog "github.com/yyhuni/lunafox/contracts/enginemanifest/packagecatalog"
	"github.com/yyhuni/lunafox/contracts/enginemanifest/packagemanifest"
	"github.com/yyhuni/lunafox/contracts/enginemanifest/runtimeimage"
	"github.com/yyhuni/lunafox/contracts/ociartifact"
	"github.com/yyhuni/lunafox/contracts/versioning"
)

const (
	packageBuildResultsSchemaVersion = "lunafox.engine-package-build-results.v1"
	maxReleasePackageArchiveBytes    = int64(64 << 20)
	maxReleasePackageExpandedBytes   = int64(32 << 20)
	maxReleasePackageTarOverhead     = int64(64 << 10)
)

func decodePackageBuildResults(payload []byte, source string) (PackageBuildResults, error) {
	if err := rejectDuplicateJSONFields(bytes.NewReader(payload)); err != nil {
		return PackageBuildResults{}, fmt.Errorf("decode package build results %q: %w", source, err)
	}
	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.DisallowUnknownFields()
	var results PackageBuildResults
	if err := decoder.Decode(&results); err != nil {
		return PackageBuildResults{}, fmt.Errorf("decode package build results %q: %w", source, err)
	}
	if err := consumeJSONEOF(decoder); err != nil {
		return PackageBuildResults{}, fmt.Errorf("decode package build results %q: %w", source, err)
	}
	return results, nil
}

func validatePackageBuildResultsShape(results PackageBuildResults, expectedMode string) error {
	if results.SchemaVersion != packageBuildResultsSchemaVersion {
		return fmt.Errorf("unsupported package build results schemaVersion %q", results.SchemaVersion)
	}
	if results.Mode != buildModeDevelopment && results.Mode != buildModeProduction {
		return fmt.Errorf("unsupported package build results mode %q", results.Mode)
	}
	if expectedMode != "" && results.Mode != expectedMode {
		return fmt.Errorf("package build results mode %q does not match expected mode %q", results.Mode, expectedMode)
	}
	if len(results.Packages) == 0 {
		return fmt.Errorf("package build results packages cannot be empty")
	}
	lastEngineID := ""
	buildVersion := ""
	seen := make(map[string]struct{}, len(results.Packages))
	for index, artifact := range results.Packages {
		if artifact.EngineID == "" || artifact.EngineID != strings.TrimSpace(artifact.EngineID) {
			return fmt.Errorf("package build results packages[%d].engineId is required and must be canonical", index)
		}
		if _, duplicate := seen[artifact.EngineID]; duplicate {
			return fmt.Errorf("package build results contains duplicate engineId %q", artifact.EngineID)
		}
		if lastEngineID != "" && artifact.EngineID <= lastEngineID {
			return fmt.Errorf("package build results packages must be ordered by canonical engineId")
		}
		seen[artifact.EngineID] = struct{}{}
		lastEngineID = artifact.EngineID

		if artifact.EngineVersion == "" || artifact.EngineVersion != strings.TrimSpace(artifact.EngineVersion) || !versioning.IsValidSemVer(artifact.EngineVersion) {
			return fmt.Errorf("package build result %q engineVersion must be one canonical semantic version", artifact.EngineID)
		}
		if buildVersion == "" {
			buildVersion = artifact.EngineVersion
		} else if artifact.EngineVersion != buildVersion {
			return fmt.Errorf("package build results must use one Engine Package version, got %q and %q", buildVersion, artifact.EngineVersion)
		}

		if err := validatePackageArchivePath(artifact.ArchivePath); err != nil {
			return fmt.Errorf("package build result %q archivePath: %w", artifact.EngineID, err)
		}
		if _, err := ociartifact.ParsePackageDigest(artifact.PackageDigest); err != nil {
			return fmt.Errorf("package build result %q: %w", artifact.EngineID, err)
		}
		if !sha256DigestPattern.MatchString(artifact.RuntimeImageDigest) {
			return fmt.Errorf("package build result %q runtimeImageDigest must be a canonical sha256 digest", artifact.EngineID)
		}
		candidates, err := runtimeimage.ParseFirstPartyCandidates(artifact.EngineID, artifact.RuntimeImageRefs)
		if err != nil {
			return fmt.Errorf("package build result %q runtimeImageRefs: %w", artifact.EngineID, err)
		}
		if string(candidates.RuntimeImageDigest) != artifact.RuntimeImageDigest {
			return fmt.Errorf("package build result %q Runtime Image refs digest %q does not match runtimeImageDigest %q", artifact.EngineID, candidates.RuntimeImageDigest, artifact.RuntimeImageDigest)
		}
	}
	return nil
}

func validatePackageArchivePath(value string) error {
	if value == "" || value != strings.TrimSpace(value) || strings.IndexByte(value, 0) >= 0 {
		return fmt.Errorf("must be a non-empty canonical path")
	}
	if filepath.Clean(value) != value {
		return fmt.Errorf("must be a clean canonical path")
	}
	if !strings.HasSuffix(value, packagemanifest.EnginePackageArchiveSuffix) {
		return fmt.Errorf("must end with %s", packagemanifest.EnginePackageArchiveSuffix)
	}
	return nil
}

func validatePackageArtifacts(
	discovery Discovery,
	imageResults RuntimeImageBuildResults,
	packageResults PackageBuildResults,
	packagesRoot string,
	packageResultsPath string,
	expectedMode string,
) error {
	if err := validateBuildResults(discovery, imageResults, expectedMode); err != nil {
		return err
	}
	if err := validatePackageBuildResultsShape(packageResults, expectedMode); err != nil {
		return err
	}
	if packageResults.Mode != imageResults.Mode {
		return fmt.Errorf("package build results mode %q does not match Runtime Image build results mode %q", packageResults.Mode, imageResults.Mode)
	}
	if len(packageResults.Packages) != len(discovery.Engines) {
		return fmt.Errorf("package build results package count %d does not match discovery count %d", len(packageResults.Packages), len(discovery.Engines))
	}

	root, err := realDirectoryPath(packagesRoot, "packages root")
	if err != nil {
		return err
	}
	imageByID := make(map[string]RuntimeImageBuildResult, len(imageResults.Engines))
	for _, result := range imageResults.Engines {
		imageByID[result.EngineID] = result
	}

	expectedRootEntries := make(map[string]bool, len(discovery.Engines)*2+1)
	for index, source := range discovery.Engines {
		artifact := packageResults.Packages[index]
		if artifact.EngineID != source.EngineID {
			return fmt.Errorf("package build result at index %d identifies %q, want discovered engine %q", index, artifact.EngineID, source.EngineID)
		}
		imageResult, ok := imageByID[source.EngineID]
		if !ok {
			return fmt.Errorf("Runtime Image build results is missing package engine %q", source.EngineID)
		}
		if artifact.RuntimeImageDigest != imageResult.IndexDigest {
			return fmt.Errorf("package build result %q runtimeImageDigest %q does not match verified indexDigest %q", source.EngineID, artifact.RuntimeImageDigest, imageResult.IndexDigest)
		}
		if !slices.Equal(artifact.RuntimeImageRefs, imageResult.Refs) {
			return fmt.Errorf("package build result %q Runtime Image refs do not exactly match verified build receipt", source.EngineID)
		}

		expectedArchive := filepath.Join(root, source.EngineID+"-"+artifact.EngineVersion+packagemanifest.EnginePackageArchiveSuffix)
		actualArchive, err := filepath.Abs(artifact.ArchivePath)
		if err != nil {
			return fmt.Errorf("resolve package build result %q archivePath: %w", source.EngineID, err)
		}
		if actualArchive != expectedArchive {
			return fmt.Errorf("package build result %q archivePath %q does not identify canonical archive %q", source.EngineID, artifact.ArchivePath, expectedArchive)
		}
		expectedRootEntries[source.Directory] = true
		expectedRootEntries[filepath.Base(expectedArchive)] = false

		expectedEntries, err := packageEntries(discovery.EngineRoot, source, imageResult, artifact.EngineVersion)
		if err != nil {
			return err
		}
		archiveEntries, actualDigest, err := readCanonicalPackageArchive(expectedArchive)
		if err != nil {
			return fmt.Errorf("validate package archive for %s: %w", source.EngineID, err)
		}
		if string(actualDigest) != artifact.PackageDigest {
			return fmt.Errorf("package build result %q packageDigest mismatch: archive bytes are %s, receipt records %s", source.EngineID, actualDigest, artifact.PackageDigest)
		}
		if err := comparePackageEntries("archive", archiveEntries, "publisher inputs", expectedEntries); err != nil {
			return fmt.Errorf("package %s: %w", source.EngineID, err)
		}

		expandedRoot := filepath.Join(root, source.Directory)
		expandedEntries, layout, err := readExpandedPackage(expandedRoot)
		if err != nil {
			return fmt.Errorf("validate expanded package for %s: %w", source.EngineID, err)
		}
		if err := comparePackageEntries("expanded package", expandedEntries, "archive", archiveEntries); err != nil {
			return fmt.Errorf("package %s: %w", source.EngineID, err)
		}
		if err := comparePackageEntries("expanded package", expandedEntries, "publisher inputs", expectedEntries); err != nil {
			return fmt.Errorf("package %s: %w", source.EngineID, err)
		}
		manifest := layout.Definition.PackageManifest
		if manifest.EngineID != artifact.EngineID || manifest.EngineVersion != artifact.EngineVersion {
			return fmt.Errorf("package %q manifest identity/version does not match package build result", source.EngineID)
		}
		if !slices.Equal(manifest.RuntimeImage.Refs, artifact.RuntimeImageRefs) {
			return fmt.Errorf("package %q manifest Runtime Image refs do not match package build result", source.EngineID)
		}
	}

	if packageResultsPath != "" {
		resultsAbs, err := filepath.Abs(packageResultsPath)
		if err != nil {
			return fmt.Errorf("resolve package build results path: %w", err)
		}
		relative, err := filepath.Rel(root, resultsAbs)
		if err != nil {
			return fmt.Errorf("resolve package build results relative path: %w", err)
		}
		if relative != "." && relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator)) && filepath.Dir(relative) == "." {
			expectedRootEntries[relative] = false
		}
	}
	if err := validatePackageRootInventory(root, expectedRootEntries); err != nil {
		return err
	}
	return nil
}

func validatePackageReleaseEvolution(previous, current PackageBuildResults) error {
	if err := validatePackageBuildResultsShape(previous, ""); err != nil {
		return fmt.Errorf("previous package build results: %w", err)
	}
	if err := validatePackageBuildResultsShape(current, ""); err != nil {
		return fmt.Errorf("current package build results: %w", err)
	}
	previousByEngine := make(map[string]PackageBuildArtifact, len(previous.Packages))
	for _, artifact := range previous.Packages {
		previousByEngine[artifact.EngineID] = artifact
	}
	for _, artifact := range current.Packages {
		old, ok := previousByEngine[artifact.EngineID]
		if !ok || old.RuntimeImageDigest == artifact.RuntimeImageDigest {
			continue
		}
		if old.EngineVersion == artifact.EngineVersion {
			return fmt.Errorf("Runtime Image digest changed for %s without a new Engine Package release version", artifact.EngineID)
		}
	}
	return nil
}

func realDirectoryPath(value, field string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", fmt.Errorf("%s is required", field)
	}
	absolute, err := filepath.Abs(value)
	if err != nil {
		return "", fmt.Errorf("resolve %s: %w", field, err)
	}
	info, err := os.Lstat(absolute)
	if err != nil {
		return "", fmt.Errorf("inspect %s %q: %w", field, absolute, err)
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return "", fmt.Errorf("%s %q must be a real directory", field, absolute)
	}
	return absolute, nil
}

func validatePackageRootInventory(root string, expected map[string]bool) error {
	entries, err := os.ReadDir(root)
	if err != nil {
		return fmt.Errorf("read packages root %q: %w", root, err)
	}
	seen := make(map[string]struct{}, len(entries))
	for _, entry := range entries {
		wantDirectory, ok := expected[entry.Name()]
		if !ok {
			return fmt.Errorf("packages root contains unexpected entry %q", entry.Name())
		}
		info, err := os.Lstat(filepath.Join(root, entry.Name()))
		if err != nil {
			return fmt.Errorf("inspect packages root entry %q: %w", entry.Name(), err)
		}
		if info.Mode()&os.ModeSymlink != 0 || wantDirectory != info.IsDir() || (!wantDirectory && !info.Mode().IsRegular()) {
			kind := "regular file"
			if wantDirectory {
				kind = "real directory"
			}
			return fmt.Errorf("packages root entry %q must be a %s", entry.Name(), kind)
		}
		seen[entry.Name()] = struct{}{}
	}
	for name := range expected {
		if _, ok := seen[name]; !ok {
			return fmt.Errorf("packages root is missing required entry %q", name)
		}
	}
	return nil
}

func readCanonicalPackageArchive(path string) ([]enginepackagecatalog.PackageLayoutEntry, ociartifact.PackageDigest, error) {
	payload, err := readBoundedRegularFile(path, maxReleasePackageArchiveBytes, true)
	if err != nil {
		return nil, "", err
	}
	digest, _, err := packagemanifest.ComputePackageDigest(bytes.NewReader(payload))
	if err != nil {
		return nil, "", err
	}
	entries, err := decodePackageArchiveEntries(payload, path)
	if err != nil {
		return nil, "", err
	}
	var canonical bytes.Buffer
	canonicalDigest, _, err := enginepackagebuild.WriteEnginePackageArchive(&canonical, entries)
	if err != nil {
		return nil, "", fmt.Errorf("rebuild canonical package archive: %w", err)
	}
	if !bytes.Equal(payload, canonical.Bytes()) || canonicalDigest != digest {
		return nil, "", fmt.Errorf("archive bytes are not the canonical deterministic package v2 serialization")
	}
	return entries, digest, nil
}

func decodePackageArchiveEntries(payload []byte, source string) ([]enginepackagecatalog.PackageLayoutEntry, error) {
	buffered := bufio.NewReader(bytes.NewReader(payload))
	gz, err := gzip.NewReader(buffered)
	if err != nil {
		return nil, fmt.Errorf("open package gzip stream: %w", err)
	}
	gz.Multistream(false)
	decompressed := &io.LimitedReader{R: gz, N: maxReleasePackageExpandedBytes + maxReleasePackageTarOverhead + 1}
	tarReader := tar.NewReader(decompressed)
	requiredPaths := enginepackagecatalog.RequiredEnginePackagePaths()
	entries := make([]enginepackagecatalog.PackageLayoutEntry, 0, len(requiredPaths))
	var totalPayload int64
	for {
		header, nextErr := tarReader.Next()
		if nextErr == io.EOF {
			break
		}
		if nextErr != nil {
			_ = gz.Close()
			return nil, fmt.Errorf("read package tar stream: %w", nextErr)
		}
		if len(entries) >= len(requiredPaths) {
			_ = gz.Close()
			return nil, fmt.Errorf("package archive contains more than %d files", len(requiredPaths))
		}
		if header.Typeflag != tar.TypeReg {
			_ = gz.Close()
			return nil, fmt.Errorf("package archive entry %q must be a canonical regular file", header.Name)
		}
		if header.Name != requiredPaths[len(entries)] {
			_ = gz.Close()
			return nil, fmt.Errorf("package archive entry %d is %q, want canonical path %q", len(entries), header.Name, requiredPaths[len(entries)])
		}
		if header.Size < 0 || header.Size > maxReleasePackageExpandedBytes-totalPayload {
			_ = gz.Close()
			return nil, fmt.Errorf("package expanded payload exceeds maximum size %d", maxReleasePackageExpandedBytes)
		}
		entry := enginepackagecatalog.PackageLayoutEntry{Path: header.Name, Mode: fs.FileMode(header.Mode)}
		if err := enginepackagecatalog.ValidatePackageLayoutEntry(entry); err != nil {
			_ = gz.Close()
			return nil, err
		}
		entry.Payload, err = io.ReadAll(tarReader)
		if err != nil {
			_ = gz.Close()
			return nil, fmt.Errorf("read package archive entry %q: %w", header.Name, err)
		}
		if int64(len(entry.Payload)) != header.Size {
			_ = gz.Close()
			return nil, fmt.Errorf("package archive entry %q size changed while reading", header.Name)
		}
		totalPayload += int64(len(entry.Payload))
		entries = append(entries, entry)
	}
	if len(entries) != len(requiredPaths) {
		_ = gz.Close()
		return nil, fmt.Errorf("package archive contains %d files, want exactly %d", len(entries), len(requiredPaths))
	}
	var trailing [1]byte
	read, finishErr := decompressed.Read(trailing[:])
	if finishErr != nil && finishErr != io.EOF {
		_ = gz.Close()
		return nil, fmt.Errorf("finish package gzip stream: %w", finishErr)
	}
	if read != 0 || finishErr == nil || decompressed.N == 0 {
		_ = gz.Close()
		return nil, fmt.Errorf("package tar stream has trailing or oversized uncompressed bytes")
	}
	if err := gz.Close(); err != nil {
		return nil, fmt.Errorf("close package gzip stream: %w", err)
	}
	if _, err := buffered.Peek(1); err == nil {
		return nil, fmt.Errorf("package archive must contain exactly one gzip member and no trailing bytes")
	} else if err != io.EOF {
		return nil, fmt.Errorf("inspect package archive trailer: %w", err)
	}
	if _, err := enginepackagecatalog.DecodeEnginePackageLayout(entries, source); err != nil {
		return nil, err
	}
	return entries, nil
}

func readExpandedPackage(root string) ([]enginepackagecatalog.PackageLayoutEntry, enginepackagecatalog.EnginePackageLayout, error) {
	root, err := realDirectoryPath(root, "expanded package root")
	if err != nil {
		return nil, enginepackagecatalog.EnginePackageLayout{}, err
	}
	for _, directory := range []string{root, filepath.Join(root, "locales")} {
		info, err := os.Lstat(directory)
		if err != nil {
			return nil, enginepackagecatalog.EnginePackageLayout{}, fmt.Errorf("inspect expanded package directory %q: %w", directory, err)
		}
		if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 || info.Mode().Perm() != 0o755 {
			return nil, enginepackagecatalog.EnginePackageLayout{}, fmt.Errorf("expanded package directory %q must have canonical mode 0755 and must not be a symlink", directory)
		}
	}
	layout, err := enginepackagecatalog.LoadEnginePackageLayoutFromRoot(root, maxReleasePackageExpandedBytes)
	if err != nil {
		return nil, enginepackagecatalog.EnginePackageLayout{}, err
	}
	paths := enginepackagecatalog.RequiredEnginePackagePaths()
	entries := make([]enginepackagecatalog.PackageLayoutEntry, 0, len(paths))
	for _, relativePath := range paths {
		absolute := filepath.Join(root, filepath.FromSlash(relativePath))
		info, err := os.Lstat(absolute)
		if err != nil {
			return nil, enginepackagecatalog.EnginePackageLayout{}, fmt.Errorf("inspect expanded package file %q: %w", relativePath, err)
		}
		if !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 || info.Mode().Perm() != 0o644 {
			return nil, enginepackagecatalog.EnginePackageLayout{}, fmt.Errorf("expanded package file %q must have canonical mode 0644 and must be a regular non-symlink file", relativePath)
		}
		payload, err := readBoundedRegularFile(absolute, maxReleasePackageExpandedBytes, false)
		if err != nil {
			return nil, enginepackagecatalog.EnginePackageLayout{}, err
		}
		entries = append(entries, enginepackagecatalog.PackageLayoutEntry{Path: relativePath, Mode: info.Mode(), Payload: payload})
	}
	return entries, layout, nil
}

func readBoundedRegularFile(path string, maxBytes int64, requireNonEmpty bool) ([]byte, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return nil, fmt.Errorf("inspect regular file %q: %w", path, err)
	}
	if !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
		return nil, fmt.Errorf("%q must be a regular non-symlink file", path)
	}
	if info.Size() < 0 || info.Size() > maxBytes || requireNonEmpty && info.Size() == 0 {
		return nil, fmt.Errorf("%q size %d is outside allowed range 1..%d", path, info.Size(), maxBytes)
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open regular file %q: %w", path, err)
	}
	defer file.Close()
	payload, err := io.ReadAll(io.LimitReader(file, maxBytes+1))
	if err != nil {
		return nil, fmt.Errorf("read regular file %q: %w", path, err)
	}
	if int64(len(payload)) != info.Size() || int64(len(payload)) > maxBytes {
		return nil, fmt.Errorf("regular file %q changed or exceeded its limit while reading", path)
	}
	return payload, nil
}

func comparePackageEntries(leftName string, left []enginepackagecatalog.PackageLayoutEntry, rightName string, right []enginepackagecatalog.PackageLayoutEntry) error {
	if len(left) != len(right) {
		return fmt.Errorf("%s has %d files but %s has %d", leftName, len(left), rightName, len(right))
	}
	for index := range left {
		if left[index].Path != right[index].Path {
			return fmt.Errorf("%s path %q does not match %s path %q", leftName, left[index].Path, rightName, right[index].Path)
		}
		if !bytes.Equal(left[index].Payload, right[index].Payload) {
			return fmt.Errorf("%s file %q bytes do not match %s", leftName, left[index].Path, rightName)
		}
	}
	return nil
}
