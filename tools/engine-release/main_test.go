package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	enginepackagecatalog "github.com/yyhuni/lunafox/contracts/enginemanifest/packagecatalog"
	"github.com/yyhuni/lunafox/contracts/enginemanifest/packagemanifest"
)

const releaseTestDigestA = "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
const releaseTestDigestB = "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"

func TestDiscoverEngineSourcesIsDynamicAndDerivesRepository(t *testing.T) {
	root := writeEngineSourceTree(t, true)
	discovery, err := discoverEngineSources(root)
	if err != nil {
		t.Fatalf("discoverEngineSources() error = %v", err)
	}
	if discovery.SchemaVersion != discoverySchemaVersion || len(discovery.Engines) != 1 {
		t.Fatalf("unexpected discovery: %#v", discovery)
	}
	source := discovery.Engines[0]
	if source.EngineID != "engine.lunafox.subdomain_discovery" || source.Repository != "lunafox-engine-runtime-subdomain-discovery" {
		t.Fatalf("unexpected discovered source: %#v", source)
	}
	if source.Dockerfile != "subdomain_discovery/Dockerfile" || source.BuildContext != "." {
		t.Fatalf("unexpected build inputs: %#v", source)
	}
}

func TestDiscoverEngineSourcesCollectsEveryValidatedDefinitionInCanonicalOrder(t *testing.T) {
	root := t.TempDir()
	writeEngineSource(t, root, "website_discovery", "engine.lunafox.website_discovery", true)
	writeEngineSource(t, root, "port_scan", "engine.lunafox.port_scan", true)
	writeEngineSource(t, root, "http2_probe", "engine.lunafox.http2_probe", true)

	discovery, err := discoverEngineSources(root)
	if err != nil {
		t.Fatalf("discoverEngineSources() error = %v", err)
	}
	if got, want := len(discovery.Engines), 3; got != want {
		t.Fatalf("discovered engine count = %d, want %d", got, want)
	}
	gotIDs := []string{discovery.Engines[0].EngineID, discovery.Engines[1].EngineID, discovery.Engines[2].EngineID}
	wantIDs := []string{"engine.lunafox.http2_probe", "engine.lunafox.port_scan", "engine.lunafox.website_discovery"}
	if !reflect.DeepEqual(gotIDs, wantIDs) {
		t.Fatalf("discovered engine IDs = %#v, want %#v", gotIDs, wantIDs)
	}
}

func TestRunCommandSelectsOneCanonicalDiscoveredEngine(t *testing.T) {
	root := t.TempDir()
	writeEngineSource(t, root, "port_scan", "engine.lunafox.port_scan", true)
	writeEngineSource(t, root, "website_discovery", "engine.lunafox.website_discovery", true)

	var output strings.Builder
	if err := runCommand("discover", root, "engine.lunafox.port_scan", "", "", "", "", "", "", "", &output); err != nil {
		t.Fatalf("runCommand(discover selected Engine) error = %v", err)
	}
	var discovery Discovery
	if err := json.Unmarshal([]byte(output.String()), &discovery); err != nil {
		t.Fatalf("decode selected discovery output: %v", err)
	}
	if got, want := []string{discovery.Engines[0].EngineID}, []string{"engine.lunafox.port_scan"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("selected discovery engines = %#v, want %#v", got, want)
	}
}

func TestRunCommandRejectsUnknownOrNonCanonicalEngineSelection(t *testing.T) {
	root := writeEngineSourceTree(t, true)
	for _, engineID := range []string{"engine.lunafox.unknown", " engine.lunafox.subdomain_discovery"} {
		t.Run(engineID, func(t *testing.T) {
			err := runCommand("discover", root, engineID, "", "", "", "", "", "", "", &strings.Builder{})
			if err == nil || !strings.Contains(err.Error(), "-engine-id") {
				t.Fatalf("runCommand(discover, %q) error = %v, want selected Engine rejection", engineID, err)
			}
		})
	}
}

