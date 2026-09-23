package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/yyhuni/lunafox/server/internal/modules/upgrade/domain"
	model "github.com/yyhuni/lunafox/server/internal/modules/upgrade/repository/persistence"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type UpgradeOperationRepository struct{ db *gorm.DB }

func NewUpgradeOperationRepository(db *gorm.DB) *UpgradeOperationRepository {
	if db == nil {
		panic("upgrade operation database is required")
	}
	return &UpgradeOperationRepository{db: db}
}

func (repository *UpgradeOperationRepository) CreateOrGet(ctx context.Context, operation *domain.Operation) (*domain.Operation, bool, error) {
	if repository == nil || repository.db == nil || operation == nil {
		return nil, false, fmt.Errorf("upgrade operation repository is not configured")
	}
	if err := operation.Validate(); err != nil {
		return nil, false, err
	}
	var result *domain.Operation
	created := false
	err := repository.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var existing model.Operation
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("request_id = ?", operation.RequestID).First(&existing).Error
		if err == nil {
			if existing.ManifestID != operation.ManifestID || existing.ManifestDigest != operation.ManifestDigest {
				return domain.ErrUpgradeRequestConflict
			}
			result = fromModel(&existing)
			return nil
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		var active model.Operation
		activeQuery := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("status NOT IN ?", terminalStatuses())
		if err := activeQuery.First(&active).Error; err == nil {
			return domain.ErrUpgradeAlreadyRunning
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		record, err := toModel(operation)
		if err != nil {
			return err
		}
		if err := tx.Create(record).Error; err != nil {
			// A concurrent creator may win either unique index. Re-read the
			// request row and expose deterministic domain semantics.
			var replay model.Operation
			if readErr := tx.Where("request_id = ?", operation.RequestID).First(&replay).Error; readErr == nil {
				if replay.ManifestID != operation.ManifestID || replay.ManifestDigest != operation.ManifestDigest {
					return domain.ErrUpgradeRequestConflict
				}
				result = fromModel(&replay)
				return nil
			}
			return err
		}
		result = fromModel(record)
		created = true
		return nil
	})
	if err != nil {
		return nil, false, err
	}
	return result, created, nil
}

func (repository *UpgradeOperationRepository) Get(ctx context.Context, operationID string) (*domain.Operation, error) {
	if strings.TrimSpace(operationID) == "" {
		return nil, domain.ErrUpgradeNotFound
	}
	var record model.Operation
	if err := repository.db.WithContext(ctx).Where("id = ?", operationID).First(&record).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrUpgradeNotFound
		}
		return nil, err
	}
	return fromModel(&record), nil
}

// GetByRequest is used by the application idempotency fast path. A missing
// request is reported as ErrUpgradeNotFound so callers can continue creating
// a new Operation without interpreting storage errors as an empty result.
func (repository *UpgradeOperationRepository) GetByRequest(ctx context.Context, requestID string) (*domain.Operation, error) {
	if repository == nil || repository.db == nil {
		return nil, domain.ErrUpgradeNotFound
	}
	requestID = strings.TrimSpace(requestID)
	if requestID == "" {
		return nil, domain.ErrUpgradeNotFound
	}
	var record model.Operation
	if err := repository.db.WithContext(ctx).Where("request_id = ?", requestID).First(&record).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrUpgradeNotFound
		}
		return nil, err
	}
	return fromModel(&record), nil
}

