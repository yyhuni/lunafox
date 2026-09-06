package containercontract

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
	"testing"

	engineexecutionpb "github.com/yyhuni/lunafox/engine-go/protocol"
)

func readDockerfile(t *testing.T, directory string) string {
	t.Helper()
	payload, err := os.ReadFile(filepath.Join("..", directory, "Dockerfile"))
	if err != nil {
		t.Fatalf("read %s Dockerfile: %v", directory, err)
	}
	return string(payload)
}

func requireMarker(t *testing.T, payload, marker string) {
	t.Helper()
	if !strings.Contains(payload, marker) {
		t.Errorf("Dockerfile missing %q", marker)
	}
}

func parseDockerfileInstructions(t *testing.T, payload string) []string {
	t.Helper()
	scanner := bufio.NewScanner(strings.NewReader(payload))
	var instructions []string
	var logical strings.Builder
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		continued := strings.HasSuffix(line, "\\")
		line = strings.TrimSpace(strings.TrimSuffix(line, "\\"))
		if logical.Len() > 0 {
			logical.WriteByte(' ')
		}
		logical.WriteString(line)
		if continued {
			continue
		}
		instructions = append(instructions, strings.Join(strings.Fields(logical.String()), " "))
		logical.Reset()
	}
	if err := scanner.Err(); err != nil {
		t.Fatalf("scan Dockerfile: %v", err)
	}
	if logical.Len() != 0 {
		t.Fatal("Dockerfile ends with an unterminated continuation")
	}
	return instructions
}

func stageInstructions(t *testing.T, instructions []string, from string) []string {
	t.Helper()
	start := -1
	for index, instruction := range instructions {
		if instruction == from {
			start = index
			break
		}
	}
	if start < 0 {
		t.Fatalf("Dockerfile is missing stage %q", from)
	}
	end := len(instructions)
	for index := start + 1; index < len(instructions); index++ {
		if strings.HasPrefix(instructions[index], "FROM ") {
			end = index
			break
		}
	}
	return instructions[start:end]
}

func requireInstruction(t *testing.T, instructions []string, want string) {
	t.Helper()
	want = strings.Join(strings.Fields(want), " ")
	for _, instruction := range instructions {
		if instruction == want {
			return
		}
	}
	t.Errorf("Dockerfile instruction missing %q", want)
}

func requireInstructionContains(t *testing.T, instructions []string, marker string) {
	t.Helper()
	for _, instruction := range instructions {
		if strings.Contains(instruction, marker) {
			return
		}
	}
	t.Errorf("Dockerfile instruction missing marker %q", marker)
}

func requireSingleDirective(t *testing.T, instructions []string, directive, want string) {
	t.Helper()
	var found []string
	for _, instruction := range instructions {
		if strings.HasPrefix(instruction, directive+" ") {
			found = append(found, instruction)
		}
	}
	if len(found) != 1 || found[0] != want {
		t.Errorf("%s instructions = %q, want exactly %q", directive, found, want)
	}
}

