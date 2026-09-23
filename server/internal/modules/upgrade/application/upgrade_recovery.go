package application

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"
	"unicode"

	"github.com/yyhuni/lunafox/server/internal/modules/upgrade/domain"
)

// HostUpgradeEvent is the sanitized event received from the host upgrader.
// It intentionally carries no command output, paths, credentials, or image
// refs. The operation and manifest digest are the replay fence.
type HostUpgradeEvent struct {
	OperationID                string
	ManifestDigest             string
	Stage                      string
	Diagnostic                 string
	Migration                  string
	ObservedDigests            map[string]string
	ExecutionMode              domain.ExecutionMode
	PlanDigest                 string
	BaselineStateDigest        string
	TouchedServices            []string
	ConfirmedDeploymentVersion string
	UpdatedAt                  time.Time
	// StageUpdatedAt is the last lifecycle checkpoint. UpdatedAt may be newer
	// because a progress heartbeat was appended, but that must not feed the
	// stalled-stage watchdog or create a synthetic StageTimes entry.
	StageUpdatedAt time.Time
	ProgressEvents []domain.ProgressEvent
	// FromJournal marks an observation read from the deployment-scoped,
	// schema-validated checkpoint. A journal may legitimately skip transient
	// stages while the Server was down; live/untrusted adapters must leave this
	// false and use the strict transition graph below.
	FromJournal bool
}

