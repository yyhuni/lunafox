package application

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/blang/semver"
	"github.com/google/uuid"
	"github.com/yyhuni/lunafox/contracts/ociartifact"
	"github.com/yyhuni/lunafox/contracts/releasemanifest"
	"github.com/yyhuni/lunafox/server/internal/modules/upgrade/domain"
)

const hostDispatchTimeout = 15 * time.Second

// ServiceConfig contains only server-owned dependencies. Manifest and policy
// readers are deliberately interfaces so HTTP code cannot smuggle deployment
// paths, commands, or image references into this use case.
type ServiceConfig struct {
	ManifestSource           ManifestSource
	MigrationPolicySource    MigrationPolicySource
	Repository               domain.Repository
	Authorizer               ActiveSuperuserAuthorizer
	Dispatcher               HostUpgradeDispatcher
	Coordinator              PreDispatchCoordinator
	AgentSource              AgentUpgradeSource
	Verifier                 UpgradeVerifier
	CurrentVersion           string
	Registry                 string
	Now                      func() time.Time
	AgentVerificationTimeout time.Duration
}

// Service orchestrates update checks and the durable Upgrade Operation
// lifecycle. The host dispatcher only accepts a bounded handoff and owns the
// long-running Compose work outside the request lifecycle.
type Service struct {
	manifestSource           ManifestSource
	migrationPolicySource    MigrationPolicySource
	repository               domain.Repository
	authorizer               ActiveSuperuserAuthorizer
	dispatcher               HostUpgradeDispatcher
	coordinator              PreDispatchCoordinator
	agentSource              AgentUpgradeSource
	verifier                 UpgradeVerifier
	currentVersion           string
	registry                 string
	now                      func() time.Time
	agentVerificationTimeout time.Duration
	mu                       sync.Mutex
	agentNotifyMu            sync.Mutex
	agentNotifiedOperations  map[string]struct{}
}

var (
	ErrUpgradeDependency = errors.New("upgrade service dependency is required")
)

// NewService constructs the Server upgrade application service.
func NewService(config ServiceConfig) (*Service, error) {
	if config.ManifestSource == nil || config.MigrationPolicySource == nil || config.Repository == nil || config.Authorizer == nil {
		return nil, ErrUpgradeDependency
	}
	currentVersion := strings.TrimSpace(config.CurrentVersion)
	if currentVersion != "" {
		if _, err := semver.Parse(currentVersion); err != nil {
			return nil, fmt.Errorf("current release version is invalid: %w", err)
		}
	}
	registry := strings.TrimSpace(config.Registry)
	if registry != "" && registry != "docker.io" && registry != "ghcr.io" {
		return nil, fmt.Errorf("upgrade registry must be docker.io or ghcr.io")
	}
	now := config.Now
	if now == nil {
		now = func() time.Time { return time.Now().UTC() }
	}
	verificationTimeout := config.AgentVerificationTimeout
	if verificationTimeout <= 0 {
		verificationTimeout = 15 * time.Minute
	}
	return &Service{
		manifestSource:           config.ManifestSource,
		migrationPolicySource:    config.MigrationPolicySource,
		repository:               config.Repository,
		authorizer:               config.Authorizer,
		dispatcher:               config.Dispatcher,
		coordinator:              config.Coordinator,
		agentSource:              config.AgentSource,
		verifier:                 config.Verifier,
		currentVersion:           currentVersion,
		registry:                 registry,
		now:                      now,
		agentVerificationTimeout: verificationTimeout,
		agentNotifiedOperations:  make(map[string]struct{}),
	}, nil
}

// NewUpgradeService is the descriptive constructor alias used by bootstrap
// wiring and keeps the module's public entry point discoverable.
func NewUpgradeService(config ServiceConfig) (*Service, error) {
	return NewService(config)
}

type CheckForUpdatesResult struct {
	CurrentVersion string
	HasUpdate      bool
	Manifest       ManifestSummary
	Eligible       bool
	Diagnostic     *domain.Diagnostic
}

// ManifestSummary is the safe, UI-facing projection of a validated release.
// It contains identities and digests only; image refs remain host-owned.
type ManifestSummary struct {
	ManifestID                string
	ManifestDigest            string
	ReleaseVersion            string
	DeploymentMode            string
	CompatibilityRange        string
	MaintenanceWindowMinutes  int
	RequiresAdminConfirmation bool
	HasDatabaseMigration      bool
	MigrationType             string
	MigrationID               string
	MigrationChecksum         string
	MigrationPolicyVersion    int
	RuntimeImageDigests       map[string]string
	EngineDigests             []string
}

