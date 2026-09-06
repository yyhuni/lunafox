package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	enginepackagebuild "github.com/yyhuni/lunafox/contracts/enginemanifest/enginepackagebuild"
	enginepackagecatalog "github.com/yyhuni/lunafox/contracts/enginemanifest/packagecatalog"
	"github.com/yyhuni/lunafox/contracts/enginemanifest/packagemanifest"
	"github.com/yyhuni/lunafox/contracts/versioning"
)

// PackageBuildArtifact is the only output projection needed by a publisher
// after an archive is written. The package digest is the exact compressed
// archive identity; RuntimeImageDigest remains distinct and is never copied
// into package.json as a sibling field.
type PackageBuildArtifact struct {
	EngineID           string   `json:"engineId"`
	EngineVersion      string   `json:"engineVersion"`
	ArchivePath        string   `json:"archivePath"`
	PackageDigest      string   `json:"packageDigest"`
	RuntimeImageDigest string   `json:"runtimeImageDigest"`
	RuntimeImageRefs   []string `json:"runtimeImageRefs"`
}

// PackageBuildResults is the closed publisher receipt for one package build.
// It is verified against discovery, the Runtime Image receipt, expanded files,
// and exact archive bytes before release tooling may consume it.
type PackageBuildResults struct {
	SchemaVersion string                 `json:"schemaVersion"`
	Mode          string                 `json:"mode"`
	Packages      []PackageBuildArtifact `json:"packages"`
}

func buildPackages(discovery Discovery, results RuntimeImageBuildResults, version, outputRoot string) ([]PackageBuildArtifact, error) {
	rawVersion := version
	version = strings.TrimSpace(version)
	if version == "" || rawVersion != version {
		return nil, fmt.Errorf("engine package version is required and must be canonical")
	}
	if !versioning.IsValidSemVer(version) {
		return nil, fmt.Errorf("%s", versioning.SemVerFieldMessage("engine package version"))
	}
	outputRoot = strings.TrimSpace(outputRoot)
	if outputRoot == "" {
		return nil, fmt.Errorf("engine package output root is required")
	}
	if err := validateBuildResults(discovery, results, results.Mode); err != nil {
		return nil, fmt.Errorf("validate image build results before package generation: %w", err)
	}
	if err := os.MkdirAll(outputRoot, 0o755); err != nil {
		return nil, fmt.Errorf("create engine package output root: %w", err)
	}
	byID := make(map[string]RuntimeImageBuildResult, len(results.Engines))
	for _, result := range results.Engines {
		byID[result.EngineID] = result
	}
	artifacts := make([]PackageBuildArtifact, 0, len(discovery.Engines))
	for _, source := range discovery.Engines {
		result := byID[source.EngineID]
		artifact, err := buildOnePackage(discovery.EngineRoot, source, result, version, outputRoot)
		if err != nil {
			return nil, err
		}
		artifacts = append(artifacts, artifact)
	}
	return artifacts, nil
}