// ReconcileHostEvent folds an idempotent host checkpoint into the Server
// Operation. A host completion event only advances to verifying because the
// receipt is not sufficient evidence for overall success.
func (service *Service) ReconcileHostEvent(ctx context.Context, event HostUpgradeEvent) (*domain.Operation, error) {
	if service == nil || service.repository == nil {
		return nil, ErrUpgradeDependency
	}
	if strings.TrimSpace(event.OperationID) == "" || strings.TrimSpace(event.ManifestDigest) == "" {
		return nil, fmt.Errorf("host upgrade event operationId and manifestDigest are required")
	}
	operation, err := service.repository.Get(ctx, event.OperationID)
	if err != nil {
		return nil, err
	}
	if operation.ManifestDigest != event.ManifestDigest {
		return nil, domain.ErrReleaseManifestTargetMismatch
	}
	if err := validatePersistedFrontendOnlyOperation(operation); err != nil {
		return nil, err
	}
	// Non-journal adapters do not pass through the file reader's receipt
	// boundary. Keep their scoped evidence vocabulary just as narrow so a
	// replacement adapter cannot persist full-upgrade digests into a
	// frontend-only operation.
	if !event.FromJournal && operation.EffectiveExecutionMode() == domain.ExecutionModeFrontendOnly {
		if err := validateFrontendOnlyObservedDigests(event.ObservedDigests, "host observed digest"); err != nil {
			return nil, err
		}
	}
	if event.FromJournal {
		if err := validateHostEventScope(operation, event); err != nil {
			return nil, err
		}
		if err := validateFrontendOnlyJournalOperationStage(operation); err != nil {
			return nil, err
		}
		if err := validateJournalObservedDigests(operation, event.ObservedDigests); err != nil {
			return nil, err
		}
	}
	if err := domain.ValidateProgressEvents(event.ProgressEvents); err != nil {
		return nil, fmt.Errorf("invalid host progress events: %w", err)
	}
	if err := validateFrontendOnlyJournalProgress(operation, event.FromJournal, event.ProgressEvents); err != nil {
		return nil, err
	}
	if err := validateFrontendOnlyJournalMigration(operation, event.FromJournal, event.Migration); err != nil {
		return nil, err
	}
	status, migrationStatus, err := mapHostStage(event.Stage, event.Migration)
	if err != nil {
		return nil, err
	}
	if err := validateFrontendOnlyJournalStage(operation, event.FromJournal, status); err != nil {
		return nil, err
	}
	previousStatus := operation.Status
	// A failed/unknown migration is a stronger safety signal than any later
	// stage event. Also infer unknown when the host reports a generic failure
	// after the explicit migrating checkpoint but omits migration metadata.
	effectiveMigration := mergeMigrationStatus(operation.MigrationStatus, migrationStatus)
	if operation.Status == domain.StatusMigrating && status == domain.StatusFailed && migrationStatus == "" {
		effectiveMigration = domain.MigrationStatusUnknown
	}
	if effectiveMigration == domain.MigrationStatusFailed || effectiveMigration == domain.MigrationStatusUnknown {
		status = domain.StatusNeedsRecovery
	}
	// A journal checkpoint is durable evidence from the host executor, but it may
	// be the first checkpoint observed after a Server restart. Preserve the
	// migration safety fence while allowing a forward replay over stages whose
	// events were emitted while the Server was unavailable.
	if event.FromJournal && statusRank(status) > statusRank(operation.Status) &&
		operation.MigrationType != "none" && statusRank(status) >= statusRank(domain.StatusRestarting) &&
		effectiveMigration != domain.MigrationStatusSucceeded {
		status = domain.StatusNeedsRecovery
	}

	// Host events are observations, not commands. A terminal Server result is
	// authoritative except for the explicit safety escalation above. A stale
	// event may still contribute newer migration evidence, but it can never move
	// the lifecycle backwards.
	if operation.Status.IsTerminal() {
		if status != domain.StatusNeedsRecovery || operation.Status == domain.StatusSucceeded {
			return operation, nil
		}
	}
	if !operation.Status.IsTerminal() && statusRank(status) < statusRank(operation.Status) {
		status = operation.Status
	}
	mergedProgress, progressChanged, err := domain.MergeProgressEvents(operation.ProgressEvents, event.ProgressEvents)
	if err != nil {
		return nil, fmt.Errorf("merge host progress events: %w", err)
	}
	journalRecoveryTransition := false
	if err := domain.ValidateExecutionTransition(operation.EffectiveExecutionMode(), operation.Status, status); err != nil {
		// A delayed checkpoint that is not a safety escalation is harmless. Do not
		// turn an idempotent host retry into an API-visible failure.
		if statusRank(status) < statusRank(operation.Status) {
			return operation, nil
		}
		// A validated journal checkpoint may be the first durable observation
		// after downtime. It can replay a forward phase without manufacturing a
		// success result; final success still requires Server/Agent evidence.
		if !event.FromJournal {
			return nil, err
		}
		if recoveryErr := domain.ValidateJournalRecoveryTransition(operation.EffectiveExecutionMode(), operation.MigrationType, effectiveMigration, operation.Status, status); recoveryErr != nil {
			return nil, err
		}
		journalRecoveryTransition = true
	}
	eventAt := event.UpdatedAt.UTC()
	eventStageAt := event.StageUpdatedAt.UTC()
	if eventStageAt.IsZero() {
		// Old journals have no separate stage timestamp. Their UpdatedAt is the
		// best available checkpoint evidence and remains backward compatible.
		eventStageAt = eventAt
	}
	stageAt := operation.StageTimes[status].UTC()
	migrationChanged := effectiveMigration != "" && effectiveMigration != operation.MigrationStatus
	candidateDiagnostic := operation.Diagnostic
	if event.Diagnostic != "" && (operation.Diagnostic == "" || status == domain.StatusNeedsRecovery) {
		candidateDiagnostic = sanitizeUpgradeDiagnostic(event.Diagnostic)
	}
	diagnosticChanged := candidateDiagnostic != operation.Diagnostic
	observedChanged := false
	for key, value := range event.ObservedDigests {
		if operation.ObservedDigests == nil || operation.ObservedDigests[key] != value {
			observedChanged = true
			break
		}
	}
	checkpointAdvanced := !eventStageAt.IsZero() && (stageAt.IsZero() || eventStageAt.After(stageAt))
	if status == operation.Status && !checkpointAdvanced && !migrationChanged && !diagnosticChanged && !observedChanged && !progressChanged {
		// The recovery job reads the same durable checkpoint on every tick. A
		// replay must not turn that read into synthetic stage progress, otherwise
		// the watchdog could never classify a genuinely stalled operation.
		// Agent verification may still have new control-plane evidence, so keep
		// that reconciliation hook active even when the host checkpoint itself is
		// unchanged.
		return service.reconcileAfterHostEvent(ctx, operation)
	}
	now := service.now().UTC()
	if !eventAt.IsZero() && eventAt.After(now) {
		now = eventAt
	}
	if checkpointAdvanced && !eventStageAt.IsZero() && eventStageAt.After(now) {
		now = eventStageAt
	}
	if !now.After(operation.UpdatedAt) {
		now = operation.UpdatedAt.Add(time.Nanosecond)
	}
	operation.Status = status
	if effectiveMigration != "" {
		operation.MigrationStatus = effectiveMigration
	}
	operation.Diagnostic = candidateDiagnostic
	if len(event.ObservedDigests) > 0 {
		if operation.ObservedDigests == nil {
			operation.ObservedDigests = map[string]string{}
		}
		for key, value := range event.ObservedDigests {
			operation.ObservedDigests[key] = value
		}
	}
	operation.ProgressEvents = mergedProgress
	operation.UpdatedAt = now
	if checkpointAdvanced || status != previousStatus {
		if operation.StageTimes == nil {
			operation.StageTimes = map[domain.Status]time.Time{}
		}
		checkpointAt := eventStageAt
		if checkpointAt.IsZero() {
			checkpointAt = now
		}
		if previous, ok := operation.StageTimes[status]; !ok || checkpointAt.After(previous) {
			operation.StageTimes[status] = checkpointAt
		}
	}
	if status.IsTerminal() && (checkpointAdvanced || status != previousStatus) {
		operation.CompletedAt = &now
	}
	updated, err := service.persistReconciledOperation(ctx, operation, previousStatus, journalRecoveryTransition)
	if err != nil {
		return nil, err
	}
	return service.reconcileAfterHostEvent(ctx, updated)
}