func TestRunCommandValidatesSelectedEngineReceipt(t *testing.T) {
	root := t.TempDir()
	writeEngineSource(t, root, "port_scan", "engine.lunafox.port_scan", true)
	writeEngineSource(t, root, "website_discovery", "engine.lunafox.website_discovery", true)
	fullDiscovery, err := discoverEngineSources(root)
	if err != nil {
		t.Fatal(err)
	}
	selectedDiscovery, err := selectEngineDiscovery(fullDiscovery, "engine.lunafox.port_scan")
	if err != nil {
		t.Fatal(err)
	}
	payload, err := json.Marshal(validDevelopmentResults(selectedDiscovery))
	if err != nil {
		t.Fatal(err)
	}
	resultsPath := filepath.Join(t.TempDir(), "build-results.json")
	if err := os.WriteFile(resultsPath, payload, 0o644); err != nil {
		t.Fatal(err)
	}

	if err := runCommand("validate-build-results", root, "engine.lunafox.port_scan", resultsPath, "", "", "", "", buildModeDevelopment, "", &strings.Builder{}); err != nil {
		t.Fatalf("runCommand(validate selected receipt) error = %v", err)
	}
}

func TestRunCommandRejectsSelectedEngineForPackageCommands(t *testing.T) {
	root := writeEngineSourceTree(t, true)
	for _, command := range []string{"build-packages", "validate-package-build-results", "validate-package-artifacts"} {
		t.Run(command, func(t *testing.T) {
			err := runCommand(command, root, "engine.lunafox.subdomain_discovery", "", "", "", "", "", "", "", &strings.Builder{})
			if err == nil || !strings.Contains(err.Error(), "requires the complete discovered Engine set") {
				t.Fatalf("runCommand(%s with selected Engine) error = %v, want package partial-receipt rejection", command, err)
			}
		})
	}
}