type CreateUpgradeOperationInput struct {
	RequestID      string
	ManifestID     string
	ManifestDigest string
	Confirmed      bool
	ImageRefs      []string
	ImageRef       string
}

// CheckForUpdates returns a validated candidate and policy diagnostic. Policy
// ineligibility is returned as data so the frontend can explain the risk;
// malformed manifests and unavailable policy sources remain hard errors.
func (service *Service) CheckForUpdates(ctx context.Context, userID int) (CheckForUpdatesResult, error) {
	if err := service.authorize(ctx, userID); err != nil {
		return CheckForUpdatesResult{}, err
	}
	manifest, err := service.manifestSource.Load()
	if err != nil {
		return CheckForUpdatesResult{}, err
	}
	summary, err := summarizeManifest(manifest)
	if err != nil {
		return CheckForUpdatesResult{}, err
	}
	hasUpdate, err := service.hasNewerRelease(manifest.ReleaseVersion)
	if err != nil {
		return CheckForUpdatesResult{}, err
	}
	result := CheckForUpdatesResult{
		CurrentVersion: service.currentVersion,
		HasUpdate:      hasUpdate,
		Manifest:       summary,
	}
	if compatibilityErr := service.checkReleaseCompatibility(manifest.Upgrade.CompatibilityRange); compatibilityErr != nil {
		if diagnostic, ok := domain.DiagnosticOf(compatibilityErr); ok && diagnostic.Code == domain.ErrorCodeReleaseCompatibilityUnsupported {
			result.Diagnostic = &diagnostic
			return result, nil
		}
		return CheckForUpdatesResult{}, compatibilityErr
	}
	policy, err := service.migrationPolicySource.Load(ctx)
	if err != nil {
		return CheckForUpdatesResult{}, err
	}
	eligibility, eligibilityErr := domain.EvaluateMigration(manifest, policy)
	result.Eligible = eligibilityErr == nil && eligibility.Supported
	if eligibilityErr != nil {
		if diagnostic, ok := domain.DiagnosticOf(eligibilityErr); ok {
			result.Diagnostic = &diagnostic
		}
		return result, nil
	}
	return result, nil
}

