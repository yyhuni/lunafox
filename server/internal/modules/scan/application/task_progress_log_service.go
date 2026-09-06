package application

import (
	"context"

	"github.com/yyhuni/lunafox/server/internal/pkg/dberrors"
)

type TaskProgressLogService struct {
	logQueryStore   TaskProgressLogQueryStore
	logCommandStore TaskProgressLogCommandStore
	scanLookup      TaskProgressLogScanLookup
}

func NewTaskProgressLogService(logQueryStore TaskProgressLogQueryStore, logCommandStore TaskProgressLogCommandStore, scanLookup TaskProgressLogScanLookup) *TaskProgressLogService {
	return &TaskProgressLogService{logQueryStore: logQueryStore, logCommandStore: logCommandStore, scanLookup: scanLookup}
}

func NewTaskProgressLogApplicationService(logQueryStore TaskProgressLogQueryStore, logCommandStore TaskProgressLogCommandStore, scanLookup TaskProgressLogScanLookup) TaskProgressLogApplicationService {
	return NewTaskProgressLogService(logQueryStore, logCommandStore, scanLookup)
}

func (service *TaskProgressLogService) ListByScanID(ctx context.Context, scanID int, query *TaskProgressLogListQuery) ([]TaskProgressLogEntry, bool, error) {
	_ = ctx
	_, err := service.scanLookup.GetTaskProgressLogRefByID(scanID)
	if err != nil {
		if dberrors.IsRecordNotFound(err) {
			return nil, false, ErrScanNotFound
		}
		return nil, false, err
	}

	afterID, limit := query.normalize()

	logs, err := service.logQueryStore.FindByScanIDWithCursor(scanID, afterID, limit+1)
	if err != nil {
		return nil, false, err
	}
	hasMore := len(logs) > limit
	if hasMore {
		logs = logs[:limit]
	}
	return logs, hasMore, nil
}

func (service *TaskProgressLogService) BatchCreateTaskProgressLogs(ctx context.Context, request *TaskProgressLogBatchCreateRequest) (int, int, error) {
	if err := ctx.Err(); err != nil {
		return 0, 0, err
	}
	scanID, taskID, requestID, items := request.normalize()
	// ScanID is persisted with every event so PostgreSQL can route and later
	// reclaim whole scan ranges without deriving scope through scan_task.
	_, err := service.scanLookup.GetTaskProgressLogRefByID(scanID)
	if err != nil {
		if dberrors.IsRecordNotFound(err) {
			return 0, 0, ErrScanNotFound
		}
		return 0, 0, err
	}
	if len(items) == 0 {
		return 0, 0, nil
	}
	logs := make([]TaskProgressLogEntry, len(items))
	for index, item := range items {
		emittedAt := item.EmittedAt
		logs[index] = TaskProgressLogEntry{
			ScanID:    scanID,
			TaskID:    taskID,
			RequestID: requestID,
			Sequence:  item.Sequence,
			Level:     item.Level,
			Content:   item.Content,
			EmittedAt: &emittedAt,
		}
	}
	return service.logCommandStore.BatchCreateTaskProgressLogs(ctx, logs)
}
