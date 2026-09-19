package upgrader

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/yyhuni/lunafox/contracts/releasemanifest"
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