// CreateOperation performs the final server-side review after administrator
// confirmation. A replay returns the persisted operation and never dispatches
// another host action.
func (service *Service) CreateOperation(ctx context.Context, userID int, input CreateUpgradeOperationInput) (*domain.Operation, bool, error) {
	if err := service.authorize(ctx, userID); err != nil {
		return nil, false, err
	}
	requestID, err := canonicalUUID(input.RequestID)
	if err != nil {
		return nil, false, err
	}
	if !input.Confirmed {
		return nil, false, domain.ErrUpgradeConfirmationRequired
	}
	// Check the durable request binding before reading the current manifest.
	// This is important when a browser retries after the deployment manifest
	// has advanced: the original request must replay its Operation, while a
	// different target using the same requestId must be rejected.
	if lookup, ok := service.repository.(RequestLookupRepository); ok {
		existing, lookupErr := lookup.GetByRequest(ctx, requestID)
		if lookupErr == nil && existing != nil {
			if existing.ManifestID != strings.TrimSpace(input.ManifestID) || existing.ManifestDigest != strings.TrimSpace(input.ManifestDigest) {
				return nil, false, domain.ErrUpgradeRequestConflict
			}
			return existing, false, nil
		}
		if lookupErr != nil && !errors.Is(lookupErr, domain.ErrUpgradeNotFound) {
			return nil, false, lookupErr
		}
	}
	manifest, err := service.manifestSource.Load()
	if err != nil {
		return nil, false, err
	}
	if err := domain.ValidateTargetRequest(domain.TargetRequest{
		ManifestID: input.ManifestID, ManifestDigest: input.ManifestDigest,
		ImageRefs: input.ImageRefs, ImageRef: input.ImageRef,
	}, manifest); err != nil {
		return nil, false, err
	}
	hasUpdate, err := service.hasNewerRelease(manifest.ReleaseVersion)
	if err != nil {
		return nil, false, err
	}
	if !hasUpdate {
		return nil, false, domain.NewUpgradeNoUpdateAvailable()
	}
	if err := service.checkReleaseCompatibility(manifest.Upgrade.CompatibilityRange); err != nil {
		return nil, false, err
	}
	policy, err := service.migrationPolicySource.Load(ctx)
	if err != nil {
		return nil, false, err
	}
	eligibility, err := domain.EvaluateMigration(manifest, policy)
	if err != nil || !eligibility.Supported {
		if err != nil {
			return nil, false, err
		}
		return nil, false, domain.NewMigrationUnsupported("release is not supported by the active migration policy")
	}
	service.mu.Lock()
	defer service.mu.Unlock()
	now := service.now().UTC()
	var agentExpectations []domain.AgentExpectation
	if service.agentSource != nil {
		target, targetErr := service.agentTarget(manifest)
		if targetErr != nil {
			return nil, false, targetErr
		}
		agentExpectations, err = service.agentSource.Snapshot(ctx, target)
		if err != nil {
			return nil, false, fmt.Errorf("snapshot agents for upgrade: %w", err)
		}
	}
	operationID := uuid.NewString()
	deadline := now.Add(service.agentVerificationTimeout)
	operation := &domain.Operation{
		OperationID:               operationID,
		RequestID:                 requestID,
		OperatorID:                userID,
		ManifestID:                manifest.Upgrade.ManifestID,
		ManifestDigest:            manifest.Digest(),
		ReleaseVersion:            manifest.ReleaseVersion,
		CompatibilityRange:        manifest.Upgrade.CompatibilityRange,
		MaintenanceWindowMinutes:  manifest.Upgrade.MaintenanceWindowMinutes,
		Status:                    domain.StatusQueued,
		MigrationStatus:           domain.MigrationStatusNotStarted,
		MigrationType:             eligibility.MigrationType,
		MigrationID:               eligibility.MigrationID,
		MigrationChecksum:         eligibility.Checksum,
		AgentDesiredVersion:       manifest.ReleaseVersion,
		AgentTargetDigest:         runtimeAgentDigest(manifest),
		AgentSummary:              summarizeAgentExpectations(agentExpectations),
		AgentExpectations:         agentExpectations,
		AgentVerificationDeadline: &deadline,
		ObservedDigests:           map[string]string{},
		StageTimes:                map[domain.Status]time.Time{domain.StatusQueued: now},
		CreatedAt:                 now,
		UpdatedAt:                 now,
	}
	createdOperation, created, err := service.repository.CreateOrGet(ctx, operation)
	if err != nil {
		return nil, false, err
	}
	if !created {
		return createdOperation, false, nil
	}
	if service.coordinator != nil {
		createdOperation.Status = domain.StatusStopping
		createdOperation.UpdatedAt = service.now().UTC()
		if createdOperation.StageTimes == nil {
			createdOperation.StageTimes = map[domain.Status]time.Time{}
		}
		createdOperation.StageTimes[domain.StatusStopping] = createdOperation.UpdatedAt
		if err := service.repository.Update(ctx, createdOperation); err != nil {
			return createdOperation, true, err
		}
		var preparation PreparationResult
		var err error
		if operationCoordinator, ok := service.coordinator.(UpgradePreDispatchCoordinator); ok {
			preparation, err = operationCoordinator.PrepareForUpgrade(ctx, createdOperation.OperationID)
		} else {
			preparation, err = service.coordinator.Prepare(ctx)
		}
		if err != nil {
			service.markPreparationFailed(createdOperation)
			return createdOperation, true, err
		}
		createdOperation.CancelledScanCount = preparation.CancelledScanCount
		createdOperation.CancelledTaskCount = preparation.CancelledTaskCount
		createdOperation.UpdatedAt = service.now().UTC()
		if err := service.repository.Update(ctx, createdOperation); err != nil {
			return createdOperation, true, err
		}
	}
	if err := service.dispatch(ctx, createdOperation, HostUpgradeActionStart); err != nil {
		return createdOperation, true, err
	}
	return createdOperation, true, nil
}

func (service *Service) markPreparationFailed(operation *domain.Operation) {
	if operation == nil {
		return
	}
	now := service.now().UTC()
	operation.Status = domain.StatusFailed
	operation.Diagnostic = "active work could not be cancelled before upgrade handoff"
	operation.UpdatedAt = now
	operation.CompletedAt = &now
	if operation.StageTimes == nil {
		operation.StageTimes = map[domain.Status]time.Time{}
	}
	operation.StageTimes[domain.StatusFailed] = now
	_ = service.repository.Update(context.Background(), operation)
}

