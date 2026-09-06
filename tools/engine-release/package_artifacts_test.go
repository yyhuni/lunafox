package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	enginepackagecatalog "github.com/yyhuni/lunafox/contracts/enginemanifest/packagecatalog"
	"github.com/yyhuni/lunafox/contracts/enginemanifest/packagemanifest"
)

func TestDecodePackageBuildResultsIsClosed(t *testing.T) {
	valid := `{"schemaVersion":"lunafox.engine-package-build-results.v1","mode":"development","packages":[{"engineId":"engine.lunafox.subdomain_discovery","engineVersion":"1.2.3","archivePath":"package.lfengine.tar.gz","packageDigest":"` + releaseTestDigestA + `","runtimeImageDigest":"` + releaseTestDigestA + `","runtimeImageRefs":["localhost:5000/lunafox-engine-runtime-subdomain-discovery@` + releaseTestDigestA + `"]}]}`
	results, err := decodePackageBuildResults([]byte(valid), "fixture")
	if err != nil {
		t.Fatalf("decodePackageBuildResults() error = %v", err)
	}
	if err := validatePackageBuildResultsShape(results, buildModeDevelopment); err != nil {
		t.Fatalf("validatePackageBuildResultsShape() error = %v", err)
	}

	tests := []struct {
		name    string
		payload string
		want    string
	}{
		{name: "unknown field", payload: strings.Replace(valid, `"mode":"development"`, `"mode":"development","extra":true`, 1), want: "unknown field"},
		{name: "duplicate field", payload: strings.Replace(valid, `"mode":"development"`, `"mode":"development","mode":"development"`, 1), want: "duplicate JSON field"},
		{name: "trailing JSON", payload: valid + `{}`, want: "trailing"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := decodePackageBuildResults([]byte(test.payload), "fixture")
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("decodePackageBuildResults() error = %v, want %q", err, test.want)
			}
		})
	}
}

func TestValidatePackageArtifactsBindsReceiptExpandedAndCanonicalArchive(t *testing.T) {
	discovery, imageResults, packageResults, outputRoot, resultsPath := validPackageArtifactFixture(t)
	if err := validatePackageArtifacts(discovery, imageResults, packageResults, outputRoot, resultsPath, buildModeDevelopment); err != nil {
		t.Fatalf("validatePackageArtifacts() error = %v", err)
	}

	t.Run("forged package digest", func(t *testing.T) {
		discovery, images, packages, root, receipt := validPackageArtifactFixture(t)
		packages.Packages[0].PackageDigest = releaseTestDigestB
		writePackageBuildResultsFixture(t, receipt, packages)
		err := validatePackageArtifacts(discovery, images, packages, root, receipt, buildModeDevelopment)
		if err == nil || !strings.Contains(err.Error(), "packageDigest mismatch") {
			t.Fatalf("validatePackageArtifacts() error = %v", err)
		}
	})

	t.Run("forged archive path", func(t *testing.T) {
		discovery, images, packages, root, receipt := validPackageArtifactFixture(t)
		packages.Packages[0].ArchivePath = filepath.Join(root, "other.lfengine.tar.gz")
		writePackageBuildResultsFixture(t, receipt, packages)
		err := validatePackageArtifacts(discovery, images, packages, root, receipt, buildModeDevelopment)
		if err == nil || !strings.Contains(err.Error(), "canonical archive") {
			t.Fatalf("validatePackageArtifacts() error = %v", err)
		}
	})

	t.Run("noncanonical archive metadata with forged digest", func(t *testing.T) {
		discovery, images, packages, root, receipt := validPackageArtifactFixture(t)
		artifact := &packages.Packages[0]
		expected, err := packageEntries(discovery.EngineRoot, discovery.Engines[0], images.Engines[0], artifact.EngineVersion)
		if err != nil {
			t.Fatal(err)
		}
		writeNoncanonicalPackageArchive(t, artifact.ArchivePath, expected)
		payload, err := os.ReadFile(artifact.ArchivePath)
		if err != nil {
			t.Fatal(err)
		}
		digest, _, err := packagemanifest.ComputePackageDigest(bytes.NewReader(payload))
		if err != nil {
			t.Fatal(err)
		}
		artifact.PackageDigest = string(digest)
		writePackageBuildResultsFixture(t, receipt, packages)
		err = validatePackageArtifacts(discovery, images, packages, root, receipt, buildModeDevelopment)
		if err == nil || !strings.Contains(err.Error(), "canonical deterministic") {
			t.Fatalf("validatePackageArtifacts() error = %v", err)
		}
	})

	t.Run("expanded bytes differ from archive", func(t *testing.T) {
		discovery, images, packages, root, receipt := validPackageArtifactFixture(t)
		localePath := filepath.Join(root, discovery.Engines[0].Directory, "locales", "en.json")
		payload, err := os.ReadFile(localePath)
		if err != nil {
			t.Fatal(err)
		}
		payload = bytes.Replace(payload, []byte("Subdomain"), []byte("Changed  "), 1)
		if err := os.WriteFile(localePath, payload, 0o644); err != nil {
			t.Fatal(err)
		}
		err = validatePackageArtifacts(discovery, images, packages, root, receipt, buildModeDevelopment)
		if err == nil || !strings.Contains(err.Error(), "bytes do not match archive") {
			t.Fatalf("validatePackageArtifacts() error = %v", err)
		}
	})

	t.Run("symlinked expanded member", func(t *testing.T) {
		discovery, images, packages, root, receipt := validPackageArtifactFixture(t)
		enginePath := filepath.Join(root, discovery.Engines[0].Directory, "engine.json")
		if err := os.Remove(enginePath); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(filepath.Join(discovery.EngineRoot, discovery.Engines[0].Directory, "engine.json"), enginePath); err != nil {
			t.Fatal(err)
		}
		err := validatePackageArtifacts(discovery, images, packages, root, receipt, buildModeDevelopment)
		if err == nil || !strings.Contains(err.Error(), "regular file") {
			t.Fatalf("validatePackageArtifacts() error = %v", err)
		}
	})

	t.Run("symlinked archive", func(t *testing.T) {
		discovery, images, packages, root, receipt := validPackageArtifactFixture(t)
		archivePath := packages.Packages[0].ArchivePath
		realArchivePath := archivePath + ".real"
		if err := os.Rename(archivePath, realArchivePath); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(realArchivePath, archivePath); err != nil {
			t.Fatal(err)
		}
		err := validatePackageArtifacts(discovery, images, packages, root, receipt, buildModeDevelopment)
		if err == nil || !strings.Contains(err.Error(), "regular non-symlink file") {
			t.Fatalf("validatePackageArtifacts() error = %v", err)
		}
	})

	t.Run("noncanonical expanded metadata", func(t *testing.T) {
		discovery, images, packages, root, receipt := validPackageArtifactFixture(t)
		enginePath := filepath.Join(root, discovery.Engines[0].Directory, "engine.json")
		if err := os.Chmod(enginePath, 0o600); err != nil {
			t.Fatal(err)
		}
		err := validatePackageArtifacts(discovery, images, packages, root, receipt, buildModeDevelopment)
		if err == nil || !strings.Contains(err.Error(), "canonical mode 0644") {
			t.Fatalf("validatePackageArtifacts() error = %v", err)
		}
	})

	t.Run("extra root artifact", func(t *testing.T) {
		discovery, images, packages, root, receipt := validPackageArtifactFixture(t)
		if err := os.WriteFile(filepath.Join(root, "unreported.lfengine.tar.gz"), []byte("extra"), 0o644); err != nil {
			t.Fatal(err)
		}
		err := validatePackageArtifacts(discovery, images, packages, root, receipt, buildModeDevelopment)
		if err == nil || !strings.Contains(err.Error(), "unexpected entry") {
			t.Fatalf("validatePackageArtifacts() error = %v", err)
		}
	})
}

