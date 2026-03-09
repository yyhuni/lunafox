package repository

import (
	"encoding/json"

	scandomain "github.com/yyhuni/lunafox/server/internal/modules/scan/domain"
	model "github.com/yyhuni/lunafox/server/internal/modules/scan/repository/persistence"
	"github.com/yyhuni/lunafox/server/internal/pkg/timeutil"
	"gorm.io/datatypes"
)

type ScanTargetRecord = scandomain.QueryTargetRef

type ScanRecord = scandomain.QueryScan

type ScanCreateRecord = scandomain.CreateScan

type ScanTaskCreateRecord = scandomain.CreateScanTask

type ScanTaskRecord = scandomain.TaskRecord

type TaskRuntimeScanTargetRecord = scandomain.TaskTargetRef

type TaskRuntimeScanRecord = scandomain.TaskScanRecord

type ScanLogRecord = scandomain.ScanLogEntry

type ScanLogScanRefRecord = scandomain.ScanLogScanRef

func scanTargetModelToRecord(item *model.ScanTargetRef) *ScanTargetRecord {
	if item == nil {
		return nil
	}

	return &ScanTargetRecord{
		ID:        item.ID,
		Name:      item.Name,
		Type:      item.Type,
		CreatedAt: timeutil.ToUTC(item.CreatedAt),
	}
}

func scanTargetModelToTaskRuntimeRecord(item *model.ScanTargetRef) *TaskRuntimeScanTargetRecord {
	if item == nil {
		return nil
	}

	return &TaskRuntimeScanTargetRecord{
		ID:   item.ID,
		Name: item.Name,
		Type: item.Type,
	}
}

func scanModelToTaskRuntimeRecord(item *model.Scan) *TaskRuntimeScanRecord {
	if item == nil {
		return nil
	}

	return &TaskRuntimeScanRecord{
		ID:            item.ID,
		TargetID:      item.TargetID,
		Configuration: decodeJSONMap(item.Configuration),
		Status:        item.Status,
		Target:        scanTargetModelToTaskRuntimeRecord(item.Target),
	}
}

func scanModelToScanLogScanRefRecord(item *model.Scan) *ScanLogScanRefRecord {
	if item == nil {
		return nil
	}

	return &ScanLogScanRefRecord{
		ID:     item.ID,
		Status: item.Status,
	}
}

func scanModelToRecord(item *model.Scan) *ScanRecord {
	if item == nil {
		return nil
	}

	return &ScanRecord{
		ID:                     item.ID,
		TargetID:               item.TargetID,
		WorkflowIDs:            append([]byte(nil), item.WorkflowIDs...),
		Configuration:          decodeJSONMap(item.Configuration),
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
		ID:            item.ID,
		TargetID:      item.TargetID,
		WorkflowIDs:   append([]byte(nil), item.WorkflowIDs...),
		Configuration: encodeJSONMap(item.Configuration),
		ScanMode:      item.ScanMode,
		Status:        item.Status,
		CreatedAt:     timeutil.ToUTC(item.CreatedAt),
	}
}

func scanTaskCreateRecordsToModel(items []ScanTaskCreateRecord) []model.ScanTask {
	results := make([]model.ScanTask, 0, len(items))
	for index := range items {
		results = append(results, model.ScanTask{
			Stage:          items[index].Stage,
			WorkflowID:     items[index].WorkflowID,
			WorkflowConfig: encodeJSONMap(items[index].WorkflowConfig),
			FailureKind:    "",
			Status:         items[index].Status,
		})
	}
	return results
}

func scanTaskModelToRecord(item *model.ScanTask) *ScanTaskRecord {
	if item == nil {
		return nil
	}

	return &ScanTaskRecord{
		ID:             item.ID,
		ScanID:         item.ScanID,
		Stage:          item.Stage,
		WorkflowID:     item.WorkflowID,
		Status:         item.Status,
		AgentID:        item.AgentID,
		WorkflowConfig: decodeJSONMap(item.WorkflowConfig),
		FailureKind:    item.FailureKind,
	}
}

func scanLogModelToRecord(item *model.ScanLog) *ScanLogRecord {
	if item == nil {
		return nil
	}

	return &ScanLogRecord{
		ID:        item.ID,
		ScanID:    item.ScanID,
		Level:     item.Level,
		Content:   item.Content,
		CreatedAt: timeutil.ToUTC(item.CreatedAt),
	}
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

	return &model.ScanLog{
		ID:        item.ID,
		ScanID:    item.ScanID,
		Level:     item.Level,
		Content:   item.Content,
		CreatedAt: timeutil.ToUTC(item.CreatedAt),
	}
}

func scanLogRecordListToModel(items []ScanLogRecord) []model.ScanLog {
	results := make([]model.ScanLog, 0, len(items))
	for index := range items {
		results = append(results, *scanLogRecordToModel(&items[index]))
	}
	return results
}

func encodeJSONMap(value map[string]any) datatypes.JSON {
	if len(value) == 0 {
		return datatypes.JSON([]byte("{}"))
	}

	payload, err := json.Marshal(value)
	if err != nil {
		return datatypes.JSON([]byte("{}"))
	}

	return datatypes.JSON(payload)
}

func decodeJSONMap(value datatypes.JSON) map[string]any {
	if len(value) == 0 {
		return nil
	}

	var decoded map[string]any
	if err := json.Unmarshal(value, &decoded); err != nil {
		return nil
	}

	return decoded
}
