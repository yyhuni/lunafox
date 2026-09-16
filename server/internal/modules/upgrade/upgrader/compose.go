package upgrader

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/yyhuni/lunafox/contracts/releasemanifest"
)

const (
	defaultDockerBinary          = "docker"
	defaultComposeFile           = "docker/docker-compose.yml"
	defaultEnvFile               = "docker/.env"
	publicComposeFile            = "compose.yaml"
	publicEnvFile                = ".env"
	publicPersistentOverrideFile = "compose.override.yaml"
	publicProjectName            = "lunafox"
)

var observedDigestPattern = regexp.MustCompile(`sha256:[0-9a-f]{64}`)

type RunResult struct {
	Stdout string
	Stderr string
}

// CommandRunner receives a binary and argv separately. Implementations must
// use exec.CommandContext (or an equivalent direct process API); no shell
// parser is involved in the privileged upgrade path.
type CommandRunner interface {
	Run(context.Context, string, []string, string) (RunResult, error)
}

// OSCommandRunner executes a fixed binary with a separately supplied argv.
// It intentionally does not invoke a shell; callers cannot turn Compose
// arguments into a shell script through this runner.
type OSCommandRunner struct{}

func NewOSCommandRunner() *OSCommandRunner {
	return &OSCommandRunner{}
}

