package containercontract

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"net"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/yyhuni/lunafox/contracts/executionartifact"
	engineexecutionpb "github.com/yyhuni/lunafox/engine-go/protocol"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/proto"
)

const testImageDigest = "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"

func TestReportKeepsSIGTERMEvidencePerEngine(t *testing.T) {
	payload, err := json.Marshal(Report{Engines: []EngineReport{{
		EngineID:        "engine.lunafox.subdomain_discovery",
		SIGTERMVerified: true,
	}}})
	if err != nil {
		t.Fatalf("marshal report: %v", err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(payload, &decoded); err != nil {
		t.Fatalf("decode report: %v", err)
	}
	if _, exists := decoded["sigtermVerified"]; exists {
		t.Fatal("SIGTERM evidence must not be a report-wide field")
	}
	engines, ok := decoded["engines"].([]any)
	if !ok || len(engines) != 1 {
		t.Fatalf("decoded engines = %#v", decoded["engines"])
	}
	engine, ok := engines[0].(map[string]any)
	if !ok || engine["sigtermVerified"] != true {
		t.Fatalf("per-Engine SIGTERM evidence = %#v", engines[0])
	}
}

func TestSelectReceiptImagesRequiresBothPlatformsAndImmutableRefs(t *testing.T) {
	inventory := loadTestInventory(t)
	receipt := testBuildReceipt(inventory)

	images, err := selectReceiptImages(inventory, receipt)
	if err != nil {
		t.Fatalf("selectReceiptImages() error = %v", err)
	}
	if len(images) != len(inventory.Engines) {
		t.Fatalf("selected image count = %d, want %d", len(images), len(inventory.Engines))
	}
	for _, image := range images {
		if !strings.HasSuffix(image.reference, "@"+testImageDigest) {
			t.Fatalf("selected mutable image reference %q", image.reference)
		}
	}

	receipt.Engines[0].Platforms = []string{"linux/amd64", "linux/amd64"}
	if _, err := selectReceiptImages(inventory, receipt); err == nil || !strings.Contains(err.Error(), "linux/amd64 and linux/arm64") {
		t.Fatalf("duplicate receipt platform error = %v", err)
	}
}

func TestSelectDirectImagesRequiresClosedDirectorySetAndCanonicalDigestRefs(t *testing.T) {
	inventory := loadTestInventory(t)
	values := directImageValues(inventory)
	images, err := selectDirectImages(inventory, values)
	if err != nil {
		t.Fatalf("selectDirectImages() error = %v", err)
	}
	if len(images) != len(inventory.Engines) {
		t.Fatalf("selected image count = %d, want %d", len(images), len(inventory.Engines))
	}

	bad := append([]string(nil), values...)
	bad[0] = strings.Replace(bad[0], "@"+testImageDigest, ":latest", 1)
	if _, err := selectDirectImages(inventory, bad); err == nil || !strings.Contains(err.Error(), "repository@sha256") {
		t.Fatalf("tag-only direct ref error = %v", err)
	}

	unknown := append([]string(nil), values...)
	unknown[0] = "other=" + strings.SplitN(values[0], "=", 2)[1]
	if _, err := selectDirectImages(inventory, unknown); err == nil || !strings.Contains(err.Error(), "missing directory") {
		t.Fatalf("unknown direct directory error = %v", err)
	}
}

func TestDecodeStrictJSONRejectsUnknownDuplicateAndTrailingFields(t *testing.T) {
	valid := `{"schemaVersion":"lunafox.engine-runtime-image-build-results.v1","mode":"development","engines":[]}`
	for name, payload := range map[string]string{
		"unknown":   `{"schemaVersion":"lunafox.engine-runtime-image-build-results.v1","mode":"development","engines":[],"other":true}`,
		"duplicate": `{"schemaVersion":"lunafox.engine-runtime-image-build-results.v1","mode":"development","mode":"production","engines":[]}`,
		"trailing":  valid + `{}`,
	} {
		t.Run(name, func(t *testing.T) {
			var receipt runtimeImageBuildResults
			if err := decodeStrictJSON([]byte(payload), "receipt.json", &receipt); err == nil {
				t.Fatal("decodeStrictJSON() succeeded, want rejection")
			}
		})
	}
}

func TestLoadToolInventoryRequiresDiscoveredSourceClosureAndConformanceProfiles(t *testing.T) {
	canonical := loadTestInventory(t)
	port := inventoryEngineByDirectory(t, canonical, "port_scan")
	website := inventoryEngineByDirectory(t, canonical, "website_discovery")

	t.Run("discovers added inventory Engine without a fixed count", func(t *testing.T) {
		root := t.TempDir()
		copyTestEngineSource(t, root, port.Directory)
		copyTestEngineSource(t, root, website.Directory)
		writeTestInventory(t, root, toolInventory{SchemaVersion: toolInventorySchema, Engines: []inventoryEngine{website, port}, ProductImages: canonical.ProductImages})

		inventory, err := loadToolInventory(root)
		if err != nil {
			t.Fatalf("loadToolInventory() error = %v", err)
		}
		if len(inventory.Engines) != 2 {
			t.Fatalf("discovered inventory count = %d, want 2", len(inventory.Engines))
		}
		if inventory.Engines[0].EngineID >= inventory.Engines[1].EngineID {
			t.Fatalf("discovered inventory is not ordered by engineId: %q, %q", inventory.Engines[0].EngineID, inventory.Engines[1].EngineID)
		}
		for _, engine := range inventory.Engines {
			wantScript := filepath.Join(root, "extensions", "engines", engine.Directory, "tests", "container", conformanceScriptFileName)
			if engine.conformanceScriptPath != wantScript {
				t.Fatalf("Engine %q conformance script = %q, want %q", engine.Directory, engine.conformanceScriptPath, wantScript)
			}
		}
	})

	t.Run("rejects missing profile", func(t *testing.T) {
		root := t.TempDir()
		copyTestEngineSource(t, root, port.Directory)
		writeTestInventory(t, root, toolInventory{SchemaVersion: toolInventorySchema, Engines: []inventoryEngine{port}, ProductImages: canonical.ProductImages})
		if err := os.Remove(filepath.Join(root, "extensions", "engines", port.Directory, "tests", "container", "image-conformance.json")); err != nil {
			t.Fatal(err)
		}
		if _, err := loadToolInventory(root); err == nil || !strings.Contains(err.Error(), "image conformance profile") {
			t.Fatalf("missing profile error = %v", err)
		}
	})

	t.Run("rejects missing container conformance script", func(t *testing.T) {
		root := t.TempDir()
		copyTestEngineSource(t, root, port.Directory)
		writeTestInventory(t, root, toolInventory{SchemaVersion: toolInventorySchema, Engines: []inventoryEngine{port}, ProductImages: canonical.ProductImages})
		if err := os.Remove(filepath.Join(root, "extensions", "engines", port.Directory, "tests", "container", conformanceScriptFileName)); err != nil {
			t.Fatal(err)
		}
		if _, err := loadToolInventory(root); err == nil || !strings.Contains(err.Error(), "container conformance script") {
			t.Fatalf("missing script error = %v", err)
		}
	})

	t.Run("rejects symlinked container conformance script", func(t *testing.T) {
		root := t.TempDir()
		copyTestEngineSource(t, root, port.Directory)
		writeTestInventory(t, root, toolInventory{SchemaVersion: toolInventorySchema, Engines: []inventoryEngine{port}, ProductImages: canonical.ProductImages})
		scriptPath := filepath.Join(root, "extensions", "engines", port.Directory, "tests", "container", conformanceScriptFileName)
		if err := os.Remove(scriptPath); err != nil {
			t.Fatal(err)
		}
		outsidePath := filepath.Join(root, "outside-conformance.sh")
		if err := os.WriteFile(outsidePath, []byte("#!/bin/sh\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(outsidePath, scriptPath); err != nil {
			t.Fatal(err)
		}
		if _, err := loadToolInventory(root); err == nil || !strings.Contains(err.Error(), "regular non-symlink") {
			t.Fatalf("symlinked script error = %v", err)
		}
	})

	t.Run("rejects symlinked container-test directory", func(t *testing.T) {
		root := t.TempDir()
		copyTestEngineSource(t, root, port.Directory)
		writeTestInventory(t, root, toolInventory{SchemaVersion: toolInventorySchema, Engines: []inventoryEngine{port}, ProductImages: canonical.ProductImages})
		containerRoot := filepath.Join(root, "extensions", "engines", port.Directory, "tests", "container")
		outsideRoot := filepath.Join(root, "outside-container-tests")
		if err := os.Rename(containerRoot, outsideRoot); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(outsideRoot, containerRoot); err != nil {
			t.Fatal(err)
		}
		if _, err := loadToolInventory(root); err == nil || !strings.Contains(err.Error(), "container-test directory must be a non-symlink directory") {
			t.Fatalf("symlinked container-test directory error = %v", err)
		}
	})

	t.Run("rejects unsafe profile fixture path", func(t *testing.T) {
		root := t.TempDir()
		copyTestEngineSource(t, root, port.Directory)
		writeTestInventory(t, root, toolInventory{SchemaVersion: toolInventorySchema, Engines: []inventoryEngine{port}, ProductImages: canonical.ProductImages})
		profilePath := filepath.Join(root, "extensions", "engines", port.Directory, "tests", "container", "image-conformance.json")
		payload, err := os.ReadFile(profilePath)
		if err != nil {
			t.Fatal(err)
		}
		payload = []byte(strings.Replace(string(payload), "tool-stubs/naabu.sh", "../escape.sh", 1))
		if err := os.WriteFile(profilePath, payload, 0o600); err != nil {
			t.Fatal(err)
		}
		if _, err := loadToolInventory(root); err == nil || !strings.Contains(err.Error(), "escapes the Engine container-test directory") {
			t.Fatalf("unsafe profile error = %v", err)
		}
	})

	t.Run("rejects retired image conformance command metadata", func(t *testing.T) {
		root := t.TempDir()
		copyTestEngineSource(t, root, port.Directory)
		writeTestInventory(t, root, toolInventory{SchemaVersion: toolInventorySchema, Engines: []inventoryEngine{port}, ProductImages: canonical.ProductImages})
		inventoryPath := filepath.Join(root, "extensions", "engines", "container", "tool-inventory.json")
		payload, err := os.ReadFile(inventoryPath)
		if err != nil {
			t.Fatal(err)
		}
		old := `"engineBinary":"port-scan-runtime-engine"`
		new := old + `,"conformanceBinary":"port-scan-conformance"`
		if !strings.Contains(string(payload), old) {
			t.Fatalf("test inventory does not contain %s", old)
		}
		if err := os.WriteFile(inventoryPath, []byte(strings.Replace(string(payload), old, new, 1)), 0o600); err != nil {
			t.Fatal(err)
		}
		if _, err := loadToolInventory(root); err == nil || !strings.Contains(err.Error(), "conformanceBinary") {
			t.Fatalf("retired conformance metadata error = %v", err)
		}
	})
}

func TestMergeCommandEnvironmentReplacesCrossBuildValues(t *testing.T) {
	merged := mergeCommandEnvironment(
		[]string{"PATH=/usr/bin", "GOOS=darwin", "GOARCH=arm64"},
		[]string{"CGO_ENABLED=0", "GOOS=linux", "GOARCH=amd64"},
	)
	want := []string{"PATH=/usr/bin", "CGO_ENABLED=0", "GOOS=linux", "GOARCH=amd64"}
	if !reflect.DeepEqual(merged, want) {
		t.Fatalf("merged environment = %q, want %q", merged, want)
	}
}

func TestBuildFixtureContextUsesCanonicalNamespacesAndLegalNoOpInputs(t *testing.T) {
	inventory := loadTestInventory(t)
	for _, engine := range inventory.Engines {
		t.Run(engine.Directory, func(t *testing.T) {
			fixture, err := newRuntimeFixture(engine, fixtureOptions{})
			if err != nil {
				t.Fatalf("newRuntimeFixture() error = %v", err)
			}
			defer fixture.Close()

			payload, err := os.ReadFile(fixture.context)
			if err != nil {
				t.Fatal(err)
			}
			var executionContext engineexecutionpb.EngineExecutionContext
			if err := proto.Unmarshal(payload, &executionContext); err != nil {
				t.Fatalf("decode fixture Context: %v", err)
			}
			if fixture.workspace == "" {
				t.Fatal("fixture workspace is required")
			}
			if executionContext.GetTarget().GetType() != engine.profile.Target.Type || executionContext.GetTarget().GetValue() != engine.profile.Target.Value {
				t.Fatalf("fixture target = %#v, want profile %#v", executionContext.GetTarget(), engine.profile.Target)
			}
			assertFixtureMount(t, fixture.mounts, engineexecutionpb.ContextFilePath, true)
			assertFixtureMount(t, fixture.mounts, engineexecutionpb.CredentialFilePath, true)
			assertFixtureMount(t, fixture.mounts, engineexecutionpb.SocketDirectoryPath, true)
			assertFixtureMount(t, fixture.mounts, engineexecutionpb.WorkspacePath, false)
			assertFixtureMount(t, fixture.mounts, engineexecutionpb.InputsDirectoryPath, true)
			assertFixtureInputsUnmaterialized(t, fixture)
			for _, resource := range engine.profile.PlatformResources {
				assertFixtureMount(t, fixture.mounts, resource.Path, true)
			}
			for _, artifact := range engine.profile.RuntimeArtifacts {
				assertFixtureMount(t, fixture.mounts, artifact.Path, true)
				source := mountSource(t, fixture.mounts, artifact.Path)
				info, err := os.Stat(source)
				if err != nil || !info.IsDir() || info.Mode().Perm() != 0o700 {
					t.Fatalf("runtime artifact %q is not a 0700 directory fixture: info=%v err=%v", artifact.Path, info, err)
				}
				for _, entry := range artifact.Entries {
					filePath := filepath.Join(source, filepath.FromSlash(entry.Name))
					entryInfo, err := os.Stat(filePath)
					if err != nil || !entryInfo.Mode().IsRegular() || entryInfo.Mode().Perm() != 0o400 {
						t.Fatalf("runtime artifact entry %q info=%v err=%v, want 0400 regular file", entry.Name, entryInfo, err)
					}
					payload, err := os.ReadFile(filePath)
					if err != nil || string(payload) != entry.Content {
						t.Fatalf("runtime artifact entry %q payload = %q err=%v", entry.Name, payload, err)
					}
				}
			}
			assertFixtureEnabledSections(t, &executionContext, engine.profile.NoOpEnabledSections)
		})
	}
}

func TestSharedFixtureBuilderDoesNotSpecialCaseNucleiEngineID(t *testing.T) {
	payload, err := os.ReadFile("conformance.go")
	if err != nil {
		t.Fatalf("read shared conformance runner: %v", err)
	}
	if strings.Contains(string(payload), "engine.lunafox.nuclei_vulnerability") {
		t.Fatal("shared conformance runner must use profile-declared runtime artifacts instead of an Engine ID branch")
	}
}

func TestToolSubprocessFixturesKeepInputsLazyAndUseReadOnlyCanonicalOverrides(t *testing.T) {
	inventory := loadTestInventory(t)
	for _, engine := range inventory.Engines {
		t.Run(engine.Directory, func(t *testing.T) {
			fixture, err := newRuntimeFixture(engine, fixtureOptions{exerciseTools: true})
			if err != nil {
				t.Fatalf("newRuntimeFixture() error = %v", err)
			}
			defer fixture.Close()
			assertFixtureInputsUnmaterialized(t, fixture)
			for _, stub := range engine.profile.ToolStubs {
				assertFixtureMount(t, fixture.mounts, "/opt/lunafox-tools/bin/"+stub.Tool, true)
			}
			for _, resource := range engine.profile.PlatformResources {
				assertFixtureMount(t, fixture.mounts, resource.Path, true)
			}
			payload, err := os.ReadFile(fixture.context)
			if err != nil {
				t.Fatal(err)
			}
			var executionContext engineexecutionpb.EngineExecutionContext
			if err := proto.Unmarshal(payload, &executionContext); err != nil {
				t.Fatal(err)
			}
			assertFixtureEnabledSections(t, &executionContext, engine.profile.ToolEnabledSections)
		})
	}
}

func TestFixtureInputServiceMaterializesOnlyRequestedZeroRecordRegistryRoles(t *testing.T) {
	engine := loadTestInventory(t).Engines[0]
	fixture, err := newRuntimeFixture(engine, fixtureOptions{})
	if err != nil {
		t.Fatalf("newRuntimeFixture() error = %v", err)
	}
	defer fixture.Close()
	assertFixtureInputsUnmaterialized(t, fixture)

	connection, err := grpc.NewClient(
		"passthrough:///engine-conformance-input",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) {
			return (&net.Dialer{}).DialContext(ctx, "unix", filepath.Join(fixture.socketDir, "engine.sock"))
		}),
	)
	if err != nil {
		t.Fatal(err)
	}
	defer connection.Close()
	client := engineexecutionpb.NewEngineExecutionInputServiceClient(connection)
	requestContext := metadata.NewOutgoingContext(context.Background(), metadata.Pairs("authorization", "Bearer "+fixture.server.token))
	descriptors := executionartifact.ExecutionInputRegistry()
	if len(descriptors) == 0 {
		t.Fatal("execution input Registry is empty")
	}

	request := func(descriptor executionartifact.Descriptor) {
		response, err := client.MaterializeExecutionInput(requestContext, &engineexecutionpb.MaterializeExecutionInputRequest{Role: descriptor.RoleID})
		if err != nil || response.GetPath() != descriptor.ContainerPath {
			t.Fatalf("MaterializeExecutionInput(%s) = %#v, %v", descriptor.RoleID, response, err)
		}
		assertFixtureZeroRecordInput(t, fixture, descriptor)
	}
	request(descriptors[0])
	for _, descriptor := range descriptors[1:] {
		if got := fixture.server.inputRequestCount(descriptor.RoleID); got != 0 {
			t.Fatalf("unused role %s made %d fixture requests", descriptor.RoleID, got)
		}
		hostPath, err := fixtureInputHostPath(fixture.inputs, descriptor.ContainerPath)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := os.Lstat(hostPath); !os.IsNotExist(err) {
			t.Fatalf("unused role %s was materialized at %q: %v", descriptor.RoleID, hostPath, err)
		}
	}
	for _, descriptor := range descriptors[1:] {
		request(descriptor)
	}
}

func TestSIGTERMFixturesBlockAfterEveryBuiltinToolActuallyStarts(t *testing.T) {
	inventory := loadTestInventory(t)
	options := sigtermFixtureOptions()
	if options.gateProgressAck || !options.exerciseTools || !options.holdToolOpen {
		t.Fatalf("SIGTERM fixture options = %#v", options)
	}
	for _, engine := range inventory.Engines {
		t.Run(engine.Directory, func(t *testing.T) {
			fixture, err := newRuntimeFixture(engine, options)
			if err != nil {
				t.Fatalf("newRuntimeFixture() error = %v", err)
			}
			defer fixture.Close()
			if fixture.server.progressGate != nil {
				t.Fatal("SIGTERM fixture must not stop before the tool process starts")
			}
			if fixture.toolStarted == "" || filepath.Dir(fixture.toolStarted) != fixture.workspace {
				t.Fatalf("SIGTERM tool start marker = %q", fixture.toolStarted)
			}
			toolName := strings.TrimSuffix(filepath.Base(fixture.toolStarted), ".started")
			if toolName != engine.profile.SIGTERMTool {
				t.Fatalf("SIGTERM tool = %q, want profile tool %q", toolName, engine.profile.SIGTERMTool)
			}
			stubPath := mountSource(t, fixture.mounts, "/opt/lunafox-tools/bin/"+toolName)
			stub, err := os.ReadFile(stubPath)
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(string(stub), "/workspace/"+toolName+".started") || !strings.Contains(string(stub), "exec sleep 300") {
				t.Fatalf("SIGTERM %s stub does not publish and hold its start marker", toolName)
			}
			assertFixtureInputsUnmaterialized(t, fixture)
			for _, stub := range engine.profile.ToolStubs {
				assertFixtureMount(t, fixture.mounts, "/opt/lunafox-tools/bin/"+stub.Tool, true)
			}
		})
	}
}

func TestVerifyImageLocalProbesUseDeclaredToolAndReadOnlyResourceFixture(t *testing.T) {
	var engine inventoryEngine
	var probe conformanceImageLocalProbe
	for _, candidate := range loadTestInventory(t).Engines {
		for _, candidateProbe := range candidate.profile.ImageLocalProbes {
			if candidateProbe.PlatformResource == "" {
				continue
			}
			engine = candidate
			probe = candidateProbe
			break
		}
		if engine.Directory != "" {
			break
		}
	}
	if engine.Directory == "" {
		t.Fatal("inventory has no image-local probe with a declared resource")
	}
	executor := &ephemeralProbeExecutor{startOutput: []byte(probe.OutputContains + "\n")}
	r := &runner{
		options: Options{DockerCommand: "docker", Platform: "linux/amd64"},
		exec:    executor,
	}
	image := selectedImage{engine: engine, reference: directImageReference(engine)}

	if err := r.verifyImageLocalProbes(context.Background(), image); err != nil {
		t.Fatalf("verifyImageLocalProbes() error = %v", err)
	}
	resource, ok := profilePlatformResource(engine.profile, probe.PlatformResource)
	if !ok {
		t.Fatalf("probe references unavailable resource %q", probe.PlatformResource)
	}
	args := executor.created().Args
	if !containsPair(args, "--entrypoint", "/opt/lunafox-tools/bin/"+probe.Tool) {
		t.Fatalf("probe does not override to its declared image-local binary: %q", args)
	}
	if len(args) < len(probe.Args) || !reflect.DeepEqual(args[len(args)-len(probe.Args):], probe.Args) {
		t.Fatalf("probe args = %q, want declared suffix %q", args, probe.Args)
	}
	mountIndex := -1
	for index, value := range args {
		if value == "-v" {
			mountIndex = index
			break
		}
	}
	if mountIndex < 0 || mountIndex+1 >= len(args) || !strings.HasSuffix(args[mountIndex+1], ":"+resource.Path+":ro") {
		t.Fatalf("declared probe resource mount is not read-only: %q", args)
	}
}

func TestCreateDefaultContainerDoesNotOverrideImageExecutionOrSecurityDefaults(t *testing.T) {
	inventory := loadTestInventory(t)
	engine := inventory.Engines[0]
	executor := &captureCommandExecutor{output: []byte("container-id\n")}
	r := &runner{
		options: Options{DockerCommand: "docker", Platform: "linux/arm64"},
		exec:    executor,
	}
	fixture := &runtimeFixture{mounts: []expectedMount{
		{Source: "/tmp/execution.pb", Target: engineexecutionpb.ContextFilePath, ReadOnly: true},
		{Type: "volume", Source: "lf-fixture-volume", Target: engineexecutionpb.SocketDirectoryPath, ReadOnly: true, Subpath: "socket"},
		{Source: "/tmp/workspace", Target: engineexecutionpb.WorkspacePath, ReadOnly: false},
	}}
	image := selectedImage{engine: engine, reference: directImageReference(engine)}

	containerID, err := r.createDefaultContainer(context.Background(), image, fixture)
	if err != nil {
		t.Fatalf("createDefaultContainer() error = %v", err)
	}
	if containerID != "container-id" {
		t.Fatalf("container ID = %q", containerID)
	}
	args := executor.last().Args
	for _, forbidden := range []string{"--entrypoint", "--user", "--network", "--privileged", "--cap-add", "--cap-drop", "--security-opt", "--env"} {
		if contains(args, forbidden) {
			t.Fatalf("docker create arguments override %s: %q", forbidden, args)
		}
	}
	if args[len(args)-1] != image.reference {
		t.Fatalf("image ref is not the final argument: %q", args)
	}
	if !containsPair(args, "--platform", "linux/arm64") || !containsPair(args, "--oom-score-adj", "500") {
		t.Fatalf("docker create arguments missing platform/OOM contract: %q", args)
	}
	if !containsPair(args, "--mount", "type=volume,source=lf-fixture-volume,target="+engineexecutionpb.SocketDirectoryPath+",volume-subpath=socket,readonly") {
		t.Fatalf("docker create arguments missing the read-only socket volume subpath: %q", args)
	}
}

func TestPullImageForPlatformEvictsOnlyTheExactReferenceBeforePulling(t *testing.T) {
	executor := &captureCommandExecutor{}
	r := &runner{options: Options{DockerCommand: "docker", Platform: "linux/amd64"}, exec: executor}
	const reference = "localhost:5001/lunafox/lunafox-engine-runtime-port-scan@" + testImageDigest

	if err := r.pullImageForPlatform(context.Background(), reference); err != nil {
		t.Fatalf("pullImageForPlatform() error = %v", err)
	}
	executor.mu.Lock()
	commands := append([]commandSpec(nil), executor.commands...)
	executor.mu.Unlock()
	want := [][]string{
		{"image", "rm", "--force", reference},
		{"pull", "--platform", "linux/amd64", reference},
	}
	if len(commands) != len(want) {
		t.Fatalf("Docker command count = %d, want %d: %#v", len(commands), len(want), commands)
	}
	for index := range want {
		if !reflect.DeepEqual(commands[index].Args, want[index]) {
			t.Fatalf("Docker command %d = %q, want %q", index, commands[index].Args, want[index])
		}
	}
}

func TestValidateImageInspectRequiresPlatformAndCanonicalImageDefaults(t *testing.T) {
	engine := loadTestInventory(t).Engines[0]
	var inspect dockerImageInspect
	inspect.ID = "sha256:image"
	inspect.OS = "linux"
	inspect.Architecture = "amd64"
	inspect.Config.User = "0:0"
	inspect.Config.WorkingDir = engineexecutionpb.WorkspacePath
	inspect.Config.Entrypoint = []string{"/opt/lunafox-engine/bin/" + engine.EngineBinary}
	inspect.Config.StopSignal = "SIGTERM"
	inspect.Config.Env = []string{"PATH=/opt/lunafox-tools/bin:/usr/bin"}
	if err := validateImageInspect(inspect, engine, "linux/amd64"); err != nil {
		t.Fatalf("validateImageInspect() error = %v", err)
	}

	inspect.Architecture = "arm64"
	if err := validateImageInspect(inspect, engine, "linux/amd64"); err == nil || !strings.Contains(err.Error(), "image platform") {
		t.Fatalf("architecture mismatch error = %v", err)
	}
	inspect.Architecture = "amd64"
	inspect.Config.Env = append(inspect.Config.Env, "LUNAFOX_EXECUTION_CONTEXT_FILE=/fallback")
	if err := validateImageInspect(inspect, engine, "linux/amd64"); err == nil || !strings.Contains(err.Error(), "bakes forbidden") {
		t.Fatalf("bootstrap fallback error = %v", err)
	}
}

func TestValidateCreatedContainerRequiresDefaultProfileAndExactMountModes(t *testing.T) {
	engine := loadTestInventory(t).Engines[0]
	fixture := &runtimeFixture{mounts: []expectedMount{
		{Source: "/private/tmp/context", Target: engineexecutionpb.ContextFilePath, ReadOnly: true},
		{Type: "volume", Source: "lf-fixture-volume", Target: engineexecutionpb.SocketDirectoryPath, ReadOnly: true, Subpath: "socket"},
		{Source: "/private/tmp/workspace", Target: engineexecutionpb.WorkspacePath, ReadOnly: false},
	}}
	var container dockerContainerInspect
	container.Config.User = "0:0"
	container.Config.WorkingDir = engineexecutionpb.WorkspacePath
	container.Config.Entrypoint = []string{"/opt/lunafox-engine/bin/" + engine.EngineBinary}
	container.Config.StopSignal = "SIGTERM"
	container.Config.Env = []string{"PATH=/opt/lunafox-tools/bin:/usr/bin"}
	container.HostConfig.NetworkMode = "default"
	container.HostConfig.OomScoreAdj = 500
	container.Mounts = append(container.Mounts,
		struct {
			Type        string `json:"Type"`
			Source      string `json:"Source"`
			Name        string `json:"Name"`
			Destination string `json:"Destination"`
			RW          bool   `json:"RW"`
		}{Type: "bind", Source: fixture.mounts[0].Source, Destination: fixture.mounts[0].Target, RW: false},
		struct {
			Type        string `json:"Type"`
			Source      string `json:"Source"`
			Name        string `json:"Name"`
			Destination string `json:"Destination"`
			RW          bool   `json:"RW"`
		}{Type: "volume", Source: "/var/lib/docker/volumes/lf-fixture-volume/_data", Name: fixture.mounts[1].Source, Destination: fixture.mounts[1].Target, RW: false},
		struct {
			Type        string `json:"Type"`
			Source      string `json:"Source"`
			Name        string `json:"Name"`
			Destination string `json:"Destination"`
			RW          bool   `json:"RW"`
		}{Type: "bind", Source: fixture.mounts[2].Source, Destination: fixture.mounts[2].Target, RW: true},
	)
	if err := validateCreatedContainer(container, engine, fixture); err != nil {
		t.Fatalf("validateCreatedContainer() error = %v", err)
	}

	privileged := container
	privileged.HostConfig.Privileged = true
	if err := validateCreatedContainer(privileged, engine, fixture); err == nil || !strings.Contains(err.Error(), "nonprivileged") {
		t.Fatalf("privileged profile error = %v", err)
	}
	wrongMode := container
	wrongMode.Mounts = append([]struct {
		Type        string `json:"Type"`
		Source      string `json:"Source"`
		Name        string `json:"Name"`
		Destination string `json:"Destination"`
		RW          bool   `json:"RW"`
	}(nil), container.Mounts...)
	wrongMode.Mounts[0].RW = true
	if err := validateCreatedContainer(wrongMode, engine, fixture); err == nil || !strings.Contains(err.Error(), "source/mode drift") {
		t.Fatalf("read-only mount error = %v", err)
	}
}

func TestDockerDefaultNetworkModeAcceptsDaemonRepresentations(t *testing.T) {
	for _, mode := range []string{"default", "bridge"} {
		if !isDockerDefaultNetworkMode(mode) {
			t.Fatalf("isDockerDefaultNetworkMode(%q) = false, want true", mode)
		}
	}
	for _, mode := range []string{"host", "none", "container:other", ""} {
		if isDockerDefaultNetworkMode(mode) {
			t.Fatalf("isDockerDefaultNetworkMode(%q) = true, want false", mode)
		}
	}
}

func TestDockerCommandTimeoutsBoundPullRunWaitAndOrdinaryOperations(t *testing.T) {
	for _, test := range []struct {
		name string
		args []string
		want time.Duration
	}{
		{name: "pull", args: []string{"pull", "image"}, want: defaultDockerPullTimeout},
		{name: "run", args: []string{"run", "image"}, want: defaultDockerRunTimeout},
		{name: "wait", args: []string{"container", "wait", "id"}, want: defaultDockerRunTimeout},
		{name: "inspect", args: []string{"container", "inspect", "id"}, want: defaultOperationTimeout},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := dockerCommandTimeout(test.args); got != test.want {
				t.Fatalf("dockerCommandTimeout(%q) = %s, want %s", test.args, got, test.want)
			}
		})
	}
}

