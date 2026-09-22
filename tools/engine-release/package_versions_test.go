package main

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
)

func TestPackageVersionMapBindsSelectedDiscoveryAndArchive(t *testing.T) {
	root := writeEngineSourceTree(t, true)
	discovery, err := discoverEngineSources(root)
	if err != nil {
		t.Fatal(err)
	}
	document := PackageVersionMap{
		SchemaVersion: packageVersionMapSchemaVersion,
		PlanDigest:    releaseTestDigestA,
		Versions: []PackageVersionMapEntry{{
			EngineID:       discovery.Engines[0].EngineID,
			PackageVersion: "0.0.0+" + strings.Repeat("c", 64),
		}},
	}
	payload, err := json.Marshal(document)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := decodePackageVersionMap(payload, "fixture")
	if err != nil {
		t.Fatalf("decodePackageVersionMap() error = %v", err)
	}
	versions, err := packageVersionsForDiscovery(decoded, discovery)
	if err != nil {
		t.Fatalf("packageVersionsForDiscovery() error = %v", err)
	}

	outputRoot := filepath.Join(t.TempDir(), "packages")
	artifacts, err := buildPackagesWithVersionMap(discovery, validDevelopmentResults(discovery), "", versions, outputRoot)
	if err != nil {
		t.Fatalf("buildPackagesWithVersionMap() error = %v", err)
	}
	if got, want := artifacts[0].EngineVersion, document.Versions[0].PackageVersion; got != want {
		t.Fatalf("archive engineVersion = %q, want %q", got, want)
	}
	if err := validatePackageArtifactsWithVersionMap(discovery, validDevelopmentResults(discovery), PackageBuildResults{
		SchemaVersion: packageBuildResultsSchemaVersion,
		Mode:          buildModeDevelopment,
		Packages:      artifacts,
	}, outputRoot, "", buildModeDevelopment, versions); err != nil {
		t.Fatalf("validatePackageArtifactsWithVersionMap() error = %v", err)
	}
}

func TestPackageVersionMapRejectsMalformedAndMismatchedInputs(t *testing.T) {
	root := t.TempDir()
	writeEngineSource(t, root, "port_scan", "engine.lunafox.port_scan", true)
	writeEngineSource(t, root, "website_discovery", "engine.lunafox.website_discovery", true)
	discovery, err := discoverEngineSources(root)
	if err != nil {
		t.Fatal(err)
	}

	for _, payload := range [][]byte{
		[]byte(`{"schemaVersion":"lunafox.engine-package-version-map.v1","planDigest":"` + releaseTestDigestA + `","versions":[{"engineId":"engine.lunafox.port_scan","packageVersion":"0.0.0+` + strings.Repeat("a", 64) + `"}],"unexpected":true}`),
		[]byte(`{"schemaVersion":"lunafox.engine-package-version-map.v1","planDigest":"` + releaseTestDigestA + `","versions":[{"engineId":"engine.lunafox.port_scan","packageVersion":"not-semver"}]}`),
	} {
		if _, err := decodePackageVersionMap(payload, "fixture"); err == nil {
			t.Fatalf("decodePackageVersionMap(%s) unexpectedly succeeded", payload)
		}
	}

	document := PackageVersionMap{
		SchemaVersion: packageVersionMapSchemaVersion,
		PlanDigest:    releaseTestDigestA,
		Versions: []PackageVersionMapEntry{{
			EngineID:       discovery.Engines[0].EngineID,
			PackageVersion: "0.0.0+" + strings.Repeat("a", 64),
		}},
	}
	if _, err := packageVersionsForDiscovery(document, discovery); err == nil || !strings.Contains(err.Error(), "count") {
		t.Fatalf("packageVersionsForDiscovery() error = %v, want selected-set rejection", err)
	}

	versions := map[string]string{
		discovery.Engines[0].EngineID: "0.0.0+" + strings.Repeat("a", 64),
		discovery.Engines[1].EngineID: "0.0.0+" + strings.Repeat("b", 64),
	}
	results := PackageBuildResults{SchemaVersion: packageBuildResultsSchemaVersion, Mode: buildModeDevelopment}
	for index, source := range discovery.Engines {
		results.Packages = append(results.Packages, PackageBuildArtifact{
			EngineID:           source.EngineID,
			EngineVersion:      versions[source.EngineID],
			ArchivePath:        filepath.Join("packages", source.EngineID+".lfengine.tar.gz"),
			PackageDigest:      releaseTestDigestA,
			RuntimeImageDigest: releaseTestDigestA,
			RuntimeImageRefs:   []string{"localhost:5000/" + source.Repository + "@" + releaseTestDigestA},
		})
		if index == 0 {
			continue
		}
	}
	if err := validatePackageBuildResultsShapeWithVersions(results, buildModeDevelopment, versions); err != nil {
		t.Fatalf("validatePackageBuildResultsShapeWithVersions() error = %v", err)
	}
	if err := validatePackageBuildResultsShape(results, buildModeDevelopment); err == nil || !strings.Contains(err.Error(), "one Engine Package version") {
		t.Fatalf("legacy shape validation error = %v, want single-version rejection", err)
	}
}
