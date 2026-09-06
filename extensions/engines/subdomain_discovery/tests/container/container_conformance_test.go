package container_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func readSibling(t *testing.T, name string) string {
	t.Helper()
	path := filepath.Join("..", "..", name)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
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

func TestDockerfileDefinesAdditiveSubdomainRuntimeImage(t *testing.T) {
	dockerfile := readSibling(t, "Dockerfile")
	for _, marker := range []string{
		"FROM --platform=$BUILDPLATFORM public.ecr.aws/docker/library/golang:${TOOLS_GO_VERSION}-bookworm AS go-tools",
		"FROM --platform=$TARGETPLATFORM ${UBUNTU_BASE} AS native-tools",
		"FROM --platform=$BUILDPLATFORM public.ecr.aws/docker/library/golang:${ENGINE_GO_VERSION}-bookworm AS engine-builder",
		"FROM --platform=$TARGETPLATFORM ${UBUNTU_BASE} AS runtime-base",
		"FROM runtime-base AS verify",
		"FROM runtime-base AS runtime",
		"COPY --from=contracts . /contracts",
		"RUN --mount=type=bind,source=subdomain_discovery/tests/container,target=/run/lunafox-conformance",
		"/bin/sh /run/lunafox-conformance/container-conformance.sh",
		": > /conformance-passed",
		"RUN --mount=type=bind,from=verify,source=/conformance-passed,target=/run/.conformance-passed",
		"test -f /run/.conformance-passed",
		"GOBIN= go install",
		"massdns ${MASSDNS_VERSION}",
		"ENTRYPOINT [\"/opt/lunafox-engine/bin/subdomain-discovery-runtime-engine\"]",
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
		"COPY subdomain_discovery/tests/container/container-conformance.sh",
		"COPY --from=verify /conformance-passed",
		"/usr/local/bin/subdomain-discovery-conformance",
	} {
		if strings.Contains(dockerfile, forbidden) {
			t.Fatalf("Dockerfile must not retain image-resident conformance payload %q", forbidden)
		}
	}

	for _, version := range []string{
		"SUBFINDER_VERSION=v2.12.0",
		"PUREDNS_VERSION=v2.1.2-0.20260223162428-46bd4c963ec2",
		"MASSDNS_VERSION=v1.1.0",
	} {
		if !strings.Contains(dockerfile, version) {
			t.Errorf("Dockerfile missing fixed tool version %q", version)
		}
	}

	if !strings.Contains(dockerfile, "\nUSER 0:0\n") {
		t.Fatal("runtime image must explicitly retain the root identity")
	}
	for _, forbidden := range []string{
		"LUNAFOX_EXECUTION_CONTEXT_FILE=",
		"LUNAFOX_ENGINE_EXECUTION_ENDPOINT=",
		"LUNAFOX_ENGINE_EXECUTION_CREDENTIAL_FILE=",
	} {
		if strings.Contains(dockerfile, forbidden) {
			t.Fatalf("Dockerfile must not hard-code platform bootstrap environment %q", forbidden)
		}
	}
}

func TestContainerConformanceChecksToolsAndVersions(t *testing.T) {
	script := readContainerAsset(t, "container-conformance.sh")
	for _, marker := range []string{
		"engine_path=\"${1:-/opt/lunafox-engine/bin/subdomain-discovery-runtime-engine}\"",
		"tools_dir=\"${2:-/opt/lunafox-tools/bin}\"",
		"versions_file=\"${3:-/opt/lunafox-tools/VERSIONS}\"",
		"engine executable is missing or not executable",
		"assert_executable subfinder",
		"assert_executable puredns",
		"assert_executable massdns",
		"assert_help_probe massdns.real",
		"assert_file_executable massdns.real",
		"assert_version_output subfinder v2.12.0",
			"assert_version_output puredns v2.1.2",
		"assert_version_output massdns v1.1.0",
		"resolved_massdns=\"$(command -v massdns)\"",
	} {
		if !strings.Contains(script, marker) {
			t.Errorf("conformance script missing %q", marker)
		}
	}
}

func TestContainerConformanceRejectsNonRunnableMassDNSPayload(t *testing.T) {
	for _, invalidTool := range []string{"massdns.real"} {
		t.Run(invalidTool, func(t *testing.T) {
			root := t.TempDir()
			toolsDir := filepath.Join(root, "tools")
			if err := os.Mkdir(toolsDir, 0o700); err != nil {
				t.Fatal(err)
			}
			enginePath := filepath.Join(root, "engine")
			writeExecutable(t, enginePath, []byte("#!/bin/sh\nexit 0\n"))
			versionsPath := filepath.Join(root, "VERSIONS")
			if err := os.WriteFile(versionsPath, []byte("subfinder=v2.12.0\npuredns=v2.1.2-0.20260223162428-46bd4c963ec2\nmassdns=v1.1.0\n"), 0o600); err != nil {
				t.Fatal(err)
			}
			for name, version := range map[string]string{
				"subfinder": "v2.12.0",
				"puredns":   "v2.1.2",
				"massdns":   "v1.1.0",
			} {
				writeExecutable(t, filepath.Join(toolsDir, name), []byte("#!/bin/sh\nprintf '%s\\n' '"+name+" "+version+"'\n"))
			}
			for _, name := range []string{"massdns.real"} {
				payload := []byte("#!/bin/sh\nprintf '%s\\n' usage\n")
				if name == invalidTool {
					// A truncated ELF header reproduces the non-runnable payload class
					// seen when a manifest contains a binary for the wrong architecture.
					payload = []byte{0x7f, 'E', 'L', 'F', 2, 1, 1, 0}
				}
				writeExecutable(t, filepath.Join(toolsDir, name), payload)
			}

			command := exec.Command("/bin/sh", "container-conformance.sh", enginePath, toolsDir, versionsPath)
			command.Env = environmentWithPath(toolsDir + string(os.PathListSeparator) + os.Getenv("PATH"))
			output, err := command.CombinedOutput()
			if err == nil {
				t.Fatalf("conformance accepted non-runnable %s payload: %s", invalidTool, output)
			}
			if !strings.Contains(string(output), invalidTool+" help probe failed") {
				t.Fatalf("conformance error for %s = %q", invalidTool, output)
			}
		})
	}
}

func writeExecutable(t *testing.T, path string, payload []byte) {
	t.Helper()
	if err := os.WriteFile(path, payload, 0o700); err != nil {
		t.Fatal(err)
	}
}

func environmentWithPath(path string) []string {
	environment := os.Environ()
	for index, value := range environment {
		if strings.HasPrefix(value, "PATH=") {
			environment[index] = "PATH=" + path
			return environment
		}
	}
	return append(environment, "PATH="+path)
}

func TestEngineAPIImageBoundaryIsDocumented(t *testing.T) {
	readme := readSibling(t, "README.md")
	for _, marker := range []string{
		"Engine Runtime Image",
		"Engine API v2",
		"Execution.Target",
		"SubfinderProviderConfig",
		"No legacy server entrypoint",
		"Dockerfile",
		"contracts",
	} {
		if !strings.Contains(readme, marker) {
			t.Errorf("README missing Engine API v2 boundary marker %q", marker)
		}
	}
}