func TestCommandWithTimeoutFailsAStuckOperation(t *testing.T) {
	r := &runner{exec: blockingCommandExecutor{}}
	started := time.Now()
	_, err := r.commandWithTimeout(context.Background(), 20*time.Millisecond, "stuck operation", commandSpec{Name: "blocked"})
	if err == nil || !strings.Contains(err.Error(), "timed out after") || !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("commandWithTimeout() error = %v", err)
	}
	if elapsed := time.Since(started); elapsed > time.Second {
		t.Fatalf("commandWithTimeout() elapsed = %s, want bounded execution", elapsed)
	}
}

func TestRemoveContainerUsesIndependentBoundedCleanupContext(t *testing.T) {
	executor := &deadlineCommandExecutor{}
	r := &runner{options: Options{DockerCommand: "docker"}, exec: executor}
	r.removeContainer("container-id")
	if !executor.sawDeadline || executor.contextErr != nil {
		t.Fatalf("cleanup context deadline=%t err=%v", executor.sawDeadline, executor.contextErr)
	}
	if !reflect.DeepEqual(executor.spec.Args, []string{"container", "rm", "--force", "container-id"}) {
		t.Fatalf("cleanup command args = %q", executor.spec.Args)
	}
}

func TestEphemeralProbeCancellationStillRemovesDaemonContainer(t *testing.T) {
	executor := &cancelledEphemeralProbeExecutor{}
	r := &runner{options: Options{DockerCommand: "docker"}, exec: executor}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	_, err := r.runEphemeralContainer(ctx, "run", "--rm", "fixture-image", "fixture-command")
	if err == nil || !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("runEphemeralContainer() error = %v", err)
	}
	if !executor.removed() {
		t.Fatal("cancelled ephemeral probe did not remove its daemon-side container")
	}
	create := executor.created()
	if len(create.Args) < 4 || contains(create.Args, "--rm") || create.Args[2] != "--name" || !strings.HasPrefix(create.Args[3], "lf-engine-conformance-probe-") {
		t.Fatalf("ephemeral create command does not use runner-owned cleanup: %q", create.Args)
	}
}