func (runner *OSCommandRunner) Run(ctx context.Context, binary string, args []string, dir string) (RunResult, error) {
	if runner == nil {
		return RunResult{}, fmt.Errorf("os command runner is not configured")
	}
	if strings.TrimSpace(binary) == "" {
		return RunResult{}, fmt.Errorf("command binary is required")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	command := exec.CommandContext(ctx, binary, args...)
	command.Dir = dir
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr
	err := command.Run()
	result := RunResult{Stdout: stdout.String(), Stderr: stderr.String()}
	if err != nil {
		return result, err
	}
	return result, nil
}

// ComposeExecutor implements the host-side Compose part of the upgrade. It
// never accepts Compose/image/path values from Request; all paths and image
// references come from its fixed deployment root and validated manifest.
type ComposeExecutor struct {
	Runner       CommandRunner
	DockerBinary string
	PublicLayout bool
	Registry     string
}

func NewComposeExecutor(runner CommandRunner) *ComposeExecutor {
	return &ComposeExecutor{Runner: runner, DockerBinary: defaultDockerBinary}
}

// NewPublicComposeExecutor creates the fixed executor used by the released
// Compose package. Registry is deployment-owned and never comes from a browser
// request, so a manifest cannot silently mix Docker Hub and GHCR references.
func NewPublicComposeExecutor(runner CommandRunner, registry string) (*ComposeExecutor, error) {
	registry = strings.TrimSpace(registry)
	if registry != "docker.io" && registry != "ghcr.io" {
		return nil, fmt.Errorf("unsupported upgrade registry %q", registry)
	}
	return &ComposeExecutor{Runner: runner, DockerBinary: defaultDockerBinary, PublicLayout: true, Registry: registry}, nil
}

func (executor *ComposeExecutor) Execute(ctx context.Context, request Request, store *JournalStore) error {
	if executor == nil || executor.Runner == nil {
		return fmt.Errorf("compose executor is not configured")
	}
	if store == nil {
		return fmt.Errorf("journal store is not configured")
	}
	// Interfaces may pass a nil context when resuming from a persisted journal.
	// Normalize it before handing control to a runner so every command receives
	// a usable cancellation/timeout boundary.
	if ctx == nil {
		ctx = context.Background()
	}
	if err := request.Validate(); err != nil {
		return err
	}
	current, err := store.LoadCurrent()
	if errors.Is(err, ErrJournalNotFound) {
		now := nowUTC()
		current = Journal{SchemaVersion: JournalSchema, OperationID: request.OperationID, ManifestDigest: request.ManifestDigest, Stage: StageQueued, StartedAt: now, UpdatedAt: now}
	} else if err != nil {
		return err
	}
	if current.OperationID != request.OperationID || current.ManifestDigest != request.ManifestDigest {
		return ErrReplayDigestMismatch
	}
	if IsTerminal(current.Stage) {
		return nil
	}
	stage := current.Stage
	manifestPath, err := store.ManifestPath(request.ManifestDigest)
	if err != nil {
		return executor.failCheckpoint(store, request, failureStageFor(migrationEvidenceForJournal(current)), "manifest path resolution failed")
	}
	manifest, err := releasemanifest.Load(manifestPath)
	if err != nil {
		return executor.failCheckpoint(store, request, failureStageFor(migrationEvidenceForJournal(current)), "manifest validation failed")
	}
	if manifest.Digest() != request.ManifestDigest {
		return executor.failCheckpoint(store, request, failureStageFor(migrationEvidenceForJournal(current)), "manifest digest mismatch")
	}
	migration := manifest.Upgrade.DatabaseMigration
	// Bind migration identity to the same immutable manifest used for image
	// selection. This evidence survives a Server restart and lets the control
	// plane distinguish a completed migration from an uncertain one.
	migrationStatus := current.MigrationStatus
	if migrationStatus == "" {
		migrationStatus = MigrationStatusNotStarted
	}
	// A journal is the durable evidence for a migration that may already have
	// touched the database. Never replace an existing identity with metadata
	// from a newly read manifest, even when an operator (or a damaged handoff)
	// reuses the operation id. The digest check above normally makes this
	// impossible; keeping the explicit fence protects the recovery boundary if
	// either artifact was edited independently.
	if (current.MigrationID != "" && current.MigrationID != migration.MigrationID) ||
		(current.MigrationChecksum != "" && current.MigrationChecksum != migration.Checksum) {
		return executor.failCheckpoint(store, request, failureStageFor(migrationEvidenceForJournal(current)), "migration identity mismatch")
	}
	if current.MigrationID != migration.MigrationID || current.MigrationChecksum != migration.Checksum || current.MigrationStatus != migrationStatus {
		current.MigrationID = migration.MigrationID
		current.MigrationChecksum = migration.Checksum
		current.MigrationStatus = migrationStatus
		current.UpdatedAt = nowUTC()
		if err := store.Save(current); err != nil {
			return err
		}
	}
	// A restart at the migration checkpoint is recoverable only when the
	// journal proves a successful migration.  A running/not-started command has
	// an unknowable commit outcome; preserve that uncertainty and never rerun
	// the migration automatically.  If success was durably recorded before a
	// crash, continue with service verification instead.
	if stage == StageMigrating {
		if !migration.HasDatabaseMigration {
			return executor.failCheckpoint(store, request, StageNeedsRecovery, "migration checkpoint conflicts with manifest")
		}
		switch migrationStatus {
		case MigrationStatusSucceeded:
			if _, err := store.Checkpoint(request.OperationID, request.ManifestDigest, StageRestarting, "", nil); err != nil {
				return err
			}
			stage = StageRestarting
		case MigrationStatusRunning, MigrationStatusNotStarted:
			if _, setErr := store.SetMigration(request.OperationID, request.ManifestDigest, migration.MigrationID, migration.Checksum, MigrationStatusUnknown); setErr != nil {
				return setErr
			}
			return executor.failCheckpoint(store, request, StageNeedsRecovery, "database migration outcome is unknown after restart")
		case MigrationStatusFailed, MigrationStatusUnknown:
			return executor.failCheckpoint(store, request, StageNeedsRecovery, "database migration requires recovery")
		default:
			return executor.failCheckpoint(store, request, StageNeedsRecovery, "database migration status is invalid")
		}
	}
	migrationStarted := migrationStartedAtStage(stage, migration.HasDatabaseMigration)
	migrationNeedsRecovery := migrationStarted && migrationStatus != MigrationStatusSucceeded
	if err := validateHostMigration(manifest); err != nil {
		// The Server normally performs this policy gate. Repeat it before any
		// Compose mutation so a stale or compromised handoff cannot update
		// services with a migration the host process is unable to recover.
		return executor.failCheckpoint(store, request, failureStageFor(migrationStarted), "unsupported database migration")
	}
	if err := executor.validateFixedDeploymentFiles(store.DeploymentRoot()); err != nil {
		return executor.failCheckpoint(store, request, failureStageFor(migrationStarted), "deployment files are not available")
	}
	overridePath, err := store.ComposeOverridePath(request.OperationID)
	if err != nil {
		return err
	}
	// Regenerating the immutable override is safe on resume and repairs a
	// checkpoint that was written just before a power loss.
	if err := executor.writeComposeOverride(overridePath, manifest); err != nil {
		return executor.failCheckpoint(store, request, failureStageFor(migrationStarted), "compose override generation failed")
	}
	base := executor.composeArgv(store.DeploymentRoot(), overridePath)
	services := executor.observedServices()
	// A migration is considered started for every post-migration checkpoint.
	// If the process died while the migration command was in flight, the
	// explicit StageMigrating branch above records recovery instead of retrying
	// an operation whose outcome is unknown.

	switch stage {
	case StageQueued, StageStopping:
		if _, err := store.Checkpoint(request.OperationID, request.ManifestDigest, StagePreflight, "", nil); err != nil {
			return err
		}
		fallthrough
	case StagePreflight:
		if _, err := executor.Runner.Run(ctx, executor.binary(), append(append([]string(nil), base...), "config", "--quiet"), store.DeploymentRoot()); err != nil {
			return executor.failCheckpoint(store, request, StageFailed, "compose preflight failed")
		}
		if _, err := store.Checkpoint(request.OperationID, request.ManifestDigest, StageUpdating, "", nil); err != nil {
			return err
		}
		fallthrough
	case StageUpdating:
		if _, err := executor.Runner.Run(ctx, executor.binary(), append(append([]string(nil), base...), append([]string{"pull"}, executor.pullServices()...)...), store.DeploymentRoot()); err != nil {
			return executor.failCheckpoint(store, request, StageFailed, "compose image pull failed")
		}
		if _, err := executor.Runner.Run(ctx, executor.binary(), append(append([]string(nil), base...), executor.coreUpdateArgs()...), store.DeploymentRoot()); err != nil {
			return executor.failCheckpoint(store, request, StageFailed, "compose service update failed")
		}
		if migration.HasDatabaseMigration {
			if _, err := store.Checkpoint(request.OperationID, request.ManifestDigest, StageMigrating, "", nil); err != nil {
				return err
			}
			migrationStarted = true
			if _, err := store.SetMigration(request.OperationID, request.ManifestDigest, migration.MigrationID, migration.Checksum, MigrationStatusRunning); err != nil {
				return err
			}
			if _, err := executor.Runner.Run(ctx, executor.binary(), append(append([]string(nil), base...), "run", "--rm", "--no-deps", "server", "/usr/local/bin/server", "migrate", "up"), store.DeploymentRoot()); err != nil {
				status := MigrationStatusFailed
				// A cancelled/expired process may have committed before it was
				// interrupted, so its outcome is unknowable. A completed process
				// returning a non-zero exit is a definite migration failure.
				if ctx != nil && ctx.Err() != nil {
					status = MigrationStatusUnknown
				}
				if _, setErr := store.SetMigration(request.OperationID, request.ManifestDigest, migration.MigrationID, migration.Checksum, status); setErr != nil {
					return setErr
				}
				return executor.failCheckpoint(store, request, StageNeedsRecovery, "database migration failed or is unknown")
			}
			if _, err := store.SetMigration(request.OperationID, request.ManifestDigest, migration.MigrationID, migration.Checksum, MigrationStatusSucceeded); err != nil {
				return err
			}
		}
		if _, err := store.Checkpoint(request.OperationID, request.ManifestDigest, StageRestarting, "", nil); err != nil {
			return err
		}
		stage = StageRestarting
	case StageRestarting, StageAgentVerifying, StageVerifying:
		// The host mutation and any migration checkpoint already happened. Only
		// repeat idempotent health and digest observations on recovery.
	default:
		return executor.failCheckpoint(store, request, StageFailed, "unsupported execution stage")
	}
	if executor.PublicLayout && stage == StageRestarting {
		if _, err := executor.Runner.Run(ctx, executor.binary(), append(append([]string(nil), base...), executor.agentUpdateArgs()...), store.DeploymentRoot()); err != nil {
			if migrationStarted {
				return executor.failCheckpoint(store, request, StageNeedsRecovery, "resident Agent update failed after migration")
			}
			return executor.failCheckpoint(store, request, StageFailed, "resident Agent update failed")
		}
	}

	healthResult, err := executor.Runner.Run(ctx, executor.binary(), append(append([]string(nil), base...), append([]string{"ps", "--format", "json"}, services...)...), store.DeploymentRoot())
	if err != nil || !healthyComposeOutput(healthResult.Stdout, services) {
		if migrationStarted {
			return executor.failCheckpoint(store, request, StageNeedsRecovery, "services are unhealthy after migration")
		}
		return executor.failCheckpoint(store, request, StageFailed, "services are unhealthy")
	}
	// The host can establish that Compose is ready, but only the Server can
	// verify Agent reconnect/update_required readiness. Emit that boundary
	// explicitly before the final verification checkpoint.
	if _, err := store.Checkpoint(request.OperationID, request.ManifestDigest, StageAgentVerifying, "", nil); err != nil {
		return err
	}
	if _, err := store.Checkpoint(request.OperationID, request.ManifestDigest, StageVerifying, "", nil); err != nil {
		return err
	}
	observed := make(map[string]string, len(services))
	for _, service := range services {
		// Image inspection belongs to the Docker CLI, not the Compose
		// subcommand. Keep it as a separate fixed argv so a Compose plugin
		// cannot reinterpret the verification request.
		result, inspectErr := executor.Runner.Run(ctx, executor.binary(), []string{"image", "inspect", "--format", "{{index .RepoDigests 0}}", executor.imageRefForService(manifest, service)}, store.DeploymentRoot())
		if inspectErr != nil {
			if migrationStarted {
				return executor.failCheckpoint(store, request, StageNeedsRecovery, "observed image verification failed after migration")
			}
			return executor.failCheckpoint(store, request, StageFailed, "observed image verification failed")
		}
		digest := observedDigest(result.Stdout)
		want, _ := manifest.RuntimeImageDigest(service)
		if digest != want {
			if migrationStarted {
				return executor.failCheckpoint(store, request, StageNeedsRecovery, "observed image digest differs after migration")
			}
			return executor.failCheckpoint(store, request, StageFailed, "observed image digest differs")
		}
		observed[service] = digest
	}
	if executor.PublicLayout {
		if err := executor.writeComposeOverride(filepath.Join(store.DeploymentRoot(), publicPersistentOverrideFile), manifest); err != nil {
			return executor.failCheckpoint(store, request, failureStageFor(migrationStarted), "persistent Compose override installation failed")
		}
	}
	receipt := Receipt{SchemaVersion: JournalSchema, OperationID: request.OperationID, ManifestDigest: request.ManifestDigest, CompletedAt: nowUTC(), Services: services, ObservedImages: observed}
	if err := store.SaveReceipt(receipt); err != nil {
		return err
	}
	if migrationNeedsRecovery {
		// A repair may collect fresh service/digest evidence, but an unknown or
		// failed migration result remains a recovery fence and cannot become
		// succeeded merely because Compose is healthy.
		return executor.failCheckpoint(store, request, StageNeedsRecovery, "database migration result requires recovery")
	}
	// StageVerifying remains the journal state. The receipt proves only that
	// host deployment completed; Server and Agent evidence decide succeeded.
	return nil
}

func validateHostMigration(manifest *releasemanifest.Manifest) error {
	if manifest == nil {
		return fmt.Errorf("release manifest is required")
	}
	migration := manifest.Upgrade.DatabaseMigration
	if !migration.HasDatabaseMigration {
		if migration.MigrationType != "none" || migration.MigrationID != "" || migration.Checksum != "" {
			return fmt.Errorf("no-migration release has unexpected migration metadata")
		}
		return nil
	}
	if migration.MigrationType != "compatible" {
		return fmt.Errorf("migration type %q is not supported by host executor", migration.MigrationType)
	}
	return nil
}

func migrationStartedAtStage(stage Stage, hasMigration bool) bool {
	if !hasMigration {
		return false
	}
	switch stage {
	case StageMigrating, StageRestarting, StageAgentVerifying, StageVerifying:
		return true
	default:
		return false
	}
}

func migrationEvidenceForJournal(journal Journal) bool {
	if journal.MigrationStatus == MigrationStatusRunning || journal.MigrationStatus == MigrationStatusSucceeded || journal.MigrationStatus == MigrationStatusFailed || journal.MigrationStatus == MigrationStatusUnknown {
		return true
	}
	// Reaching the explicit migrating checkpoint is itself evidence that the
	// migration boundary was crossed, even if a crash happened before the
	// identity fields could be flushed.
	if journal.Stage == StageMigrating {
		return true
	}
	// The executor binds identity before running the command. For a later
	// post-migration checkpoint, that identity distinguishes a migration from a
	// no-migration code restart whose status is merely `not_started`.
	if journal.Stage == StageRestarting || journal.Stage == StageAgentVerifying || journal.Stage == StageVerifying {
		return journal.MigrationID != "" || journal.MigrationChecksum != ""
	}
	return false
}

func failureStageFor(migrationEvidence bool) Stage {
	if migrationEvidence {
		return StageNeedsRecovery
	}
	return StageFailed
}

func (executor *ComposeExecutor) failCheckpoint(store *JournalStore, request Request, stage Stage, diagnostic string) error {
	_, err := store.Checkpoint(request.OperationID, request.ManifestDigest, stage, diagnostic, nil)
	return err
}

func (executor *ComposeExecutor) binary() string {
	if strings.TrimSpace(executor.DockerBinary) == "" {
		return defaultDockerBinary
	}
	return executor.DockerBinary
}

// ComposeArgv is the only builder used by the host executor. The generated
// override contains images only; secrets remain in the existing deployment
// `.env` file and never enter the journal directory.
func ComposeArgv(deploymentRoot, overridePath string) []string {
	return []string{
		"compose",
		"--project-directory", deploymentRoot,
		"--env-file", filepath.Join(deploymentRoot, defaultEnvFile),
		"-f", filepath.Join(deploymentRoot, defaultComposeFile),
		"-f", overridePath,
	}
}

func (executor *ComposeExecutor) composeArgv(deploymentRoot, overridePath string) []string {
	if executor == nil || !executor.PublicLayout {
		return ComposeArgv(deploymentRoot, overridePath)
	}
	return []string{
		"compose",
		"--project-name", publicProjectName,
		"--project-directory", deploymentRoot,
		"--env-file", filepath.Join(deploymentRoot, publicEnvFile),
		"-f", filepath.Join(deploymentRoot, publicComposeFile),
		"-f", overridePath,
	}
}

func (executor *ComposeExecutor) observedServices() []string {
	if executor != nil && executor.PublicLayout {
		return []string{"server", "frontend", "nginx", "agent"}
	}
	return []string{"server", "frontend", "nginx"}
}

func (executor *ComposeExecutor) pullServices() []string {
	if executor != nil && executor.PublicLayout {
		return []string{"server", "frontend", "nginx", "agent", "bootstrap"}
	}
	return executor.observedServices()
}

func (executor *ComposeExecutor) coreUpdateArgs() []string {
	args := []string{"up", "-d", "--no-build", "--force-recreate"}
	if executor != nil && executor.PublicLayout {
		args = append(args, "--no-deps", "--wait", "--wait-timeout", "300")
	}
	return append(args, "server", "frontend", "nginx")
}

func (executor *ComposeExecutor) agentUpdateArgs() []string {
	return []string{"up", "-d", "--no-build", "--force-recreate", "--no-deps", "--wait", "--wait-timeout", "300", "agent"}
}

type composeOverride struct {
	Services map[string]composeOverrideService `json:"services"`
}

type composeOverrideService struct {
	Image       string            `json:"image"`
	Environment map[string]string `json:"environment,omitempty"`
	Command     []string          `json:"command,omitempty"`
}

func (executor *ComposeExecutor) writeComposeOverride(path string, manifest *releasemanifest.Manifest) error {
	if manifest == nil {
		return fmt.Errorf("release manifest is required")
	}
	services := make(map[string]composeOverrideService)
	for _, name := range []string{"server", "frontend", "nginx"} {
		image, err := executor.selectedImageRef(manifest, name)
		if err != nil {
			return err
		}
		services[name] = composeOverrideService{Image: image}
	}
	if executor != nil && executor.PublicLayout {
		agent, err := executor.selectedImageRef(manifest, "agent")
		if err != nil {
			return err
		}
		bootstrap, err := executor.selectedImageRef(manifest, "bootstrap")
		if err != nil {
			return err
		}
		services["server"] = composeOverrideService{Image: services["server"].Image, Environment: map[string]string{
			"RELEASE_VERSION":         manifest.ReleaseVersion,
			"AGENT_VERSION":           manifest.ReleaseVersion,
			"AGENT_IMAGE_REF":         agent,
			"RELEASE_REGISTRY":        executor.Registry,
			"ENGINE_INSTALL_REGISTRY": executor.Registry,
		}}
		services["agent"] = composeOverrideService{Image: agent, Environment: map[string]string{"AGENT_VERSION": manifest.ReleaseVersion}}
		services["agent-preflight"] = composeOverrideService{Image: agent, Environment: map[string]string{"AGENT_IMAGE_REF": agent}}
		for _, name := range []string{"config-init", "migrate", "cert-init"} {
			services[name] = composeOverrideService{Image: bootstrap}
		}
		services["bootstrap"] = composeOverrideService{Image: bootstrap, Environment: map[string]string{
			"RELEASE_VERSION":         manifest.ReleaseVersion,
			"AGENT_VERSION":           manifest.ReleaseVersion,
			"ENGINE_INSTALL_REGISTRY": executor.Registry,
		}}
		services["upgrader"] = composeOverrideService{
			Image: bootstrap,
			Command: []string{
				"--root-dir", "/deployment",
				"--layout", "public",
				"--registry", executor.Registry,
			},
		}
	}
	override := composeOverride{Services: services}
	data, err := json.MarshalIndent(override, "", "  ")
	if err != nil {
		return err
	}
	return atomicWrite(path, append(data, '\n'), 0o600)
}

func (executor *ComposeExecutor) validateFixedDeploymentFiles(root string) error {
	required := []string{defaultManifestName, defaultComposeFile, defaultEnvFile}
	if executor != nil && executor.PublicLayout {
		required = []string{publicComposeFile, publicEnvFile}
	}
	for _, relative := range required {
		path := filepath.Join(root, relative)
		info, err := os.Lstat(path)
		if err != nil || info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
			return fmt.Errorf("invalid deployment file %s", relative)
		}
	}
	return nil
}

