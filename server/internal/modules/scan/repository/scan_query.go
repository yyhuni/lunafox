package repository

import (
	"context"
	"strings"

	model "github.com/yyhuni/lunafox/server/internal/modules/scan/repository/persistence"
	"github.com/yyhuni/lunafox/server/internal/pkg/dbtx"
	"github.com/yyhuni/lunafox/server/internal/pkg/scope"
	"gorm.io/gorm"
)

// GetByID finds a scan by ID. Normal reads exclude soft-deleted rows by default.
func (r *ScanRepository) GetByID(id int) (*ScanRecord, error) {
	return r.GetByIDContext(context.Background(), id)
}

// GetByIDContext preserves a caller-owned cancellation/deadline through the lookup.
func (r *ScanRepository) GetByIDContext(ctx context.Context, id int) (*ScanRecord, error) {
	var scan model.Scan
	err := dbtx.Resolve(ctx, r.db).WithContext(ctx).Where("id = ? AND deleted_at IS NULL", id).
		First(&scan).Error
	if err != nil {
		return nil, err
	}
	return scanModelToRecord(&scan)
}

// GetDetailByID loads the scan detail projection with target and current Agent state.
func (r *ScanRepository) GetDetailByID(id int) (*ScanRecord, error) {
	return r.GetDetailByIDContext(context.Background(), id)
}

// GetDetailByIDContext preserves a caller-owned cancellation/deadline through scan detail reads.
func (r *ScanRepository) GetDetailByIDContext(ctx context.Context, id int) (*ScanRecord, error) {
	var scan model.Scan
	db := dbtx.Resolve(ctx, r.db).WithContext(ctx)
	err := db.Where("scan.id = ? AND scan.deleted_at IS NULL", id).
		Preload("Target").
		Joins("LEFT JOIN agent ON agent.id = scan.agent_id").
		Joins("LEFT JOIN agent_runtime_status ON agent_runtime_status.agent_id = agent.id").
		Select(`scan.*,
			agent.display_name AS agent_name,
			agent.status AS agent_status,
			agent_runtime_status.health_state AS agent_health_state,
			CASE WHEN scan.agent_id IS NOT NULL AND agent.id IS NULL THEN TRUE ELSE FALSE END AS agent_deleted`).
		First(&scan).Error
	if err != nil {
		return nil, err
	}

	tasks, err := r.loadRuntimeTasks(ctx, scan.ID)
	if err != nil {
		return nil, err
	}

	record, err := scanModelToRecord(&scan)
	if err != nil {
		return nil, err
	}
	runtimeTasks, err := scanTaskRuntimeRowsToRuntimeTaskRecord(tasks)
	if err != nil {
		return nil, err
	}
	record.RuntimeTasks = runtimeTasks
	record.PlannedEngineIDs = plannedEngineIDsFromRuntimeTasks(tasks)
	return record, nil
}

func (r *ScanRepository) loadRuntimeTasks(ctx context.Context, scanID int) ([]scanTaskRuntimeRow, error) {
	var tasks []scanTaskRuntimeRow
	err := NewScanTaskRepository(dbtx.Resolve(ctx, r.db)).(*scanTaskRepository).
		scanTaskRuntimeQuery(ctx).
		Where("st.scan_id = ?", scanID).
		Order("st.stage_order ASC, st.id ASC").
		Find(&tasks).Error
	return tasks, err
}

// GetScanForScanTask finds the scan projection required by scan task runtime flows.
func (r *ScanRepository) GetScanForScanTask(scanID int) (*ScanTaskRuntimeScanRecord, error) {
	var scan model.Scan
	err := r.db.Where("id = ? AND deleted_at IS NULL", scanID).
		Preload("Target").
		First(&scan).Error
	if err != nil {
		return nil, err
	}
	return scanModelToScanTaskRuntimeScanRecord(&scan), nil
}

// FindTaskProgressLogScanRefByID finds the minimal scan projection required by task-progress-log flows.
func (r *ScanRepository) FindTaskProgressLogScanRefByID(id int) (*TaskProgressLogScanRefRecord, error) {
	var scan model.Scan
	err := r.db.Where("id = ? AND deleted_at IS NULL", id).
		Select("id", "status").
		First(&scan).Error
	if err != nil {
		return nil, err
	}
	return scanModelToTaskProgressLogScanRefRecord(&scan), nil
}

// FindByIDs finds scans by IDs (excluding soft deleted).
func (r *ScanRepository) FindByIDs(ids []int) ([]ScanRecord, error) {
	if len(ids) == 0 {
		return []ScanRecord{}, nil
	}

	var scans []model.Scan
	err := r.db.Where("id IN ? AND deleted_at IS NULL", ids).
		Find(&scans).Error
	if err != nil {
		return nil, err
	}

	return scanModelListToRecord(scans)
}

// List returns paginated scans with filters. Soft-deleted rows are excluded.
func (r *ScanRepository) List(page, pageSize int, targetID int, status, filter, orderBy string) ([]ScanRecord, int64, error) {
	return r.ListContext(context.Background(), page, pageSize, targetID, status, filter, orderBy)
}

