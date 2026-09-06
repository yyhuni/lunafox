package agentdata

import (
	"context"
	"errors"
	"strings"
	"time"

	agentdatav1 "github.com/yyhuni/lunafox/contracts/gen/lunafox/agent/data/v1"
	taskprogressv1 "github.com/yyhuni/lunafox/contracts/gen/lunafox/taskprogress/v1"
	"github.com/yyhuni/lunafox/contracts/resourcenames"
	scanapp "github.com/yyhuni/lunafox/server/internal/modules/scan/application"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	maxTaskProgressLogEntries           = 1000
	maxTaskProgressLogContentBytes      = 16 * 1024
	maxTaskProgressLogBatchContentBytes = 4 * 1024 * 1024
)

type TaskProgressLogDataPlane interface {
	WriteTaskProgressLogs(ctx context.Context, batch TaskProgressLogBatch) (accepted int, duplicates int, err error)
}

type TaskProgressLogBatch struct {
	ScanID       int
	TaskID       int
	AgentID      int
	SessionID    string
	SessionEpoch int64
	RequestID    string
	Entries      []TaskProgressLogEntry
}

type TaskProgressLogEntry struct {
	Sequence  int64
	Level     string
	Content   string
	EmittedAt time.Time
}

func (s *DataPlaneService) BatchWriteTaskProgressLogs(ctx context.Context, req *agentdatav1.BatchWriteTaskProgressLogsRequest) (*agentdatav1.BatchWriteTaskProgressLogsResponse, error) {
	lease, err := s.requireAuthenticatedAgentSession(ctx)
	if err != nil {
		return nil, err
	}
	if s.taskProgressLogs == nil {
		return nil, status.Error(codes.Unimplemented, errDataPlaneUnimplemented)
	}
	batch, err := taskProgressLogBatchFromProto(req)
	if err != nil {
		return nil, err
	}
	batch.AgentID = lease.AgentID
	batch.SessionID = lease.SessionID
	batch.SessionEpoch = lease.SessionEpoch
	accepted, duplicates, err := s.taskProgressLogs.WriteTaskProgressLogs(ctx, batch)
	if err != nil {
		return nil, mapTaskProgressLogWriteError(err)
	}
	return &agentdatav1.BatchWriteTaskProgressLogsResponse{Summary: &agentdatav1.BatchWriteSummary{
		AcceptedItems:  int32(accepted),
		DuplicateItems: int32(duplicates),
		TotalItems:     int32(len(batch.Entries)),
	}}, nil
}

func taskProgressLogBatchFromProto(req *agentdatav1.BatchWriteTaskProgressLogsRequest) (TaskProgressLogBatch, error) {
	if req == nil {
		return TaskProgressLogBatch{}, status.Error(codes.InvalidArgument, "request is required")
	}
	scanID, taskID, err := resourcenames.ParseTask(req.GetTask())
	if err != nil {
		return TaskProgressLogBatch{}, status.Error(codes.InvalidArgument, err.Error())
	}
	requestID := strings.TrimSpace(req.RequestId)
	if requestID == "" {
		return TaskProgressLogBatch{}, status.Error(codes.InvalidArgument, "request_id is required")
	}
	if len(req.Entries) == 0 {
		return TaskProgressLogBatch{}, status.Error(codes.InvalidArgument, "entries must not be empty")
	}
	if len(req.Entries) > maxTaskProgressLogEntries {
		return TaskProgressLogBatch{}, status.Errorf(codes.ResourceExhausted, "entries must not exceed %d", maxTaskProgressLogEntries)
	}

	entries := make([]TaskProgressLogEntry, 0, len(req.Entries))
	totalContentBytes := 0
	for index, entry := range req.Entries {
		mapped, err := taskProgressLogEntryFromProto(index, entry)
		if err != nil {
			return TaskProgressLogBatch{}, err
		}
		totalContentBytes += len([]byte(mapped.Content))
		if totalContentBytes > maxTaskProgressLogBatchContentBytes {
			return TaskProgressLogBatch{}, status.Errorf(codes.ResourceExhausted, "entries total content bytes must not exceed %d", maxTaskProgressLogBatchContentBytes)
		}
		entries = append(entries, mapped)
	}
	return TaskProgressLogBatch{
		ScanID:    scanID,
		TaskID:    taskID,
		RequestID: requestID,
		Entries:   entries,
	}, nil
}

func taskProgressLogEntryFromProto(index int, entry *taskprogressv1.TaskProgressLogEntry) (TaskProgressLogEntry, error) {
	if entry == nil {
		return TaskProgressLogEntry{}, status.Errorf(codes.InvalidArgument, "entries[%d] is required", index)
	}
	if entry.Sequence <= 0 {
		return TaskProgressLogEntry{}, status.Errorf(codes.InvalidArgument, "entries[%d].sequence is required", index)
	}
	level, err := taskProgressLogLevelFromProto(entry.Level)
	if err != nil {
		return TaskProgressLogEntry{}, status.Errorf(codes.InvalidArgument, "entries[%d].level is invalid", index)
	}
	if strings.TrimSpace(entry.Content) == "" {
		return TaskProgressLogEntry{}, status.Errorf(codes.InvalidArgument, "entries[%d].content is required", index)
	}
	if len([]byte(entry.Content)) > maxTaskProgressLogContentBytes {
		return TaskProgressLogEntry{}, status.Errorf(codes.ResourceExhausted, "entries[%d].content must not exceed %d bytes", index, maxTaskProgressLogContentBytes)
	}
	if entry.EmittedAt == nil || !entry.EmittedAt.IsValid() {
		return TaskProgressLogEntry{}, status.Errorf(codes.InvalidArgument, "entries[%d].emitted_at is required", index)
	}
	return TaskProgressLogEntry{
		Sequence:  entry.Sequence,
		Level:     level,
		Content:   entry.Content,
		EmittedAt: entry.EmittedAt.AsTime(),
	}, nil
}

func taskProgressLogLevelFromProto(level taskprogressv1.TaskProgressLogLevel) (string, error) {
	switch level {
	case taskprogressv1.TaskProgressLogLevel_TASK_PROGRESS_LOG_LEVEL_INFO:
		return "info", nil
	case taskprogressv1.TaskProgressLogLevel_TASK_PROGRESS_LOG_LEVEL_WARNING:
		return "warning", nil
	case taskprogressv1.TaskProgressLogLevel_TASK_PROGRESS_LOG_LEVEL_ERROR:
		return "error", nil
	default:
		return "", errors.New("unsupported task progress log level")
	}
}

func mapTaskProgressLogWriteError(err error) error {
	switch {
	case errors.Is(err, context.Canceled):
		return status.Error(codes.Canceled, "task progress log write cancelled")
	case errors.Is(err, context.DeadlineExceeded):
		return status.Error(codes.DeadlineExceeded, "task progress log write deadline exceeded")
	case errors.Is(err, scanapp.ErrScanNotFound), errors.Is(err, scanapp.ErrScanTaskNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, scanapp.ErrScanTaskNotOwned), errors.Is(err, errTaskProgressLogOwnershipMismatch):
		return status.Error(codes.PermissionDenied, err.Error())
	case errors.Is(err, errTaskProgressLogNotRunning), errors.Is(err, errTaskProgressLogSessionMismatch):
		return status.Error(codes.FailedPrecondition, err.Error())
	default:
		return status.Error(codes.Unavailable, "task progress log persistence is unavailable")
	}
}