func TestValidatePackageReleaseEvolutionUsesStrictReceipts(t *testing.T) {
	_, _, previous, _, _ := validPackageArtifactFixture(t)
	current := clonePackageBuildResults(previous)
	current.Packages[0].RuntimeImageDigest = releaseTestDigestB
	current.Packages[0].RuntimeImageRefs = []string{"localhost:5000/lunafox-engine-runtime-subdomain-discovery@" + releaseTestDigestB}
	if err := validatePackageReleaseEvolution(previous, current); err == nil || !strings.Contains(err.Error(), "without a new Engine Package release version") {
		t.Fatalf("validatePackageReleaseEvolution() error = %v", err)
	}
	current.Packages[0].EngineVersion = "1.2.4"
	if err := validatePackageReleaseEvolution(previous, current); err != nil {
		t.Fatalf("validatePackageReleaseEvolution() error = %v", err)
	}

	invalid := clonePackageBuildResults(current)
	invalid.Packages = append(invalid.Packages, invalid.Packages[0])
	if err := validatePackageReleaseEvolution(previous, invalid); err == nil || !strings.Contains(err.Error(), "duplicate engineId") {
		t.Fatalf("validatePackageReleaseEvolution() error = %v", err)
	}
}

func validPackageArtifactFixture(t *testing.T) (Discovery, RuntimeImageBuildResults, PackageBuildResults, string, string) {
	t.Helper()
	engineRoot := writeEngineSourceTree(t, true)
	discovery, err := discoverEngineSources(engineRoot)
	if err != nil {
		t.Fatal(err)
	}
	imageResults := validDevelopmentResults(discovery)
	outputRoot := filepath.Join(t.TempDir(), "packages")
	artifacts, err := buildPackages(discovery, imageResults, "1.2.3", outputRoot)
	if err != nil {
		t.Fatal(err)
	}
	packageResults := PackageBuildResults{
		SchemaVersion: packageBuildResultsSchemaVersion,
		Mode:          buildModeDevelopment,
		Packages:      artifacts,
	}
	resultsPath := filepath.Join(outputRoot, "build-results.json")
	writePackageBuildResultsFixture(t, resultsPath, packageResults)
	return discovery, imageResults, packageResults, outputRoot, resultsPath
}

func writePackageBuildResultsFixture(t *testing.T, path string, results PackageBuildResults) {
	t.Helper()
	payload, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, append(payload, '\n'), 0o644); err != nil {
		t.Fatal(err)
	}
}

func writeNoncanonicalPackageArchive(t *testing.T, path string, entries []enginepackagecatalog.PackageLayoutEntry) {
	t.Helper()
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	gz := gzip.NewWriter(file)
	tarWriter := tar.NewWriter(gz)
	for _, entry := range entries {
		header := &tar.Header{
			Name:     entry.Path,
			Mode:     0o644,
			Size:     int64(len(entry.Payload)),
			ModTime:  time.Unix(1, 0),
			Typeflag: tar.TypeReg,
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
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
}

func clonePackageBuildResults(input PackageBuildResults) PackageBuildResults {
	output := input
	output.Packages = append([]PackageBuildArtifact(nil), input.Packages...)
	for index := range output.Packages {
		output.Packages[index].RuntimeImageRefs = append([]string(nil), input.Packages[index].RuntimeImageRefs...)
	}
	return output
}
