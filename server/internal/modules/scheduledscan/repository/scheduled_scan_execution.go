package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	scandomain "github.com/yyhuni/lunafox/server/internal/modules/scan/domain"
	scheduledapp "github.com/yyhuni/lunafox/server/internal/modules/scheduledscan/application"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func (repo *ScheduledScanRepository) ListDueSchedules(ctx context.Context, evaluationAt time.Time) ([]scheduledapp.DueSchedule, error) {
	var rows []struct {
		ID          int
		NextRunTime time.Time `gorm:"column:next_run_time"`
	}
	err := activeScheduledScanQuery(repo.db.WithContext(ctx).Model(&scheduledScanModel{})).
		Select("scheduled_scan.id, scheduled_scan.next_run_time").
		Where("scheduled_scan.is_enabled = TRUE AND scheduled_scan.next_run_time IS NOT NULL AND scheduled_scan.next_run_time <= ?", evaluationAt.UTC()).
		Order("scheduled_scan.next_run_time ASC").
		Order("scheduled_scan.id ASC").
		Limit(scheduledapp.DueScheduleBatchSize).
		Find(&rows).Error
	if err != nil {
		return nil, err
	}
	out := make([]scheduledapp.DueSchedule, 0, len(rows))
	for _, row := range rows {
		out = append(out, scheduledapp.DueSchedule{ID: row.ID, NextRunTime: row.NextRunTime.UTC()})
	}
	return out, nil
}

func (repo *ScheduledScanRepository) MaterializeDue(ctx context.Context, scheduledScanID int, evaluationAt time.Time) (bool, error) {
	if repo == nil || repo.db == nil || repo.calculator == nil || scheduledScanID <= 0 {
		return false, scheduledapp.ErrScheduledScanInvalidArgument
	}
	evaluationAt = evaluationAt.UTC()
	committed := false
	err := repo.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		ownership, exists, err := scheduledScanRuntimeOwnership(tx, scheduledScanID)
		if err != nil || !exists {
			return err
		}
		active, err := lockActiveScheduledScanRuntimeTarget(tx, ownership)
		if err != nil || !active {
			return err
		}
		var schedule scheduledScanModel
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", scheduledScanID).First(&schedule).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil
			}
			return err
		}
		if !sameOptionalInt(ownership, schedule.TargetID) {
			// An Update can rebind a Schedule between the initial ownership read
			// and its row lock. Leave it for the next reconciliation pass, which
			// will lock the actual owner before creating an occurrence.
			return nil
		}
		if !schedule.IsEnabled || schedule.NextRunTime == nil || schedule.NextRunTime.After(evaluationAt) {
			return nil
		}

		scheduledFor, err := repo.calculator.LatestAtOrBefore(schedule.CronExpression, *schedule.NextRunTime, evaluationAt)
		if err != nil {
			return err
		}
		nextRunTime, err := repo.calculator.AdvanceAfter(schedule.CronExpression, evaluationAt)
		if err != nil {
			return err
		}
		occurrence := &scheduledScanOccurrenceModel{
			ScheduledScanID: scheduledScanID,
			ScheduledFor:    scheduledFor.UTC(),
		}
		if err := tx.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "scheduled_scan_id"}, {Name: "scheduled_for"}},
			DoNothing: true,
		}).Create(occurrence).Error; err != nil {
			return err
		}
		if err := tx.Model(&scheduledScanModel{}).Where("id = ?", scheduledScanID).Update("next_run_time", nextRunTime.UTC()).Error; err != nil {
			return err
		}
		committed = true
		return nil
	})
	return committed, err
}

