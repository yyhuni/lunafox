package repository

import (
	"context"
	"time"

	scheduledapp "github.com/yyhuni/lunafox/server/internal/modules/scheduledscan/application"
	"github.com/yyhuni/lunafox/server/internal/pkg/scope"
)

// occurrenceStatusPredicateSQL mirrors application.DeriveOccurrenceStatus in
// SQL for the status aggregation. The two MUST stay in sync; the repository
// contract tests assert every status branch against the Go derivation.
const occurrenceStatusPredicateSQL = `COUNT(CASE WHEN dispatched_at IS NOT NULL THEN 1 END) AS succeeded,
COUNT(CASE WHEN dispatched_at IS NULL AND failure_kind IS NOT NULL THEN 1 END) AS failed,
COUNT(CASE WHEN dispatched_at IS NULL AND failure_kind IS NULL AND attempted_at IS NOT NULL AND next_retry_at IS NOT NULL THEN 1 END) AS retrying,
COUNT(CASE WHEN dispatched_at IS NULL AND failure_kind IS NULL AND attempted_at IS NOT NULL AND next_retry_at IS NULL THEN 1 END) AS dispatching,
COUNT(CASE WHEN attempted_at IS NULL THEN 1 END) AS pending`

func (repo *ScheduledScanRepository) ListOccurrences(ctx context.Context, query scheduledapp.OccurrenceHistoryQuery) ([]scheduledapp.OccurrenceSnapshot, int64, error) {
	if repo == nil || repo.db == nil || query.ScheduledScanID <= 0 {
		return nil, 0, scheduledapp.ErrScheduledScanInvalidArgument
	}
	base := repo.db.WithContext(ctx).Model(&scheduledScanOccurrenceModel{}).
		Where("scheduled_scan_id = ?", query.ScheduledScanID)
	var total int64
	if err := base.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var models []scheduledScanOccurrenceModel
	if err := base.
		Scopes(scope.WithPagination(query.Page, query.PageSize)).
		Order("scheduled_for DESC").
		Order("id DESC").
		Find(&models).Error; err != nil {
		return nil, 0, err
	}
	out := make([]scheduledapp.OccurrenceSnapshot, 0, len(models))
	for index := range models {
		out = append(out, occurrenceModelToSnapshot(&models[index]))
	}
	return out, total, nil
}

func (repo *ScheduledScanRepository) CountOccurrencesByStatus(ctx context.Context, scheduledScanID int) (scheduledapp.OccurrenceStatusCounts, error) {
	if repo == nil || repo.db == nil || scheduledScanID <= 0 {
		return scheduledapp.OccurrenceStatusCounts{}, scheduledapp.ErrScheduledScanInvalidArgument
	}
	var row struct {
		Succeeded   int64
		Failed      int64
		Retrying    int64
		Dispatching int64
		Pending     int64
	}
	err := repo.db.WithContext(ctx).Model(&scheduledScanOccurrenceModel{}).
		Select(occurrenceStatusPredicateSQL).
		Where("scheduled_scan_id = ?", scheduledScanID).
		Scan(&row).Error
	if err != nil {
		return scheduledapp.OccurrenceStatusCounts{}, err
	}
	return scheduledapp.OccurrenceStatusCounts{
		Succeeded:   row.Succeeded,
		Failed:      row.Failed,
		Retrying:    row.Retrying,
		Dispatching: row.Dispatching,
		Pending:     row.Pending,
	}, nil
}

func occurrenceModelToSnapshot(model *scheduledScanOccurrenceModel) scheduledapp.OccurrenceSnapshot {
	return scheduledapp.OccurrenceSnapshot{
		ID:               model.ID,
		ScheduledScanID:  model.ScheduledScanID,
		ScheduledFor:     model.ScheduledFor,
		AttemptedAt:      cloneTimePointer(model.AttemptedAt),
		DispatchedAt:     cloneTimePointer(model.DispatchedAt),
		FailureKind:      cloneStringPointer(model.FailureKind),
		FailureMessage:   cloneStringPointer(model.FailureMessage),
		RetryCount:       model.RetryCount,
		NextRetryAt:      cloneTimePointer(model.NextRetryAt),
		LastFailureCause: cloneStringPointer(model.LastFailureCause),
	}
}

func cloneTimePointer(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}

func cloneStringPointer(value *string) *string {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}

var _ scheduledapp.ScheduledScanOccurrenceStore = (*ScheduledScanRepository)(nil)