// validateFrontendOnlyJournalMigration runs before stage mapping so malformed
// migration evidence cannot turn a scoped operation into needs_recovery or
// violate its no-migration persistence invariant.
func validateFrontendOnlyJournalMigration(operation *domain.Operation, fromJournal bool, migration string) error {
	if !fromJournal || operation == nil || operation.EffectiveExecutionMode() != domain.ExecutionModeFrontendOnly {
		return nil
	}
	migration = strings.TrimSpace(migration)
	if migration == "" || migration == string(domain.MigrationStatusNotStarted) {
		return nil
	}
	return fmt.Errorf("frontend-only host journal migration status %q is not allowed", migration)
}

// validateFrontendOnlyJournalProgress prevents a malformed checkpoint from
// persisting full-upgrade lifecycle claims into the scoped operation log.
func validateFrontendOnlyJournalProgress(operation *domain.Operation, fromJournal bool, events []domain.ProgressEvent) error {
	if !fromJournal || operation == nil || operation.EffectiveExecutionMode() != domain.ExecutionModeFrontendOnly {
		return nil
	}
	for _, event := range events {
		if !frontendOnlyJournalStageAllowed(event.Stage) {
			return fmt.Errorf("frontend-only host journal progress stage %q is not allowed", event.Stage)
		}
	}
	return nil
}

// validateFrontendOnlyJournalOperationStage prevents the forward-replay
// exception from repairing a corrupted full-upgrade phase into a valid scoped
// phase. A frontend-only operation can never have owned these phases.
func validateFrontendOnlyJournalOperationStage(operation *domain.Operation) error {
	if operation == nil || operation.EffectiveExecutionMode() != domain.ExecutionModeFrontendOnly {
		return nil
	}
	if !frontendOnlyJournalStageForbidden(operation.Status) {
		return nil
	}
	return fmt.Errorf("frontend-only persisted operation stage %q cannot be replayed from journal", operation.Status)
}

// validatePersistedFrontendOnlyOperation closes the gap between repository
// reads and update-time domain validation. Repositories deliberately preserve
// legacy rows and therefore do not validate every projection on read; a
// scoped row must nevertheless be rejected before even an idempotent journal
// replay can expose or reconcile its forbidden full-upgrade evidence.
func validatePersistedFrontendOnlyOperation(operation *domain.Operation) error {
	if operation == nil || operation.ExecutionMode != domain.ExecutionModeFrontendOnly {
		return nil
	}
	if err := operation.Validate(); err != nil {
		return fmt.Errorf("persisted frontend-only operation is invalid: %w", err)
	}
	if err := validateFrontendOnlyJournalOperationStage(operation); err != nil {
		return err
	}
	if err := validateFrontendOnlyObservedDigests(operation.ObservedDigests, "persisted frontend-only operation observed digest"); err != nil {
		return err
	}
	for stage := range operation.StageTimes {
		if !stage.Valid() || frontendOnlyJournalStageForbidden(stage) {
			return fmt.Errorf("persisted frontend-only operation stage history %q is not allowed", stage)
		}
	}
	for _, progress := range operation.ProgressEvents {
		if !frontendOnlyOperationStageAllowed(progress.Stage) {
			return fmt.Errorf("persisted frontend-only operation progress stage %q is not allowed", progress.Stage)
		}
	}
	return nil
}