func TestBuiltinImagesShareCanonicalContainerContract(t *testing.T) {
	for _, engine := range loadTestInventory(t).Engines {
		t.Run(engine.Directory, func(t *testing.T) {
			dockerfile := readDockerfile(t, engine.Directory)
			instructions := parseDockerfileInstructions(t, dockerfile)
			runtimeBase := stageInstructions(t, instructions, "FROM --platform=$TARGETPLATFORM ${UBUNTU_BASE} AS runtime-base")
			verify := stageInstructions(t, instructions, "FROM runtime-base AS verify")
			runtime := stageInstructions(t, instructions, "FROM runtime-base AS runtime")
			runtimeEnvironment := "ENV PATH=/opt/lunafox-tools/bin:$PATH"
			if engine.Directory == "url_collection" {
				// Its pinned Python tools are installed outside Python's default site path.
				runtimeEnvironment += " PYTHONPATH=/opt/lunafox-tools/python"
			}

			requireInstruction(t, instructions, "ARG ENGINE_IMAGE_VERSION")
			requireInstruction(t, instructions, "ARG ENGINE_IMAGE_SOURCE=https://github.com/yyhuni/lunafox")
			requireSingleDirective(t, runtimeBase, "FROM", "FROM --platform=$TARGETPLATFORM ${UBUNTU_BASE} AS runtime-base")
			requireSingleDirective(t, runtimeBase, "LABEL", "LABEL org.opencontainers.image.source=\"${ENGINE_IMAGE_SOURCE}\" org.opencontainers.image.version=\"${ENGINE_IMAGE_VERSION}\"")
			requireInstruction(t, runtimeBase, "RUN test -n \"$ENGINE_IMAGE_VERSION\" && test -n \"$ENGINE_IMAGE_SOURCE\"")
			requireSingleDirective(t, runtimeBase, "ENV", runtimeEnvironment)
			requireSingleDirective(t, runtimeBase, "WORKDIR", "WORKDIR "+engineexecutionpb.WorkspacePath)
			requireSingleDirective(t, runtime, "FROM", "FROM runtime-base AS runtime")
			requireInstruction(t, runtime, "RUN --mount=type=bind,from=verify,source=/conformance-passed,target=/run/.conformance-passed test -f /run/.conformance-passed")
			requireSingleDirective(t, runtime, "USER", "USER 0:0")
			requireSingleDirective(t, runtime, "STOPSIGNAL", "STOPSIGNAL SIGTERM")
			requireSingleDirective(t, runtime, "ENTRYPOINT", "ENTRYPOINT [\"/opt/lunafox-engine/bin/"+engine.EngineBinary+"\"]")
			requireSingleDirective(t, runtime, "CMD", "CMD []")

			for _, path := range []string{
				engineexecutionpb.ContextDirectoryPath,
				engineexecutionpb.CredentialDirectoryPath,
				engineexecutionpb.InputsDirectoryPath,
				engineexecutionpb.ConfigResourcesDirectoryPath,
				engineexecutionpb.PlatformResourcesDirectoryPath,
				engineexecutionpb.SocketDirectoryPath,
				engineexecutionpb.WorkspacePath,
			} {
				requireInstructionContains(t, runtimeBase, path)
			}
			for _, path := range []string{
				engineexecutionpb.ContextFilePath,
				engineexecutionpb.CredentialFilePath,
				engineexecutionpb.ExecutionEndpointPath,
				engineexecutionpb.SubdomainsInputPath,
				engineexecutionpb.HostPortsInputPath,
				engineexecutionpb.WebsiteURLsInputPath,
				engineexecutionpb.EndpointURLsInputPath,
			} {
				// File targets remain absent until Agent attaches task-scoped
				// mounts; the Dockerfile records them only as contract comments.
				requireMarker(t, dockerfile, path)
			}
			requireInstructionContains(t, runtimeBase, "/opt/lunafox-engine/bin/"+engine.EngineBinary)
			for _, tool := range engine.Tools {
				requireInstructionContains(t, runtimeBase, "/opt/lunafox-tools/bin/"+tool.Name)
			}
			requireInstructionContains(t, verify, "--mount=type=bind,source="+engine.Directory+"/tests/container,target=/run/lunafox-conformance")
			requireInstructionContains(t, verify, "/bin/sh /run/lunafox-conformance/container-conformance.sh /opt/lunafox-engine/bin/"+engine.EngineBinary)
			requireInstructionContains(t, verify, ": > /conformance-passed")
			for _, forbidden := range []string{
				"COPY " + engine.Directory + "/tests/container",
				"/usr/local/bin/" + strings.ReplaceAll(engine.EngineBinary, "-runtime-engine", "") + "-conformance",
				"COPY --from=verify",
			} {
				if strings.Contains(dockerfile, forbidden) {
					t.Fatalf("Dockerfile must not retain image-resident conformance payload %q", forbidden)
				}
			}
			for _, instruction := range instructions {
				if strings.HasPrefix(instruction, "VOLUME ") {
					t.Errorf("Dockerfile must not declare an image-owned volume: %q", instruction)
				}
				if strings.HasPrefix(instruction, "ENV ") {
					if strings.Contains(instruction, "LUNAFOX_") {
						t.Errorf("Dockerfile must not bake platform bootstrap environment values: %q", instruction)
					}
				}
			}
			for _, retired := range []string{
				"LUNAFOX_EXECUTION_CONTEXT_FILE",
				"LUNAFOX_ENGINE_EXECUTION_ENDPOINT",
				"LUNAFOX_ENGINE_EXECUTION_CREDENTIAL_FILE",
			} {
				if strings.Contains(dockerfile, retired) {
					t.Errorf("Dockerfile must not mention retired bootstrap environment %q", retired)
				}
			}
			if strings.Contains(dockerfile, "LUNAFOX_ENGINE_EXECUTION_RESOURCE") {
				t.Fatal("Dockerfile contains the forbidden generic execution resource bootstrap")
			}
		})
	}
}

func TestBuiltinImagesUseTargetPlatformEngineBuilds(t *testing.T) {
	for _, engine := range loadTestInventory(t).Engines {
		t.Run(engine.Directory, func(t *testing.T) {
			dockerfile := readDockerfile(t, engine.Directory)
			instructions := parseDockerfileInstructions(t, dockerfile)
			engineBuilder := stageInstructions(t, instructions, "FROM --platform=$BUILDPLATFORM public.ecr.aws/docker/library/golang:${ENGINE_GO_VERSION}-bookworm AS engine-builder")
			runtimeBase := stageInstructions(t, instructions, "FROM --platform=$TARGETPLATFORM ${UBUNTU_BASE} AS runtime-base")
			verify := stageInstructions(t, instructions, "FROM runtime-base AS verify")
			requireInstruction(t, instructions, "FROM --platform=$BUILDPLATFORM public.ecr.aws/docker/library/golang:${ENGINE_GO_VERSION}-bookworm AS engine-builder")
			requireInstruction(t, instructions, "COPY --from=engine-go . /engine-go")
			requireInstruction(t, engineBuilder, "WORKDIR /src/"+engine.Directory)
			requireInstruction(t, engineBuilder, "COPY "+engine.Directory+"/go.mod "+engine.Directory+"/go.sum ./")
			requireInstructionContains(t, engineBuilder, "go mod download")
			requireInstruction(t, engineBuilder, "COPY "+engine.Directory+" ./")
			requireInstructionContains(t, engineBuilder, "go build -trimpath -ldflags='-s -w'")
			requireInstructionContains(t, engineBuilder, "./cmd/")
			requireInstructionContains(t, instructions, "go build -trimpath -ldflags='-s -w'")
			requireInstructionContains(t, runtimeBase, "COPY --from=engine-builder /out/"+engine.EngineBinary+" /opt/lunafox-engine/bin/"+engine.EngineBinary)
			requireInstructionContains(t, verify, "/bin/sh /run/lunafox-conformance/container-conformance.sh")
			if strings.Contains(dockerfile, "GOBIN=\"") || strings.Contains(dockerfile, "GOBIN='") {
				t.Fatal("cross-platform Go tool build sets a non-empty GOBIN")
			}
			if strings.Contains(dockerfile, "GOBIN= go install") {
				requireInstructionContains(t, instructions, "${TARGETOS}_${TARGETARCH}")
			}
		})
	}
}
