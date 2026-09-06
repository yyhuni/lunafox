package container_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func readSibling(t *testing.T, name string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "..", name))
	if err != nil {
		t.Fatalf("read %s: %v", name, err)
	}
	return string(data)
}

func readContainerAsset(t *testing.T, name string) string {
	t.Helper()
	data, err := os.ReadFile(name)
	if err != nil {
		t.Fatalf("read container asset %s: %v", name, err)
	}
	return string(data)
}

func TestDockerfileDefinesPinnedFingerprintRuntimeImage(t *testing.T) {
	dockerfile := readSibling(t, "Dockerfile")
	for _, marker := range []string{
		"FROM --platform=$TARGETPLATFORM rust:1.94-bookworm AS observer-ward-builder",
		"git clone --branch \"$OBSERVER_WARD_VERSION\"",
		"https://github.com/yyhuni/observer_ward_for_luna.git",
		"OBSERVER_WARD_VERSION=v2026.6.28-lunafox.1",
		"OBSERVER_WARD_COMMIT=65801cf6d4b3dd4bb07ea7a1cf6e849713c42ea6",
		"refs/tags/${OBSERVER_WARD_VERSION}^{commit}",
		"cd observer_ward",
		"cd observer_ward; \\\n    rustup target add \"$target\"",
		"cargo build --locked --release --target",
		"target/$target/release/observer_ward",
		"FROM --platform=$BUILDPLATFORM public.ecr.aws/docker/library/golang:${ENGINE_GO_VERSION}-bookworm AS engine-builder",
		"FROM --platform=$TARGETPLATFORM ${UBUNTU_BASE} AS runtime-base",
		"FROM runtime-base AS verify",
		"FROM runtime-base AS runtime",
		"COPY --from=contracts . /contracts",
		"RUN --mount=type=bind,source=fingerprint_detection/tests/container,target=/run/lunafox-conformance",
		"/bin/sh /run/lunafox-conformance/container-conformance.sh",
		": > /conformance-passed",
		"RUN --mount=type=bind,from=verify,source=/conformance-passed,target=/run/.conformance-passed",
		"test -f /run/.conformance-passed",
		"/opt/lunafox-tools/bin/observer_ward",
		"/opt/lunafox-tools/VERSIONS",
		"ENTRYPOINT [\"/opt/lunafox-engine/bin/fingerprint-detection-runtime-engine\"]",
		"CMD []",
		"STOPSIGNAL SIGTERM",
		"USER 0:0",
	} {
		if !strings.Contains(dockerfile, marker) {
			t.Errorf("Dockerfile missing %q", marker)
		}
	}
	for _, forbidden := range []string{
		"COPY fingerprint_detection/tests/container/container-conformance.sh",
		"COPY --from=verify /conformance-passed",
		"LUNAFOX_EXECUTION_CONTEXT_FILE=",
		"LUNAFOX_ENGINE_EXECUTION_ENDPOINT=",
		"LUNAFOX_ENGINE_EXECUTION_CREDENTIAL_FILE=",
	} {
		if strings.Contains(dockerfile, forbidden) {
			t.Fatalf("Dockerfile must not contain %q", forbidden)
		}
	}
}

func TestFingerprintContainerConformancePinsToolAndCorpus(t *testing.T) {
	script := readContainerAsset(t, "container-conformance.sh")
	for _, marker := range []string{
		"observer-ward is not executable",
		"observer_ward_commit=",
		"65801cf6d4b3dd4bb07ea7a1cf6e849713c42ea6",
		"observer-ward --version",
		"observer_ward.real\" --help",
		"v2026.6.28-lunafox.1",
		"command -v docker",
	} {
		if !strings.Contains(script, marker) {
			t.Errorf("conformance script missing %q", marker)
		}
	}
}

func TestFingerprintContainerProfileBindsTypedInputAndPlatformCorpus(t *testing.T) {
	profile := readContainerAsset(t, "image-conformance.json")
	for _, marker := range []string{
		`"kind": "websiteURLs"`,
		`"resultType": "asset.website_technology.v1"`,
		`"tool": "observer-ward"`,
		`"fingerprintLibraryFingerPrintHub"`,
		`"application/vnd.lunafox.fingerprint-library.fingerprinthub.v1+json"`,
		`"/run/lunafox/resources/platform/fingerprintLibraryFingerPrintHub/fingerprinthub_web.json"`,
	} {
		if !strings.Contains(profile, marker) {
			t.Errorf("profile missing %q", marker)
		}
	}
}

func TestObserverWardStubPreservesExactInputTarget(t *testing.T) {
	stub := readContainerAsset(t, "tool-stubs/observer-ward.sh")
	for _, marker := range []string{
		`while IFS= read -r candidate`,
		`\"input_target\":\"$candidate\"`,
		`\"target\":\"$candidate\"`,
	} {
		if !strings.Contains(stub, marker) {
			t.Errorf("Observer Ward stub missing %q", marker)
		}
	}
}