// GetOperation returns one durable operation after the same server-side
// authority check used by create/retry.
func (service *Service) GetOperation(ctx context.Context, userID int, operationID string) (*domain.Operation, error) {
	if err := service.authorize(ctx, userID); err != nil {
		return nil, err
	}
	operationID, err := canonicalUUID(operationID)
	if err != nil {
		return nil, domain.ErrUpgradeNotFound
	}
	return service.repository.Get(ctx, operationID)
}

// RetryOperation explicitly reuses the persisted release target. It resets
// only retryable terminal states and sends one resume handoff for that same
// operation; succeeded operations and active operations are never restarted.
func (service *Service) RetryOperation(ctx context.Context, userID int, operationID string, confirmed bool) (*domain.Operation, error) {
	if err := service.authorize(ctx, userID); err != nil {
		return nil, err
	}
	if !confirmed {
		return nil, domain.ErrUpgradeConfirmationRequired
	}
	operationID, err := canonicalUUID(operationID)
	if err != nil {
		return nil, domain.ErrUpgradeNotFound
	}
	service.mu.Lock()
	defer service.mu.Unlock()
	operation, err := service.repository.Get(ctx, operationID)
	if err != nil {
		return nil, err
	}
	if operation.Status != domain.StatusFailed && operation.Status != domain.StatusNeedsRecovery && operation.Status != domain.StatusNeedsAttention {
		return nil, domain.ErrUpgradeRetryNotAllowed
	}
	manifest, err := service.loadTargetManifest(operation.ManifestDigest)
	if err != nil {
		return nil, err
	}
	if err := domain.EnsureSameTarget(domain.Target{ManifestID: operation.ManifestID, ManifestDigest: operation.ManifestDigest}, domain.Target{ManifestID: manifest.Upgrade.ManifestID, ManifestDigest: manifest.Digest()}); err != nil {
		return nil, err
	}
	policy, err := service.migrationPolicySource.Load(ctx)
	if err != nil {
		return nil, err
	}
	eligibility, err := domain.EvaluateMigration(manifest, policy)
	if err != nil {
		return nil, err
	}
	if !eligibility.Supported {
		return nil, domain.NewMigrationUnsupported("release is not supported by the active migration policy")
	}
	now := service.now().UTC()
	expectedStatus := operation.Status
	// Once a migration checkpoint was crossed, a retry must resume at the
	// recovery/verification boundary. Reopening at queued would run the same
	// migration again after an uncertain database outcome.
	operation.Status = retryStartStatus(operation.Status, operation.MigrationStatus)
	operation.MigrationStatus = retryMigrationStatus(operation.MigrationStatus)
	operation.Diagnostic = ""
	operation.CompletedAt = nil
	operation.UpdatedAt = now
	if operation.StageTimes == nil {
		operation.StageTimes = map[domain.Status]time.Time{}
	}
	operation.StageTimes[operation.Status] = now
	if retryRepository, ok := service.repository.(domain.RetryRepository); ok {
		if err := retryRepository.ResetForRetry(ctx, operation, expectedStatus); err != nil {
			return nil, err
		}
	} else if err := service.repository.Update(ctx, operation); err != nil {
		return nil, err
	}
	if err := service.dispatch(ctx, operation, HostUpgradeActionRepair); err != nil {
		return operation, err
	}
	return operation, nil
}

func (service *Service) loadTargetManifest(digest string) (*releasemanifest.Manifest, error) {
	if source, ok := service.manifestSource.(TargetManifestSource); ok {
		return source.LoadTarget(digest)
	}
	return service.manifestSource.Load()
}

func (service *Service) hasNewerRelease(candidate string) (bool, error) {
	if service.currentVersion == "" {
		return true, nil
	}
	targetVersion, err := semver.Parse(strings.TrimSpace(candidate))
	if err != nil {
		return false, domain.WrapManifestInvalid(fmt.Errorf("release version is invalid: %w", err))
	}
	currentVersion, err := semver.Parse(service.currentVersion)
	if err != nil {
		return false, fmt.Errorf("current release version is invalid: %w", err)
	}
	return targetVersion.GT(currentVersion), nil
}

func (service *Service) checkReleaseCompatibility(rawRange string) error {
	compatibilityRange, err := semver.ParseRange(rawRange)
	if err != nil {
		return domain.WrapManifestInvalid(fmt.Errorf("upgrade compatibility range is invalid: %w", err))
	}
	if compatibilityRange == nil {
		return domain.WrapManifestInvalid(errors.New("upgrade compatibility range is invalid"))
	}
	if service.currentVersion == "" {
		return nil
	}
	currentVersion, err := semver.Parse(service.currentVersion)
	if err != nil {
		return fmt.Errorf("current release version is invalid: %w", err)
	}
	if !compatibilityRange(currentVersion) {
		return domain.NewReleaseCompatibilityUnsupported()
	}
	return nil
}

