package repository

import (
	"context"
	"strings"
	"time"
	"unicode/utf8"

	cleanupdomain "github.com/yyhuni/lunafox/server/internal/modules/targetcleanup/domain"
	model "github.com/yyhuni/lunafox/server/internal/modules/targetcleanup/repository/persistence"
	"gorm.io/gorm"
)

const maxTargetCleanupLastErrorBytes = 2000

type TargetCleanupRepository struct {
	db *gorm.DB
}

func NewTargetCleanupRepository(db *gorm.DB) *TargetCleanupRepository {
	return &TargetCleanupRepository{db: db}
}

func (repo *TargetCleanupRepository) ListDue(ctx context.Context, dueAt time.Time, limit int) ([]cleanupdomain.CleanupJob, error) {
	if repo == nil || repo.db == nil || limit <= 0 {
		return nil, nil
	}
	var rows []model.TargetCleanupJob
	if err := repo.db.WithContext(ctx).
		Where("status = ? AND next_retry_at <= ?", string(cleanupdomain.CleanupJobPending), dueAt.UTC()).
		Order("next_retry_at ASC").
		Order("id ASC").
		Limit(limit).
		Find(&rows).Error; err != nil {
		return nil, err
	}
	jobs := make([]cleanupdomain.CleanupJob, 0, len(rows))
	for index := range rows {
		jobs = append(jobs, targetCleanupJobModelToDomain(&rows[index]))
	}
	return jobs, nil
}

func (repo *TargetCleanupRepository) RecordFailure(ctx context.Context, jobID, retryCount int, nextRetryAt time.Time, errorClass string) error {
	if repo == nil || repo.db == nil || jobID <= 0 || retryCount <= 0 {
		return nil
	}
	return repo.db.WithContext(ctx).Model(&model.TargetCleanupJob{}).
		Where("id = ? AND status = ?", jobID, string(cleanupdomain.CleanupJobPending)).
		Updates(map[string]any{
			"retry_count":   retryCount,
			"next_retry_at": nextRetryAt.UTC(),
			"last_error":    boundedTargetCleanupDiagnostic(errorClass),
		}).Error
}

func (repo *TargetCleanupRepository) Defer(ctx context.Context, jobID int, nextRetryAt time.Time) error {
	if repo == nil || repo.db == nil || jobID <= 0 {
		return nil
	}
	return repo.db.WithContext(ctx).Model(&model.TargetCleanupJob{}).
		Where("id = ? AND status = ?", jobID, string(cleanupdomain.CleanupJobPending)).
		Update("next_retry_at", nextRetryAt.UTC()).Error
}

func (repo *TargetCleanupRepository) InspectBacklog(ctx context.Context) (cleanupdomain.CleanupBacklog, error) {
	if repo == nil || repo.db == nil {
		return cleanupdomain.CleanupBacklog{}, nil
	}
	query := repo.db.WithContext(ctx).Model(&model.TargetCleanupJob{}).Where("status = ?", string(cleanupdomain.CleanupJobPending))
	var backlog cleanupdomain.CleanupBacklog
	if err := query.Count(&backlog.UnfinishedCount).Error; err != nil {
		return cleanupdomain.CleanupBacklog{}, err
	}
	if backlog.UnfinishedCount == 0 {
		return backlog, nil
	}
	var oldest model.TargetCleanupJob
	if err := query.Select("created_at").Order("created_at ASC").Take(&oldest).Error; err != nil {
		return cleanupdomain.CleanupBacklog{}, err
	}
	backlog.OldestCreatedAt = cloneCleanupTime(&oldest.CreatedAt)
	return backlog, nil
}

func targetCleanupJobModelToDomain(row *model.TargetCleanupJob) cleanupdomain.CleanupJob {
	if row == nil {
		return cleanupdomain.CleanupJob{}
	}
	return cleanupdomain.CleanupJob{
		ID: row.ID, TargetID: row.TargetID, Status: cleanupdomain.CleanupJobStatus(row.Status), RetryCount: row.RetryCount,
		NextRetryAt: row.NextRetryAt.UTC(), LastError: row.LastError, CompletedAt: cloneCleanupTime(row.CompletedAt),
		CreatedAt: row.CreatedAt.UTC(), UpdatedAt: row.UpdatedAt.UTC(),
	}
}

func cloneCleanupTime(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	copy := value.UTC()
	return &copy
}

func boundedTargetCleanupDiagnostic(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "database_or_reconciliation"
	}
	value = strings.Map(func(r rune) rune {
		if r < 0x20 || r == 0x7f {
			return -1
		}
		return r
	}, value)
	for len(value) > maxTargetCleanupLastErrorBytes {
		value = value[:len(value)-1]
	}
	if !utf8.ValidString(value) {
		return "database_or_reconciliation"
	}
	return value
}
