package application

import "time"

const (
	defaultTaskProgressLogLimit = 200
	maxTaskProgressLogLimit     = 1000
)

type TaskProgressLogCreateItem struct {
	Sequence  int64
	Level     string
	Content   string
	EmittedAt time.Time
}

type TaskProgressLogListQuery struct {
	AfterID int64
	Limit   int
}

func (query *TaskProgressLogListQuery) normalize() (afterID int64, limit int) {
	if query == nil {
		return 0, defaultTaskProgressLogLimit
	}
	afterID = query.AfterID
	if afterID < 0 {
		afterID = 0
	}
	limit = query.Limit
	if limit <= 0 {
		limit = defaultTaskProgressLogLimit
	}
	if limit > maxTaskProgressLogLimit {
		limit = maxTaskProgressLogLimit
	}
	return afterID, limit
}

type TaskProgressLogBatchCreateRequest struct {
	ScanID    int
	TaskID    int
	RequestID string
	Items     []TaskProgressLogCreateItem
}

func (request *TaskProgressLogBatchCreateRequest) normalize() (scanID int, taskID int, requestID string, items []TaskProgressLogCreateItem) {
	if request == nil {
		return 0, 0, "", nil
	}
	return request.ScanID, request.TaskID, request.RequestID, request.Items
}
