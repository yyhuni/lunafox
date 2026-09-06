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

func TestDockerfileDefinesIndependentWebsiteRuntimeImage(t *testing.T) {
	dockerfile := readSibling(t, "Dockerfile")
	for _, marker := range []string{
		"FROM --platform=$BUILDPLATFORM public.ecr.aws/docker/library/golang:${TOOLS_GO_VERSION}-bookworm AS go-tools",
		"FROM --platform=$BUILDPLATFORM public.ecr.aws/docker/library/golang:${ENGINE_GO_VERSION}-bookworm AS engine-builder",
		"FROM --platform=$TARGETPLATFORM ${UBUNTU_BASE} AS runtime-base",
		"FROM runtime-base AS verify",
		"FROM runtime-base AS runtime",
		"COPY --from=contracts . /contracts",
		"GOBIN= go install",
		"RUN --mount=type=bind,source=website_discovery/tests/container,target=/run/lunafox-conformance",
		"/bin/sh /run/lunafox-conformance/container-conformance.sh",
		": > /conformance-passed",
		"RUN --mount=type=bind,from=verify,source=/conformance-passed,target=/run/.conformance-passed",
		"test -f /run/.conformance-passed",
		"ENTRYPOINT [\"/opt/lunafox-engine/bin/website-discovery-runtime-engine\"]",
		"CMD []",
		"STOPSIGNAL SIGTERM",
		"USER 0:0",
		"LABEL org.opencontainers.image.source=\"${ENGINE_IMAGE_SOURCE}\"",
		"org.opencontainers.image.version=\"${ENGINE_IMAGE_VERSION}\"",
		"WORKDIR /workspace",
		"/run/lunafox/context",
		"/run/lunafox/credential",
		"/run/lunafox/inputs",
		"/run/lunafox/resources/config",
		"/run/lunafox/resources/platform",
		"/run/lunafox/socket",
	} {
		if !strings.Contains(dockerfile, marker) {
			t.Errorf("Dockerfile missing %q", marker)
		}
	}
	for _, forbidden := range []string{
		"COPY website_discovery/tests/container/container-conformance.sh",
		"COPY --from=verify /conformance-passed",
		"/usr/local/bin/website-discovery-conformance",
	} {
		if strings.Contains(dockerfile, forbidden) {
			t.Fatalf("Dockerfile must not retain image-resident conformance payload %q", forbidden)
		}
	}
	if !strings.Contains(dockerfile, "HTTPX_VERSION=v1.8.1") {
		t.Fatal("Dockerfile must pin httpx v1.8.1")
	}
	if !strings.Contains(dockerfile, "\nUSER 0:0\n") {
		t.Fatal("website runtime image must explicitly retain the root identity")
	}
	for _, forbidden := range []string{
		"LUNAFOX_EXECUTION_CONTEXT_FILE=",
		"LUNAFOX_ENGINE_EXECUTION_ENDPOINT=",
		"LUNAFOX_ENGINE_EXECUTION_CREDENTIAL_FILE=",
		"CapAdd",
		"CapDrop",
		"SecurityOpt",
		"Privileged",
		"worker/Dockerfile",
	} {
		if strings.Contains(dockerfile, forbidden) {
			t.Fatalf("website image must not contain early profile/fallback override %q", forbidden)
		}
	}
}

func TestWebsiteImageConformanceChecksEngineHttpxAndSmoke(t *testing.T) {
	script := readContainerAsset(t, "container-conformance.sh")
	for _, marker := range []string{
		"engine_path=\"${1:-/opt/lunafox-engine/bin/website-discovery-runtime-engine}\"",
		"httpx is not executable",
		"httpx version output did not contain v1.8.1",
		"httpx -list",
		"httpx empty-list smoke failed",
		"command -v docker",
	} {
		if !strings.Contains(script, marker) {
			t.Errorf("conformance script missing %q", marker)
		}
	}
}

func TestWebsiteRuntimeArgvContractRemainsEngineOwned(t *testing.T) {
	commandTests := readSibling(t, "runtime/command_test.go")
	for _, marker := range []string{
		"buildHTTPXCommand",
		"-list",
		"-json",
		"-silent",
		"-no-color",
		"-status-code",
		"-o",
	} {
		if !strings.Contains(commandTests, marker) {
			t.Errorf("website argv regression test missing %q", marker)
		}
	}
}

func TestWebsiteRuntimeImageBuildContextIsDocumented(t *testing.T) {
	readme := readSibling(t, "README.md")
	for _, marker := range []string{
		"Engine Runtime Image",
		"--build-context contracts=./contracts",
		"--build-context engine-go=./engine-go",
		"Engine API v2",
		"Execution.Input.HostPorts",
		"zero-record",
		"Target baseline",
		"legacy server",
		"minimal",
	} {
		if !strings.Contains(readme, marker) {
			t.Errorf("README missing website image boundary marker %q", marker)
		}
	}
}