func TestRunImageLocalConformanceCopiesSourceOwnedScriptAndCleansItUp(t *testing.T) {
	engine := inventoryEngineByDirectory(t, loadTestInventory(t), "port_scan")
	image := selectedImage{engine: engine, reference: directImageReference(engine)}

	t.Run("copies the script through Docker without a bind mount", func(t *testing.T) {
		executor := &ephemeralProbeExecutor{}
		r := &runner{
			options: Options{DockerCommand: "docker", Platform: "linux/amd64"},
			exec:    executor,
		}

		if err := r.runImageLocalConformance(context.Background(), image); err != nil {
			t.Fatalf("runImageLocalConformance() error = %v", err)
		}
		create := executor.created()
		if !containsPair(create.Args, "--platform", "linux/amd64") {
			t.Fatalf("conformance container does not select the requested platform: %q", create.Args)
		}
		if !containsPair(create.Args, "--network", "none") {
			t.Fatalf("conformance container must not require network access: %q", create.Args)
		}
		if !containsPair(create.Args, "--entrypoint", "/bin/sh") {
			t.Fatalf("conformance container does not execute the source script through /bin/sh: %q", create.Args)
		}
		if contains(create.Args, "-v") || contains(create.Args, "--mount") {
			t.Fatalf("conformance container must not bind mount a source-owned script: %q", create.Args)
		}
		if contains(create.Args, "/usr/local/bin/port-scan-conformance") {
			t.Fatalf("conformance container still depends on an image-resident command: %q", create.Args)
		}
		wantTail := []string{
			image.reference,
			conformanceScriptContainerPath,
			"/opt/lunafox-engine/bin/" + engine.EngineBinary,
		}
		if len(create.Args) < len(wantTail) || !reflect.DeepEqual(create.Args[len(create.Args)-len(wantTail):], wantTail) {
			t.Fatalf("conformance container command tail = %q, want %q", create.Args, wantTail)
		}
		commands := executor.commandsSnapshot()
		copy := commandWithPrefix(commands, "cp")
		if copy.Name == "" || !reflect.DeepEqual(copy.Args, []string{"cp", engine.conformanceScriptPath, "container-id:" + conformanceScriptContainerPath}) {
			t.Fatalf("conformance script copy command = %q, want Docker API copy from source-owned script", copy.Args)
		}
		if commandWithPrefix(commands, "container", "rm").Name == "" {
			t.Fatal("conformance container was not removed after a successful copy and execution")
		}
	})

	t.Run("cleans up when the disposable container fails", func(t *testing.T) {
		executor := &ephemeralProbeExecutor{startErr: errors.New("container start failed")}
		r := &runner{
			options: Options{DockerCommand: "docker", Platform: "linux/amd64"},
			exec:    executor,
		}

		if err := r.runImageLocalConformance(context.Background(), image); err == nil || !strings.Contains(err.Error(), "container start failed") {
			t.Fatalf("runImageLocalConformance() error = %v, want disposable-container failure", err)
		}
		if commandWithPrefix(executor.commandsSnapshot(), "container", "rm").Name == "" {
			t.Fatal("failed conformance run did not remove its temporary container")
		}
	})
}