func (repo *ScheduledScanRepository) SelectAttemptCandidate(ctx context.Context) (*scheduledapp.OccurrenceCandidate, error) {
	var row struct {
		ID              int64
		ScheduledScanID int       `gorm:"column:scheduled_scan_id"`
		ScheduledFor    time.Time `gorm:"column:scheduled_for"`
	}
	query := `
			SELECT occurrence.id, occurrence.scheduled_scan_id, occurrence.scheduled_for
		FROM scheduled_scan AS schedule
		JOIN scheduled_scan_occurrence AS occurrence
			ON occurrence.scheduled_scan_id = schedule.id
		LEFT JOIN target AS active_target
			ON active_target.id = schedule.target_id AND active_target.deleted_at IS NULL
		WHERE schedule.is_enabled = TRUE
			AND (schedule.target_id IS NULL OR active_target.id IS NOT NULL)
			AND occurrence.attempted_at IS NULL
			AND occurrence.id = (
				SELECT oldest.id
				FROM scheduled_scan_occurrence AS oldest
				WHERE oldest.scheduled_scan_id = schedule.id
					AND oldest.attempted_at IS NULL
				ORDER BY oldest.scheduled_for ASC, oldest.id ASC
				LIMIT 1
			)
			ORDER BY schedule.last_run_time ASC NULLS FIRST, schedule.id ASC, occurrence.id ASC
			LIMIT 1`
	if repo.db.Dialector.Name() == "postgres" {
		query = `
			SELECT occurrence.id, occurrence.scheduled_scan_id, occurrence.scheduled_for
			FROM scheduled_scan AS schedule
			JOIN LATERAL (
				SELECT oldest.id, oldest.scheduled_scan_id, oldest.scheduled_for
				FROM scheduled_scan_occurrence AS oldest
				WHERE oldest.scheduled_scan_id = schedule.id
					AND oldest.attempted_at IS NULL
				ORDER BY oldest.scheduled_for ASC, oldest.id ASC
				LIMIT 1
			) AS occurrence ON TRUE
			LEFT JOIN target AS active_target
				ON active_target.id = schedule.target_id AND active_target.deleted_at IS NULL
			WHERE schedule.is_enabled = TRUE
				AND (schedule.target_id IS NULL OR active_target.id IS NOT NULL)
			ORDER BY schedule.last_run_time ASC NULLS FIRST, schedule.id ASC, occurrence.id ASC
			LIMIT 1`
	}
	result := repo.db.WithContext(ctx).Raw(query).Scan(&row)
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 0 || row.ID == 0 {
		return nil, nil
	}
	return &scheduledapp.OccurrenceCandidate{
		ID:              row.ID,
		ScheduledScanID: row.ScheduledScanID,
		ScheduledFor:    row.ScheduledFor.UTC(),
	}, nil
}

func (repo *ScheduledScanRepository) StartAttempt(
	ctx context.Context,
	candidate scheduledapp.OccurrenceCandidate,
	attemptedAt time.Time,
) (*scheduledapp.FrozenDispatchInput, error) {
	if repo == nil || repo.db == nil || candidate.ID <= 0 || candidate.ScheduledScanID <= 0 {
		return nil, nil
	}
	attemptedAt = attemptedAt.UTC()
	var frozen *scheduledapp.FrozenDispatchInput
	err := repo.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		ownership, exists, err := scheduledScanRuntimeOwnership(tx, candidate.ScheduledScanID)
		if err != nil || !exists {
			return err
		}
		active, err := lockActiveScheduledScanRuntimeTarget(tx, ownership)
		if err != nil || !active {
			return err
		}
		var schedule scheduledScanModel
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", candidate.ScheduledScanID).First(&schedule).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil
			}
			return err
		}
		if !sameOptionalInt(ownership, schedule.TargetID) {
			return nil
		}
		if !schedule.IsEnabled {
			return nil
		}

		var occurrence scheduledScanOccurrenceModel
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ? AND scheduled_scan_id = ? AND attempted_at IS NULL", candidate.ID, candidate.ScheduledScanID).
			First(&occurrence).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil
			}
			return err
		}
		result := tx.Model(&scheduledScanOccurrenceModel{}).
			Where("id = ? AND attempted_at IS NULL", occurrence.ID).
			Update("attempted_at", attemptedAt)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return nil
		}
		if err := tx.Model(&scheduledScanModel{}).Where("id = ?", schedule.ID).Updates(map[string]any{
			"run_count":     gorm.Expr("run_count + 1"),
			"last_run_time": attemptedAt,
		}).Error; err != nil {
			return err
		}
		configuration, err := decodeJSONMapStrict(schedule.Configuration)
		if err != nil {
			return fmt.Errorf("decode scheduled scan configuration: %w", err)
		}
		inputSource, ok := scandomain.ParseDatabaseInputSource(schedule.InputSource)
		if !ok {
			return fmt.Errorf("%w: persisted value %q", scandomain.ErrInvalidInputSource, schedule.InputSource)
		}
		targetIDs, targetScoped, err := freezeScheduledScanTargetIDs(tx, &schedule)
		if err != nil {
			return err
		}
		frozen = &scheduledapp.FrozenDispatchInput{
			OccurrenceID:    occurrence.ID,
			ScheduledScanID: schedule.ID,
			ScheduledFor:    occurrence.ScheduledFor.UTC(),
			ScanWorkflowID:  schedule.ScanWorkflowID,
			Configuration:   configuration,
			InputSource:     inputSource,
			TargetIDs:       append([]int(nil), targetIDs...),
			TargetScoped:    targetScoped,
			AgentID:         cloneIntPtr(schedule.AgentID),
		}
		return nil
	})
	return frozen, err
}