// validateFrontendOnlyJournalStage keeps the journal replay exception narrow.
// Frontend-only execution never owns the stop, migration, or Agent-verification
// phases; accepting one of those checkpoints here would let a malformed durable
// observation bypass ValidateExecutionTransition and strand the operation in a
// lifecycle state that the scoped host executor cannot produce.
func validateFrontendOnlyJournalStage(operation *domain.Operation, fromJournal bool, status domain.Status) error {
	if !fromJournal || operation == nil || operation.EffectiveExecutionMode() != domain.ExecutionModeFrontendOnly {
		return nil
	}
	if !frontendOnlyJournalStageForbidden(status) {
		return nil
	}
	return fmt.Errorf("frontend-only host journal stage %q is not allowed", status)
}

func frontendOnlyJournalStageForbidden(status domain.Status) bool {
	switch status {
	case domain.StatusStopping, domain.StatusMigrating, domain.StatusAgentVerifying:
		return true
	default:
		return false
	}
}

func frontendOnlyJournalStageAllowed(status domain.Status) bool {
	if frontendOnlyJournalStageForbidden(status) {
		return false
	}
	switch status {
	case domain.StatusQueued, domain.StatusPreflight, domain.StatusUpdating,
		domain.StatusRestarting, domain.StatusVerifying, domain.StatusFailed,
		domain.StatusNeedsRecovery, domain.StatusNeedsAttention:
		return true
	default:
		return false
	}
}

func frontendOnlyOperationStageAllowed(status domain.Status) bool {
	return status.Valid() && !frontendOnlyJournalStageForbidden(status)
}

// validateJournalObservedDigests repeats the receipt service boundary at the
// recovery application boundary. JournalEventReader is an interface, so a
// replacement reader must not gain a wider evidence vocabulary than the file
// reader's schema-validated receipt.
func validateJournalObservedDigests(operation *domain.Operation, observed map[string]string) error {
	if operation == nil {
		return ErrUpgradeDependency
	}
	if operation.EffectiveExecutionMode() == domain.ExecutionModeFrontendOnly {
		return validateFrontendOnlyObservedDigests(observed, "host journal observed digest")
	}
	for service, digest := range observed {
		if !journalObservedDigestServiceAllowed(operation, service) {
			return fmt.Errorf("host journal observed digest service %q is not allowed", service)
		}
		if !upgradeObservedDigestPattern.MatchString(digest) {
			return fmt.Errorf("host journal observed digest for %q is invalid", service)
		}
	}
	return nil
}

func validateFrontendOnlyObservedDigests(observed map[string]string, label string) error {
	for service, digest := range observed {
		if service != "frontend" {
			return fmt.Errorf("%s service %q is not allowed", label, service)
		}
		if !upgradeObservedDigestPattern.MatchString(digest) {
			return fmt.Errorf("%s for %q is invalid", label, service)
		}
	}
	return nil
}

func journalObservedDigestServiceAllowed(operation *domain.Operation, service string) bool {
	if operation != nil && operation.EffectiveExecutionMode() == domain.ExecutionModeFrontendOnly {
		return service == "frontend"
	}
	switch service {
	case "server", "frontend", "nginx", "agent":
		return true
	default:
		return false
	}
}

