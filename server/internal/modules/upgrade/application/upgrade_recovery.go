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
	OperationID     string
	ManifestDigest  string
	Stage           string
	Diagnostic      string
	Migration       string
	ObservedDigests map[string]string
	UpdatedAt       time.Time
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
	status, migrationStatus, err := mapHostStage(event.Stage, event.Migration)
	if err != nil {
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
	if err := domain.ValidateTransition(operation.Status, status); err != nil {
		// A delayed checkpoint that is not a safety escalation is harmless. Do not
		// turn an idempotent host retry into an API-visible failure.
		if statusRank(status) < statusRank(operation.Status) {
			return operation, nil
		}
		// A validated journal checkpoint may be the first durable observation
		// after downtime. It can replay a forward phase without manufacturing a
		// success result; final success still requires Server/Agent evidence.
		if !(event.FromJournal && statusRank(status) > statusRank(operation.Status) && statusRank(status) <= statusRank(domain.StatusVerifying)) {
			return nil, err
		}
		// Continue with the validated journal observation.
	}
	now := event.UpdatedAt.UTC()
	if now.IsZero() || now.Before(operation.UpdatedAt) {
		now = service.now().UTC()
	}
	if !now.After(operation.UpdatedAt) {
		now = operation.UpdatedAt.Add(time.Nanosecond)
	}
	operation.Status = status
	if effectiveMigration != "" {
		operation.MigrationStatus = effectiveMigration
	}
	if event.Diagnostic != "" && (operation.Diagnostic == "" || status == domain.StatusNeedsRecovery) {
		operation.Diagnostic = sanitizeUpgradeDiagnostic(event.Diagnostic)
	}
	if len(event.ObservedDigests) > 0 {
		if operation.ObservedDigests == nil {
			operation.ObservedDigests = map[string]string{}
		}
		for key, value := range event.ObservedDigests {
			operation.ObservedDigests[key] = value
		}
	}
	operation.UpdatedAt = now
	if operation.StageTimes == nil {
		operation.StageTimes = map[domain.Status]time.Time{}
	}
	operation.StageTimes[status] = now
	if status.IsTerminal() {
		operation.CompletedAt = &now
	}
	updated, err := service.persistReconciledOperation(ctx, operation, previousStatus)
	if err != nil {
		return nil, err
	}
	return service.reconcileAfterHostEvent(ctx, updated)
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
	if err := domain.ValidateTransition(previous, status); err != nil {
		return nil, err
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
	return service.persistReconciledOperation(ctx, operation, previous)
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

func (service *Service) persistReconciledOperation(ctx context.Context, operation *domain.Operation, expected domain.Status) (*domain.Operation, error) {
	if transitionRepository, ok := service.repository.(domain.TransitionRepository); ok {
		if err := transitionRepository.UpdateTransition(ctx, operation, expected); err != nil {
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
	if err := service.repository.Update(ctx, operation); err != nil {
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
