package upgrader

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/yyhuni/lunafox/contracts/releasemanifest"
	"go.yaml.in/yaml/v3"
)

type recordingRunner struct {
	commands  []recordedCommand
	responses map[string]RunResult
	errors    map[string]error
	err       error
}

type recordedCommand struct {
	binary string
	args   []string
	dir    string
}

type frontendOnlyRunner struct {
	commands []recordedCommand
}

type frontendEdgeProbeStub struct {
	observations []PublicFrontendObservation
	errors       []error
	calls        int
}

func (probe *frontendEdgeProbeStub) Probe(_ context.Context, _, _, _, _ string) (PublicFrontendObservation, error) {
	probe.calls++
	index := probe.calls - 1
	if index < len(probe.errors) && probe.errors[index] != nil {
		return PublicFrontendObservation{}, probe.errors[index]
	}
	if index >= len(probe.observations) {
		return PublicFrontendObservation{}, errors.New("public edge has not converged")
	}
	return probe.observations[index], nil
}

func (runner *frontendOnlyRunner) Run(_ context.Context, binary string, args []string, dir string) (RunResult, error) {
	runner.commands = append(runner.commands, recordedCommand{binary: binary, args: append([]string(nil), args...), dir: dir})
	joined := strings.Join(args, " ")
	switch {
	case strings.Contains(joined, "ps --format json frontend"):
		return RunResult{Stdout: `[{"Service":"frontend","State":"running","Health":"healthy"}]`}, nil
	case strings.Contains(joined, "ps -q frontend"):
		return RunResult{Stdout: "abcdef0123456789\n"}, nil
	case strings.HasPrefix(joined, "exec ") && strings.Contains(joined, " node -e "):
		return RunResult{Stdout: `{"statusCode":200,"contentType":"text/html","observedAt":"2026-09-21T00:00:00Z"}`}, nil
	case strings.Contains(joined, "inspect --format"):
		return RunResult{Stdout: `"ghcr.io/yyhuni/lunafox-frontend@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"`}, nil
	default:
		return RunResult{}, nil
	}
}

func (runner *recordingRunner) Run(_ context.Context, binary string, args []string, dir string) (RunResult, error) {
	runner.commands = append(runner.commands, recordedCommand{binary: binary, args: append([]string(nil), args...), dir: dir})
	if runner.err != nil {
		return RunResult{}, runner.err
	}
	for key, response := range runner.responses {
		if strings.Contains(strings.Join(args, " "), key) {
			if commandErr := runner.errors[key]; commandErr != nil {
				return response, commandErr
			}
			return response, nil
		}
	}
	for key, commandErr := range runner.errors {
		if strings.Contains(strings.Join(args, " "), key) {
			return RunResult{}, commandErr
		}
	}
	if strings.Contains(strings.Join(args, " "), "ps --format") {
		return RunResult{Stdout: `[{"Service":"server","State":"running"},{"Service":"frontend","State":"healthy"},{"Service":"nginx","State":"running"},{"Service":"agent","State":"running"}]`}, nil
	}
	if strings.Contains(strings.Join(args, " "), "image inspect") {
		return RunResult{Stdout: "docker.io/yyhuni/lunafox@" + testManifestDigest}, nil
	}
	return RunResult{}, nil
}

func TestComposeArgvHasFixedProjectAndNoShellCommand(t *testing.T) {
	args := ComposeArgv("/opt/lunafox", "/opt/lunafox/.lunafox/upgrade/operations/op-1.compose.json")
	want := []string{"compose", "--project-directory", "/opt/lunafox", "--env-file", "/opt/lunafox/docker/.env", "-f", "/opt/lunafox/docker/docker-compose.yml", "-f", "/opt/lunafox/.lunafox/upgrade/operations/op-1.compose.json"}
	if strings.Join(args, "\x00") != strings.Join(want, "\x00") {
		t.Fatalf("ComposeArgv() = %#v, want %#v", args, want)
	}
	for _, arg := range args {
		if strings.ContainsAny(arg, ";&|\n") {
			t.Fatalf("argv contains shell metacharacter in %q", arg)
		}
	}
}