func (executor *ComposeExecutor) imageRefForService(manifest *releasemanifest.Manifest, service string) string {
	ref, _ := executor.selectedImageRef(manifest, service)
	return ref
}

func (executor *ComposeExecutor) selectedImageRef(manifest *releasemanifest.Manifest, service string) (string, error) {
	refs, err := manifest.RuntimeImageRefs(service)
	if err != nil {
		return "", err
	}
	if executor == nil || !executor.PublicLayout {
		return refs[0], nil
	}
	prefix := executor.Registry + "/"
	for _, ref := range refs {
		if strings.HasPrefix(ref, prefix) {
			return ref, nil
		}
	}
	return "", fmt.Errorf("manifest has no %s image for registry %s", service, executor.Registry)
}

func observedDigest(output string) string {
	return observedDigestPattern.FindString(strings.ToLower(output))
}

func healthyComposeOutput(raw string, services []string) bool {
	if strings.TrimSpace(raw) == "" {
		return false
	}
	// Compose emits either a JSON array or one JSON object per line depending
	// on the CLI version. Parse both forms and require each fixed service to be
	// running; when Compose reports a health field, it must be healthy as well.
	var records []map[string]any
	if err := json.Unmarshal([]byte(raw), &records); err != nil {
		for _, line := range strings.Split(raw, "\n") {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			var record map[string]any
			if err := json.Unmarshal([]byte(line), &record); err != nil {
				return false
			}
			records = append(records, record)
		}
	}
	if len(records) == 0 {
		return false
	}
	statuses := make(map[string]map[string]string, len(records))
	for _, record := range records {
		service := jsonStringField(record, "Service")
		if service == "" {
			service = jsonStringField(record, "service")
		}
		if service == "" {
			name := jsonStringField(record, "Name")
			if name == "" {
				name = jsonStringField(record, "name")
			}
			for _, candidate := range services {
				if name == candidate || strings.HasPrefix(name, candidate+"-") {
					service = candidate
					break
				}
			}
		}
		if service == "" {
			continue
		}
		statuses[service] = map[string]string{
			"state":  strings.ToLower(strings.TrimSpace(firstJSONField(record, "State", "state"))),
			"health": strings.ToLower(strings.TrimSpace(firstJSONField(record, "Health", "health"))),
		}
	}
	for _, service := range services {
		status, found := statuses[service]
		if !found {
			return false
		}
		if status["state"] != "running" && status["state"] != "healthy" {
			return false
		}
		if health := status["health"]; health != "" && health != "healthy" {
			return false
		}
	}
	return true
}

func jsonStringField(record map[string]any, key string) string {
	value, _ := record[key].(string)
	return value
}

func firstJSONField(record map[string]any, keys ...string) string {
	for _, key := range keys {
		if value := jsonStringField(record, key); value != "" {
			return value
		}
	}
	return ""
}

func nowUTC() time.Time { return time.Now().UTC() }
