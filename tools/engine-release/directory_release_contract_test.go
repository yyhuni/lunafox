package main

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"

	enginepackagecatalog "github.com/yyhuni/lunafox/contracts/enginemanifest/packagecatalog"
)

const directoryReleaseEngineID = "engine.lunafox.directory_scan"

func TestDirectorySourceBuildsExactProductionPackage(t *testing.T) {
	discovery := directoryReleaseDiscovery(t)
	imageResults := directoryProductionImageResults(discovery.Engines[0])
	if err := validateBuildResults(discovery, imageResults, buildModeProduction); err != nil {
		t.Fatalf("validate Directory production Runtime Image receipt: %v", err)
	}

	outputRoot := filepath.Join(t.TempDir(), "packages")
	artifacts, err := buildPackages(discovery, imageResults, "1.2.3", outputRoot)
	if err != nil {
		t.Fatalf("build Directory Package v2: %v", err)
	}
	if len(artifacts) != 1 {
		t.Fatalf("Directory package artifact count = %d, want 1", len(artifacts))
	}
	packageResults := PackageBuildResults{
		SchemaVersion: packageBuildResultsSchemaVersion,
		Mode:          buildModeProduction,
		Packages:      artifacts,
	}
	receiptPath := filepath.Join(outputRoot, "build-results.json")
	writePackageBuildResultsFixture(t, receiptPath, packageResults)
	if err := validatePackageArtifacts(discovery, imageResults, packageResults, outputRoot, receiptPath, buildModeProduction); err != nil {
		t.Fatalf("validate Directory Package v2 artifacts: %v", err)
	}

	layout, err := enginepackagecatalog.LoadEnginePackageLayoutFromRoot(
		filepath.Join(outputRoot, "directory_scan"),
		1024*1024,
	)
	if err != nil {
		t.Fatalf("load generated Directory Package v2: %v", err)
	}
	definition := layout.Definition
	if definition.PackageManifest.EngineID != directoryReleaseEngineID ||
		definition.EngineDefinition.EngineID != directoryReleaseEngineID {
		t.Fatalf("Directory package identity drifted: package=%q engine=%q",
			definition.PackageManifest.EngineID,
			definition.EngineDefinition.EngineID,
		)
	}
	if definition.EngineDefinition.Execution.EngineAPIMajor != 2 {
		t.Fatalf("Directory Engine API major = %d, want 2", definition.EngineDefinition.Execution.EngineAPIMajor)
	}
	if got, want := definition.PackageManifest.RuntimeImage.Refs, imageResults.Engines[0].Refs; !reflect.DeepEqual(got, want) {
		t.Fatalf("Directory Runtime Image refs = %#v, want %#v", got, want)
	}
	for _, ref := range definition.PackageManifest.RuntimeImage.Refs {
		if strings.Contains(ref, ":latest") || !strings.Contains(ref, "@"+releaseTestDigestA) {
			t.Fatalf("Directory Package v2 contains mutable Runtime Image ref %q", ref)
		}
	}

	registry := enginepackagecatalog.NewEngineDefinitionRegistry()
	if err := registry.Register(definition); err != nil {
		t.Fatalf("register generated Directory package in catalog: %v", err)
	}
	registered, found := registry.Get(directoryReleaseEngineID)
	if !found || registered.PackageManifest.EngineID != directoryReleaseEngineID {
		t.Fatalf("Directory package was not cataloged by exact identity: found=%t definition=%#v", found, registered)
	}
}