func validateHostEventScope(operation *domain.Operation, event HostUpgradeEvent) error {
	if operation == nil {
		return ErrUpgradeDependency
	}
	if operation.ExecutionMode == "" {
		if event.ExecutionMode != "" || event.PlanDigest != "" || event.BaselineStateDigest != "" || len(event.TouchedServices) != 0 || event.ConfirmedDeploymentVersion != "" {
			return fmt.Errorf("schema-v1 operation cannot accept scoped host journal")
		}
		return nil
	}
	summary := domain.PlanSummary{TouchedServices: append([]string(nil), event.TouchedServices...)}
	if err := domain.ValidateScopePlan(event.ExecutionMode, summary, event.PlanDigest, event.BaselineStateDigest, event.ConfirmedDeploymentVersion); err != nil {
		return fmt.Errorf("host journal scope is invalid: %w", err)
	}
	if event.ExecutionMode != operation.ExecutionMode || event.PlanDigest != operation.PlanDigest || event.BaselineStateDigest != operation.BaselineDeploymentDigest || event.ConfirmedDeploymentVersion != operation.ConfirmedDeploymentVersion || !sameScopeServices(event.TouchedServices, operation.PlanSummary.TouchedServices) {
		return fmt.Errorf("host journal scope does not match persisted operation")
	}
	return nil
}