func TestReportingServerHoldsStatusOnlyAckUntilExplicitRelease(t *testing.T) {
	tokenBytes := make([]byte, 32)
	for index := range tokenBytes {
		tokenBytes[index] = byte(index + 1)
	}
	token := base64.RawURLEncoding.EncodeToString(tokenBytes)
	root, err := createShortTempDir("lf-uds-*")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(root) })
	socketPath := filepath.Join(root, "engine.sock")
	server, err := startReportingServer(socketPath, token, reportingServerOptions{gateProgressAck: true})
	if err != nil {
		t.Fatal(err)
	}
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	connection, err := grpc.NewClient(
		"passthrough:///engine-conformance",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) {
			return (&net.Dialer{}).DialContext(ctx, "unix", socketPath)
		}),
	)
	if err != nil {
		t.Fatal(err)
	}
	defer connection.Close()
	client := engineexecutionpb.NewEngineExecutionReportingServiceClient(connection)
	authorized := metadata.NewOutgoingContext(ctx, metadata.Pairs("authorization", "Bearer "+token))
	type result struct {
		response *engineexecutionpb.ReportProgressResponse
		err      error
	}
	resultChannel := make(chan result, 1)
	go func() {
		response, callErr := client.ReportProgress(authorized, &engineexecutionpb.ReportProgressRequest{Message: "ready"})
		resultChannel <- result{response: response, err: callErr}
	}()
	if err := waitForEvent(ctx, server.progressReceived, time.Second, "test progress request"); err != nil {
		t.Fatal(err)
	}
	select {
	case completed := <-resultChannel:
		t.Fatalf("progress RPC completed before acknowledgement release: %+v", completed)
	case <-time.After(50 * time.Millisecond):
	}
	server.ReleaseProgress()
	select {
	case completed := <-resultChannel:
		if completed.err != nil {
			t.Fatalf("ReportProgress() error = %v", completed.err)
		}
		if completed.response == nil || proto.Size(completed.response) != 0 {
			t.Fatalf("progress acknowledgement is not an empty status-only response: %#v", completed.response)
		}
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
}