func TestDirectoryProductionReleaseRejectsImageAndPackageDrift(t *testing.T) {
	discovery := directoryReleaseDiscovery(t)

	t.Run("mutable Runtime Image ref", func(t *testing.T) {
		results := directoryProductionImageResults(discovery.Engines[0])
		results.Engines[0].Refs[0] = "docker.io/lunafox/lunafox-engine-runtime-directory-scan:latest"
		results.Engines[0].SourceRef = results.Engines[0].Refs[0]
		err := validateBuildResults(discovery, results, buildModeProduction)
		if err == nil || !strings.Contains(err.Error(), "digest") {
			t.Fatalf("validate mutable Directory Runtime Image ref error = %v", err)
		}
	})

	t.Run("single-platform production publication", func(t *testing.T) {
		results := directoryProductionImageResults(discovery.Engines[0])
		results.Engines[0].Platforms = []string{"linux/amd64"}
		err := validateBuildResults(discovery, results, buildModeProduction)
		if err == nil || !strings.Contains(err.Error(), "linux/amd64 and linux/arm64 exactly once") {
			t.Fatalf("validate single-platform Directory publication error = %v", err)
		}
	})

	t.Run("exact dual-platform production publication", func(t *testing.T) {
		results := directoryProductionImageResults(discovery.Engines[0])
		if err := validateBuildResults(discovery, results, buildModeProduction); err != nil {
			t.Fatalf("validate exact dual-platform Directory publication: %v", err)
		}
	})

	t.Run("package receipt identity", func(t *testing.T) {
		imageResults := directoryProductionImageResults(discovery.Engines[0])
		outputRoot := filepath.Join(t.TempDir(), "packages")
		artifacts, err := buildPackages(discovery, imageResults, "1.2.3", outputRoot)
		if err != nil {
			t.Fatal(err)
		}
		artifacts[0].EngineID = "engine.lunafox.directory_scanner"
		artifacts[0].RuntimeImageRefs = []string{
			"docker.io/lunafox/lunafox-engine-runtime-directory-scanner@" + releaseTestDigestA,
			"ghcr.io/lunafox/lunafox-engine-runtime-directory-scanner@" + releaseTestDigestA,
		}
		packageResults := PackageBuildResults{
			SchemaVersion: packageBuildResultsSchemaVersion,
			Mode:          buildModeProduction,
			Packages:      artifacts,
		}
		err = validatePackageArtifacts(discovery, imageResults, packageResults, outputRoot, "", buildModeProduction)
		if err == nil || !strings.Contains(err.Error(), "want discovered engine") {
			t.Fatalf("validate Directory package receipt identity error = %v", err)
		}
	})

	t.Run("package internal identity", func(t *testing.T) {
		imageResults := directoryProductionImageResults(discovery.Engines[0])
		entries, err := packageEntries(discovery.EngineRoot, discovery.Engines[0], imageResults.Engines[0], "1.2.3")
		if err != nil {
			t.Fatal(err)
		}
		packagePayload := append([]byte(nil), entries[0].Payload...)
		enginePayload := append([]byte(nil), entries[1].Payload...)
		packagePayload = bytes.Replace(packagePayload, []byte(directoryReleaseEngineID), []byte("engine.lunafox.directory_scanner"), 1)
		_, err = enginepackagecatalog.DecodeEnginePackageDefinition(packagePayload, enginePayload, "package.json", "engine.json")
		if err == nil || !strings.Contains(err.Error(), "identity mismatch") {
			t.Fatalf("decode drifted Directory package identity error = %v", err)
		}
	})
}

func directoryReleaseDiscovery(t *testing.T) Discovery {
	t.Helper()
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve Directory release contract test path")
	}
	engineRoot := filepath.Clean(filepath.Join(filepath.Dir(filename), "..", "..", "extensions", "engines"))
	discovery, err := discoverEngineSources(engineRoot)
	if err != nil {
		t.Fatalf("discover repository Engine sources: %v", err)
	}
	var directorySources []EngineSource
	for _, source := range discovery.Engines {
		if source.EngineID == directoryReleaseEngineID {
			directorySources = append(directorySources, source)
		}
	}
	if len(directorySources) != 1 {
		t.Fatalf("discovered Directory source count = %d, want 1", len(directorySources))
	}
	source := directorySources[0]
	if source.Directory != "directory_scan" ||
		source.Dockerfile != "directory_scan/Dockerfile" ||
		source.BuildContext != "." ||
		source.Repository != "lunafox-engine-runtime-directory-scan" {
		t.Fatalf("unexpected Directory release source: %#v", source)
	}
	if _, err := os.Stat(filepath.Join(engineRoot, source.Directory, "Dockerfile")); err != nil {
		t.Fatalf("inspect Directory Dockerfile: %v", err)
	}
	return Discovery{
		SchemaVersion: discovery.SchemaVersion,
		EngineRoot:    discovery.EngineRoot,
		Engines:       directorySources,
	}
}

func directoryProductionImageResults(source EngineSource) RuntimeImageBuildResults {
	dockerRef := "docker.io/lunafox/" + source.Repository + "@" + releaseTestDigestA
	ghcrRef := "ghcr.io/lunafox/" + source.Repository + "@" + releaseTestDigestA
	return RuntimeImageBuildResults{
		SchemaVersion: buildResultsSchemaVersion,
		Mode:          buildModeProduction,
		Engines: []RuntimeImageBuildResult{{
			EngineID:       source.EngineID,
			Dockerfile:     source.Dockerfile,
			BuildContext:   source.BuildContext,
			Repository:     source.Repository,
			BuildCount:     1,
			IndexDigest:    releaseTestDigestA,
			IndexMediaType: ociImageIndexMediaType,
			Platforms:      []string{"linux/amd64", "linux/arm64"},
			Refs:           []string{dockerRef, ghcrRef},
			SourceRef:      dockerRef,
			CopiedRefs:     []string{ghcrRef},
		}},
	}
}