func scheduledScanRuntimeOwnership(tx *gorm.DB, scheduledScanID int) (*int, bool, error) {
	var ownership struct {
		TargetID *int `gorm:"column:target_id"`
	}
	if err := tx.Model(&scheduledScanModel{}).Select("target_id").Where("id = ?", scheduledScanID).Take(&ownership).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, false, nil
		}
		return nil, false, err
	}
	return cloneIntPtr(ownership.TargetID), true, nil
}

func lockActiveScheduledScanRuntimeTarget(tx *gorm.DB, targetID *int) (bool, error) {
	if targetID == nil {
		return true, nil
	}
	var target targetRefModel
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("id = ? AND deleted_at IS NULL", *targetID).
		First(&target).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func sameOptionalInt(left, right *int) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}
	return *left == *right
}

func freezeScheduledScanTargetIDs(tx *gorm.DB, schedule *scheduledScanModel) ([]int, bool, error) {
	if tx == nil || schedule == nil {
		return nil, false, scheduledapp.ErrScheduledScanInvalidArgument
	}
	if schedule.TargetID != nil {
		if *schedule.TargetID <= 0 || schedule.OrganizationID != nil {
			return nil, false, scheduledapp.ErrScheduledScanInvalidArgument
		}
		return []int{*schedule.TargetID}, true, nil
	}
	if schedule.OrganizationID == nil || *schedule.OrganizationID <= 0 {
		return nil, false, scheduledapp.ErrScheduledScanInvalidArgument
	}

	var targetIDs []int
	if err := tx.Table("organization_target AS membership").
		Select("membership.target_id").
		Joins("JOIN target AS active_target ON active_target.id = membership.target_id AND active_target.deleted_at IS NULL").
		Where("membership.organization_id = ?", *schedule.OrganizationID).
		Order("membership.target_id ASC").
		Scan(&targetIDs).Error; err != nil {
		return nil, false, err
	}
	return targetIDs, false, nil
}

func (repo *ScheduledScanRepository) RecordOutcome(
	ctx context.Context,
	occurrenceID int64,
	outcome scheduledapp.HandoffOutcome,
	recordedAt time.Time,
) (bool, error) {
	if repo == nil || repo.db == nil || occurrenceID <= 0 {
		return false, nil
	}
	updates := map[string]any{}
	counterColumn := "failed_handoff_count"
	if outcome.Completed() {
		updates["dispatched_at"] = recordedAt.UTC()
		updates["failure_kind"] = nil
		updates["failure_message"] = nil
		counterColumn = "successful_handoff_count"
	} else {
		kind := strings.TrimSpace(string(outcome.Kind))
		message := strings.TrimSpace(outcome.Message)
		if kind == "" || len(kind) > 100 || message == "" || len(message) > 2000 {
			return false, fmt.Errorf("invalid scheduled scan handoff outcome")
		}
		updates["dispatched_at"] = nil
		updates["failure_kind"] = kind
		updates["failure_message"] = message
	}

	recorded := false
	err := repo.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var occurrence struct {
			ScheduledScanID int `gorm:"column:scheduled_scan_id"`
		}
		if err := tx.Model(&scheduledScanOccurrenceModel{}).
			Select("scheduled_scan_id").
			Where("id = ?", occurrenceID).
			Take(&occurrence).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil
			}
			return err
		}

		// Conditional settlement is the idempotency gate. The aggregate update
		// must commit with it so a retry can never create a second outcome count.
		result := tx.Model(&scheduledScanOccurrenceModel{}).
			Where("id = ? AND attempted_at IS NOT NULL AND dispatched_at IS NULL AND failure_kind IS NULL AND failure_message IS NULL", occurrenceID).
			Updates(updates)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return nil
		}

		result = tx.Model(&scheduledScanModel{}).
			Where("id = ?", occurrence.ScheduledScanID).
			UpdateColumn(counterColumn, gorm.Expr(counterColumn+" + ?", 1))
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return fmt.Errorf("scheduled scan %d disappeared while recording handoff outcome", occurrence.ScheduledScanID)
		}
		recorded = true
		return nil
	})
	if err != nil {
		return false, err
	}
	return recorded, nil
}

func (repo *ScheduledScanRepository) DeleteOccurrenceBatch(ctx context.Context, attemptedAtCutoff time.Time) (int64, error) {
	result := repo.db.WithContext(ctx).Exec(`
		DELETE FROM scheduled_scan_occurrence
		WHERE id IN (
			SELECT id
			FROM scheduled_scan_occurrence
			WHERE attempted_at IS NOT NULL AND attempted_at <= ?
			ORDER BY attempted_at ASC, id ASC
			LIMIT ?
		)`, attemptedAtCutoff.UTC(), scheduledapp.OccurrenceDeleteBatchSize)
	return result.RowsAffected, result.Error
}