func TestPublicComposeExecutorUsesFixedLayoutAndPersistsTarget(t *testing.T) {
	root := newShortRoot(t)
	for relative, content := range map[string]string{
		publicComposeFile: "services: {}\n",
		publicEnvFile:     "PUBLIC_PORT=443\n",
	} {
		if err := os.WriteFile(filepath.Join(root, relative), []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	store, err := NewPublicJournalStore(root)
	if err != nil {
		t.Fatal(err)
	}
	raw := releaseManifestFixtureBytes(t)
	temporaryManifest := filepath.Join(root, "candidate.yaml")
	if err := os.WriteFile(temporaryManifest, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	manifest, err := loadManifestForTest(temporaryManifest)
	if err != nil {
		t.Fatal(err)
	}
	manifestPath, err := store.ManifestPath(manifest.Digest())
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(manifestPath, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	executor, err := NewPublicComposeExecutor(&recordingRunner{}, "ghcr.io")
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	request := Request{SchemaVersion: RequestSchema, OperationID: "op-public", Action: ActionStart, ManifestDigest: manifest.Digest()}
	if err := store.Save(Journal{SchemaVersion: JournalSchema, OperationID: request.OperationID, ManifestDigest: request.ManifestDigest, Stage: StageQueued, StartedAt: now, UpdatedAt: now}); err != nil {
		t.Fatal(err)
	}
	runner := executor.Runner.(*recordingRunner)
	if err := executor.Execute(context.Background(), request, store); err != nil {
		t.Fatal(err)
	}
	commands := make([]string, 0, len(runner.commands))
	for _, command := range runner.commands {
		commands = append(commands, strings.Join(command.args, " "))
	}
	joined := strings.Join(commands, "\n")
	for _, required := range []string{
		"--project-name lunafox",
		"-f " + filepath.Join(root, publicComposeFile),
		"pull server frontend nginx agent bootstrap",
		"up -d --no-build --force-recreate --no-deps --wait --wait-timeout 300 server frontend nginx",
		"up -d --no-build --force-recreate --no-deps --wait --wait-timeout 300 agent",
	} {
		if !strings.Contains(joined, required) {
			t.Fatalf("public command trace is missing %q:\n%s", required, joined)
		}
	}
	overrideBytes, err := os.ReadFile(filepath.Join(root, publicPersistentOverrideFile))
	if err != nil {
		t.Fatal(err)
	}
	var override composeOverride
	if err := json.Unmarshal(overrideBytes, &override); err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(override.Services["agent"].Image, "ghcr.io/") {
		t.Fatalf("resident Agent registry = %q, want ghcr.io", override.Services["agent"].Image)
	}
	if override.Services["server"].Environment["RELEASE_VERSION"] != manifest.ReleaseVersion {
		t.Fatalf("persisted Server release version = %q, want %q", override.Services["server"].Environment["RELEASE_VERSION"], manifest.ReleaseVersion)
	}
	for _, service := range []string{"server", "bootstrap"} {
		if override.Services[service].Environment["ENGINE_INSTALL_REGISTRY"] != "ghcr.io" {
			t.Fatalf("persisted %s Engine registry = %q, want ghcr.io", service, override.Services[service].Environment["ENGINE_INSTALL_REGISTRY"])
		}
	}
	if override.Services["server"].Environment["RELEASE_REGISTRY"] != "ghcr.io" {
		t.Fatalf("persisted Server release registry = %q, want ghcr.io", override.Services["server"].Environment["RELEASE_REGISTRY"])
	}
	upgrader := override.Services["upgrader"]
	if !strings.HasPrefix(upgrader.Image, "ghcr.io/") {
		t.Fatalf("persistent upgrader registry = %q, want ghcr.io", upgrader.Image)
	}
	if got := strings.Join(upgrader.Command, " "); got != "--root-dir /deployment --layout public --registry ghcr.io" {
		t.Fatalf("persisted upgrader command = %q", got)
	}
	for name, service := range override.Services {
		if strings.HasPrefix(service.Image, "docker.io/yyhuni/lunafox-") {
			t.Fatalf("persistent override mixes Docker Hub image for %s: %s", name, service.Image)
		}
	}
}

func TestNewPublicComposeExecutorRejectsUnknownRegistry(t *testing.T) {
	if _, err := NewPublicComposeExecutor(&recordingRunner{}, "example.invalid"); err == nil {
		t.Fatal("unknown registry was accepted")
	}
}

func TestBuildFrontendOnlyComposePlanSealsFrontendAndKeepsConfirmedOverride(t *testing.T) {
	root := newShortRoot(t)
	executor, err := NewPublicComposeExecutor(&recordingRunner{}, "ghcr.io")
	if err != nil {
		t.Fatal(err)
	}
	operationPath := filepath.Join(root, ".lunafox", "upgrade", "operations", "frontend.compose.yaml")
	plan, err := executor.BuildFrontendOnlyComposePlan(root, operationPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := plan.Validate(); err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(plan.PullArgs, " "); !strings.HasSuffix(got, "pull frontend") {
		t.Fatalf("pull argv = %q, want frontend-only pull", got)
	}
	if got := strings.Join(plan.UpdateArgs, " "); !strings.HasSuffix(got, "--wait-timeout 300 frontend") {
		t.Fatalf("update argv = %q, want sealed frontend update", got)
	}
	if got := strings.Join(plan.HealthArgs, " "); !strings.HasSuffix(got, "ps --format json frontend") {
		t.Fatalf("health argv = %q, want frontend-only health", got)
	}
	for _, args := range [][]string{plan.PullArgs, plan.UpdateArgs, plan.HealthArgs} {
		joined := strings.Join(args, " ")
		for _, forbidden := range []string{" server", " nginx", " agent", " bootstrap", " migrate"} {
			if strings.Contains(joined, forbidden) {
				t.Fatalf("frontend plan contains forbidden service %q: %s", forbidden, joined)
			}
		}
	}
	if !containsArg(plan.BaseArgs, filepath.Join(root, publicPersistentOverrideFile)) {
		t.Fatalf("base argv omits confirmed override: %#v", plan.BaseArgs)
	}
	if !containsArg(plan.BaseArgs, operationPath) {
		t.Fatalf("base argv omits staged operation patch: %#v", plan.BaseArgs)
	}
}

func TestStageFrontendOnlyComposeOverrideChangesOnlyFrontendImage(t *testing.T) {
	root := newShortRoot(t)
	confirmedOverride := `services:
  server:
    image: ghcr.io/yyhuni/lunafox-server@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa
    environment:
      RELEASE_VERSION: 1.2.3
  frontend:
    image: ghcr.io/yyhuni/lunafox-frontend@sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb
    restart: unless-stopped
`
	for relative, content := range map[string]string{
		publicComposeFile:            "services: {}\n",
		publicEnvFile:                "PUBLIC_PORT=443\n",
		publicPersistentOverrideFile: confirmedOverride,
	} {
		if err := os.WriteFile(filepath.Join(root, relative), []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	store, err := NewPublicJournalStore(root)
	if err != nil {
		t.Fatal(err)
	}
	manifestBytes := releaseManifestFixtureBytes(t)
	manifestPath := filepath.Join(root, "candidate.yaml")
	if err := os.WriteFile(manifestPath, manifestBytes, 0o600); err != nil {
		t.Fatal(err)
	}
	manifest, err := loadManifestForTest(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(mustManifestPath(t, store, manifest.Digest()), manifestBytes, 0o600); err != nil {
		t.Fatal(err)
	}
	executor, err := NewPublicComposeExecutor(&recordingRunner{}, "ghcr.io")
	if err != nil {
		t.Fatal(err)
	}
	operationID := "frontend-only-stage"
	patchPath, err := executor.StageFrontendOnlyComposeOverride(store, operationID, manifest)
	if err != nil {
		t.Fatal(err)
	}
	patchBytes, err := os.ReadFile(patchPath)
	if err != nil {
		t.Fatal(err)
	}
	var patch map[string]any
	if err := yaml.Unmarshal(patchBytes, &patch); err != nil {
		t.Fatal(err)
	}
	services, ok := patch["services"].(map[string]any)
	if !ok || len(services) != 1 {
		t.Fatalf("staged patch services = %#v, want only frontend", patch["services"])
	}
	frontend, ok := services[FrontendOnlyService].(map[string]any)
	if !ok || len(frontend) != 1 {
		t.Fatalf("staged frontend patch = %#v, want only image", services[FrontendOnlyService])
	}
	if _, ok := frontend["image"].(string); !ok {
		t.Fatalf("staged frontend image = %#v", frontend["image"])
	}
	confirmed, err := os.ReadFile(filepath.Join(root, publicPersistentOverrideFile))
	if err != nil {
		t.Fatal(err)
	}
	if string(confirmed) != confirmedOverride {
		t.Fatalf("persistent override changed unexpectedly: %s", confirmed)
	}
}

func TestStageFrontendOnlyComposeOverrideRejectsMutableBaseline(t *testing.T) {
	root := newShortRoot(t)
	for relative, content := range map[string]string{
		publicComposeFile:            "services: {}\n",
		publicEnvFile:                "PUBLIC_PORT=443\n",
		publicPersistentOverrideFile: "services:\n  frontend:\n    image: ghcr.io/yyhuni/lunafox-frontend:latest\n",
	} {
		if err := os.WriteFile(filepath.Join(root, relative), []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	store, err := NewPublicJournalStore(root)
	if err != nil {
		t.Fatal(err)
	}
	manifestPath := filepath.Join(root, "candidate.yaml")
	manifestBytes := releaseManifestFixtureBytes(t)
	if err := os.WriteFile(manifestPath, manifestBytes, 0o600); err != nil {
		t.Fatal(err)
	}
	manifest, err := loadManifestForTest(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	executor, err := NewPublicComposeExecutor(&recordingRunner{}, "ghcr.io")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := executor.StageFrontendOnlyComposeOverride(store, "mutable-baseline", manifest); err == nil {
		t.Fatal("mutable confirmed frontend image was accepted")
	}
}

func TestPromoteFrontendOnlyComposeOverridePreservesUntouchedServices(t *testing.T) {
	root := newShortRoot(t)
	confirmedOverride := `services:
  server:
    image: ghcr.io/yyhuni/lunafox-server@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa
    environment:
      RELEASE_VERSION: 1.2.3
  frontend:
    image: ghcr.io/yyhuni/lunafox-frontend@sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb
    restart: unless-stopped
`
	for relative, content := range map[string]string{
		publicComposeFile:            "services: {}\n",
		publicEnvFile:                "PUBLIC_PORT=443\n",
		publicPersistentOverrideFile: confirmedOverride,
	} {
		if err := os.WriteFile(filepath.Join(root, relative), []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	store, err := NewPublicJournalStore(root)
	if err != nil {
		t.Fatal(err)
	}
	manifestBytes := releaseManifestFixtureBytes(t)
	manifestPath := filepath.Join(root, "candidate.yaml")
	if err := os.WriteFile(manifestPath, manifestBytes, 0o600); err != nil {
		t.Fatal(err)
	}
	manifest, err := loadManifestForTest(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	executor, err := NewPublicComposeExecutor(&recordingRunner{}, "ghcr.io")
	if err != nil {
		t.Fatal(err)
	}
	operationID := "frontend-only-promote"
	if _, err := executor.StageFrontendOnlyComposeOverride(store, operationID, manifest); err != nil {
		t.Fatal(err)
	}
	if err := executor.PromoteFrontendOnlyComposeOverride(store, operationID, manifest); err != nil {
		t.Fatal(err)
	}
	promoted, err := readConfirmedComposeOverride(filepath.Join(root, publicPersistentOverrideFile))
	if err != nil {
		t.Fatal(err)
	}
	services := promoted["services"].(map[string]any)
	server := services["server"].(map[string]any)
	if server["image"] != "ghcr.io/yyhuni/lunafox-server@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa" {
		t.Fatalf("server image changed during frontend promotion: %#v", server["image"])
	}
	serverEnvironment := server["environment"].(map[string]any)
	if serverEnvironment["RELEASE_VERSION"] != "1.2.3" {
		t.Fatalf("server environment changed during frontend promotion: %#v", serverEnvironment)
	}
	frontend := services[FrontendOnlyService].(map[string]any)
	wantFrontend, err := executor.selectedImageRef(manifest, FrontendOnlyService)
	if err != nil {
		t.Fatal(err)
	}
	if frontend["image"] != wantFrontend || frontend["restart"] != "unless-stopped" {
		t.Fatalf("frontend promotion did not preserve service fields: %#v", frontend)
	}
}

func mustManifestPath(t *testing.T, store *JournalStore, digest string) string {
	t.Helper()
	path, err := store.ManifestPath(digest)
	if err != nil {
		t.Fatal(err)
	}
	return path
}

func TestOSCommandRunnerUsesStructuredArguments(t *testing.T) {
	runner := NewOSCommandRunner()
	result, err := runner.Run(context.Background(), "printf", []string{"%s", "argv-is-data"}, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if result.Stdout != "argv-is-data" {
		t.Fatalf("stdout = %q, want argv-is-data", result.Stdout)
	}
	// A shell metacharacter remains data because the runner calls the binary
	// directly instead of passing the argv through sh -c.
	result, err = runner.Run(context.Background(), "printf", []string{"%s", "$(touch should-not-exist)"}, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if result.Stdout != "$(touch should-not-exist)" {
		t.Fatalf("shell metacharacter was interpreted: %q", result.Stdout)
	}
}

func TestComposeExecutorRunsFixedLifecycleAndWritesReceipt(t *testing.T) {
	root := newComposeRoot(t)
	store, err := NewJournalStore(root)
	if err != nil {
		t.Fatal(err)
	}
	runner := &recordingRunner{}
	executor := NewComposeExecutor(runner)
	manifest, err := loadManifestForTest(filepath.Join(root, defaultManifestName))
	if err != nil {
		t.Fatal(err)
	}
	request := Request{SchemaVersion: RequestSchema, OperationID: "op-compose", Action: ActionStart, ManifestDigest: manifest.Digest()}
	now := time.Now().UTC()
	if err := store.Save(Journal{SchemaVersion: JournalSchema, OperationID: request.OperationID, ManifestDigest: request.ManifestDigest, Stage: StageQueued, StartedAt: now, UpdatedAt: now}); err != nil {
		t.Fatal(err)
	}
	if err := executor.Execute(context.Background(), request, store); err != nil {
		t.Fatal(err)
	}
	if len(runner.commands) != 7 {
		t.Fatalf("runner command count = %d, want 7", len(runner.commands))
	}
	for index, command := range runner.commands {
		if command.binary != "docker" || command.dir != root {
			t.Fatalf("command %d = %#v, want docker in deployment root", index, command)
		}
	}
	if _, err := store.LoadReceipt(request.OperationID); err != nil {
		t.Fatalf("LoadReceipt() = %v", err)
	}
	current, err := store.LoadCurrent()
	if err != nil {
		t.Fatal(err)
	}
	if current.Stage != StageVerifying {
		t.Fatalf("current stage = %q, want verifying until Server evidence arrives", current.Stage)
	}
	if _, err := os.Stat(filepath.Join(store.Directory(), HistoryDirectory, request.OperationID+OverrideFileSuffix)); err != nil {
		t.Fatalf("compose override missing: %v", err)
	}
	keys := make([]string, 0, len(current.ProgressEvents))
	for _, event := range current.ProgressEvents {
		keys = append(keys, event.MessageKey)
		if strings.ContainsAny(event.Message, "/\\\n\r") || strings.Contains(strings.ToLower(event.Message), "compose") {
			t.Fatalf("unsafe progress event = %#v", event)
		}
	}
	wantKeys := []string{
		ProgressPreflightStarted,
		ProgressPullImagesStarted,
		ProgressServicesUpdateStarted,
		ProgressServicesUpdated,
		ProgressHealthCheckStarted,
		ProgressAgentVerificationStarted,
		ProgressDigestVerificationStarted,
		ProgressHostExecutionCompleted,
	}
	if strings.Join(keys, ",") != strings.Join(wantKeys, ",") {
		t.Fatalf("progress keys = %#v, want %#v", keys, wantKeys)
	}
}

func TestComposeExecutorRunsFullPathForPinnedLegacyAlpha114(t *testing.T) {
	root := newComposeRoot(t)
	manifestPath := filepath.Join(root, defaultManifestName)
	legacyBytes, err := os.ReadFile(legacyAlpha114ManifestFixturePath(t))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(manifestPath, legacyBytes, 0o600); err != nil {
		t.Fatal(err)
	}
	manifest, err := releasemanifest.LoadLegacyAlpha114(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	store, err := NewJournalStore(root)
	if err != nil {
		t.Fatal(err)
	}
	request := Request{SchemaVersion: RequestSchema, OperationID: "legacy-alpha114-full", Action: ActionStart, ManifestDigest: manifest.Digest()}
	now := time.Now().UTC()
	if err := store.Save(Journal{SchemaVersion: JournalSchema, OperationID: request.OperationID, ManifestDigest: request.ManifestDigest, Stage: StageQueued, StartedAt: now, UpdatedAt: now}); err != nil {
		t.Fatal(err)
	}
	runner := &recordingRunner{responses: make(map[string]RunResult)}
	for _, service := range []string{"server", "frontend", "nginx"} {
		refs, refsErr := manifest.RuntimeImageRefs(service)
		if refsErr != nil {
			t.Fatal(refsErr)
		}
		runner.responses[refs[0]] = RunResult{Stdout: refs[0]}
	}
	if err := NewComposeExecutor(runner).Execute(context.Background(), request, store); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	current, err := store.LoadCurrent()
	if err != nil {
		t.Fatal(err)
	}
	if current.Stage != StageVerifying {
		t.Fatalf("legacy full path stage = %q, want %q", current.Stage, StageVerifying)
	}
}

func TestComposeExecutorRunsFullPathForRegisteredAlpha164Bridge(t *testing.T) {
	root := newComposeRoot(t)
	manifestPath := filepath.Join(root, defaultManifestName)
	bridgeBytes := alpha164BridgeManifestFixtureBytes(t)
	if err := os.WriteFile(manifestPath, bridgeBytes, 0o600); err != nil {
		t.Fatal(err)
	}
	manifest, err := loadManifestWithLegacyCompatibility(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	if manifest.HasRuntimeComposition() {
		t.Fatal("bridge manifest unexpectedly has composition evidence")
	}
	store, err := NewJournalStore(root)
	if err != nil {
		t.Fatal(err)
	}
	request := Request{SchemaVersion: RequestSchema, OperationID: "alpha164-bridge-full", Action: ActionStart, ManifestDigest: manifest.Digest()}
	now := time.Now().UTC()
	if err := store.Save(Journal{SchemaVersion: JournalSchema, OperationID: request.OperationID, ManifestDigest: request.ManifestDigest, Stage: StageQueued, StartedAt: now, UpdatedAt: now}); err != nil {
		t.Fatal(err)
	}
	runner := &recordingRunner{responses: make(map[string]RunResult)}
	for _, service := range []string{"server", "frontend", "nginx"} {
		refs, refsErr := manifest.RuntimeImageRefs(service)
		if refsErr != nil {
			t.Fatal(refsErr)
		}
		runner.responses[refs[0]] = RunResult{Stdout: refs[0]}
	}
	if err := NewComposeExecutor(runner).Execute(context.Background(), request, store); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	current, err := store.LoadCurrent()
	if err != nil {
		t.Fatal(err)
	}
	if current.Stage != StageVerifying {
		t.Fatalf("bridge full path stage = %q, want %q", current.Stage, StageVerifying)
	}
}

func TestComposeExecutorFrontendOnlyUsesSealedCommandsAndV2Receipt(t *testing.T) {
	root := newShortRoot(t)
	for relative, content := range map[string]string{
		publicComposeFile:            "services: {}\n",
		publicEnvFile:                "PUBLIC_HOST=localhost\nPUBLIC_PORT=443\nPUBLIC_URL=https://localhost\n",
		publicPersistentOverrideFile: "services:\n  frontend:\n    image: ghcr.io/yyhuni/lunafox-frontend@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa\n",
	} {
		if err := os.WriteFile(filepath.Join(root, relative), []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	store, err := NewPublicJournalStore(root)
	if err != nil {
		t.Fatal(err)
	}
	manifestPath := filepath.Join(root, "candidate.yaml")
	manifestBytes := releaseManifestFixtureBytes(t)
	if err := os.WriteFile(manifestPath, manifestBytes, 0o600); err != nil {
		t.Fatal(err)
	}
	manifest, err := loadManifestForTest(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(mustManifestPath(t, store, manifest.Digest()), manifestBytes, 0o600); err != nil {
		t.Fatal(err)
	}
	runner := &frontendOnlyRunner{}
	executor, err := NewPublicComposeExecutor(runner, "ghcr.io")
	if err != nil {
		t.Fatal(err)
	}
	executor.PublicFrontendProbe = &frontendEdgeProbeStub{observations: []PublicFrontendObservation{{StatusCode: 200, ContentType: "text/html", ObservedAt: time.Now().UTC()}}}
	executor.FrontendEdgeProbeAttempts = 1
	request := Request{
		SchemaVersion:              ScopedRequestSchema,
		OperationID:                "frontend-only-execute",
		Action:                     ActionStart,
		ManifestDigest:             manifest.Digest(),
		ExecutionMode:              ExecutionModeFrontendOnly,
		PlanDigest:                 "sha256:" + strings.Repeat("1", 64),
		BaselineStateDigest:        "sha256:" + strings.Repeat("2", 64),
		TouchedServices:            []string{FrontendOnlyService},
		ConfirmedDeploymentVersion: "1.2.2",
	}
	now := time.Now().UTC()
	if err := store.Save(Journal{
		SchemaVersion:              ScopedJournalSchema,
		OperationID:                request.OperationID,
		ManifestDigest:             request.ManifestDigest,
		Stage:                      StageQueued,
		ExecutionMode:              request.ExecutionMode,
		PlanDigest:                 request.PlanDigest,
		BaselineStateDigest:        request.BaselineStateDigest,
		TouchedServices:            request.TouchedServices,
		ConfirmedDeploymentVersion: request.ConfirmedDeploymentVersion,
		StartedAt:                  now,
		UpdatedAt:                  now,
	}); err != nil {
		t.Fatal(err)
	}
	if err := executor.Execute(context.Background(), request, store); err != nil {
		t.Fatal(err)
	}
	if len(runner.commands) != 5 {
		t.Fatalf("frontend-only command count = %d, want pull/up/ps/ps -q/inspect", len(runner.commands))
	}
	joined := make([]string, 0, len(runner.commands))
	for _, command := range runner.commands {
		joined = append(joined, strings.Join(command.args, " "))
	}
	trace := strings.Join(joined, "\n")
	for _, forbidden := range []string{"pull server", "pull nginx", "pull agent", "migrate", "upgrader", "agent"} {
		if strings.Contains(trace, forbidden) {
			t.Fatalf("frontend-only trace contains forbidden operation %q:\n%s", forbidden, trace)
		}
	}
	if !strings.Contains(trace, "pull frontend") || !strings.Contains(trace, "--wait-timeout 300 frontend") || !strings.Contains(trace, "ps --format json frontend") || !strings.Contains(trace, "ps -q frontend") || !strings.Contains(trace, "inspect --format") {
		t.Fatalf("frontend-only trace is incomplete:\n%s", trace)
	}
	receipt, err := store.LoadReceipt(request.OperationID)
	if err != nil {
		t.Fatal(err)
	}
	if receipt.SchemaVersion != ScopedJournalSchema || receipt.ExecutionMode != ExecutionModeFrontendOnly || !sameStringSlice(receipt.Services, []string{FrontendOnlyService}) {
		t.Fatalf("frontend-only receipt = %#v", receipt)
	}
	current, err := store.LoadCurrent()
	if err != nil {
		t.Fatal(err)
	}
	if current.Stage != StageVerifying {
		t.Fatalf("frontend-only stage = %q, want verifying until confirm", current.Stage)
	}
	confirmed, err := os.ReadFile(filepath.Join(root, publicPersistentOverrideFile))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(confirmed), "sha256:"+strings.Repeat("a", 64)) {
		t.Fatal("frontend-only execution promoted the persistent override before confirm")
	}
	if _, err := os.Stat(filepath.Join(store.Directory(), HistoryDirectory, request.OperationID+OverrideFileSuffix)); err != nil {
		t.Fatalf("staged frontend override is missing: %v", err)
	}
	commandCount := len(runner.commands)
	resume := request
	resume.Action = ActionResume
	if err := executor.Execute(context.Background(), resume, store); err != nil {
		t.Fatalf("resume verified frontend-only handoff: %v", err)
	}
	if len(runner.commands) != commandCount {
		t.Fatalf("resume reran verified frontend-only commands: %#v", runner.commands[commandCount:])
	}
}

func TestComposeExecutorFrontendOnlyRejectsStoppingStage(t *testing.T) {
	root := newShortRoot(t)
	for relative, content := range map[string]string{
		publicComposeFile:            "services: {}\n",
		publicEnvFile:                "PUBLIC_HOST=localhost\nPUBLIC_PORT=443\nPUBLIC_URL=https://localhost\n",
		publicPersistentOverrideFile: "services:\n  frontend:\n    image: ghcr.io/yyhuni/lunafox-frontend@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa\n",
	} {
		if err := os.WriteFile(filepath.Join(root, relative), []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	store, err := NewPublicJournalStore(root)
	if err != nil {
		t.Fatal(err)
	}
	manifestBytes := releaseManifestFixtureBytes(t)
	manifestPath := filepath.Join(root, "candidate.yaml")
	if err := os.WriteFile(manifestPath, manifestBytes, 0o600); err != nil {
		t.Fatal(err)
	}
	manifest, err := loadManifestForTest(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(mustManifestPath(t, store, manifest.Digest()), manifestBytes, 0o600); err != nil {
		t.Fatal(err)
	}
	runner := &frontendOnlyRunner{}
	executor, err := NewPublicComposeExecutor(runner, "ghcr.io")
	if err != nil {
		t.Fatal(err)
	}
	request := Request{
		SchemaVersion:              ScopedRequestSchema,
		OperationID:                "frontend-only-stopping",
		Action:                     ActionStart,
		ManifestDigest:             manifest.Digest(),
		ExecutionMode:              ExecutionModeFrontendOnly,
		PlanDigest:                 "sha256:" + strings.Repeat("1", 64),
		BaselineStateDigest:        "sha256:" + strings.Repeat("2", 64),
		TouchedServices:            []string{FrontendOnlyService},
		ConfirmedDeploymentVersion: "1.2.2",
	}
	now := time.Now().UTC()
	if err := store.Save(Journal{
		SchemaVersion:              ScopedJournalSchema,
		OperationID:                request.OperationID,
		ManifestDigest:             request.ManifestDigest,
		Stage:                      StageStopping,
		ExecutionMode:              request.ExecutionMode,
		PlanDigest:                 request.PlanDigest,
		BaselineStateDigest:        request.BaselineStateDigest,
		TouchedServices:            request.TouchedServices,
		ConfirmedDeploymentVersion: request.ConfirmedDeploymentVersion,
		StartedAt:                  now,
		UpdatedAt:                  now,
	}); err != nil {
		t.Fatal(err)
	}
	if err := executor.Execute(context.Background(), request, store); err != nil {
		t.Fatal(err)
	}
	current, err := store.LoadCurrent()
	if err != nil {
		t.Fatal(err)
	}
	if current.Stage != StageFailed || !strings.Contains(current.Diagnostic, "stopping stage") {
		t.Fatalf("stopping frontend-only journal = %#v, want fail-closed diagnostic", current)
	}
	for _, command := range runner.commands {
		joined := strings.Join(command.args, " ")
		if strings.Contains(joined, "pull frontend") || strings.Contains(joined, "up -d") {
			t.Fatalf("stopping frontend-only stage executed Compose mutation: %q", joined)
		}
	}
}

func TestDockerFrontendEdgeProbeUsesCandidateNetworkAndFixedNginxRequest(t *testing.T) {
	runner := &frontendOnlyRunner{}
	probe := NewDockerFrontendEdgeProbe(runner, "docker")
	observation, err := probe.Probe(context.Background(), "/deployment", "abcdef0123456789", "example.test", "frontend-edge-probe")
	if err != nil {
		t.Fatal(err)
	}
	if observation.StatusCode != 200 || observation.ContentType != "text/html" {
		t.Fatalf("edge observation = %#v", observation)
	}
	if len(runner.commands) != 1 {
		t.Fatalf("probe command count = %d, want 1", len(runner.commands))
	}
	command := runner.commands[0]
	if command.binary != "docker" || command.dir != "/deployment" {
		t.Fatalf("probe command = %#v", command)
	}
	joined := strings.Join(command.args, " ")
	for _, required := range []string{"exec abcdef0123456789 node -e", "frontend-edge-probe", "example.test", "nginx"} {
		if !strings.Contains(joined, required) {
			t.Fatalf("probe command missing %q: %s", required, joined)
		}
	}
	if strings.Contains(joined, "sh -c") {
		t.Fatalf("probe command invokes a shell: %s", joined)
	}
}

func TestComposeExecutorFrontendOnlyFailsClosedWhenPublicEdgeDoesNotConverge(t *testing.T) {
	root := newShortRoot(t)
	confirmed := "services:\n  frontend:\n    image: ghcr.io/yyhuni/lunafox-frontend@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa\n"
	for relative, content := range map[string]string{
		publicComposeFile:            "services: {}\n",
		publicEnvFile:                "PUBLIC_HOST=localhost\nPUBLIC_PORT=443\nPUBLIC_URL=https://localhost\n",
		publicPersistentOverrideFile: confirmed,
	} {
		if err := os.WriteFile(filepath.Join(root, relative), []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	store, err := NewPublicJournalStore(root)
	if err != nil {
		t.Fatal(err)
	}
	manifestBytes := releaseManifestFixtureBytes(t)
	manifestPath := filepath.Join(root, "candidate.yaml")
	if err := os.WriteFile(manifestPath, manifestBytes, 0o600); err != nil {
		t.Fatal(err)
	}
	manifest, err := loadManifestForTest(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(mustManifestPath(t, store, manifest.Digest()), manifestBytes, 0o600); err != nil {
		t.Fatal(err)
	}
	request := Request{
		SchemaVersion:              ScopedRequestSchema,
		OperationID:                "frontend-only-edge-failure",
		Action:                     ActionStart,
		ManifestDigest:             manifest.Digest(),
		ExecutionMode:              ExecutionModeFrontendOnly,
		PlanDigest:                 "sha256:" + strings.Repeat("1", 64),
		BaselineStateDigest:        "sha256:" + strings.Repeat("2", 64),
		TouchedServices:            []string{FrontendOnlyService},
		ConfirmedDeploymentVersion: "1.2.2",
	}
	now := time.Now().UTC()
	if err := store.Save(Journal{
		SchemaVersion:              ScopedJournalSchema,
		OperationID:                request.OperationID,
		ManifestDigest:             request.ManifestDigest,
		Stage:                      StageQueued,
		ExecutionMode:              request.ExecutionMode,
		PlanDigest:                 request.PlanDigest,
		BaselineStateDigest:        request.BaselineStateDigest,
		TouchedServices:            request.TouchedServices,
		ConfirmedDeploymentVersion: request.ConfirmedDeploymentVersion,
		StartedAt:                  now,
		UpdatedAt:                  now,
	}); err != nil {
		t.Fatal(err)
	}
	runner := &frontendOnlyRunner{}
	executor, err := NewPublicComposeExecutor(runner, "ghcr.io")
	if err != nil {
		t.Fatal(err)
	}
	probe := &frontendEdgeProbeStub{errors: []error{errors.New("resolver still points at the old frontend"), errors.New("edge unavailable")}}
	executor.PublicFrontendProbe = probe
	executor.FrontendEdgeProbeAttempts = 2
	executor.FrontendEdgeProbeDelay = 0
	if err := executor.Execute(context.Background(), request, store); err != nil {
		t.Fatal(err)
	}
	if probe.calls != 2 {
		t.Fatalf("edge probe attempts = %d, want 2", probe.calls)
	}
	current, err := store.LoadCurrent()
	if err != nil {
		t.Fatal(err)
	}
	if current.Stage != StageFailed {
		t.Fatalf("stage after edge failure = %q, want failed", current.Stage)
	}
	if _, err := store.LoadReceipt(request.OperationID); !errors.Is(err, ErrJournalNotFound) {
		t.Fatalf("receipt after edge failure = %v, want absent", err)
	}
	contents, err := os.ReadFile(filepath.Join(root, publicPersistentOverrideFile))
	if err != nil {
		t.Fatal(err)
	}
	if string(contents) != confirmed {
		t.Fatalf("persistent override changed after edge failure: %s", contents)
	}
}

func TestLoadPublicEdgeConfigRejectsNonCanonicalPublicURL(t *testing.T) {
	root := newShortRoot(t)
	if err := os.WriteFile(filepath.Join(root, publicEnvFile), []byte("PUBLIC_HOST=example.test\nPUBLIC_PORT=8443\nPUBLIC_URL=http://example.test:8443\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := loadPublicEdgeConfig(root); err == nil {
		t.Fatal("non-HTTPS public URL was accepted")
	}
}

func TestComposeReceiptDoesNotAdvanceJournalToSucceeded(t *testing.T) {
	root := newComposeRoot(t)
	store, err := NewJournalStore(root)
	if err != nil {
		t.Fatal(err)
	}
	manifest, err := loadManifestForTest(filepath.Join(root, defaultManifestName))
	if err != nil {
		t.Fatal(err)
	}
	request := Request{SchemaVersion: RequestSchema, OperationID: "op-receipt-boundary", Action: ActionStart, ManifestDigest: manifest.Digest()}
	now := time.Now().UTC()
	if err := store.Save(Journal{SchemaVersion: JournalSchema, OperationID: request.OperationID, ManifestDigest: request.ManifestDigest, Stage: StageQueued, StartedAt: now, UpdatedAt: now}); err != nil {
		t.Fatal(err)
	}
	if err := NewComposeExecutor(&recordingRunner{}).Execute(context.Background(), request, store); err != nil {
		t.Fatal(err)
	}
	current, err := store.LoadCurrent()
	if err != nil {
		t.Fatal(err)
	}
	if current.Stage != StageVerifying {
		t.Fatalf("journal stage = %q, want verifying until Server/Agent evidence", current.Stage)
	}
	if IsTerminal(current.Stage) {
		t.Fatalf("receipt incorrectly made journal terminal: %q", current.Stage)
	}
	receipt, err := store.LoadReceipt(request.OperationID)
	if err != nil {
		t.Fatal(err)
	}
	if receipt.OperationID != request.OperationID || receipt.ManifestDigest != request.ManifestDigest {
		t.Fatalf("receipt identity = %#v, want operation/digest binding", receipt)
	}
}

func TestComposeExecutorNormalizesNilContext(t *testing.T) {
	root := newComposeRoot(t)
	store, err := NewJournalStore(root)
	if err != nil {
		t.Fatal(err)
	}
	manifest, err := loadManifestForTest(filepath.Join(root, defaultManifestName))
	if err != nil {
		t.Fatal(err)
	}
	request := Request{SchemaVersion: RequestSchema, OperationID: "op-nil-context", Action: ActionStart, ManifestDigest: manifest.Digest()}
	now := time.Now().UTC()
	if err := store.Save(Journal{SchemaVersion: JournalSchema, OperationID: request.OperationID, ManifestDigest: request.ManifestDigest, Stage: StageQueued, StartedAt: now, UpdatedAt: now}); err != nil {
		t.Fatal(err)
	}
	delegate := &recordingRunner{}
	runner := &nonNilContextRunner{delegate: delegate}
	var nilContext context.Context
	if err := NewComposeExecutor(runner).Execute(nilContext, request, store); err != nil {
		t.Fatalf("Execute(nil) = %v", err)
	}
	if runner.sawNil {
		t.Fatal("ComposeExecutor passed a nil context to the command runner")
	}
}

func TestComposeExecutorMigrationFailureIsNeedsRecovery(t *testing.T) {
	root := newComposeRoot(t)
	manifestPath := filepath.Join(root, defaultManifestName)
	raw := releaseManifestFixtureBytes(t)
	raw = []byte(strings.Replace(string(raw), "hasDatabaseMigration: false", "hasDatabaseMigration: true", 1))
	raw = []byte(strings.Replace(string(raw), "migrationType: \"none\"", "migrationType: \"compatible\"", 1))
	// The parser only accepts checksummed migration metadata; use a valid
	// fixture by replacing all fields in the existing manifest at once.
	raw = []byte(strings.Replace(string(raw), `migrationId: ""
    checksum: ""`, `migrationId: 000002_upgrade
    checksum: sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb`, 1))
	if err := os.WriteFile(manifestPath, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	manifest, err := loadManifestForTest(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	store, err := NewJournalStore(root)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	request := Request{SchemaVersion: RequestSchema, OperationID: "op-migration", Action: ActionStart, ManifestDigest: manifest.Digest()}
	if err := store.Save(Journal{SchemaVersion: JournalSchema, OperationID: request.OperationID, ManifestDigest: request.ManifestDigest, Stage: StageQueued, StartedAt: now, UpdatedAt: now}); err != nil {
		t.Fatal(err)
	}
	runner := &recordingRunner{errors: map[string]error{"run --rm": errors.New("migration failed")}}
	// Once migration has started, the journal's needs_recovery checkpoint is
	// the durable outcome. Execute returns nil so the daemon cannot overwrite
	// that terminal safety state with a generic request error.
	if err := NewComposeExecutor(runner).Execute(context.Background(), request, store); err != nil {
		t.Fatalf("Execute() = %v, want nil after durable needs_recovery checkpoint", err)
	}
	var migrationArgs []string
	for _, command := range runner.commands {
		if strings.Contains(strings.Join(command.args, " "), "run --rm") {
			migrationArgs = command.args
			break
		}
	}
	if strings.Join(migrationArgs, " ") == "" || !strings.HasSuffix(strings.Join(migrationArgs, " "), "server /usr/local/bin/server migrate up") {
		t.Fatalf("migration command = %#v, want fixed server migrate up entry point", migrationArgs)
	}
	current, err := store.LoadCurrent()
	if err != nil {
		t.Fatal(err)
	}
	if current.Stage != StageNeedsRecovery {
		t.Fatalf("current stage = %q, want needs_recovery after migration began", current.Stage)
	}
	if current.MigrationStatus != MigrationStatusFailed {
		t.Fatalf("migration status = %q, want failed for a definite non-zero exit", current.MigrationStatus)
	}
}

func TestComposeExecutorRetriesSameManifestAfterPreMigrationFailure(t *testing.T) {
	root := newComposeRoot(t)
	manifest := writeMigrationManifestFixture(t, root)
	store, err := NewJournalStore(root)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	request := Request{SchemaVersion: RequestSchema, OperationID: "op-pre-migration-retry", Action: ActionStart, ManifestDigest: manifest.Digest()}
	if err := store.Save(Journal{SchemaVersion: JournalSchema, OperationID: request.OperationID, ManifestDigest: request.ManifestDigest, Stage: StageQueued, StartedAt: now, UpdatedAt: now}); err != nil {
		t.Fatal(err)
	}
	// A pull failure happens before the migration checkpoint. It is therefore
	// safe to reopen the same operation at updating and retry the exact target.
	first := &recordingRunner{errors: map[string]error{"pull": errors.New("registry unavailable")}}
	if err := NewComposeExecutor(first).Execute(context.Background(), request, store); err != nil {
		t.Fatalf("first Execute() = %v, want durable failed checkpoint", err)
	}
	failed, err := store.LoadCurrent()
	if err != nil {
		t.Fatal(err)
	}
	if failed.Stage != StageFailed || failed.RepairStage != StageUpdating {
		t.Fatalf("pre-migration failure journal = %#v, want failed/updating", failed)
	}
	if failed.MigrationStatus != MigrationStatusNotStarted {
		t.Fatalf("migration status after pre-migration failure = %q, want not_started", failed.MigrationStatus)
	}
	repaired, err := store.ResetForRepair(request.OperationID, request.ManifestDigest)
	if err != nil {
		t.Fatal(err)
	}
	if repaired.Stage != StageUpdating {
		t.Fatalf("repair stage = %q, want updating", repaired.Stage)
	}
	second := &recordingRunner{}
	if err := NewComposeExecutor(second).Execute(context.Background(), request, store); err != nil {
		t.Fatal(err)
	}
	completed, err := store.LoadCurrent()
	if err != nil {
		t.Fatal(err)
	}
	if completed.Stage != StageVerifying || completed.OperationID != request.OperationID || completed.ManifestDigest != request.ManifestDigest {
		t.Fatalf("retried journal = %#v, want verifying with the same target", completed)
	}
	if completed.MigrationStatus != MigrationStatusSucceeded {
		t.Fatalf("retried migration status = %q, want succeeded", completed.MigrationStatus)
	}
	foundMigration := false
	for _, command := range second.commands {
		if strings.Contains(strings.Join(command.args, " "), "server /usr/local/bin/server migrate up") {
			foundMigration = true
		}
	}
	if !foundMigration {
		t.Fatalf("retry commands = %#v, want the fixed migration entrypoint", second.commands)
	}
	if _, err := store.LoadReceipt(request.OperationID); err != nil {
		t.Fatalf("retry receipt = %v", err)
	}
}

func TestComposeExecutorInterruptedMigrationIsUnknownAndNeedsRecovery(t *testing.T) {
	root := newComposeRoot(t)
	manifest := writeMigrationManifestFixture(t, root)
	store, err := NewJournalStore(root)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	request := Request{SchemaVersion: RequestSchema, OperationID: "op-interrupted-migration", Action: ActionStart, ManifestDigest: manifest.Digest()}
	if err := store.Save(Journal{SchemaVersion: JournalSchema, OperationID: request.OperationID, ManifestDigest: request.ManifestDigest, Stage: StageQueued, StartedAt: now, UpdatedAt: now}); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	runner := &cancelMigrationRunner{cancel: cancel}
	if err := NewComposeExecutor(runner).Execute(ctx, request, store); err != nil {
		t.Fatalf("Execute() = %v, want durable recovery checkpoint", err)
	}
	current, err := store.LoadCurrent()
	if err != nil {
		t.Fatal(err)
	}
	if current.Stage != StageNeedsRecovery || current.MigrationStatus != MigrationStatusUnknown {
		t.Fatalf("interrupted migration journal = %#v, want needs_recovery/unknown", current)
	}
	for _, command := range runner.commands {
		if strings.Contains(strings.Join(command.args, " "), "migrate down") {
			t.Fatalf("interrupted migration attempted a down migration: %#v", command)
		}
	}
}

func TestComposeExecutorSuccessfulMigrationHealthFailureHasNoAutomaticRollback(t *testing.T) {
	root := newComposeRoot(t)
	manifest := writeMigrationManifestFixture(t, root)
	store, err := NewJournalStore(root)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	request := Request{SchemaVersion: RequestSchema, OperationID: "op-migration-health-failure", Action: ActionStart, ManifestDigest: manifest.Digest()}
	if err := store.Save(Journal{SchemaVersion: JournalSchema, OperationID: request.OperationID, ManifestDigest: request.ManifestDigest, Stage: StageQueued, StartedAt: now, UpdatedAt: now}); err != nil {
		t.Fatal(err)
	}
	runner := &recordingRunner{responses: map[string]RunResult{
		"ps --format": {Stdout: `[{"Service":"server","State":"running"},{"Service":"frontend","State":"exited","Health":"unhealthy"},{"Service":"nginx","State":"running"}]`},
	}}
	if err := NewComposeExecutor(runner).Execute(context.Background(), request, store); err != nil {
		t.Fatalf("Execute() = %v, want durable recovery checkpoint", err)
	}
	current, err := store.LoadCurrent()
	if err != nil {
		t.Fatal(err)
	}
	if current.Stage != StageNeedsRecovery || current.MigrationStatus != MigrationStatusSucceeded {
		t.Fatalf("post-migration health journal = %#v, want needs_recovery/succeeded", current)
	}
	for _, command := range runner.commands {
		joined := strings.Join(command.args, " ")
		if strings.Contains(joined, "migrate down") || strings.Contains(joined, "rollback") {
			t.Fatalf("post-migration health failure attempted an automatic rollback: %#v", command)
		}
	}
	if _, err := store.LoadReceipt(request.OperationID); !errors.Is(err, ErrJournalNotFound) {
		t.Fatalf("receipt after unhealthy verification = %v, want no receipt", err)
	}
}

func TestComposeExecutorNoMigrationHealthFailureRemainsRetryableWithoutRollback(t *testing.T) {
	root := newComposeRoot(t)
	store, err := NewJournalStore(root)
	if err != nil {
		t.Fatal(err)
	}
	manifest, err := loadManifestForTest(filepath.Join(root, defaultManifestName))
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	request := Request{SchemaVersion: RequestSchema, OperationID: "op-code-health-failure", Action: ActionStart, ManifestDigest: manifest.Digest()}
	if err := store.Save(Journal{SchemaVersion: JournalSchema, OperationID: request.OperationID, ManifestDigest: request.ManifestDigest, Stage: StageQueued, StartedAt: now, UpdatedAt: now}); err != nil {
		t.Fatal(err)
	}
	runner := &recordingRunner{responses: map[string]RunResult{
		"ps --format": {Stdout: `[{"Service":"server","State":"running"},{"Service":"frontend","State":"exited","Health":"unhealthy"},{"Service":"nginx","State":"running"}]`},
	}}
	if err := NewComposeExecutor(runner).Execute(context.Background(), request, store); err != nil {
		t.Fatalf("Execute() = %v, want durable failed checkpoint", err)
	}
	current, err := store.LoadCurrent()
	if err != nil {
		t.Fatal(err)
	}
	if current.Stage != StageFailed {
		t.Fatalf("no-migration health journal = %#v, want failed", current)
	}
	for _, command := range runner.commands {
		joined := strings.Join(command.args, " ")
		if strings.Contains(joined, "migrate down") || strings.Contains(joined, "rollback") {
			t.Fatalf("no-migration health failure attempted an implicit rollback: %#v", command)
		}
	}
}

func TestComposeExecutorMigrationRestartWithDurableSuccessSkipsMigration(t *testing.T) {
	root := newComposeRoot(t)
	manifestPath := filepath.Join(root, defaultManifestName)
	raw, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	raw = []byte(strings.Replace(string(raw), "hasDatabaseMigration: false", "hasDatabaseMigration: true", 1))
	raw = []byte(strings.Replace(string(raw), "migrationType: \"none\"", "migrationType: \"compatible\"", 1))
	raw = []byte(strings.Replace(string(raw), `migrationId: ""
    checksum: ""`, `migrationId: 000002_upgrade
    checksum: sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb`, 1))
	if err := os.WriteFile(manifestPath, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	manifest, err := loadManifestForTest(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	store, err := NewJournalStore(root)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	request := Request{SchemaVersion: RequestSchema, OperationID: "op-migration-success", Action: ActionResume, ManifestDigest: manifest.Digest()}
	if err := store.Save(Journal{
		SchemaVersion: JournalSchema, OperationID: request.OperationID, ManifestDigest: request.ManifestDigest,
		Stage: StageMigrating, MigrationID: manifest.Upgrade.DatabaseMigration.MigrationID,
		MigrationChecksum: manifest.Upgrade.DatabaseMigration.Checksum, MigrationStatus: MigrationStatusSucceeded,
		StartedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatal(err)
	}
	runner := &recordingRunner{}
	if err := NewComposeExecutor(runner).Execute(context.Background(), request, store); err != nil {
		t.Fatal(err)
	}
	for _, command := range runner.commands {
		if strings.Contains(strings.Join(command.args, " "), "migrate up") {
			t.Fatalf("durably successful migration was rerun: %#v", command)
		}
	}
	current, err := store.LoadCurrent()
	if err != nil {
		t.Fatal(err)
	}
	if current.Stage != StageVerifying || current.MigrationStatus != MigrationStatusSucceeded {
		t.Fatalf("recovered journal = %#v, want verifying/succeeded", current)
	}
}

func TestComposeExecutorDoesNotRerunMigrationAfterRestart(t *testing.T) {
	root := newComposeRoot(t)
	store, err := NewJournalStore(root)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	request := Request{SchemaVersion: RequestSchema, OperationID: "op-migrating-restart", Action: ActionResume, ManifestDigest: testManifestDigest}
	if err := store.Save(Journal{SchemaVersion: JournalSchema, OperationID: request.OperationID, ManifestDigest: request.ManifestDigest, Stage: StageMigrating, StartedAt: now, UpdatedAt: now}); err != nil {
		t.Fatal(err)
	}
	runner := &recordingRunner{}
	if err := NewComposeExecutor(runner).Execute(context.Background(), request, store); err != nil {
		t.Fatal(err)
	}
	if len(runner.commands) != 0 {
		t.Fatalf("migration restart ran Docker commands: %#v", runner.commands)
	}
	current, err := store.LoadCurrent()
	if err != nil {
		t.Fatal(err)
	}
	if current.Stage != StageNeedsRecovery {
		t.Fatalf("current stage = %q, want needs_recovery", current.Stage)
	}
}

func TestComposeExecutorResumesRestartingWithVerificationOnly(t *testing.T) {
	root := newComposeRoot(t)
	store, err := NewJournalStore(root)
	if err != nil {
		t.Fatal(err)
	}
	manifest, err := loadManifestForTest(filepath.Join(root, defaultManifestName))
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	request := Request{SchemaVersion: RequestSchema, OperationID: "op-restarting", Action: ActionResume, ManifestDigest: manifest.Digest()}
	if err := store.Save(Journal{SchemaVersion: JournalSchema, OperationID: request.OperationID, ManifestDigest: request.ManifestDigest, Stage: StageRestarting, StartedAt: now, UpdatedAt: now}); err != nil {
		t.Fatal(err)
	}
	runner := &recordingRunner{}
	if err := NewComposeExecutor(runner).Execute(context.Background(), request, store); err != nil {
		t.Fatal(err)
	}
	if len(runner.commands) != 4 {
		t.Fatalf("verification-only command count = %d, want 4", len(runner.commands))
	}
	for _, command := range runner.commands {
		joined := strings.Join(command.args, " ")
		if strings.Contains(joined, " pull ") || strings.Contains(joined, " up ") || strings.Contains(joined, " run ") || strings.Contains(joined, " config ") {
			t.Fatalf("resume unexpectedly reran mutation command: %#v", command)
		}
	}
	current, err := store.LoadCurrent()
	if err != nil {
		t.Fatal(err)
	}
	if current.Stage != StageVerifying {
		t.Fatalf("current stage = %q, want verifying", current.Stage)
	}
}

func TestReceiptRequiresObservedDigestForEveryService(t *testing.T) {
	receipt := Receipt{
		SchemaVersion:  JournalSchema,
		OperationID:    "op-receipt",
		ManifestDigest: testManifestDigest,
		CompletedAt:    time.Now().UTC(),
		Services:       []string{"server", "frontend"},
		ObservedImages: map[string]string{"server": testManifestDigest},
	}
	if err := receipt.Validate(); err == nil || !strings.Contains(err.Error(), "missing observed image") {
		t.Fatalf("Receipt.Validate() = %v, want missing observed image rejection", err)
	}
}

func TestHealthyComposeOutputRequiresEveryService(t *testing.T) {
	services := []string{"server", "frontend", "nginx"}
	if healthyComposeOutput(`[{"Service":"server","State":"running"},{"Service":"frontend","State":"healthy"}]`, services) {
		t.Fatal("missing service was accepted")
	}
	if !healthyComposeOutput(`[{"Service":"server","State":"running"},{"Service":"frontend","State":"healthy"},{"Service":"nginx","State":"running"}]`, services) {
		t.Fatal("healthy services were rejected")
	}
	if healthyComposeOutput(`[{"Service":"server","State":"running"},{"Service":"frontend","State":"exited","Health":"unhealthy"},{"Service":"nginx","State":"running"}]`, services) {
		t.Fatal("unhealthy service was accepted")
	}
}

func TestComposeExecutorNeverReplacesAgentContainer(t *testing.T) {
	root := newComposeRoot(t)
	store, err := NewJournalStore(root)
	if err != nil {
		t.Fatal(err)
	}
	manifest, err := loadManifestForTest(filepath.Join(root, defaultManifestName))
	if err != nil {
		t.Fatal(err)
	}
	request := Request{SchemaVersion: RequestSchema, OperationID: "op-no-agent-replace", Action: ActionStart, ManifestDigest: manifest.Digest()}
	if err := store.Save(Journal{SchemaVersion: JournalSchema, OperationID: request.OperationID, ManifestDigest: request.ManifestDigest, Stage: StageQueued, StartedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()}); err != nil {
		t.Fatal(err)
	}
	runner := &recordingRunner{}
	if err := NewComposeExecutor(runner).Execute(context.Background(), request, store); err != nil {
		t.Fatal(err)
	}
	for _, command := range runner.commands {
		joined := strings.Join(command.args, " ")
		if strings.Contains(joined, " agent") || strings.Contains(joined, "agent@") {
			t.Fatalf("host executor attempted Agent replacement: %#v", command)
		}
	}
}

func loadManifestForTest(path string) (*releasemanifest.Manifest, error) {
	return releasemanifest.Load(path)
}

func releaseManifestFixtureBytes(t *testing.T) []byte {
	t.Helper()
	// The public projection omits the repository-root release manifest, so
	// upgrader tests must remain inside the exported upgrade module closure.
	raw, err := os.ReadFile(filepath.Join("..", "testdata", "release.manifest.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func legacyAlpha114ManifestFixturePath(t *testing.T) string {
	t.Helper()
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller() failed")
	}
	return filepath.Join(filepath.Dir(filename), "..", "..", "..", "..", "..", "scripts", "ci", "fixtures", "legacy-alpha114-release.manifest.yaml")
}

func writeMigrationManifestFixture(t *testing.T, root string) *releasemanifest.Manifest {
	t.Helper()
	manifestPath := filepath.Join(root, defaultManifestName)
	raw, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	raw = []byte(strings.Replace(string(raw), "hasDatabaseMigration: false", "hasDatabaseMigration: true", 1))
	raw = []byte(strings.Replace(string(raw), "migrationType: \"none\"", "migrationType: \"compatible\"", 1))
	raw = []byte(strings.Replace(string(raw), `migrationId: ""
    checksum: ""`, `migrationId: 000002_upgrade
    checksum: sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb`, 1))
	if err := os.WriteFile(manifestPath, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	manifest, err := loadManifestForTest(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	return manifest
}

type cancelMigrationRunner struct {
	commands []recordedCommand
	cancel   context.CancelFunc
}

type nonNilContextRunner struct {
	delegate *recordingRunner
	sawNil   bool
}

func (runner *nonNilContextRunner) Run(ctx context.Context, binary string, args []string, dir string) (RunResult, error) {
	if ctx == nil {
		runner.sawNil = true
		return RunResult{}, errors.New("runner received nil context")
	}
	return runner.delegate.Run(ctx, binary, args, dir)
}

func (runner *cancelMigrationRunner) Run(ctx context.Context, binary string, args []string, dir string) (RunResult, error) {
	runner.commands = append(runner.commands, recordedCommand{binary: binary, args: append([]string(nil), args...), dir: dir})
	joined := strings.Join(args, " ")
	if strings.Contains(joined, "run --rm") {
		runner.cancel()
		return RunResult{}, context.Canceled
	}
	if strings.Contains(joined, "ps --format") {
		return RunResult{Stdout: `[{"Service":"server","State":"running"},{"Service":"frontend","State":"healthy"},{"Service":"nginx","State":"running"}]`}, nil
	}
	if strings.Contains(joined, "image inspect") {
		return RunResult{Stdout: "docker.io/yyhuni/lunafox@" + testManifestDigest}, nil
	}
	return RunResult{}, nil
}

func newComposeRoot(t *testing.T) string {
	t.Helper()
	root := newShortRoot(t)
	if err := os.MkdirAll(filepath.Join(root, "docker"), 0o700); err != nil {
		t.Fatal(err)
	}
	for relative, content := range map[string]string{
		defaultComposeFile: "services: {}\n",
		defaultEnvFile:     "PUBLIC_PORT=443\n",
	} {
		path := filepath.Join(root, relative)
		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	raw := releaseManifestFixtureBytes(t)
	if err := os.WriteFile(filepath.Join(root, defaultManifestName), raw, 0o600); err != nil {
		t.Fatal(err)
	}
	return root
}

func newShortRoot(t *testing.T) string {
	t.Helper()
	root, err := os.MkdirTemp("/tmp", "lfu-c-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(root) })
	return root
}
