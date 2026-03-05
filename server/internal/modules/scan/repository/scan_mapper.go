package repository

import (
	scandomain "github.com/yyhuni/lunafox/server/internal/modules/scan/domain"
	model "github.com/yyhuni/lunafox/server/internal/modules/scan/repository/persistence"
	"github.com/yyhuni/lunafox/server/internal/pkg/timeutil"
)

type ScanTargetRecord = scandomain.QueryTargetRef

type ScanRecord = scandomain.QueryScan

// Create-side projections are unified with domain create projections.
type ScanCreateRecord = scandomain.CreateScan

type ScanTaskCreateRecord = scandomain.CreateScanTask

// Runtime/read-side task projection is unified with domain runtime projection.
type ScanTaskRecord = scandomain.TaskRecord

// Runtime scan lookup projection is unified with domain runtime projection.
type TaskRuntimeScanTargetRecord = scandomain.TaskTargetRef

type TaskRuntimeScanRecord = scandomain.TaskScanRecord

// Runtime scan-log projection is unified with domain runtime projection.
type ScanLogRecord = scandomain.ScanLogEntry

// Runtime scan-log scan existence projection is unified with domain runtime projection.
type ScanLogScanRefRecord = scandomain.ScanLogScanRef

func scanTargetModelToRecord(item *model.ScanTargetRef) *ScanTargetRecord {
	if item == nil {
		return nil
	}
	return &ScanTargetRecord{ID: item.ID, Name: item.Name, Type: item.Type, CreatedAt: timeutil.ToUTC(item.CreatedAt)}
}

func scanTargetModelToTaskRuntimeRecord(item *model.ScanTargetRef) *TaskRuntimeScanTargetRecord {
	if item == nil {
		return nil
	}
	return &TaskRuntimeScanTargetRecord{ID: item.ID, Name: item.Name, Type: item.Type}
}

func scanModelToTaskRuntimeRecord(item *model.Scan) *TaskRuntimeScanRecord {
	if item == nil {
		return nil
	}
	return &TaskRuntimeScanRecord{
		ID:                item.ID,
		TargetID:          item.TargetID,
		Status:            item.Status,
		YAMLConfiguration: item.YAMLConfiguration,
		Target:            scanTargetModelToTaskRuntimeRecord(item.Target),
	}
}

func scanModelToScanLogScanRefRecord(item *model.Scan) *ScanLogScanRefRecord {
	if item == nil {
		return nil
	}
	return &ScanLogScanRefRecord{ID: item.ID, Status: item.Status}
}

func scanModelToRecord(item *model.Scan) *ScanRecord {
	if item == nil {
		return nil
	}
	return &ScanRecord{
		ID:                     item.ID,
		TargetID:               item.TargetID,
		WorkflowIDs:            append([]byte(nil), item.WorkflowIDs...),
		YAMLConfiguration:      item.YAMLConfiguration,
		ScanMode:               item.ScanMode,
		Status:                 item.Status,
		ResultsDir:             item.ResultsDir,
		WorkerID:               item.WorkerID,
		ErrorMessage:           item.ErrorMessage,
		Progress:               item.Progress,
		CurrentStage:           item.CurrentStage,
		StageProgress:          append([]byte(nil), item.StageProgress...),
		CreatedAt:              timeutil.ToUTC(item.CreatedAt),
		StoppedAt:              timeutil.ToUTCPtr(item.StoppedAt),
		CachedSubdomainsCount:  item.CachedSubdomainsCount,
		CachedWebsitesCount:    item.CachedWebsitesCount,
		CachedEndpointsCount:   item.CachedEndpointsCount,
		CachedIPsCount:         item.CachedIPsCount,
		CachedDirectoriesCount: item.CachedDirectoriesCount,
		CachedScreenshotsCount: item.CachedScreenshotsCount,
		CachedVulnsTotal:       item.CachedVulnsTotal,
		CachedVulnsCritical:    item.CachedVulnsCritical,
		CachedVulnsHigh:        item.CachedVulnsHigh,
		CachedVulnsMedium:      item.CachedVulnsMedium,
		CachedVulnsLow:         item.CachedVulnsLow,
		Target:                 scanTargetModelToRecord(item.Target),
	}
}

func scanModelListToRecord(items []model.Scan) []ScanRecord {
	results := make([]ScanRecord, 0, len(items))
	for index := range items {
		results = append(results, *scanModelToRecord(&items[index]))
	}
	return results
}

func scanCreateRecordToModel(item *ScanCreateRecord) *model.Scan {
	if item == nil {
		return nil
	}
	return &model.Scan{
		ID:                item.ID,
		TargetID:          item.TargetID,
		WorkflowIDs:       append([]byte(nil), item.WorkflowIDs...),
		YAMLConfiguration: item.YAMLConfiguration,
		ScanMode:          item.ScanMode,
		Status:            item.Status,
		CreatedAt:         timeutil.ToUTC(item.CreatedAt),
	}
}

func scanTaskCreateRecordsToModel(items []ScanTaskCreateRecord) []model.ScanTask {
	results := make([]model.ScanTask, 0, len(items))
	for index := range items {
		results = append(results, model.ScanTask{
			Stage:              items[index].Stage,
			WorkflowID:         items[index].WorkflowID,
			WorkflowConfigYAML: items[index].WorkflowConfigYAML,
			Status:             items[index].Status,
		})
	}
	return results
}

func scanTaskModelToRecord(item *model.ScanTask) *ScanTaskRecord {
	if item == nil {
		return nil
	}
	return &ScanTaskRecord{
		ID:                 item.ID,
		ScanID:             item.ScanID,
		Stage:              item.Stage,
		WorkflowID:         item.WorkflowID,
		Status:             item.Status,
		AgentID:            item.AgentID,
		WorkflowConfigYAML: item.WorkflowConfigYAML,
	}
}

func scanLogModelToRecord(item *model.ScanLog) *ScanLogRecord {
	if item == nil {
		return nil
	}
	return &ScanLogRecord{ID: item.ID, ScanID: item.ScanID, Level: item.Level, Content: item.Content, CreatedAt: timeutil.ToUTC(item.CreatedAt)}
}

func scanLogModelListToRecord(items []model.ScanLog) []ScanLogRecord {
	results := make([]ScanLogRecord, 0, len(items))
	for index := range items {
		results = append(results, *scanLogModelToRecord(&items[index]))
	}
	return results
}

func scanLogRecordToModel(item *ScanLogRecord) *model.ScanLog {
	if item == nil {
		return nil
	}
	return &model.ScanLog{ID: item.ID, ScanID: item.ScanID, Level: item.Level, Content: item.Content, CreatedAt: timeutil.ToUTC(item.CreatedAt)}
}

func scanLogRecordListToModel(items []ScanLogRecord) []model.ScanLog {
	results := make([]model.ScanLog, 0, len(items))
	for index := range items {
		results = append(results, *scanLogRecordToModel(&items[index]))
	}
	return results
}