// Compatibility aliases make the application boundary easy to consume while
// retaining the explicit AIP-oriented method names above.
func (service *Service) Check(ctx context.Context, userID int) (CheckForUpdatesResult, error) {
	return service.CheckForUpdates(ctx, userID)
}

func (service *Service) Create(ctx context.Context, userID int, input CreateUpgradeOperationInput) (*domain.Operation, bool, error) {
	return service.CreateOperation(ctx, userID, input)
}

func (service *Service) Get(ctx context.Context, userID int, operationID string) (*domain.Operation, error) {
	return service.GetOperation(ctx, userID, operationID)
}

func (service *Service) Retry(ctx context.Context, userID int, operationID string, confirmed bool) (*domain.Operation, error) {
	return service.RetryOperation(ctx, userID, operationID, confirmed)
}

func (service *Service) authorize(ctx context.Context, userID int) error {
	if userID <= 0 {
		return domain.ErrUpgradeUnauthorized
	}
	allowed, err := service.authorizer.IsActiveSuperuser(ctx, userID)
	if err != nil {
		return fmt.Errorf("resolve upgrade authority: %w", err)
	}
	if !allowed {
		return domain.ErrUpgradeUnauthorized
	}
	return nil
}

func (service *Service) dispatch(ctx context.Context, operation *domain.Operation, action HostUpgradeAction) error {
	if service.dispatcher == nil || operation == nil {
		return domain.ErrUpgradeHostUnavailable
	}
	// The host handoff is intentionally detached from the browser request. A
	// bounded timeout protects only the acceptance handshake, not Compose work.
	dispatchCtx, cancel := context.WithTimeout(context.Background(), hostDispatchTimeout)
	defer cancel()
	if err := service.dispatcher.Dispatch(dispatchCtx, HostUpgradeRequest{
		OperationID: operation.OperationID, ManifestDigest: operation.ManifestDigest, Action: action,
	}); err != nil {
		service.markHostUnavailable(operation, err)
		return fmt.Errorf("dispatch host upgrader: %w", domain.ErrUpgradeHostUnavailable)
	}
	return nil
}

func (service *Service) markHostUnavailable(operation *domain.Operation, _ error) {
	if operation == nil {
		return
	}
	// Once the operation has recorded a migration checkpoint, losing the host
	// handoff no longer proves that the database is unchanged. Preserve that
	// uncertainty as needs_recovery instead of presenting a retryable failure.
	status := domain.StatusFailed
	if operation.MigrationStatus == domain.MigrationStatusRunning ||
		operation.MigrationStatus == domain.MigrationStatusSucceeded ||
		operation.MigrationStatus == domain.MigrationStatusFailed ||
		operation.MigrationStatus == domain.MigrationStatusUnknown ||
		operation.Status == domain.StatusMigrating ||
		operation.Status == domain.StatusRestarting ||
		operation.Status == domain.StatusAgentVerifying ||
		operation.Status == domain.StatusVerifying {
		status = domain.StatusNeedsRecovery
	}
	operation.Status = status
	operation.Diagnostic = "host upgrader handoff was not accepted"
	operation.UpdatedAt = service.now().UTC()
	if operation.StageTimes == nil {
		operation.StageTimes = map[domain.Status]time.Time{}
	}
	operation.StageTimes[status] = operation.UpdatedAt
	operation.CompletedAt = &operation.UpdatedAt
	_ = service.repository.Update(context.Background(), operation)
}

func canonicalUUID(value string) (string, error) {
	parsed, err := uuid.Parse(strings.TrimSpace(value))
	if err != nil || parsed.String() != value {
		return "", fmt.Errorf("requestId must be a canonical UUID")
	}
	return parsed.String(), nil
}

func retryMigrationStatus(status domain.MigrationStatus) domain.MigrationStatus {
	if status == domain.MigrationStatusRunning || status == domain.MigrationStatusSucceeded || status == domain.MigrationStatusUnknown {
		return status
	}
	return domain.MigrationStatusNotStarted
}

func retryStartStatus(previous domain.Status, migrationStatus domain.MigrationStatus) domain.Status {
	if migrationStatus != domain.MigrationStatusNotStarted || previous == domain.StatusNeedsRecovery {
		return domain.StatusRestarting
	}
	return domain.StatusQueued
}

