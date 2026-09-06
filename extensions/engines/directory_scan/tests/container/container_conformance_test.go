package container_test

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func readDirectoryFile(t *testing.T, path ...string) string {
	t.Helper()
	payload, err := os.ReadFile(filepath.Join(path...))
	if err != nil {
		t.Fatalf("read %s: %v", filepath.Join(path...), err)
	}
	return string(payload)
}

func directorySourceFile(t *testing.T, path ...string) string {
	t.Helper()
	parts := append([]string{"..", ".."}, path...)
	return readDirectoryFile(t, parts...)
}

func requireTextMarkers(t *testing.T, payload string, markers ...string) {
	t.Helper()
	for _, marker := range markers {
		if !strings.Contains(payload, marker) {
			t.Errorf("source missing %q", marker)
		}
	}
}

func TestDockerfilePinsOfficialFFUFArchivesForBothArchitectures(t *testing.T) {
	dockerfile := directorySourceFile(t, "Dockerfile")
	requireTextMarkers(t, dockerfile,
		"ARG FFUF_VERSION=v2.2.1",
		"FROM --platform=$BUILDPLATFORM alpine:3.23.2 AS ffuf-downloader",
		"case \"$TARGETARCH\" in",
		"archive=\"ffuf_2.2.1_linux_amd64.tar.gz\"",
		"checksum=\"86307885810d3c36ba4a3e9ba5178c2d9027bba0dd7f4ea39e39e7c972b62396\"",
		"archive=\"ffuf_2.2.1_linux_arm64.tar.gz\"",
		"checksum=\"89ad4f50345e6a9a48ecc8d241811d582cedf96279276b48d666b12b31260484\"",
		"*) echo \"unsupported FFUF target architecture: $TARGETARCH\" >&2; exit 1 ;;",
		"https://github.com/ffuf/ffuf/releases/download/${FFUF_VERSION}/${archive}",
		"sha256sum -c -",
		"tar -xzf \"/tmp/$archive\" -C /out ffuf",
	)

	checksumIndex := strings.Index(dockerfile, "sha256sum -c -")
	extractIndex := strings.Index(dockerfile, "tar -xzf")
	if checksumIndex < 0 || extractIndex < 0 || checksumIndex >= extractIndex {
		t.Fatal("FFUF archive must be checksum-verified before extraction")
	}
	if !strings.Contains(dockerfile, "RUN set -eux;") ||
		strings.Contains(dockerfile, "sha256sum -c - ||") ||
		strings.Contains(dockerfile, "sha256sum -c -; true") {
		t.Fatal("FFUF checksum mismatch must fail the image build without fallback")
	}
	for _, forbidden := range []string{
		"releases/latest",
		"@latest",
		"go install github.com/ffuf/ffuf",
		"go get github.com/ffuf/ffuf",
		"git clone",
		"apt-get install ffuf",
		"apk add ffuf",
	} {
		if strings.Contains(dockerfile, forbidden) {
			t.Fatalf("Directory image contains forbidden FFUF installation path %q", forbidden)
		}
	}
}

func TestDirectoryRuntimeImageUsesCanonicalLayoutAndIsolatedFFUFHome(t *testing.T) {
	dockerfile := directorySourceFile(t, "Dockerfile")
	requireTextMarkers(t, dockerfile,
		"FROM --platform=$BUILDPLATFORM public.ecr.aws/docker/library/golang:${ENGINE_GO_VERSION}-bookworm AS engine-builder",
		"FROM --platform=$TARGETPLATFORM ${UBUNTU_BASE} AS runtime-base",
		"FROM runtime-base AS verify",
		"FROM runtime-base AS runtime",
		"ENV PATH=/opt/lunafox-tools/bin:$PATH",
		"XDG_CONFIG_HOME=/run/lunafox/ffuf-config",
		"/run/lunafox/ffuf-config/ffuf",
		"COPY --from=ffuf-downloader /out/ffuf /opt/lunafox-tools/bin/ffuf",
		"COPY --from=engine-builder /out/directory-scan-engine",
		"/opt/lunafox-engine/bin/directory-scan-engine",
		"ffuf=${FFUF_VERSION}",
		"RUN --mount=type=bind,source=directory_scan/tests/container,target=/run/lunafox-conformance",
		"/bin/sh /run/lunafox-conformance/container-conformance.sh",
		"RUN --mount=type=bind,from=verify,source=/conformance-passed,target=/run/.conformance-passed",
		"ENTRYPOINT [\"/opt/lunafox-engine/bin/directory-scan-engine\"]",
		"CMD []",
		"STOPSIGNAL SIGTERM",
		"USER 0:0",
	)
	for _, path := range []string{
		"/run/lunafox/context",
		"/run/lunafox/credential",
		"/run/lunafox/inputs",
		"/run/lunafox/resources/config",
		"/run/lunafox/resources/platform",
		"/run/lunafox/socket",
		"/workspace",
	} {
		if !strings.Contains(dockerfile, path) {
			t.Errorf("Dockerfile missing canonical runtime path %q", path)
		}
	}
	for _, forbidden := range []string{
		"COPY directory_scan/tests/container/container-conformance.sh",
		"COPY --from=verify /conformance-passed",
		"ffufrc",
		"LUNAFOX_EXECUTION_CONTEXT_FILE=",
		"LUNAFOX_ENGINE_EXECUTION_ENDPOINT=",
		"LUNAFOX_ENGINE_EXECUTION_CREDENTIAL_FILE=",
		"VOLUME ",
	} {
		if strings.Contains(dockerfile, forbidden) {
			t.Fatalf("Directory image contains forbidden runtime state %q", forbidden)
		}
	}
}