func TestDiscoverEngineSourcesDoesNotReadLegacyRuntimeOrSourcePackageImageIdentity(t *testing.T) {
	root := writeEngineSourceTree(t, true)
	engineDir := filepath.Join(root, "subdomain_discovery")
	runtimeDir := filepath.Join(engineDir, "artifacts", "runtimes")
	if err := os.MkdirAll(runtimeDir, 0o755); err != nil {
		t.Fatal(err)
	}
	for path, payload := range map[string]string{
		filepath.Join(runtimeDir, "subdomain_discovery.runtime.json"): `{"registry":"evil.invalid","imageRef":"evil.invalid/runtime:latest"}`,
		filepath.Join(engineDir, "package.json"):                      `{"runtimeImage":{"refs":["evil.invalid/runtime:latest"]}}`,
	} {
		if err := os.WriteFile(path, []byte(payload), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	discovery, err := discoverEngineSources(root)
	if err != nil {
		t.Fatalf("discoverEngineSources() error = %v", err)
	}
	encoded, err := encodeDiscovery(discovery)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(encoded), "evil.invalid") || strings.Contains(string(encoded), "runtime:latest") {
		t.Fatalf("discovery consumed image identity outside engine.json: %s", encoded)
	}
}

func TestDiscoverEngineSourcesRejectsSymlinkedAuthorInputs(t *testing.T) {
	t.Run("engine definition", func(t *testing.T) {
		root := writeEngineSourceTree(t, true)
		manifest := filepath.Join(root, "subdomain_discovery", "engine.json")
		realManifest := filepath.Join(root, "real-engine.json")
		payload, err := os.ReadFile(manifest)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(realManifest, payload, 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.Remove(manifest); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(realManifest, manifest); err != nil {
			t.Fatal(err)
		}
		if _, err := discoverEngineSources(root); err == nil || !strings.Contains(err.Error(), "regular non-symlink") {
			t.Fatalf("expected symlinked engine definition rejection, got %v", err)
		}
	})

	t.Run("Dockerfile", func(t *testing.T) {
		root := writeEngineSourceTree(t, false)
		realDockerfile := filepath.Join(root, "real.Dockerfile")
		if err := os.WriteFile(realDockerfile, []byte("FROM scratch\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(realDockerfile, filepath.Join(root, "subdomain_discovery", "Dockerfile")); err != nil {
			t.Fatal(err)
		}
		if _, err := discoverEngineSources(root); err == nil || !strings.Contains(err.Error(), "regular non-symlink") {
			t.Fatalf("expected symlinked Dockerfile rejection, got %v", err)
		}
	})
}

func TestDiscoverEngineSourcesRejectsSourceImageIdentityFields(t *testing.T) {
	root := writeEngineSourceTree(t, true)
	path := filepath.Join(root, "subdomain_discovery", "engine.json")
	payload, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	payload = append(payload[:len(payload)-1], []byte(`,"runtimeImage":{"refs":["tag"]}}`)...)
	if err := os.WriteFile(path, payload, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := discoverEngineSources(root); err == nil || !strings.Contains(err.Error(), "unknown field") {
		t.Fatalf("expected source runtime image identity rejection, got %v", err)
	}
}

func TestValidateBuildResultsRequiresOneBuildAndValidPlatforms(t *testing.T) {
	root := writeEngineSourceTree(t, true)
	discovery, err := discoverEngineSources(root)
	if err != nil {
		t.Fatal(err)
	}
	results := validDevelopmentResults(discovery)
	if err := validateBuildResults(discovery, results, buildModeDevelopment); err != nil {
		t.Fatalf("validateBuildResults() error = %v", err)
	}
	results.Engines[0].BuildCount = 2
	if err := validateBuildResults(discovery, results, buildModeDevelopment); err == nil || !strings.Contains(err.Error(), "exactly one image build") {
		t.Fatalf("expected duplicate build rejection, got %v", err)
	}
	results = validDevelopmentResults(discovery)
	results.Engines[0].Platforms = []string{"linux/arm64"}
	if err := validateBuildResults(discovery, results, buildModeDevelopment); err != nil {
		t.Fatalf("expected single development platform acceptance, got %v", err)
	}
	results = validDevelopmentResults(discovery)
	results.Engines[0].Platforms = []string{"linux/amd64", "linux/amd64"}
	if err := validateBuildResults(discovery, results, buildModeDevelopment); err == nil || !strings.Contains(err.Error(), "must not repeat") {
		t.Fatalf("expected duplicate platform rejection, got %v", err)
	}
	results = validDevelopmentResults(discovery)
	results.Engines[0].Refs[0] = "localhost/lunafox-engine-runtime-subdomain-discovery@" + releaseTestDigestA
	results.Engines[0].SourceRef = results.Engines[0].Refs[0]
	if err := validateBuildResults(discovery, results, buildModeDevelopment); err == nil || !strings.Contains(err.Error(), "localhost Registry") {
		t.Fatalf("expected explicit development Registry port rejection, got %v", err)
	}
}

func TestValidateBuildResultsProductionCopyPreservesDigestAndOrder(t *testing.T) {
	root := writeEngineSourceTree(t, true)
	discovery, err := discoverEngineSources(root)
	if err != nil {
		t.Fatal(err)
	}
	result := RuntimeImageBuildResult{
		EngineID:       discovery.Engines[0].EngineID,
		Dockerfile:     discovery.Engines[0].Dockerfile,
		BuildContext:   ".",
		Repository:     discovery.Engines[0].Repository,
		BuildCount:     1,
		IndexDigest:    releaseTestDigestA,
		IndexMediaType: ociImageIndexMediaType,
		Platforms:      []string{"linux/amd64", "linux/arm64"},
		Refs: []string{
			"docker.io/acme/lunafox-engine-runtime-subdomain-discovery@" + releaseTestDigestA,
			"ghcr.io/acme/lunafox-engine-runtime-subdomain-discovery@" + releaseTestDigestA,
		},
		SourceRef:  "docker.io/acme/lunafox-engine-runtime-subdomain-discovery@" + releaseTestDigestA,
		CopiedRefs: []string{"ghcr.io/acme/lunafox-engine-runtime-subdomain-discovery@" + releaseTestDigestA},
	}
	results := RuntimeImageBuildResults{SchemaVersion: buildResultsSchemaVersion, Mode: buildModeProduction, Engines: []RuntimeImageBuildResult{result}}
	if err := validateBuildResults(discovery, results, buildModeProduction); err != nil {
		t.Fatalf("validateBuildResults() error = %v", err)
	}
	results.Engines[0].Refs[1] = "ghcr.io/acme/lunafox-engine-runtime-subdomain-discovery@" + releaseTestDigestB
	if err := validateBuildResults(discovery, results, buildModeProduction); err == nil || !strings.Contains(err.Error(), "same OCI image digest") {
		t.Fatalf("expected copied digest drift rejection, got %v", err)
	}
}

func TestValidateBuildResultsRejectsInvalidCandidateShapes(t *testing.T) {
	root := writeEngineSourceTree(t, true)
	discovery, err := discoverEngineSources(root)
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name string
		refs []string
		want string
	}{
		{name: "empty", refs: nil, want: "sourceRef and refs"},
		{name: "tag only", refs: []string{"localhost:5000/lunafox-engine-runtime-subdomain-discovery:latest"}, want: "digest separator"},
		{name: "placeholder", refs: []string{"localhost:5000/lunafox-engine-runtime-subdomain-discovery@sha256:<digest>"}, want: "lowercase sha256 digest"},
		{name: "duplicate location", refs: []string{
			"localhost:5000/lunafox-engine-runtime-subdomain-discovery@" + releaseTestDigestA,
			"localhost:5000/lunafox-engine-runtime-subdomain-discovery@" + releaseTestDigestA,
		}, want: "duplicate runtimeImage candidate location"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			results := validDevelopmentResults(discovery)
			results.Engines[0].Refs = append([]string(nil), test.refs...)
			if len(test.refs) == 0 {
				results.Engines[0].SourceRef = ""
			} else {
				results.Engines[0].SourceRef = test.refs[0]
			}
			err := validateBuildResults(discovery, results, buildModeDevelopment)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("validateBuildResults() error = %v, want %q", err, test.want)
			}
		})
	}
}

func TestBuildPackagesBindsOnlyVerifiedImageRefsAndClosedLayout(t *testing.T) {
	root := writeEngineSourceTree(t, true)
	discovery, err := discoverEngineSources(root)
	if err != nil {
		t.Fatal(err)
	}
	results := validDevelopmentResults(discovery)
	out := filepath.Join(t.TempDir(), "packages")
	artifacts, err := buildPackages(discovery, results, "1.2.3", out)
	if err != nil {
		t.Fatalf("buildPackages() error = %v", err)
	}
	if len(artifacts) != 1 || artifacts[0].PackageDigest == "" || artifacts[0].RuntimeImageDigest != releaseTestDigestA {
		t.Fatalf("unexpected package artifacts: %#v", artifacts)
	}
	layout, err := enginepackagecatalog.LoadEnginePackageLayoutFromRoot(filepath.Join(out, "subdomain_discovery"), 1024*1024)
	if err != nil {
		t.Fatalf("LoadEnginePackageLayoutFromRoot() error = %v", err)
	}
	if layout.Definition.PackageManifest.RuntimeImage.Refs[0] != results.Engines[0].Refs[0] {
		t.Fatalf("package did not preserve image ref: %#v", layout.Definition.PackageManifest.RuntimeImage.Refs)
	}
	if layout.Definition.PackageManifest.EngineVersion != "1.2.3" {
		t.Fatalf("unexpected package version: %#v", layout.Definition.PackageManifest)
	}
	if !reflect.DeepEqual(layout.Definition.PackageManifest.RuntimeImage.Refs, []string{results.Engines[0].Refs[0]}) {
		t.Fatalf("unexpected package refs: %#v", layout.Definition.PackageManifest.RuntimeImage.Refs)
	}
	archive, err := os.Stat(artifacts[0].ArchivePath)
	if err != nil || archive.Size() <= 0 {
		t.Fatalf("package archive missing or empty: %v", err)
	}

	packagePayload, err := os.ReadFile(filepath.Join(out, "subdomain_discovery", "package.json"))
	if err != nil {
		t.Fatal(err)
	}
	var manifest packagemanifest.PackageManifest
	if err := json.Unmarshal(packagePayload, &manifest); err != nil {
		t.Fatal(err)
	}
	if manifest.RuntimeImage.Refs[0] != results.Engines[0].Refs[0] {
		t.Fatalf("package.json ref mismatch: %#v", manifest)
	}
}

func TestBuildPackagesRejectsNonCanonicalVersion(t *testing.T) {
	root := writeEngineSourceTree(t, true)
	discovery, err := discoverEngineSources(root)
	if err != nil {
		t.Fatal(err)
	}
	_, err = buildPackages(discovery, validDevelopmentResults(discovery), " 1.2.3 ", t.TempDir())
	if err == nil || !strings.Contains(err.Error(), "must be canonical") {
		t.Fatalf("expected non-canonical version rejection, got %v", err)
	}
}

func TestBuildPackagesRejectsUnsafeOrNonSemVerVersion(t *testing.T) {
	root := writeEngineSourceTree(t, true)
	discovery, err := discoverEngineSources(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, version := range []string{"../../outside", "1.2.3/path", "v1.2.3", "1.2"} {
		t.Run(version, func(t *testing.T) {
			_, err := buildPackages(discovery, validDevelopmentResults(discovery), version, t.TempDir())
			if err == nil || !strings.Contains(err.Error(), "must match MAJOR.MINOR.PATCH") {
				t.Fatalf("buildPackages() error = %v, want SemVer rejection", err)
			}
		})
	}
}

func validDevelopmentResults(discovery Discovery) RuntimeImageBuildResults {
	source := discovery.Engines[0]
	return RuntimeImageBuildResults{
		SchemaVersion: buildResultsSchemaVersion,
		Mode:          buildModeDevelopment,
		Engines: []RuntimeImageBuildResult{{
			EngineID:       source.EngineID,
			Dockerfile:     source.Dockerfile,
			BuildContext:   source.BuildContext,
			Repository:     source.Repository,
			BuildCount:     1,
			IndexDigest:    releaseTestDigestA,
			IndexMediaType: ociImageIndexMediaType,
			Platforms:      []string{"linux/amd64", "linux/arm64"},
			Refs:           []string{"localhost:5000/" + source.Repository + "@" + releaseTestDigestA},
			SourceRef:      "localhost:5000/" + source.Repository + "@" + releaseTestDigestA,
		}},
	}
}

func writeEngineSourceTree(t *testing.T, includeDockerfile bool) string {
	t.Helper()
	root := t.TempDir()
	writeEngineSource(t, root, "subdomain_discovery", "engine.lunafox.subdomain_discovery", includeDockerfile)
	return root
}

func writeEngineSource(t *testing.T, root, directory, engineID string, includeDockerfile bool) {
	t.Helper()
	engineDir := filepath.Join(root, directory)
	if err := os.MkdirAll(filepath.Join(engineDir, "locales"), 0o755); err != nil {
		t.Fatal(err)
	}
	engine := `{"manifestVersion":"engine.v5","engineId":"` + engineID + `","publisher":"lunafox","execution":{"engineApiMajor":2,"supportedTargetTypes":["domain"],"configSections":[{"id":"scan","defaultEnabled":true,"params":[{"key":"timeout","type":"integer","default":1,"minimum":1}]}]}}`
	if err := os.WriteFile(filepath.Join(engineDir, "engine.json"), []byte(engine), 0o644); err != nil {
		t.Fatal(err)
	}
	locale := `{"engine":{"displayName":"Subdomain","description":"Subdomain"},"sections":{"scan":{"name":"Scan","description":"Scan","params":{"timeout":{"description":"Timeout"}}}}}`
	for _, language := range []string{"en", "zh"} {
		if err := os.WriteFile(filepath.Join(engineDir, "locales", language+".json"), []byte(locale), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if includeDockerfile {
		if err := os.WriteFile(filepath.Join(engineDir, "Dockerfile"), []byte("FROM scratch\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}