func TestReportingServerHoldsEmptyTypedResultAcknowledgementUntilExplicitRelease(t *testing.T) {
	tokenBytes := make([]byte, 32)
	for index := range tokenBytes {
		tokenBytes[index] = byte(index + 1)
	}
	token := base64.RawURLEncoding.EncodeToString(tokenBytes)
	root, err := createShortTempDir("lf-result-uds-*")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(root) })
	socketPath := filepath.Join(root, "engine.sock")
	server, err := startReportingServer(socketPath, token, reportingServerOptions{gateResultAck: true})
	if err != nil {
		t.Fatal(err)
	}
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	connection, err := grpc.NewClient(
		"passthrough:///engine-conformance",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) {
			return (&net.Dialer{}).DialContext(ctx, "unix", socketPath)
		}),
	)
	if err != nil {
		t.Fatal(err)
	}
	defer connection.Close()
	client := engineexecutionpb.NewEngineExecutionReportingServiceClient(connection)
	authorized := metadata.NewOutgoingContext(ctx, metadata.Pairs("authorization", "Bearer "+token))
	type result struct {
		response *engineexecutionpb.SubmitResultBatchResponse
		err      error
	}
	resultChannel := make(chan result, 1)
	go func() {
		response, callErr := client.SubmitResultBatch(authorized, &engineexecutionpb.SubmitResultBatchRequest{
			ResultType: "asset.website.v1",
			Items:      [][]byte{[]byte(`{"url":"https://example.com"}`)},
		})
		resultChannel <- result{response: response, err: callErr}
	}()
	if err := waitForEvent(ctx, server.resultReceived, time.Second, "test result request"); err != nil {
		t.Fatal(err)
	}
	select {
	case completed := <-resultChannel:
		t.Fatalf("result RPC completed before acknowledgement release: %+v", completed)
	case <-time.After(50 * time.Millisecond):
	}
	server.ReleaseResult()
	select {
	case completed := <-resultChannel:
		if completed.err != nil {
			t.Fatalf("SubmitResultBatch() error = %v", completed.err)
		}
		if completed.response == nil || proto.Size(completed.response) != 0 {
			t.Fatalf("result acknowledgement is not an empty status-only response: %#v", completed.response)
		}
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
	if err := server.requireResultItems("asset.website.v1", 1); err != nil {
		t.Fatal(err)
	}
}