func (repository *UpgradeOperationRepository) Update(ctx context.Context, operation *domain.Operation) error {
	if repository == nil || repository.db == nil || operation == nil {
		return fmt.Errorf("upgrade operation repository is not configured")
	}
	if err := operation.Validate(); err != nil {
		return err
	}
	record, err := toModel(operation)
	if err != nil {
		return err
	}
	// Keep the legacy Update surface usable by bootstrap/tests while still
	// preventing a delayed event from moving a durable operation backwards or
	// overwriting a terminal result. The stricter lifecycle graph is enforced by
	// UpdateTransition, which event/retry paths use in production.
	var current model.Operation
	if err := repository.db.WithContext(ctx).Where("id = ?", record.ID).First(&current).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return domain.ErrUpgradeNotFound
		}
		return err
	}
	currentStatus := domain.Status(current.Status)
	requestedStatus := domain.Status(record.Status)
	if currentStatus.IsTerminal() && currentStatus != requestedStatus {
		return fmt.Errorf("terminal upgrade operation cannot transition from %q to %q", currentStatus, requestedStatus)
	}
	if !currentStatus.IsTerminal() && statusRankForRepository(requestedStatus) < statusRankForRepository(currentStatus) {
		return fmt.Errorf("upgrade operation cannot move backwards from %q to %q", currentStatus, requestedStatus)
	}
	if !current.UpdatedAt.IsZero() && !record.UpdatedAt.IsZero() && record.UpdatedAt.Before(current.UpdatedAt) {
		// A stale host event is harmlessly ignored. Returning nil makes retries
		// idempotent while ensuring the older payload cannot overwrite evidence.
		return nil
	}
	result := repository.db.WithContext(ctx).Model(&model.Operation{}).Where("id = ? AND updated_at <= ?", record.ID, record.UpdatedAt).Updates(map[string]any{
		"status": record.Status, "migration_status": record.MigrationStatus,
		"execution_mode": record.ExecutionMode, "work_disposition": record.WorkDisposition,
		"plan_summary": record.PlanSummary, "plan_digest": record.PlanDigest,
		"baseline_deployment_digest":   record.BaselineDeploymentDigest,
		"confirmed_deployment_version": record.ConfirmedDeploymentVersion,
		"cancelled_scan_count":         record.CancelledScanCount, "cancelled_task_count": record.CancelledTaskCount,
		"agent_desired_version": record.AgentDesiredVersion, "agent_target_digest": record.AgentTargetDigest,
		"agent_expected_count": record.AgentExpectedCount, "agent_ready_count": record.AgentReadyCount,
		"agent_missing_count": record.AgentMissingCount, "agent_unhealthy_count": record.AgentUnhealthyCount,
		"agent_expectations": record.AgentExpectations, "agent_verification_deadline": record.AgentVerificationDeadline,
		"observed_digests": record.ObservedDigests, "stage_times": record.StageTimes, "progress_events": record.ProgressEvents,
		"diagnostic": record.Diagnostic, "updated_at": record.UpdatedAt, "completed_at": record.CompletedAt,
	})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return domain.ErrUpgradeNotFound
	}
	return nil
}

// UpdateTransition atomically applies one lifecycle transition using the
// caller's observed status as a compare-and-set fence. A concurrent writer
// either wins the row lock or receives ErrUpgradeTransitionConflict; it can
// never overwrite a newer phase with a late event.
func (repository *UpgradeOperationRepository) UpdateTransition(ctx context.Context, operation *domain.Operation, expected domain.Status) error {
	return repository.updateTransition(ctx, operation, expected, false)
}

// UpdateJournalRecoveryTransition atomically persists a forward journal replay
// after the application has validated its host-only provenance and evidence.
// It intentionally uses a separate domain guard so ordinary host events remain
// constrained to the strict lifecycle graph.
func (repository *UpgradeOperationRepository) UpdateJournalRecoveryTransition(ctx context.Context, operation *domain.Operation, expected domain.Status) error {
	return repository.updateTransition(ctx, operation, expected, true)
}