func TestDirectoryImageConformanceChecksPinnedFFUFAndArchitecture(t *testing.T) {
	script := readDirectoryFile(t, "container-conformance.sh")
	requireTextMarkers(t, script,
		"engine_path=\"${1:-/opt/lunafox-engine/bin/directory-scan-engine}\"",
		"tools_dir=\"${2:-/opt/lunafox-tools/bin}\"",
		"versions_file=\"${3:-/opt/lunafox-tools/VERSIONS}\"",
		"expected_config_home=\"${4:-/run/lunafox/ffuf-config}\"",
		"ffuf is not executable",
		"ffuf version: 2.2.1",
		"uname -m",
		"od -An -tx1 -j 18 -N 2",
		"x86_64:3e00",
		"aarch64:b700",
		"XDG_CONFIG_HOME is not isolated",
		"default FFUF config home is not empty",
		"command -v docker",
	)
}

func TestDirectoryImageConformanceRejectsMissingOrWrongFFUF(t *testing.T) {
	t.Run("missing FFUF", func(t *testing.T) {
		output, err := runDirectoryImageConformanceFixture(t, "")
		if err == nil || !strings.Contains(output, "ffuf is not executable") {
			t.Fatalf("missing FFUF conformance output = %q, error = %v", output, err)
		}
	})

	t.Run("wrong FFUF version", func(t *testing.T) {
		output, err := runDirectoryImageConformanceFixture(t, "#!/bin/sh\nprintf '%s\\n' 'ffuf version: 2.2.0'\n")
		if err == nil || !strings.Contains(output, "did not report 2.2.1") {
			t.Fatalf("wrong-version FFUF conformance output = %q, error = %v", output, err)
		}
	})
}

func TestDirectoryImageFixtureDeclaresFFUFProbeAndTypedResult(t *testing.T) {
	payload := readDirectoryFile(t, "image-conformance.json")
	var profile struct {
		SchemaVersion       string   `json:"schemaVersion"`
		NoOpEnabledSections []string `json:"noOpEnabledSections"`
		ToolEnabledSections []string `json:"toolEnabledSections"`
		ResultType          string   `json:"resultType"`
		ToolStubs           []struct {
			Tool   string `json:"tool"`
			Source string `json:"source"`
		} `json:"toolStubs"`
		ImageLocalProbes []struct {
			Tool           string   `json:"tool"`
			Args           []string `json:"args"`
			OutputContains string   `json:"outputContains"`
		} `json:"imageLocalProbes"`
	}
	if err := json.Unmarshal([]byte(payload), &profile); err != nil {
		t.Fatalf("decode image-conformance.json: %v", err)
	}
	if profile.SchemaVersion != "lunafox-engine-image-conformance-profile/v1" ||
		profile.ResultType != "asset.directory.v1" ||
		!reflect.DeepEqual(profile.NoOpEnabledSections, []string{"ffuf"}) ||
		!reflect.DeepEqual(profile.ToolEnabledSections, []string{"ffuf"}) {
		t.Fatalf("unexpected Directory image profile: %#v", profile)
	}
	if len(profile.ToolStubs) != 1 || profile.ToolStubs[0].Tool != "ffuf" || profile.ToolStubs[0].Source != "tool-stubs/ffuf.sh" {
		t.Fatalf("unexpected FFUF stub declaration: %#v", profile.ToolStubs)
	}
	if len(profile.ImageLocalProbes) != 1 || profile.ImageLocalProbes[0].Tool != "ffuf" ||
		!reflect.DeepEqual(profile.ImageLocalProbes[0].Args, []string{"-V"}) ||
		profile.ImageLocalProbes[0].OutputContains != "ffuf version: 2.2.1" {
		t.Fatalf("unexpected FFUF image-local probe: %#v", profile.ImageLocalProbes)
	}
}

func runDirectoryImageConformanceFixture(t *testing.T, ffufScript string) (string, error) {
	t.Helper()
	root := t.TempDir()
	enginePath := filepath.Join(root, "directory-scan-engine")
	toolsDir := filepath.Join(root, "tools")
	versionsPath := filepath.Join(root, "VERSIONS")
	configHome := filepath.Join(root, "ffuf-config")
	if err := os.MkdirAll(filepath.Join(configHome, "ffuf"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(toolsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(enginePath, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(versionsPath, []byte("ffuf=v2.2.1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if ffufScript != "" {
		if err := os.WriteFile(filepath.Join(toolsDir, "ffuf"), []byte(ffufScript), 0o755); err != nil {
			t.Fatal(err)
		}
	}

	scriptPath, err := filepath.Abs("container-conformance.sh")
	if err != nil {
		t.Fatal(err)
	}
	command := exec.Command("/bin/sh", scriptPath, enginePath, toolsDir, versionsPath, configHome)
	command.Env = append(os.Environ(),
		"XDG_CONFIG_HOME="+configHome,
		"PATH="+toolsDir+string(os.PathListSeparator)+os.Getenv("PATH"),
	)
	output, err := command.CombinedOutput()
	return string(output), err
}