func loadTestInventory(t *testing.T) toolInventory {
	t.Helper()
	inventory, err := loadToolInventory(testRepoRoot(t))
	if err != nil {
		t.Fatalf("loadToolInventory() error = %v", err)
	}
	return inventory
}

func inventoryEngineByDirectory(t *testing.T, inventory toolInventory, directory string) inventoryEngine {
	t.Helper()
	for _, engine := range inventory.Engines {
		if engine.Directory == directory {
			return engine
		}
	}
	t.Fatalf("inventory is missing Engine directory %q", directory)
	return inventoryEngine{}
}

func copyTestEngineSource(t *testing.T, root, directory string) {
	t.Helper()
	sourceRoot := filepath.Join(testRepoRoot(t), "extensions", "engines", directory)
	targetRoot := filepath.Join(root, "extensions", "engines", directory)
	err := filepath.WalkDir(sourceRoot, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		relative, err := filepath.Rel(sourceRoot, path)
		if err != nil {
			return err
		}
		target := filepath.Join(targetRoot, relative)
		if entry.IsDir() {
			return os.MkdirAll(target, 0o700)
		}
		if !entry.Type().IsRegular() {
			return fmt.Errorf("test Engine source contains nonregular file %q", path)
		}
		payload, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o700); err != nil {
			return err
		}
		return os.WriteFile(target, payload, 0o600)
	})
	if err != nil {
		t.Fatalf("copy Engine source %q: %v", directory, err)
	}
}