func (repository *UpgradeOperationRepository) updateTransition(ctx context.Context, operation *domain.Operation, expected domain.Status, journalRecovery bool) error {
	if repository == nil || repository.db == nil || operation == nil {
		return fmt.Errorf("upgrade operation repository is not configured")
	}
	if err := operation.Validate(); err != nil {
		return err
	}
	if !expected.Valid() {
		return fmt.Errorf("expected upgrade status %q is invalid", expected)
	}
	var transitionErr error
	if journalRecovery {
		transitionErr = domain.ValidateJournalRecoveryTransition(operation.EffectiveExecutionMode(), operation.MigrationType, operation.MigrationStatus, expected, operation.Status)
	} else {
		transitionErr = domain.ValidateExecutionTransition(operation.EffectiveExecutionMode(), expected, operation.Status)
	}
	if transitionErr != nil {
		return transitionErr
	}
	record, err := toModel(operation)
	if err != nil {
		return err
	}
	result := repository.db.WithContext(ctx).Model(&model.Operation{}).
		Where("id = ? AND status = ? AND manifest_digest = ?", record.ID, string(expected), record.ManifestDigest).
		Updates(map[string]any{
			"status": record.Status, "migration_status": record.MigrationStatus,
			"execution_mode": record.ExecutionMode, "work_disposition": record.WorkDisposition,
			"plan_summary": record.PlanSummary, "plan_digest": record.PlanDigest,
			"baseline_deployment_digest":   record.BaselineDeploymentDigest,
			"confirmed_deployment_version": record.ConfirmedDeploymentVersion,
			"cancelled_scan_count":         record.CancelledScanCount, "cancelled_task_count": record.CancelledTaskCount,
			"agent_desired_version": record.AgentDesiredVersion, "agent_target_digest": record.AgentTargetDigest,
			"agent_expected_count": record.AgentExpectedCount, "agent_ready_count": record.AgentReadyCount,
			"agent_missing_count": record.AgentMissingCount, "agent_unhealthy_count": record.AgentUnhealthyCount,
			"agent_expectations": record.AgentExpectations, "agent_verification_deadline": record.AgentVerificationDeadline,
			"observed_digests": record.ObservedDigests,
			"stage_times":      record.StageTimes, "progress_events": record.ProgressEvents, "diagnostic": record.Diagnostic,
			"updated_at": record.UpdatedAt, "completed_at": record.CompletedAt,
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return domain.ErrUpgradeTransitionConflict
	}
	return nil
}

// ResetForRetry is the only persistence path that may reopen a terminal
// operation. Keeping it as an explicit compare-and-set transaction prevents a
// late host event from racing a repair and silently replacing the repair's
// state.
func (repository *UpgradeOperationRepository) ResetForRetry(ctx context.Context, operation *domain.Operation, expected domain.Status) error {
	if repository == nil || repository.db == nil || operation == nil {
		return fmt.Errorf("upgrade operation repository is not configured")
	}
	if err := operation.Validate(); err != nil {
		return err
	}
	if !expected.IsTerminal() || expected == domain.StatusSucceeded {
		return fmt.Errorf("retry reset expected a retryable terminal status, got %q", expected)
	}
	if operation.Status.IsTerminal() {
		return fmt.Errorf("retry reset target must be active, got %q", operation.Status)
	}
	record, err := toModel(operation)
	if err != nil {
		return err
	}
	var rows int64
	err = repository.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var current model.Operation
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", record.ID).First(&current).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return domain.ErrUpgradeNotFound
			}
			return err
		}
		if domain.Status(current.Status) != expected {
			return domain.ErrUpgradeTransitionConflict
		}
		if current.ManifestID != record.ManifestID || current.ManifestDigest != record.ManifestDigest || current.RequestID != record.RequestID {
			return domain.ErrUpgradeTransitionConflict
		}
		result := tx.Model(&model.Operation{}).
			Where("id = ? AND status = ? AND manifest_digest = ?", record.ID, string(expected), record.ManifestDigest).
			Updates(map[string]any{
				"status": record.Status, "migration_status": record.MigrationStatus,
				"execution_mode": record.ExecutionMode, "work_disposition": record.WorkDisposition,
				"plan_summary": record.PlanSummary, "plan_digest": record.PlanDigest,
				"baseline_deployment_digest":   record.BaselineDeploymentDigest,
				"confirmed_deployment_version": record.ConfirmedDeploymentVersion,
				"cancelled_scan_count":         record.CancelledScanCount, "cancelled_task_count": record.CancelledTaskCount,
				"agent_desired_version": record.AgentDesiredVersion, "agent_target_digest": record.AgentTargetDigest,
				"agent_expected_count": record.AgentExpectedCount, "agent_ready_count": record.AgentReadyCount,
				"agent_missing_count": record.AgentMissingCount, "agent_unhealthy_count": record.AgentUnhealthyCount,
				"agent_expectations": record.AgentExpectations, "agent_verification_deadline": record.AgentVerificationDeadline,
				"observed_digests": record.ObservedDigests,
				"stage_times":      record.StageTimes, "progress_events": record.ProgressEvents, "diagnostic": record.Diagnostic,
				"updated_at": record.UpdatedAt, "completed_at": record.CompletedAt,
			})
		if result.Error != nil {
			return result.Error
		}
		rows = result.RowsAffected
		return nil
	})
	if err != nil {
		return err
	}
	if rows != 1 {
		return domain.ErrUpgradeTransitionConflict
	}
	return nil
}

func statusRankForRepository(status domain.Status) int {
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
	case domain.StatusSucceeded, domain.StatusFailed:
		return 8
	case domain.StatusNeedsRecovery, domain.StatusNeedsAttention:
		return 9
	default:
		return -1
	}
}