// ListContext preserves a caller-owned cancellation/deadline through scan list reads.
func (r *ScanRepository) ListContext(ctx context.Context, page, pageSize int, targetID int, status, filter, orderBy string) ([]ScanRecord, int64, error) {
	var scans []model.Scan
	var total int64
	db := dbtx.Resolve(ctx, r.db).WithContext(ctx)

	baseQuery := db.Model(&model.Scan{}).Where("scan.deleted_at IS NULL")

	if targetID > 0 {
		baseQuery = baseQuery.Where("scan.target_id = ?", targetID)
	}
	if status != "" {
		baseQuery = baseQuery.Where("scan.status = ?", status)
	}
	baseQuery = applyScanListFilter(baseQuery, filter)

	if err := baseQuery.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := baseQuery.
		Joins("LEFT JOIN agent ON agent.id = scan.agent_id").
		Select("scan.*, agent.display_name as agent_name").
		Preload("Target").
		Scopes(
			scope.WithPagination(page, pageSize),
			orderScanList(orderBy),
		).
		Find(&scans).Error
	if err != nil {
		return nil, 0, err
	}

	records, err := scanModelListToRecord(scans)
	if err != nil {
		return nil, 0, err
	}
	if err := r.hydrateListPlannedEngineIDs(ctx, records); err != nil {
		return nil, 0, err
	}

	return records, total, nil
}

func applyScanListFilter(query *gorm.DB, filter string) *gorm.DB {
	trimmed := strings.TrimSpace(filter)
	if trimmed == "" {
		return query
	}

	groups := scope.ParseFilter(trimmed)
	if len(groups) == 0 {
		return query.Joins("LEFT JOIN target ON target.id = scan.target_id").
			Where("LOWER(target.name) LIKE LOWER(?)", "%"+trimmed+"%")
	}

	segments := make([]scanListFilterSegment, 0, len(groups))
	joinedTarget := false
	for _, group := range groups {
		condition, args, needsTargetJoin := buildScanListFilterCondition(group.Filter)
		if condition == "" {
			continue
		}
		if needsTargetJoin {
			joinedTarget = true
		}
		if group.LogicalOp == scope.LogicalOr && len(segments) > 0 {
			last := &segments[len(segments)-1]
			last.conditions = append(last.conditions, condition)
			last.args = append(last.args, args...)
			continue
		}
		segments = append(segments, scanListFilterSegment{
			conditions: []string{condition},
			args:       append([]any(nil), args...),
		})
	}

	if joinedTarget {
		query = query.Joins("LEFT JOIN target ON target.id = scan.target_id")
	}
	for _, segment := range segments {
		if len(segment.conditions) == 1 {
			query = query.Where(segment.conditions[0], segment.args...)
			continue
		}
		query = query.Where("("+strings.Join(segment.conditions, " OR ")+")", segment.args...)
	}
	return query
}

type scanListFilterSegment struct {
	conditions []string
	args       []any
}

func buildScanListFilterCondition(filter scope.ParsedFilter) (string, []any, bool) {
	switch strings.ToLower(filter.Field) {
	case "targetname":
		return buildScanListStringCondition("target.name", filter.Operator), []any{buildScanListStringArg(filter.Operator, filter.Value)}, true
	case "status":
		return buildScanListStringCondition("scan.status", filter.Operator), []any{buildScanListStringArg(filter.Operator, filter.Value)}, false
	default:
		return "", nil, false
	}
}

func buildScanListStringCondition(column, operator string) string {
	switch operator {
	case "==":
		return column + " = ?"
	case "!=":
		return column + " != ?"
	default:
		return "LOWER(" + column + ") LIKE LOWER(?)"
	}
}

func buildScanListStringArg(operator, value string) any {
	if operator == "==" || operator == "!=" {
		return value
	}
	return "%" + value + "%"
}

func orderScanList(orderBy string) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		direction := "DESC"
		parts := strings.Fields(orderBy)
		if len(parts) == 2 && strings.EqualFold(parts[1], "asc") {
			direction = "ASC"
		}
		return db.Order("scan.created_at " + direction).Order("scan.id " + direction)
	}
}

type scanTaskPlannedEngineSummaryRow struct {
	ScanID     int
	StageOrder int
	StepOrder  int
	ID         int
	EngineID   string
}

