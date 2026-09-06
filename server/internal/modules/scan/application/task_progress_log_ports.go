package application

import (
	"context"
)

type TaskProgressLogQueryStore interface {
	FindByScanIDWithCursor(scanID int, afterID int64, limit int) ([]TaskProgressLogEntry, error)
}

type TaskProgressLogCommandStore interface {
	BatchCreateTaskProgressLogs(ctx context.Context, logs []TaskProgressLogEntry) (accepted int, duplicates int, err error)
}

type TaskProgressLogScanLookup interface {
	GetTaskProgressLogRefByID(id int) (*TaskProgressLogScanRef, error)
}

type TaskProgressLogApplicationService interface {
	ListByScanID(ctx context.Context, scanID int, query *TaskProgressLogListQuery) ([]TaskProgressLogEntry, bool, error)
	BatchCreateTaskProgressLogs(ctx context.Context, request *TaskProgressLogBatchCreateRequest) (accepted int, duplicates int, err error)
}
