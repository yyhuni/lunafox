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

func TestDockerfileDefinesIndependentPortRuntimeImage(t *testing.T) {
	dockerfile := readSibling(t, "Dockerfile")
	for _, marker := range []string{
		"FROM --platform=$BUILDPLATFORM alpine:3.23.2 AS naabu-downloader-base",
		"FROM naabu-downloader-base AS naabu-downloader",
		"FROM --platform=$BUILDPLATFORM public.ecr.aws/docker/library/golang:${ENGINE_GO_VERSION}-bookworm AS engine-builder",
		"FROM --platform=$TARGETPLATFORM ${UBUNTU_BASE} AS runtime-base",
		"FROM runtime-base AS verify",
		"FROM runtime-base AS runtime",
		"COPY --from=contracts . /contracts",
		"RUN --mount=type=bind,source=port_scan/tests/container,target=/run/lunafox-conformance",
		"/bin/sh /run/lunafox-conformance/container-conformance.sh",
		": > /conformance-passed",
		"RUN --mount=type=bind,from=verify,source=/conformance-passed,target=/run/.conformance-passed",
		"test -f /run/.conformance-passed",
		"case \"$TARGETARCH\" in",
		"archive=\"naabu_2.4.0_linux_amd64.zip\"",
		"archive=\"naabu_2.4.0_linux_arm64.zip\"",
		"unsupported Naabu target architecture",
		"wget -T 60 -O",
		"checksum=\"13972348cfa00d3fb18f8ecd4b62a9c3b57bc75ef037cea87cd4bed3463dca02\"",
		"checksum=\"a6d67c1e2a0ced4347512dd3265458b50b207908b17c9c76d38a755f5536c58b\"",
		"printf '%s  %s\\n'",
		"sha256sum -c -",
		"libpcap0.8t64",
		"ENTRYPOINT [\"/opt/lunafox-engine/bin/port-scan-runtime-engine\"]",
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
		"COPY port_scan/tests/container/container-conformance.sh",
		"COPY --from=verify /conformance-passed",
		"/usr/local/bin/port-scan-conformance",
	} {
		if strings.Contains(dockerfile, forbidden) {
			t.Fatalf("Dockerfile must not retain image-resident conformance payload %q", forbidden)
		}
	}
	if !strings.Contains(dockerfile, "NAABU_VERSION=v2.4.0") {
		t.Fatal("Dockerfile must pin naabu v2.4.0")
	}
	if strings.Contains(dockerfile, "naabu/v2/cmd/naabu@") {
		t.Fatal("port image must not cross-compile Naabu's CGO/libpcap dependency")
	}
	if strings.Contains(dockerfile, "printf '%s  %s\\\\n'") {
		t.Fatal("checksum input must contain a real newline escape, not a literal backslash")
	}
	for _, forbidden := range []string{"apk add", "apt-get install -y --no-install-recommends ca-certificates curl unzip"} {
		if strings.Contains(dockerfile, forbidden) {
			t.Fatalf("Naabu downloader must use only checksum-verified base-image tools, found %q", forbidden)
		}
	}
	if !strings.Contains(dockerfile, "\nUSER 0:0\n") {
		t.Fatal("port runtime image must explicitly retain the root identity")
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
			t.Fatalf("port image must not contain early profile/fallback override %q", forbidden)
		}
	}
}

func TestPortImageConformanceChecksEngineAndNaabu(t *testing.T) {
	script := readContainerAsset(t, "container-conformance.sh")
	for _, marker := range []string{
		"engine_path=\"${1:-/opt/lunafox-engine/bin/port-scan-runtime-engine}\"",
		"naabu is not executable",
		"expected_version=\"${marker#v}\"",
		"Current Version: $expected_version",
		"naabu version output did not report",
		"command -v docker",
	} {
		if !strings.Contains(script, marker) {
			t.Errorf("conformance script missing %q", marker)
		}
	}
}

func TestPortRuntimeArgvAndSecurityBoundaryRemainEngineOwned(t *testing.T) {
	commandTests := readSibling(t, "runtime/command_test.go")
	for _, marker := range []string{
		"buildNaabuActiveCommand",
		"buildNaabuPassiveCommand",
		"-list",
		"-passive",
		"-json",
		"-silent",
	} {
		if !strings.Contains(commandTests, marker) {
			t.Errorf("port argv regression test missing %q", marker)
		}
	}
	for _, forbidden := range []string{"CapAdd", "CapDrop", "SecurityOpt", "Privileged", "docker.sock"} {
		if strings.Contains(commandTests, forbidden) {
			t.Fatalf("port command tests must not encode an Executor security override %q", forbidden)
		}
	}
}

func TestPortRuntimeImageBuildContextIsDocumented(t *testing.T) {
	readme := readSibling(t, "README.md")
	for _, marker := range []string{
		"Engine Runtime Image",
		"--build-context contracts=./contracts",
		"--build-context engine-go=./engine-go",
		"Engine API v2",
		"Execution.Input.Subdomains",
		"zero-byte Subdomains",
		"upstream SHA-256",
		"libpcap0.8t64",
		"legacy server",
		"real SYN",
	} {
		if !strings.Contains(readme, marker) {
			t.Errorf("README missing port image boundary marker %q", marker)
		}
	}
}