func (r *ScanRepository) hydrateListPlannedEngineIDs(ctx context.Context, scans []ScanRecord) error {
	if len(scans) == 0 {
		return nil
	}

	scanIDs := make([]int, 0, len(scans))
	for index := range scans {
		scanIDs = append(scanIDs, scans[index].ID)
		scans[index].PlannedEngineIDs = []string{}
	}

	var rows []scanTaskPlannedEngineSummaryRow
	if err := dbtx.Resolve(ctx, r.db).WithContext(ctx).Model(&model.ScanTask{}).
		Select("scan_id, stage_order, step_order, id, engine_id").
		Where("scan_id IN ? AND LENGTH(resolved_execution_plan) > 0", scanIDs).
		Order("scan_id ASC, stage_order ASC, step_order ASC, id ASC").
		Find(&rows).Error; err != nil {
		return err
	}

	positionByScanID := make(map[int]int, len(scans))
	seenByScanID := make(map[int]map[string]struct{}, len(scans))
	for index := range scans {
		positionByScanID[scans[index].ID] = index
	}

	for _, row := range rows {
		position, ok := positionByScanID[row.ScanID]
		if !ok || row.EngineID == "" {
			continue
		}
		seen := seenByScanID[row.ScanID]
		if seen == nil {
			seen = map[string]struct{}{}
			seenByScanID[row.ScanID] = seen
		}
		if _, exists := seen[row.EngineID]; exists {
			continue
		}
		seen[row.EngineID] = struct{}{}
		scans[position].PlannedEngineIDs = append(scans[position].PlannedEngineIDs, row.EngineID)
	}

	return nil
}

func plannedEngineIDsFromRuntimeTasks(tasks []scanTaskRuntimeRow) []string {
	engineIDs := make([]string, 0, len(tasks))
	seen := make(map[string]struct{}, len(tasks))
	for _, task := range tasks {
		if task.EngineID == "" || !task.HasResolvedExecutionPlan {
			continue
		}
		if _, exists := seen[task.EngineID]; exists {
			continue
		}
		seen[task.EngineID] = struct{}{}
		engineIDs = append(engineIDs, task.EngineID)
	}
	return engineIDs
}

// GetGlobalStatsSummary returns the global scan statistics summary.
func (r *ScanRepository) GetGlobalStatsSummary() (*ScanStatistics, error) {
	stats := &ScanStatistics{}

	if err := r.db.Model(&model.Scan{}).Where("deleted_at IS NULL").
		Count(&stats.Total).Error; err != nil {
		return nil, err
	}
	if err := r.db.Model(&model.Scan{}).Where("deleted_at IS NULL AND status = ?", scanStatusPending).
		Count(&stats.Pending).Error; err != nil {
		return nil, err
	}
	if err := r.db.Model(&model.Scan{}).Where("deleted_at IS NULL AND status = ?", scanStatusRunning).
		Count(&stats.Running).Error; err != nil {
		return nil, err
	}
	if err := r.db.Model(&model.Scan{}).Where("deleted_at IS NULL AND status = ?", scanStatusSucceeded).
		Count(&stats.Completed).Error; err != nil {
		return nil, err
	}
	if err := r.db.Model(&model.Scan{}).Where("deleted_at IS NULL AND status = ?", scanStatusFailed).
		Count(&stats.Failed).Error; err != nil {
		return nil, err
	}
	if err := r.db.Model(&model.Scan{}).Where("deleted_at IS NULL AND status = ?", scanStatusCancelled).
		Count(&stats.Cancelled).Error; err != nil {
		return nil, err
	}

	type sumResult struct {
		TotalVulns      int64
		TotalSubdomains int64
		TotalEndpoints  int64
		TotalWebsites   int64
	}
	var sums sumResult
	if err := r.db.Model(&model.Scan{}).Where("deleted_at IS NULL").
		Select(`
			COALESCE(SUM(cached_vulns_total), 0) as total_vulns,
			COALESCE(SUM(cached_subdomains_count), 0) as total_subdomains,
			COALESCE(SUM(cached_endpoints_count), 0) as total_endpoints,
			COALESCE(SUM(cached_websites_count), 0) as total_websites
		`).
		Scan(&sums).Error; err != nil {
		return nil, err
	}

	stats.TotalVulns = sums.TotalVulns
	stats.TotalSubdomains = sums.TotalSubdomains
	stats.TotalEndpoints = sums.TotalEndpoints
	stats.TotalWebsites = sums.TotalWebsites
	stats.TotalAssets = sums.TotalSubdomains + sums.TotalEndpoints + sums.TotalWebsites

	return stats, nil
}

// GetTargetRefByScanID returns the target associated with a scan.
func (r *ScanRepository) GetTargetRefByScanID(scanID int) (*ScanTargetRecord, error) {
	return r.GetTargetRefByScanIDContext(context.Background(), scanID)
}

// GetTargetRefByScanIDContext preserves a caller-owned cancellation/deadline through the lookup.
func (r *ScanRepository) GetTargetRefByScanIDContext(ctx context.Context, scanID int) (*ScanTargetRecord, error) {
	var scan model.Scan
	err := dbtx.Resolve(ctx, r.db).WithContext(ctx).Where("id = ? AND deleted_at IS NULL", scanID).
		Preload("Target").
		First(&scan).Error
	if err != nil {
		return nil, err
	}
	if scan.Target == nil {
		return nil, gorm.ErrRecordNotFound
	}
	return scanTargetModelToRecord(scan.Target), nil
}