func (repository *UpgradeOperationRepository) FindActive(ctx context.Context) (*domain.Operation, error) {
	var record model.Operation
	if err := repository.db.WithContext(ctx).Where("status NOT IN ?", terminalStatuses()).Order("created_at ASC").First(&record).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, gorm.ErrRecordNotFound
		}
		return nil, err
	}
	return fromModel(&record), nil
}

func terminalStatuses() []string {
	return []string{string(domain.StatusSucceeded), string(domain.StatusFailed), string(domain.StatusNeedsRecovery), string(domain.StatusNeedsAttention)}
}

func toModel(operation *domain.Operation) (*model.Operation, error) {
	if err := validateOperationEvidence(operation); err != nil {
		return nil, err
	}
	observed, err := json.Marshal(nonNilStringMap(operation.ObservedDigests))
	if err != nil {
		return nil, err
	}
	stageTimes, err := json.Marshal(nonNilStageTimes(operation.StageTimes))
	if err != nil {
		return nil, err
	}
	progressEvents, err := json.Marshal(nonNilProgressEvents(operation.ProgressEvents))
	if err != nil {
		return nil, err
	}
	agentExpectations, err := json.Marshal(nonNilAgentExpectations(operation.AgentExpectations))
	if err != nil {
		return nil, err
	}
	planSummary, err := json.Marshal(planSummaryForPersistence(operation))
	if err != nil {
		return nil, err
	}
	return &model.Operation{
		ID: operation.OperationID, RequestID: operation.RequestID, OperatorID: operation.OperatorID,
		ManifestID: operation.ManifestID, ManifestDigest: operation.ManifestDigest, ReleaseVersion: operation.ReleaseVersion,
		CompatibilityRange: operation.CompatibilityRange, MaintenanceWindowMinutes: operation.MaintenanceWindowMinutes,
		Status: string(operation.Status), MigrationStatus: string(operation.MigrationStatus), MigrationType: operation.MigrationType,
		MigrationID: operation.MigrationID, MigrationChecksum: operation.MigrationChecksum,
		ExecutionMode: string(operation.ExecutionMode), WorkDisposition: string(operation.WorkDisposition),
		PlanSummary: planSummary, PlanDigest: operation.PlanDigest, BaselineDeploymentDigest: operation.BaselineDeploymentDigest,
		ConfirmedDeploymentVersion: operation.ConfirmedDeploymentVersion,
		CancelledScanCount:         operation.CancelledScanCount, CancelledTaskCount: operation.CancelledTaskCount,
		AgentDesiredVersion: operation.AgentDesiredVersion, AgentTargetDigest: operation.AgentTargetDigest,
		AgentExpectedCount: operation.AgentSummary.Expected, AgentReadyCount: operation.AgentSummary.Ready,
		AgentMissingCount: operation.AgentSummary.Missing, AgentUnhealthyCount: operation.AgentSummary.Unhealthy,
		AgentExpectations: agentExpectations, AgentVerificationDeadline: operation.AgentVerificationDeadline,
		ObservedDigests: observed, StageTimes: stageTimes, ProgressEvents: progressEvents, Diagnostic: operation.Diagnostic,
		CreatedAt: operation.CreatedAt.UTC(), UpdatedAt: operation.UpdatedAt.UTC(), CompletedAt: operation.CompletedAt,
	}, nil
}