func buildOnePackage(engineRoot string, source EngineSource, result RuntimeImageBuildResult, version, outputRoot string) (PackageBuildArtifact, error) {
	entries, err := packageEntries(engineRoot, source, result, version)
	if err != nil {
		return PackageBuildArtifact{}, err
	}

	packageDir := filepath.Join(outputRoot, source.Directory)
	if err := replaceDirectory(packageDir); err != nil {
		return PackageBuildArtifact{}, fmt.Errorf("prepare expanded package %s: %w", source.EngineID, err)
	}
	for _, entry := range entries {
		target := filepath.Join(packageDir, filepath.FromSlash(entry.Path))
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return PackageBuildArtifact{}, fmt.Errorf("create package path %s: %w", entry.Path, err)
		}
		if err := os.Chmod(filepath.Dir(target), 0o755); err != nil {
			return PackageBuildArtifact{}, fmt.Errorf("normalize package directory for %s: %w", entry.Path, err)
		}
		if err := os.WriteFile(target, entry.Payload, 0o644); err != nil {
			return PackageBuildArtifact{}, fmt.Errorf("write package path %s: %w", entry.Path, err)
		}
		if err := os.Chmod(target, 0o644); err != nil {
			return PackageBuildArtifact{}, fmt.Errorf("normalize package mode for %s: %w", entry.Path, err)
		}
	}

	archivePath := filepath.Join(outputRoot, source.EngineID+"-"+version+packagemanifest.EnginePackageArchiveSuffix)
	temporary, err := os.CreateTemp(outputRoot, ".engine-package-v2-*.tmp")
	if err != nil {
		return PackageBuildArtifact{}, fmt.Errorf("create package archive temp file: %w", err)
	}
	temporaryPath := temporary.Name()
	defer func() {
		_ = temporary.Close()
		_ = os.Remove(temporaryPath)
	}()
	packageDigest, _, err := enginepackagebuild.WriteEnginePackageArchive(temporary, entries)
	if err != nil {
		return PackageBuildArtifact{}, fmt.Errorf("write package v2 archive for %s: %w", source.EngineID, err)
	}
	if err := temporary.Close(); err != nil {
		return PackageBuildArtifact{}, fmt.Errorf("close package v2 archive for %s: %w", source.EngineID, err)
	}
	if err := os.Remove(archivePath); err != nil && !os.IsNotExist(err) {
		return PackageBuildArtifact{}, fmt.Errorf("replace package v2 archive for %s: %w", source.EngineID, err)
	}
	if err := os.Rename(temporaryPath, archivePath); err != nil {
		return PackageBuildArtifact{}, fmt.Errorf("publish package v2 archive for %s: %w", source.EngineID, err)
	}
	return PackageBuildArtifact{
		EngineID:           source.EngineID,
		EngineVersion:      version,
		ArchivePath:        archivePath,
		PackageDigest:      string(packageDigest),
		RuntimeImageDigest: result.IndexDigest,
		RuntimeImageRefs:   append([]string(nil), result.Refs...),
	}, nil
}

func packageEntries(engineRoot string, source EngineSource, result RuntimeImageBuildResult, version string) ([]enginepackagecatalog.PackageLayoutEntry, error) {
	engineDir := filepath.Join(engineRoot, filepath.FromSlash(source.Directory))
	enginePayload, err := readRegularFile(filepath.Join(engineDir, "engine.json"))
	if err != nil {
		return nil, fmt.Errorf("read %s for package v2: %w", source.EngineID, err)
	}
	localePayloads := make(map[string][]byte, 2)
	for _, locale := range []string{"en", "zh"} {
		path := filepath.Join(engineDir, "locales", locale+".json")
		payload, err := readRegularFile(path)
		if err != nil {
			return nil, fmt.Errorf("read %s locale %s: %w", source.EngineID, locale, err)
		}
		localePayloads[locale] = payload
	}
	manifest := packagemanifest.PackageManifest{
		PackageFormatVersion: packagemanifest.PackageFormatVersion,
		EngineID:             source.EngineID,
		EngineVersion:        version,
		RuntimeImage: packagemanifest.RuntimeImage{
			Refs: append([]string(nil), result.Refs...),
		},
	}
	packagePayload, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal package.json for %s: %w", source.EngineID, err)
	}
	packagePayload = append(packagePayload, '\n')
	entries := []enginepackagecatalog.PackageLayoutEntry{
		{Path: "package.json", Mode: 0o644, Payload: packagePayload},
		{Path: "engine.json", Mode: 0o644, Payload: enginePayload},
		{Path: "locales/en.json", Mode: 0o644, Payload: localePayloads["en"]},
		{Path: "locales/zh.json", Mode: 0o644, Payload: localePayloads["zh"]},
	}
	if _, err := enginepackagecatalog.DecodeEnginePackageLayout(entries, source.EngineID); err != nil {
		return nil, fmt.Errorf("validate generated package v2 for %s: %w", source.EngineID, err)
	}
	return entries, nil
}

func readRegularFile(path string) ([]byte, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
		return nil, fmt.Errorf("must be a regular non-symlink file")
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	return io.ReadAll(file)
}

func replaceDirectory(path string) error {
	if err := os.RemoveAll(path); err != nil {
		return err
	}
	return os.MkdirAll(path, 0o755)
}