func summarizeManifest(manifest *releasemanifest.Manifest) (ManifestSummary, error) {
	if manifest == nil {
		return ManifestSummary{}, domain.WrapManifestInvalid(nil)
	}
	digests, err := manifest.RuntimeImageDigests()
	if err != nil {
		return ManifestSummary{}, domain.WrapManifestInvalid(err)
	}
	engineDigests := make([]string, 0, len(manifest.EnginePackages))
	for index, packageEntry := range manifest.EnginePackages {
		if len(packageEntry.Refs) == 0 {
			return ManifestSummary{}, domain.WrapManifestInvalid(fmt.Errorf("enginePackages[%d] has no digest reference", index))
		}
		parsed, err := ociartifact.ParseDigestReference(packageEntry.Refs[0])
		if err != nil {
			return ManifestSummary{}, domain.WrapManifestInvalid(fmt.Errorf("enginePackages[%d]: %w", index, err))
		}
		engineDigests = append(engineDigests, parsed.Digest)
	}
	migration := manifest.Upgrade.DatabaseMigration
	return ManifestSummary{
		ManifestID:                manifest.Upgrade.ManifestID,
		ManifestDigest:            manifest.Digest(),
		ReleaseVersion:            manifest.ReleaseVersion,
		DeploymentMode:            manifest.Upgrade.DeploymentMode,
		CompatibilityRange:        manifest.Upgrade.CompatibilityRange,
		MaintenanceWindowMinutes:  manifest.Upgrade.MaintenanceWindowMinutes,
		RequiresAdminConfirmation: manifest.Upgrade.RequiresAdminConfirmation,
		HasDatabaseMigration:      migration.HasDatabaseMigration,
		MigrationType:             migration.MigrationType,
		MigrationID:               migration.MigrationID,
		MigrationChecksum:         migration.Checksum,
		MigrationPolicyVersion:    migration.PolicyVersion,
		RuntimeImageDigests:       digests,
		EngineDigests:             engineDigests,
	}, nil
}

func runtimeAgentDigest(manifest *releasemanifest.Manifest) string {
	if manifest == nil {
		return ""
	}
	digest, _ := manifest.RuntimeImageDigest("agent")
	return digest
}

// agentTarget projects the immutable Agent target from the server-owned
// release manifest. The image reference is used only by the existing control
// plane update_required payload; callers cannot provide or override it.
func (service *Service) agentTarget(manifest *releasemanifest.Manifest) (AgentUpgradeTarget, error) {
	if manifest == nil {
		return AgentUpgradeTarget{}, domain.WrapManifestInvalid(nil)
	}
	refs, err := manifest.RuntimeImageRefs("agent")
	if err != nil || len(refs) == 0 {
		if err == nil {
			err = fmt.Errorf("agent runtime image has no digest reference")
		}
		return AgentUpgradeTarget{}, domain.WrapManifestInvalid(err)
	}
	digest := runtimeAgentDigest(manifest)
	if digest == "" {
		return AgentUpgradeTarget{}, domain.WrapManifestInvalid(fmt.Errorf("agent runtime image digest is required"))
	}
	imageRef := refs[0]
	if service.registry != "" {
		imageRef = ""
		for _, candidate := range refs {
			parsed, parseErr := ociartifact.ParseDigestReference(candidate)
			if parseErr == nil && parsed.Registry == service.registry {
				imageRef = candidate
				break
			}
		}
		if imageRef == "" {
			return AgentUpgradeTarget{}, domain.WrapManifestInvalid(fmt.Errorf("agent runtime image has no candidate for registry %s", service.registry))
		}
	}
	return AgentUpgradeTarget{Version: manifest.ReleaseVersion, Digest: digest, ImageRef: imageRef}, nil
}

func summarizeAgentExpectations(expectations []domain.AgentExpectation) domain.AgentSummary {
	summary := domain.AgentSummary{Expected: len(expectations)}
	for _, expectation := range expectations {
		if expectation.Ready() {
			summary.Ready++
			continue
		}
		// An Agent without a fresh authenticated heartbeat is missing. A
		// connected Agent with stale version/health/runtime evidence is unhealthy.
		if !expectation.Connected || strings.TrimSpace(expectation.ObservedVersion) == "" {
			summary.Missing++
		} else {
			summary.Unhealthy++
		}
	}
	return summary
}