func sameScopeServices(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

// Recover is a short alias used by startup reconciliation callers.
func (service *Service) Recover(ctx context.Context, event HostUpgradeEvent) (*domain.Operation, error) {
	return service.ReconcileHostEvent(ctx, event)
}

// ReconcileJournalUnavailable classifies an active operation when the host
// checkpoint cannot be read. A missing/corrupt journal before the migration
// boundary remains retryable; after that boundary the database outcome is not
// provable and the operation must stay behind the recovery fence.
func (service *Service) ReconcileJournalUnavailable(ctx context.Context, diagnostic string) (*domain.Operation, error) {
	if service == nil || service.repository == nil {
		return nil, ErrUpgradeDependency
	}
	operation, err := service.repository.FindActive(ctx)
	if err != nil {
		// Repository implementations use gorm.ErrRecordNotFound for the normal
		// empty result. Do not make a clean startup fail just because no upgrade is
		// in progress; other storage errors remain visible to the recovery loop.
		if errors.Is(err, os.ErrNotExist) || isUpgradeRecordNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	if operation == nil || operation.Status.IsTerminal() {
		return operation, nil
	}
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
	previous := operation.Status
	if err := domain.ValidateExecutionTransition(operation.EffectiveExecutionMode(), previous, status); err != nil {
		// Queued/stopping rows predate the first executable checkpoint, so the
		// ordinary failed transition is not legal for them. They still need a
		// terminal, operator-visible outcome rather than a recovery loop that
		// reports the same transition error forever.
		if status == domain.StatusFailed {
			status = domain.StatusNeedsAttention
			if fallbackErr := domain.ValidateExecutionTransition(operation.EffectiveExecutionMode(), previous, status); fallbackErr != nil {
				return nil, fallbackErr
			}
		} else {
			return nil, err
		}
	}
	now := service.now().UTC()
	operation.Status = status
	operation.Diagnostic = sanitizeUpgradeDiagnostic(diagnostic)
	operation.UpdatedAt = now
	operation.CompletedAt = &now
	if operation.StageTimes == nil {
		operation.StageTimes = map[domain.Status]time.Time{}
	}
	operation.StageTimes[status] = now
	return service.persistReconciledOperation(ctx, operation, previous, false)
}

// ReconcileStalledOperation closes the gap where a host process disappears or
// keeps returning the same checkpoint forever. It compares the current phase's
// stage timestamp instead of UpdatedAt because Agent verification legitimately
// refreshes UpdatedAt while making no lifecycle progress.
func (service *Service) ReconcileStalledOperation(ctx context.Context, timeout time.Duration) (*domain.Operation, error) {
	if service == nil || service.repository == nil {
		return nil, ErrUpgradeDependency
	}
	operation, err := service.repository.FindActive(ctx)
	if err != nil {
		if isUpgradeRecordNotFound(err) || errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	if operation == nil || operation.Status.IsTerminal() || !service.operationStalled(operation, timeout) {
		return operation, nil
	}
	status := domain.StatusNeedsAttention
	if operationNeedsRecovery(operation) {
		status = domain.StatusNeedsRecovery
	}
	previous := operation.Status
	now := service.now().UTC()
	operation.Status = status
	operation.Diagnostic = sanitizeUpgradeDiagnostic(fmt.Sprintf("upgrade made no progress after the last confirmed %s stage; host or verification requires operator attention", previous))
	operation.UpdatedAt = now
	operation.CompletedAt = &now
	if operation.StageTimes == nil {
		operation.StageTimes = map[domain.Status]time.Time{}
	}
	operation.StageTimes[status] = now
	return service.persistReconciledOperation(ctx, operation, previous, false)
}

// upgradeRecoveryStallTimeout derives a bounded watchdog from the release's
// maintenance window. The grace period accounts for journal handoff and
// container health checks; the lower and upper bounds keep malformed/legacy
// values from either failing a fresh operation immediately or leaving an
// operation active forever.
func upgradeRecoveryStallTimeout(operation *domain.Operation) time.Duration {
	if operation == nil || operation.MaintenanceWindowMinutes <= 0 {
		return defaultUpgradeRecoveryStallTimeout
	}
	minutes := operation.MaintenanceWindowMinutes
	maximumWindowMinutes := int((maximumUpgradeRecoveryStallTimeout - upgradeRecoveryGracePeriod) / time.Minute)
	if minutes >= maximumWindowMinutes {
		return maximumUpgradeRecoveryStallTimeout
	}
	timeout := time.Duration(minutes)*time.Minute + upgradeRecoveryGracePeriod
	if timeout < minimumUpgradeRecoveryStallTimeout {
		return minimumUpgradeRecoveryStallTimeout
	}
	return timeout
}

func (service *Service) operationStalled(operation *domain.Operation, timeout time.Duration) bool {
	if operation == nil || operation.Status.IsTerminal() {
		return false
	}
	if timeout <= 0 {
		timeout = upgradeRecoveryStallTimeout(operation)
	}
	progressAt := service.operationProgressAt(operation)
	if progressAt.IsZero() {
		progressAt = operation.CreatedAt
	}
	now := service.now().UTC()
	return !progressAt.IsZero() && now.After(progressAt.Add(timeout))
}

func (service *Service) operationProgressAt(operation *domain.Operation) time.Time {
	if operation == nil {
		return time.Time{}
	}
	if stageAt, ok := operation.StageTimes[operation.Status]; ok {
		return stageAt.UTC()
	}
	return operation.CreatedAt.UTC()
}

func operationNeedsRecovery(operation *domain.Operation) bool {
	if operation == nil {
		return false
	}
	if operation.EffectiveExecutionMode() == domain.ExecutionModeFrontendOnly {
		return false
	}
	return operation.MigrationStatus == domain.MigrationStatusRunning ||
		operation.MigrationStatus == domain.MigrationStatusSucceeded ||
		operation.MigrationStatus == domain.MigrationStatusFailed ||
		operation.MigrationStatus == domain.MigrationStatusUnknown ||
		operation.Status == domain.StatusMigrating ||
		operation.Status == domain.StatusRestarting ||
		operation.Status == domain.StatusAgentVerifying ||
		operation.Status == domain.StatusVerifying
}

// isUpgradeRecordNotFound keeps the application package independent from the
// concrete ORM while still treating the repository's empty active query as a
// normal startup result.
func isUpgradeRecordNotFound(err error) bool {
	return strings.Contains(strings.ToLower(strings.TrimSpace(err.Error())), "record not found")
}

func mapHostStage(stage, migration string) (domain.Status, domain.MigrationStatus, error) {
	var status domain.Status
	switch strings.TrimSpace(stage) {
	case string(domain.StatusQueued):
		status = domain.StatusQueued
	case string(domain.StatusStopping):
		status = domain.StatusStopping
	case string(domain.StatusPreflight):
		status = domain.StatusPreflight
	case string(domain.StatusUpdating):
		status = domain.StatusUpdating
	case string(domain.StatusMigrating):
		status = domain.StatusMigrating
	case string(domain.StatusRestarting):
		status = domain.StatusRestarting
	case string(domain.StatusAgentVerifying):
		status = domain.StatusAgentVerifying
	case string(domain.StatusVerifying):
		status = domain.StatusVerifying
	case string(domain.StatusFailed):
		status = domain.StatusFailed
	case string(domain.StatusNeedsRecovery):
		status = domain.StatusNeedsRecovery
	case string(domain.StatusNeedsAttention):
		status = domain.StatusNeedsAttention
	case string(domain.StatusSucceeded):
		// Host receipts prove deployment only. Agent and API evidence are still
		// required before Server may mark the operation succeeded.
		status = domain.StatusVerifying
	default:
		return "", "", fmt.Errorf("unsupported host upgrade stage %q", stage)
	}
	return status, normalizeMigrationStatus(migration, status), nil
}

func normalizeMigrationStatus(value string, status domain.Status) domain.MigrationStatus {
	switch domain.MigrationStatus(strings.TrimSpace(value)) {
	case domain.MigrationStatusNotStarted, domain.MigrationStatusRunning,
		domain.MigrationStatusSucceeded, domain.MigrationStatusFailed,
		domain.MigrationStatusUnknown:
		return domain.MigrationStatus(strings.TrimSpace(value))
	}
	if status == domain.StatusMigrating {
		return domain.MigrationStatusRunning
	}
	if status == domain.StatusNeedsRecovery {
		return domain.MigrationStatusUnknown
	}
	return ""
}

func mergeMigrationStatus(current, incoming domain.MigrationStatus) domain.MigrationStatus {
	if incoming == "" {
		return current
	}
	if current == "" {
		return incoming
	}
	// Failure/unknown evidence is irreversible. In particular, a later
	// "succeeded" checkpoint cannot erase a migration outcome that may have
	// touched the database before the process was interrupted.
	rank := func(value domain.MigrationStatus) int {
		switch value {
		case domain.MigrationStatusNotStarted:
			return 0
		case domain.MigrationStatusRunning:
			return 1
		case domain.MigrationStatusSucceeded:
			return 2
		case domain.MigrationStatusFailed:
			return 3
		case domain.MigrationStatusUnknown:
			return 4
		default:
			return -1
		}
	}
	if rank(incoming) >= rank(current) {
		return incoming
	}
	return current
}

func statusRank(status domain.Status) int {
	switch status {
	case domain.StatusQueued:
		return 0
	case domain.StatusStopping:
		return 1
	case domain.StatusPreflight:
		return 2
	case domain.StatusUpdating:
		return 3
	case domain.StatusMigrating:
		return 4
	case domain.StatusRestarting:
		return 5
	case domain.StatusAgentVerifying:
		return 6
	case domain.StatusVerifying:
		return 7
	case domain.StatusFailed:
		return 8
	case domain.StatusNeedsAttention, domain.StatusNeedsRecovery, domain.StatusSucceeded:
		return 9
	default:
		return -1
	}
}

func sanitizeUpgradeDiagnostic(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	// Host events are expected to be pre-sanitized. Keep a second boundary here
	// so a compromised adapter cannot persist credentials or multiline output.
	lower := strings.ToLower(value)
	for _, marker := range []string{"authorization", "bearer ", "jwt", "password", "secret", "token", "private key", "-----begin"} {
		if strings.Contains(lower, marker) {
			return "host reported a redacted upgrade diagnostic"
		}
	}
	value = strings.Map(func(r rune) rune {
		if r == '\r' || r == '\n' || unicode.IsControl(r) {
			return ' '
		}
		return r
	}, value)
	if runes := []rune(value); len(runes) > 512 {
		value = string(runes[:512])
	}
	return strings.TrimSpace(value)
}

func (service *Service) persistReconciledOperation(ctx context.Context, operation *domain.Operation, expected domain.Status, journalRecovery bool) (*domain.Operation, error) {
	var err error
	if journalRecovery {
		journalRepository, ok := service.repository.(domain.JournalRecoveryTransitionRepository)
		if !ok {
			return nil, fmt.Errorf("upgrade repository does not support validated journal recovery transitions")
		}
		err = journalRepository.UpdateJournalRecoveryTransition(ctx, operation, expected)
	} else if transitionRepository, ok := service.repository.(domain.TransitionRepository); ok {
		err = transitionRepository.UpdateTransition(ctx, operation, expected)
	} else {
		err = service.repository.Update(ctx, operation)
	}
	if err != nil {
		if errors.Is(err, domain.ErrUpgradeTransitionConflict) {
			// Another event won the CAS. Return its durable state so callers do
			// not display an in-memory stage that was never committed.
			if latest, getErr := service.repository.Get(ctx, operation.OperationID); getErr == nil {
				return latest, nil
			}
		}
		return nil, err
	}
	return operation, nil
}

func cloneStringMap(values map[string]string) map[string]string {
	if len(values) == 0 {
		return map[string]string{}
	}
	result := make(map[string]string, len(values))
	for key, value := range values {
		result[key] = value
	}
	return result
}
