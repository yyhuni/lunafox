package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	scandomain "github.com/yyhuni/lunafox/server/internal/modules/scan/domain"
	scheduledapp "github.com/yyhuni/lunafox/server/internal/modules/scheduledscan/application"
	"github.com/yyhuni/lunafox/server/internal/pkg/scope"
	"github.com/yyhuni/lunafox/server/internal/pkg/timeutil"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type ScheduledScanRepository struct {
	db         *gorm.DB
	calculator scheduledapp.ScheduleCalculator
	now        func() time.Time
}

func NewScheduledScanRepository(db *gorm.DB, calculators ...scheduledapp.ScheduleCalculator) *ScheduledScanRepository {
	calculator := scheduledapp.ScheduleCalculator(scheduledapp.NewCronScheduleCalculator())
	if len(calculators) > 0 && calculators[0] != nil {
		calculator = calculators[0]
	}
	return &ScheduledScanRepository{db: db, calculator: calculator, now: time.Now}
}

func (repo *ScheduledScanRepository) WithClock(now func() time.Time) *ScheduledScanRepository {
	if repo != nil && now != nil {
		repo.now = now
	}
	return repo
}

type scheduledScanModel struct {
	ID                     int            `gorm:"primaryKey;autoIncrement"`
	Name                   string         `gorm:"column:name;size:200;not null"`
	ScanWorkflowID         string         `gorm:"column:scan_workflow_id;size:100;not null"`
	Configuration          datatypes.JSON `gorm:"column:configuration;type:jsonb;not null"`
	InputSource            string         `gorm:"column:input_source;size:32;not null"`
	OrganizationID         *int           `gorm:"column:organization_id"`
	TargetID               *int           `gorm:"column:target_id"`
	AgentID                *int           `gorm:"column:agent_id"`
	CronExpression         string         `gorm:"column:cron_expression;size:100;not null"`
	IsEnabled              bool           `gorm:"column:is_enabled;not null"`
	RunCount               int            `gorm:"column:run_count;not null"`
	SuccessfulHandoffCount int            `gorm:"column:successful_handoff_count;not null"`
	FailedHandoffCount     int            `gorm:"column:failed_handoff_count;not null"`
	LastRunTime            *time.Time     `gorm:"column:last_run_time"`
	NextRunTime            *time.Time     `gorm:"column:next_run_time"`
	CreatedAt              time.Time      `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt              time.Time      `gorm:"column:updated_at;autoUpdateTime"`
	Organization           *organizationRefModel
	Target                 *targetRefModel
}

func (scheduledScanModel) TableName() string {
	return "scheduled_scan"
}

type scheduledScanOccurrenceModel struct {
	ID              int64      `gorm:"primaryKey;autoIncrement"`
	ScheduledScanID int        `gorm:"column:scheduled_scan_id;not null;uniqueIndex:scheduled_scan_occurrence_identity"`
	ScheduledFor    time.Time  `gorm:"column:scheduled_for;not null;uniqueIndex:scheduled_scan_occurrence_identity"`
	AttemptedAt     *time.Time `gorm:"column:attempted_at"`
	DispatchedAt    *time.Time `gorm:"column:dispatched_at"`
	FailureKind     *string    `gorm:"column:failure_kind;size:100"`
	FailureMessage  *string    `gorm:"column:failure_message;size:2000"`
	CreatedAt       time.Time  `gorm:"column:created_at;autoCreateTime"`
}

func (scheduledScanOccurrenceModel) TableName() string {
	return "scheduled_scan_occurrence"
}

func (item *scheduledScanModel) BeforeCreate(*gorm.DB) error {
	if item.Configuration == nil {
		item.Configuration = datatypes.JSON([]byte("{}"))
	}
	return nil
}

type organizationRefModel struct {
	ID   int    `gorm:"primaryKey;column:id"`
	Name string `gorm:"column:name"`
}

func (organizationRefModel) TableName() string {
	return "organization"
}

type targetRefModel struct {
	ID        int        `gorm:"primaryKey;column:id"`
	Name      string     `gorm:"column:name"`
	DeletedAt *time.Time `gorm:"column:deleted_at"`
}

func (targetRefModel) TableName() string {
	return "target"
}

func (repo *ScheduledScanRepository) Create(ctx context.Context, scan *scheduledapp.ScheduledScanCreate) (*scheduledapp.ScheduledScan, error) {
	if repo == nil || repo.db == nil || scan == nil || repo.calculator == nil {
		return nil, scheduledapp.ErrScheduledScanInvalidArgument
	}
	model, err := scheduledScanCreateToModel(scan)
	if err != nil {
		return nil, err
	}
	err = repo.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := lockActiveScheduledScanTargets(tx, intPointersToIDs(model.TargetID)); err != nil {
			return err
		}
		referenceTime := repo.now().UTC()
		if err := repo.calculator.Validate(model.CronExpression); err != nil {
			return err
		}
		if model.IsEnabled {
			nextRunTime, err := repo.calculator.FirstAfter(model.CronExpression, referenceTime)
			if err != nil {
				return err
			}
			model.NextRunTime = &nextRunTime
		} else {
			model.NextRunTime = nil
		}
		return tx.Create(model).Error
	})
	if err != nil {
		return nil, err
	}
	return scheduledScanModelToRecord(model)
}

func (repo *ScheduledScanRepository) Update(ctx context.Context, id int, scan *scheduledapp.ScheduledScanUpdate) (*scheduledapp.ScheduledScan, error) {
	if repo == nil || repo.db == nil || id <= 0 || scan == nil || repo.calculator == nil {
		return nil, scheduledapp.ErrScheduledScanInvalidArgument
	}
	var model scheduledScanModel
	err := repo.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Read only the pre-lock ownership needed to acquire Target-first locks.
		// The locked row is read again below before any update is persisted.
		var ownership struct {
			TargetID *int `gorm:"column:target_id"`
		}
		if err := tx.Model(&scheduledScanModel{}).Select("target_id").Where("id = ?", id).First(&ownership).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return scheduledapp.ErrScheduledScanNotFound
			}
			return err
		}
		candidateTargetIDs := intPointersToIDs(ownership.TargetID, scan.TargetID)
		if err := lockActiveScheduledScanTargets(tx, candidateTargetIDs); err != nil {
			return err
		}
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", id).First(&model).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return scheduledapp.ErrScheduledScanNotFound
			}
			return err
		}

		wasEnabled := model.IsEnabled
		timeRuleChanged := scan.CronExpression != nil
		if err := applyScheduledScanUpdate(&model, scan); err != nil {
			return err
		}
		if err := lockActiveScheduledScanTargets(tx, intPointersToIDs(model.TargetID)); err != nil {
			return err
		}
		if err := repo.calculator.Validate(model.CronExpression); err != nil {
			return err
		}

		referenceTime := repo.now().UTC()
		if err := repo.applyScheduledScanStatusTransition(tx, &model, wasEnabled, referenceTime); err != nil {
			return err
		}
		if model.IsEnabled && wasEnabled && timeRuleChanged {
			nextRunTime, err := repo.calculator.FirstAfter(model.CronExpression, referenceTime)
			if err != nil {
				return err
			}
			model.NextRunTime = &nextRunTime
		}
		model.UpdatedAt = referenceTime
		return tx.Save(&model).Error
	})
	if err != nil {
		return nil, err
	}
	return scheduledScanModelToRecord(&model)
}

// BatchUpdateStatus applies all requested state assignments in one transaction.
func (repo *ScheduledScanRepository) BatchUpdateStatus(ctx context.Context, updates []scheduledapp.ScheduledScanStatusUpdate) (int, error) {
	if repo == nil || repo.db == nil || repo.calculator == nil {
		return 0, scheduledapp.ErrScheduledScanInvalidArgument
	}
	ordered, err := normalizeScheduledScanStatusUpdates(updates)
	if err != nil {
		return 0, err
	}

	err = repo.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Target rows are locked before their Schedule rows. Sorting both sets
		// keeps concurrent scheduler and management writes on one lock order.
		var ownerships []struct {
			ID       int  `gorm:"column:id"`
			TargetID *int `gorm:"column:target_id"`
		}
		ids := make([]int, 0, len(ordered))
		for _, update := range ordered {
			ids = append(ids, update.ID)
		}
		if err := tx.Model(&scheduledScanModel{}).
			Select("id", "target_id").
			Where("id IN ?", ids).
			Find(&ownerships).Error; err != nil {
			return err
		}
		if len(ownerships) != len(ordered) {
			return scheduledapp.ErrScheduledScanNotFound
		}
		targetIDs := make([]*int, 0, len(ownerships))
		for _, ownership := range ownerships {
			targetIDs = append(targetIDs, ownership.TargetID)
		}
		if err := lockActiveScheduledScanTargets(tx, intPointersToIDs(targetIDs...)); err != nil {
			return err
		}

		referenceTime := repo.now().UTC()
		for _, update := range ordered {
			var model scheduledScanModel
			if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
				Where("id = ?", update.ID).
				First(&model).Error; err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return scheduledapp.ErrScheduledScanNotFound
				}
				return err
			}
			if err := repo.calculator.Validate(model.CronExpression); err != nil {
				return err
			}
			wasEnabled := model.IsEnabled
			model.IsEnabled = update.IsEnabled
			if err := repo.applyScheduledScanStatusTransition(tx, &model, wasEnabled, referenceTime); err != nil {
				return err
			}
			model.UpdatedAt = referenceTime
			if err := tx.Save(&model).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return 0, err
	}
	return len(ordered), nil
}

func normalizeScheduledScanStatusUpdates(updates []scheduledapp.ScheduledScanStatusUpdate) ([]scheduledapp.ScheduledScanStatusUpdate, error) {
	if len(updates) == 0 || len(updates) > scheduledapp.MaxScheduledScanBatchStatusUpdates {
		return nil, fmt.Errorf("%w: batch must contain between 1 and %d scheduled scans", scheduledapp.ErrScheduledScanInvalidArgument, scheduledapp.MaxScheduledScanBatchStatusUpdates)
	}
	ordered := append([]scheduledapp.ScheduledScanStatusUpdate(nil), updates...)
	seen := make(map[int]struct{}, len(ordered))
	for _, update := range ordered {
		if update.ID <= 0 {
			return nil, fmt.Errorf("%w: scheduled scan ID must be positive", scheduledapp.ErrScheduledScanInvalidArgument)
		}
		if _, exists := seen[update.ID]; exists {
			return nil, fmt.Errorf("%w: duplicate scheduled scan ID", scheduledapp.ErrScheduledScanInvalidArgument)
		}
		seen[update.ID] = struct{}{}
	}
	sort.Slice(ordered, func(left, right int) bool {
		return ordered[left].ID < ordered[right].ID
	})
	return ordered, nil
}

// applyScheduledScanStatusTransition owns the cursor and occurrence lifecycle
// shared by single-row and atomic batch status writes.
func (repo *ScheduledScanRepository) applyScheduledScanStatusTransition(
	tx *gorm.DB,
	model *scheduledScanModel,
	wasEnabled bool,
	referenceTime time.Time,
) error {
	if !model.IsEnabled {
		model.NextRunTime = nil
		return tx.Where("scheduled_scan_id = ? AND attempted_at IS NULL", model.ID).
			Delete(&scheduledScanOccurrenceModel{}).Error
	}
	if !wasEnabled {
		nextRunTime, err := repo.calculator.FirstAfter(model.CronExpression, referenceTime)
		if err != nil {
			return err
		}
		model.NextRunTime = &nextRunTime
	}
	return nil
}

func (repo *ScheduledScanRepository) List(ctx context.Context, query scheduledapp.ScheduledScanListQuery) ([]scheduledapp.ScheduledScan, int64, error) {
	var models []scheduledScanModel
	var total int64
	base := activeScheduledScanQuery(repo.db.WithContext(ctx).Model(&scheduledScanModel{}))
	if query.TargetID > 0 {
		base = base.Where("scheduled_scan.target_id = ?", query.TargetID)
	}
	if query.OrganizationID > 0 {
		base = base.Where("scheduled_scan.organization_id = ?", query.OrganizationID)
	}
	if strings.TrimSpace(query.Search) != "" {
		base = base.Where("name ILIKE ?", "%"+strings.TrimSpace(query.Search)+"%")
	}
	if err := base.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := base.Preload("Organization").Preload("Target").
		Scopes(scope.WithPagination(query.Page, query.PageSize), scope.OrderByCreatedAtDesc()).
		Find(&models).Error; err != nil {
		return nil, 0, err
	}
	out := make([]scheduledapp.ScheduledScan, 0, len(models))
	for index := range models {
		record, err := scheduledScanModelToRecord(&models[index])
		if err != nil {
			return nil, 0, err
		}
		out = append(out, *record)
	}
	return out, total, nil
}

func (repo *ScheduledScanRepository) GetByID(ctx context.Context, id int) (*scheduledapp.ScheduledScan, error) {
	var model scheduledScanModel
	err := activeScheduledScanQuery(repo.db.WithContext(ctx).Preload("Organization").Preload("Target")).
		Where("scheduled_scan.id = ?", id).
		First(&model).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, scheduledapp.ErrScheduledScanNotFound
	}
	if err != nil {
		return nil, err
	}
	return scheduledScanModelToRecord(&model)
}

func activeScheduledScanQuery(db *gorm.DB) *gorm.DB {
	return db.Joins("LEFT JOIN target AS active_target ON active_target.id = scheduled_scan.target_id AND active_target.deleted_at IS NULL").
		Where("scheduled_scan.target_id IS NULL OR active_target.id IS NOT NULL")
}

func lockActiveScheduledScanTargets(tx *gorm.DB, targetIDs []int) error {
	if len(targetIDs) == 0 {
		return nil
	}
	for _, targetID := range targetIDs {
		var target targetRefModel
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ? AND deleted_at IS NULL", targetID).
			First(&target).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return fmt.Errorf("%w: target is unavailable", scheduledapp.ErrScheduledScanInvalidArgument)
			}
			return err
		}
	}
	return nil
}

func intPointersToIDs(values ...*int) []int {
	seen := make(map[int]struct{}, len(values))
	ids := make([]int, 0, len(values))
	for _, value := range values {
		if value == nil || *value <= 0 {
			continue
		}
		if _, exists := seen[*value]; exists {
			continue
		}
		seen[*value] = struct{}{}
		ids = append(ids, *value)
	}
	sort.Ints(ids)
	return ids
}

func (repo *ScheduledScanRepository) Delete(ctx context.Context, id int) error {
	if repo == nil || repo.db == nil || id <= 0 {
		return scheduledapp.ErrScheduledScanNotFound
	}
	return repo.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return deleteScheduledScanLocked(tx, id)
	})
}

func deleteScheduledScanLocked(tx *gorm.DB, id int) error {
	var model scheduledScanModel
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", id).First(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return scheduledapp.ErrScheduledScanNotFound
		}
		return err
	}
	// The occurrence ledger is Schedule-owned and its foreign key cascade is
	// the only deletion path that may remove these rows.
	return tx.Delete(&model).Error
}

func scheduledScanCreateToModel(scan *scheduledapp.ScheduledScanCreate) (*scheduledScanModel, error) {
	if scan == nil {
		return &scheduledScanModel{}, nil
	}
	configuration, err := encodeJSONMap(scan.Configuration)
	if err != nil {
		return nil, err
	}
	inputSource, ok := scan.InputSource.DatabaseValue()
	if !ok {
		return nil, fmt.Errorf("%w: %q", scandomain.ErrInvalidInputSource, scan.InputSource)
	}
	return &scheduledScanModel{
		Name:           scan.Name,
		ScanWorkflowID: scan.ScanWorkflowID,
		Configuration:  configuration,
		InputSource:    inputSource,
		OrganizationID: cloneIntPtr(scan.OrganizationID),
		TargetID:       cloneIntPtr(scan.TargetID),
		AgentID:        cloneIntPtr(scan.AgentID),
		CronExpression: scan.CronExpression,
		IsEnabled:      scan.IsEnabled,
	}, nil
}

func applyScheduledScanUpdate(model *scheduledScanModel, scan *scheduledapp.ScheduledScanUpdate) error {
	if model == nil || scan == nil {
		return nil
	}
	if scan.Name != nil {
		model.Name = *scan.Name
	}
	if scan.ScanWorkflowID != nil {
		model.ScanWorkflowID = *scan.ScanWorkflowID
	}
	if scan.Configuration != nil {
		configuration, err := encodeJSONMap(scan.Configuration)
		if err != nil {
			return err
		}
		model.Configuration = configuration
	}
	if scan.InputSource != nil {
		inputSource, ok := scan.InputSource.DatabaseValue()
		if !ok {
			return fmt.Errorf("%w: %q", scandomain.ErrInvalidInputSource, *scan.InputSource)
		}
		model.InputSource = inputSource
	}
	if scan.OrganizationID != nil {
		model.OrganizationID = cloneIntPtr(scan.OrganizationID)
		model.TargetID = nil
	}
	if scan.TargetID != nil {
		model.TargetID = cloneIntPtr(scan.TargetID)
		model.OrganizationID = nil
	}
	if scan.AgentSet {
		model.AgentID = cloneIntPtr(scan.AgentID)
	}
	if scan.CronExpression != nil {
		model.CronExpression = *scan.CronExpression
	}
	if scan.IsEnabled != nil {
		model.IsEnabled = *scan.IsEnabled
	}
	return nil
}

func scheduledScanModelToRecord(item *scheduledScanModel) (*scheduledapp.ScheduledScan, error) {
	if item == nil {
		return nil, nil
	}
	inputSource, ok := scandomain.ParseDatabaseInputSource(item.InputSource)
	if !ok {
		return nil, fmt.Errorf("%w: persisted value %q", scandomain.ErrInvalidInputSource, item.InputSource)
	}
	record := &scheduledapp.ScheduledScan{
		ID:                     item.ID,
		Name:                   item.Name,
		ScanWorkflowID:         item.ScanWorkflowID,
		Configuration:          decodeJSONMap(item.Configuration),
		InputSource:            inputSource,
		OrganizationID:         cloneIntPtr(item.OrganizationID),
		TargetID:               cloneIntPtr(item.TargetID),
		AgentID:                cloneIntPtr(item.AgentID),
		CronExpression:         item.CronExpression,
		IsEnabled:              item.IsEnabled,
		NextRunTime:            timeutil.ToUTCPtr(item.NextRunTime),
		LastRunTime:            timeutil.ToUTCPtr(item.LastRunTime),
		RunCount:               item.RunCount,
		SuccessfulHandoffCount: item.SuccessfulHandoffCount,
		FailedHandoffCount:     item.FailedHandoffCount,
		CreatedAt:              timeutil.ToUTC(item.CreatedAt),
		UpdatedAt:              timeutil.ToUTC(item.UpdatedAt),
	}
	if item.Organization != nil {
		record.OrganizationName = &item.Organization.Name
	}
	if item.Target != nil {
		record.TargetName = &item.Target.Name
	}
	return record, nil
}

func cloneIntPtr(value *int) *int {
	if value == nil {
		return nil
	}
	copy := *value
	return &copy
}

func encodeJSONMap(value map[string]any) (datatypes.JSON, error) {
	if len(value) == 0 {
		return datatypes.JSON([]byte("{}")), nil
	}
	payload, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	return datatypes.JSON(payload), nil
}

func decodeJSONMap(value datatypes.JSON) map[string]any {
	decoded, err := decodeJSONMapStrict(value)
	if err != nil {
		return map[string]any{}
	}
	return decoded
}

func decodeJSONMapStrict(value datatypes.JSON) (map[string]any, error) {
	if len(value) == 0 {
		return map[string]any{}, nil
	}
	var decoded map[string]any
	if err := json.Unmarshal(value, &decoded); err != nil {
		return nil, err
	}
	if decoded == nil {
		return nil, errors.New("scheduled scan configuration must be a JSON object")
	}
	return decoded, nil
}