func fromModel(record *model.Operation) *domain.Operation {
	agentExpectations := []domain.AgentExpectation{}
	_ = json.Unmarshal(record.AgentExpectations, &agentExpectations)
	observed := map[string]string{}
	_ = json.Unmarshal(record.ObservedDigests, &observed)
	stageTimesRaw := map[string]time.Time{}
	_ = json.Unmarshal(record.StageTimes, &stageTimesRaw)
	stageTimes := map[domain.Status]time.Time{}
	for key, value := range stageTimesRaw {
		stageTimes[domain.Status(key)] = value.UTC()
	}
	progressEvents := []domain.ProgressEvent{}
	_ = json.Unmarshal(record.ProgressEvents, &progressEvents)
	if err := domain.ValidateProgressEvents(progressEvents); err != nil {
		// A legacy/corrupt row must not become a browser-visible raw-output
		// channel. The DTO applies a second boundary, while repository reads fail
		// closed for the progress projection itself.
		progressEvents = []domain.ProgressEvent{}
	}
	planSummary := domain.PlanSummary{}
	_ = json.Unmarshal(record.PlanSummary, &planSummary)
	operation := &domain.Operation{OperationID: record.ID, RequestID: record.RequestID, OperatorID: record.OperatorID,
		ManifestID: record.ManifestID, ManifestDigest: record.ManifestDigest, ReleaseVersion: record.ReleaseVersion,
		CompatibilityRange: record.CompatibilityRange, MaintenanceWindowMinutes: record.MaintenanceWindowMinutes,
		Status: domain.Status(record.Status), MigrationStatus: domain.MigrationStatus(record.MigrationStatus), MigrationType: record.MigrationType,
		MigrationID: record.MigrationID, MigrationChecksum: record.MigrationChecksum,
		PlanDigest: record.PlanDigest, BaselineDeploymentDigest: record.BaselineDeploymentDigest,
		ConfirmedDeploymentVersion: record.ConfirmedDeploymentVersion, CancelledScanCount: record.CancelledScanCount,
		CancelledTaskCount: record.CancelledTaskCount, AgentDesiredVersion: record.AgentDesiredVersion, AgentTargetDigest: record.AgentTargetDigest,
		AgentSummary:      domain.AgentSummary{Expected: record.AgentExpectedCount, Ready: record.AgentReadyCount, Missing: record.AgentMissingCount, Unhealthy: record.AgentUnhealthyCount},
		AgentExpectations: agentExpectations, AgentVerificationDeadline: record.AgentVerificationDeadline,
		ObservedDigests: observed, ProgressEvents: progressEvents, Diagnostic: record.Diagnostic, StageTimes: stageTimes,
		CreatedAt: record.CreatedAt.UTC(), UpdatedAt: record.UpdatedAt.UTC(), CompletedAt: record.CompletedAt,
	}
	// An empty value is the only legacy form. Do not manufacture a scoped plan
	// from old database rows; callers project its effective full/unknown facts.
	if record.ExecutionMode != "" {
		operation.ExecutionMode = domain.ExecutionMode(record.ExecutionMode)
		operation.PlanSummary = planSummary
	}
	if record.WorkDisposition != "" {
		operation.WorkDisposition = domain.WorkDisposition(record.WorkDisposition)
	}
	return operation
}

func planSummaryForPersistence(operation *domain.Operation) domain.PlanSummary {
	if operation == nil || operation.ExecutionMode == "" {
		return domain.PlanSummary{TouchedServices: []string{}}
	}
	return operation.PlanSummary.Clone()
}

func nonNilAgentExpectations(value []domain.AgentExpectation) []domain.AgentExpectation {
	if value == nil {
		return []domain.AgentExpectation{}
	}
	return value
}

func validateOperationEvidence(operation *domain.Operation) error {
	if operation == nil {
		return fmt.Errorf("upgrade operation is required")
	}
	for _, expectation := range operation.AgentExpectations {
		if expectation.AgentID <= 0 {
			return fmt.Errorf("agent expectation requires a positive agentId")
		}
		if strings.ContainsAny(expectation.Diagnostic, "\r\n") || len(expectation.Diagnostic) > 512 {
			return fmt.Errorf("agent expectation diagnostic is invalid")
		}
	}
	if strings.ContainsAny(operation.Diagnostic, "\r\n") || len(operation.Diagnostic) > 2048 {
		return fmt.Errorf("upgrade diagnostic is invalid")
	}
	return nil
}

func nonNilStringMap(value map[string]string) map[string]string {
	if value == nil {
		return map[string]string{}
	}
	return value
}

func nonNilStageTimes(value map[domain.Status]time.Time) map[string]time.Time {
	result := map[string]time.Time{}
	for key, timestamp := range value {
		result[string(key)] = timestamp.UTC()
	}
	return result
}

func nonNilProgressEvents(value []domain.ProgressEvent) []domain.ProgressEvent {
	if value == nil {
		return []domain.ProgressEvent{}
	}
	return domain.CloneProgressEvents(value)
}

var _ domain.Repository = (*UpgradeOperationRepository)(nil)
var _ domain.RetryRepository = (*UpgradeOperationRepository)(nil)
var _ interface {
	GetByRequest(context.Context, string) (*domain.Operation, error)
} = (*UpgradeOperationRepository)(nil)