func writeTestInventory(t *testing.T, root string, inventory toolInventory) {
	t.Helper()
	payload, err := json.Marshal(inventory)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, "extensions", "engines", "container", "tool-inventory.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, payload, 0o600); err != nil {
		t.Fatal(err)
	}
}

func testRepoRoot(t *testing.T) string {
	t.Helper()
	root, err := filepath.Abs(filepath.Join("..", "..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	return root
}

func testBuildReceipt(inventory toolInventory) runtimeImageBuildResults {
	receipt := runtimeImageBuildResults{SchemaVersion: buildResultsSchemaVersion, Mode: "development"}
	for _, engine := range inventory.Engines {
		reference := directImageReference(engine)
		receipt.Engines = append(receipt.Engines, runtimeImageBuildResult{
			EngineID: engine.EngineID, BuildCount: 1, IndexDigest: testImageDigest,
			IndexMediaType: ociImageIndexMediaType,
			Platforms:      []string{"linux/amd64", "linux/arm64"},
			Refs:           []string{reference}, SourceRef: reference,
		})
	}
	return receipt
}

func directImageValues(inventory toolInventory) []string {
	values := make([]string, 0, len(inventory.Engines))
	for _, engine := range inventory.Engines {
		values = append(values, engine.Directory+"="+directImageReference(engine))
	}
	return values
}

func directImageReference(engine inventoryEngine) string {
	repository := "lunafox-engine-runtime-" + strings.ReplaceAll(strings.TrimPrefix(engine.EngineID, "engine.lunafox."), "_", "-")
	return "localhost:5000/lunafox/" + repository + "@" + testImageDigest
}

func assertFixtureMount(t *testing.T, mounts []expectedMount, target string, readOnly bool) {
	t.Helper()
	for _, mount := range mounts {
		if mount.Target == target {
			if mount.ReadOnly != readOnly {
				t.Fatalf("mount %s readOnly = %t, want %t", target, mount.ReadOnly, readOnly)
			}
			return
		}
	}
	t.Fatalf("missing fixture mount %s", target)
}

func assertFixtureInputsUnmaterialized(t *testing.T, fixture *runtimeFixture) {
	t.Helper()
	if fixture == nil || fixture.inputs == "" {
		t.Fatal("fixture inputs directory is required")
	}
	entries, err := os.ReadDir(fixture.inputs)
	if err != nil {
		t.Fatalf("read fixture inputs directory: %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("fixture inputs directory is pre-populated: %v", entries)
	}
	for _, descriptor := range executionartifact.ExecutionInputRegistry() {
		if got := fixture.server.inputRequestCount(descriptor.RoleID); got != 0 {
			t.Fatalf("fixture role %s made %d input requests before Path()", descriptor.RoleID, got)
		}
		hostPath, err := fixtureInputHostPath(fixture.inputs, descriptor.ContainerPath)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := os.Lstat(hostPath); !os.IsNotExist(err) {
			t.Fatalf("fixture role %s is materialized before Path(): %v", descriptor.RoleID, err)
		}
	}
}

func assertFixtureZeroRecordInput(t *testing.T, fixture *runtimeFixture, descriptor executionartifact.Descriptor) {
	t.Helper()
	if got := fixture.server.inputRequestCount(descriptor.RoleID); got != 1 {
		t.Fatalf("fixture role %s request count = %d, want one", descriptor.RoleID, got)
	}
	hostPath, err := fixtureInputHostPath(fixture.inputs, descriptor.ContainerPath)
	if err != nil {
		t.Fatal(err)
	}
	info, err := os.Lstat(hostPath)
	if err != nil || !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 || info.Mode().Perm() != 0o400 {
		t.Fatalf("fixture zero-record role %s is not a read-only regular file: info=%v err=%v", descriptor.RoleID, info, err)
	}
	payload, err := os.ReadFile(hostPath)
	if err != nil || len(payload) != 0 {
		t.Fatalf("fixture zero-record role %s payload = %q, %v", descriptor.RoleID, payload, err)
	}
}

func mountSource(t *testing.T, mounts []expectedMount, target string) string {
	t.Helper()
	for _, mount := range mounts {
		if mount.Target == target {
			return mount.Source
		}
	}
	t.Fatalf("missing fixture mount %s", target)
	return ""
}

func assertNonemptyMount(t *testing.T, mounts []expectedMount, target string) {
	t.Helper()
	info, err := os.Stat(mountSource(t, mounts, target))
	if err != nil || info.Size() == 0 {
		t.Fatalf("mount %s must use a nonempty fixture: info=%v err=%v", target, info, err)
	}
}

func assertFixtureEnabledSections(t *testing.T, context *engineexecutionpb.EngineExecutionContext, want []string) {
	t.Helper()
	wantSet := make(map[string]struct{}, len(want))
	for _, sectionID := range want {
		wantSet[sectionID] = struct{}{}
	}
	for _, section := range context.GetConfig().GetSections() {
		_, wantEnabled := wantSet[section.GetSectionId()]
		if section.GetEnabled() != wantEnabled {
			t.Fatalf("fixture section %q enabled=%t, want %t from profile", section.GetSectionId(), section.GetEnabled(), wantEnabled)
		}
		if !wantEnabled && len(section.GetParams()) != 0 {
			t.Fatalf("disabled fixture section %q contains scalar parameters", section.GetSectionId())
		}
	}
	for _, resource := range context.GetConfigResources() {
		if _, enabled := wantSet[resource.GetSectionId()]; !enabled {
			t.Fatalf("disabled fixture section %q contains a resource binding", resource.GetSectionId())
		}
	}
}

type captureCommandExecutor struct {
	mu       sync.Mutex
	commands []commandSpec
	output   []byte
	err      error
}

type blockingCommandExecutor struct{}

func (blockingCommandExecutor) Run(ctx context.Context, _ commandSpec) ([]byte, error) {
	<-ctx.Done()
	return nil, ctx.Err()
}

type deadlineCommandExecutor struct {
	sawDeadline bool
	contextErr  error
	spec        commandSpec
}

type ephemeralProbeExecutor struct {
	mu          sync.Mutex
	commands    []commandSpec
	startOutput []byte
	startErr    error
	onCreate    func(commandSpec)
}

type cancelledEphemeralProbeExecutor struct {
	mu        sync.Mutex
	commands  []commandSpec
	didRemove bool
}

func (executor *cancelledEphemeralProbeExecutor) Run(ctx context.Context, spec commandSpec) ([]byte, error) {
	executor.mu.Lock()
	executor.commands = append(executor.commands, spec)
	executor.mu.Unlock()
	if len(spec.Args) >= 2 && reflect.DeepEqual(spec.Args[:2], []string{"container", "create"}) {
		return []byte("container-id\n"), nil
	}
	if len(spec.Args) >= 2 && reflect.DeepEqual(spec.Args[:2], []string{"container", "start"}) {
		<-ctx.Done()
		return nil, ctx.Err()
	}
	if len(spec.Args) >= 2 && reflect.DeepEqual(spec.Args[:2], []string{"container", "rm"}) {
		executor.mu.Lock()
		executor.didRemove = true
		executor.mu.Unlock()
		return nil, nil
	}
	return nil, errors.New("unexpected cancelled ephemeral probe command")
}

func (executor *cancelledEphemeralProbeExecutor) removed() bool {
	executor.mu.Lock()
	defer executor.mu.Unlock()
	return executor.didRemove
}

func (executor *cancelledEphemeralProbeExecutor) created() commandSpec {
	executor.mu.Lock()
	defer executor.mu.Unlock()
	for _, command := range executor.commands {
		if len(command.Args) >= 2 && reflect.DeepEqual(command.Args[:2], []string{"container", "create"}) {
			return command
		}
	}
	return commandSpec{}
}

func (executor *ephemeralProbeExecutor) Run(_ context.Context, spec commandSpec) ([]byte, error) {
	executor.mu.Lock()
	executor.commands = append(executor.commands, spec)
	onCreate := executor.onCreate
	startOutput := append([]byte(nil), executor.startOutput...)
	startErr := executor.startErr
	executor.mu.Unlock()
	if len(spec.Args) >= 2 && reflect.DeepEqual(spec.Args[:2], []string{"container", "create"}) {
		if onCreate != nil {
			onCreate(spec)
		}
		return []byte("container-id\n"), nil
	}
	if len(spec.Args) >= 1 && spec.Args[0] == "cp" {
		return nil, nil
	}
	if len(spec.Args) >= 2 && reflect.DeepEqual(spec.Args[:2], []string{"container", "start"}) {
		return startOutput, startErr
	}
	if len(spec.Args) >= 2 && reflect.DeepEqual(spec.Args[:2], []string{"container", "inspect"}) {
		return []byte(`[{"State":{"Running":false,"ExitCode":0,"OOMKilled":false,"Error":""}}]`), nil
	}
	if len(spec.Args) >= 2 && reflect.DeepEqual(spec.Args[:2], []string{"container", "rm"}) {
		return nil, nil
	}
	return nil, errors.New("unexpected ephemeral probe command")
}

func (executor *ephemeralProbeExecutor) created() commandSpec {
	executor.mu.Lock()
	defer executor.mu.Unlock()
	for _, command := range executor.commands {
		if len(command.Args) >= 2 && reflect.DeepEqual(command.Args[:2], []string{"container", "create"}) {
			return command
		}
	}
	return commandSpec{}
}

func (executor *ephemeralProbeExecutor) commandsSnapshot() []commandSpec {
	executor.mu.Lock()
	defer executor.mu.Unlock()
	return append([]commandSpec(nil), executor.commands...)
}

func (executor *deadlineCommandExecutor) Run(ctx context.Context, spec commandSpec) ([]byte, error) {
	_, executor.sawDeadline = ctx.Deadline()
	executor.contextErr = ctx.Err()
	executor.spec = spec
	return nil, nil
}

func (executor *captureCommandExecutor) Run(_ context.Context, spec commandSpec) ([]byte, error) {
	executor.mu.Lock()
	defer executor.mu.Unlock()
	executor.commands = append(executor.commands, spec)
	return append([]byte(nil), executor.output...), executor.err
}

func (executor *captureCommandExecutor) last() commandSpec {
	executor.mu.Lock()
	defer executor.mu.Unlock()
	return executor.commands[len(executor.commands)-1]
}

func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func containsPair(values []string, first, second string) bool {
	for index := 0; index+1 < len(values); index++ {
		if reflect.DeepEqual(values[index:index+2], []string{first, second}) {
			return true
		}
	}
	return false
}

func commandWithPrefix(commands []commandSpec, prefix ...string) commandSpec {
	for _, command := range commands {
		if len(command.Args) < len(prefix) || !reflect.DeepEqual(command.Args[:len(prefix)], prefix) {
			continue
		}
		return command
	}
	return commandSpec{}
}
