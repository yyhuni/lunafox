package agentdata

import (
	"context"
	"errors"
	"testing"
	"time"

	scanapp "github.com/yyhuni/lunafox/server/internal/modules/scan/application"
	scandomain "github.com/yyhuni/lunafox/server/internal/modules/scan/domain"
	scanrepo "github.com/yyhuni/lunafox/server/internal/modules/scan/repository"
)

type progressTaskRepositoryStub struct {
	scanrepo.ScanTaskRepository
	task *scanrepo.ScanTaskRecord
	err  error
	ctx  context.Context
}

func (stub *progressTaskRepositoryStub) GetByID(ctx context.Context, _ int) (*scanrepo.ScanTaskRecord, error) {
	stub.ctx = ctx
	return stub.task, stub.err
}

type progressLogApplicationStub struct {
	request *scanapp.TaskProgressLogBatchCreateRequest
	calls   int
}

func (*progressLogApplicationStub) ListByScanID(context.Context, int, *scanapp.TaskProgressLogListQuery) ([]scanapp.TaskProgressLogEntry, bool, error) {
	return nil, false, nil
}

func (stub *progressLogApplicationStub) BatchCreateTaskProgressLogs(_ context.Context, request *scanapp.TaskProgressLogBatchCreateRequest) (int, int, error) {
	stub.calls++
	stub.request = request
	return 1, 0, nil
}

func TestTaskProgressLogDataPlaneRequiresRunningCurrentLease(t *testing.T) {
	agentID := 17
	sessionID := "session-17"
	epoch := int64(23)
	validTask := func() *scanrepo.ScanTaskRecord {
		return &scanrepo.ScanTaskRecord{
			ID: 101, ScanID: 12, Status: string(scandomain.TaskStatusRunning),
			AgentID: &agentID, AssignedAgentID: &agentID, AssignedSessionID: &sessionID, AssignedSessionEpoch: &epoch,
		}
	}
	validBatch := func() TaskProgressLogBatch {
		return TaskProgressLogBatch{
			ScanID: 12, TaskID: 101, AgentID: agentID, SessionID: sessionID, SessionEpoch: epoch, RequestID: "request-1",
			Entries: []TaskProgressLogEntry{{Sequence: 1, Level: "info", Content: "started", EmittedAt: time.Now().UTC()}},
		}
	}
	tests := []struct {
		name    string
		mutate  func(*scanrepo.ScanTaskRecord, *TaskProgressLogBatch)
		wantErr error
	}{
		{name: "current"},
		{name: "not running", mutate: func(task *scanrepo.ScanTaskRecord, _ *TaskProgressLogBatch) {
			task.Status = string(scandomain.TaskStatusSucceeded)
		}, wantErr: errTaskProgressLogNotRunning},
		{name: "wrong scan", mutate: func(_ *scanrepo.ScanTaskRecord, batch *TaskProgressLogBatch) { batch.ScanID++ }, wantErr: scanapp.ErrScanTaskNotOwned},
		{name: "wrong owner", mutate: func(_ *scanrepo.ScanTaskRecord, batch *TaskProgressLogBatch) { batch.AgentID++ }, wantErr: errTaskProgressLogOwnershipMismatch},
		{name: "wrong process session", mutate: func(_ *scanrepo.ScanTaskRecord, batch *TaskProgressLogBatch) { batch.SessionID = "session-other" }, wantErr: errTaskProgressLogSessionMismatch},
		{name: "stale epoch", mutate: func(_ *scanrepo.ScanTaskRecord, batch *TaskProgressLogBatch) { batch.SessionEpoch++ }, wantErr: errTaskProgressLogSessionMismatch},
		{name: "incomplete v2 tuple", mutate: func(task *scanrepo.ScanTaskRecord, _ *TaskProgressLogBatch) { task.AssignedSessionID = nil }, wantErr: errTaskProgressLogSessionMismatch},
		{name: "v2 plan cannot use legacy fallback", mutate: func(task *scanrepo.ScanTaskRecord, _ *TaskProgressLogBatch) {
			task.AssignedAgentID = nil
			task.AssignedSessionID = nil
			task.HasResolvedExecutionPlan = true
		}, wantErr: errTaskProgressLogSessionMismatch},
		{name: "v2 request cannot use legacy fallback", mutate: func(task *scanrepo.ScanTaskRecord, _ *TaskProgressLogBatch) {
			task.AssignedAgentID = nil
			task.AssignedSessionID = nil
			requestID := "550e8400-e29b-41d4-a716-446655440000"
			task.AssignedRequestID = &requestID
		}, wantErr: errTaskProgressLogSessionMismatch},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			task := validTask()
			batch := validBatch()
			if test.mutate != nil {
				test.mutate(task, &batch)
			}
			tasks := &progressTaskRepositoryStub{task: task}
			logs := &progressLogApplicationStub{}
			plane := NewTaskProgressLogDataPlane(logs, tasks)
			type contextKey struct{}
			ctx := context.WithValue(context.Background(), contextKey{}, "caller")

			accepted, duplicates, err := plane.WriteTaskProgressLogs(ctx, batch)
			if test.wantErr != nil {
				if !errors.Is(err, test.wantErr) || accepted != 0 || duplicates != 0 || logs.calls != 0 {
					t.Fatalf("WriteTaskProgressLogs() = %d, %d, %v, calls=%d; want %v", accepted, duplicates, err, logs.calls, test.wantErr)
				}
				return
			}
			if err != nil || accepted != 1 || duplicates != 0 || logs.calls != 1 || logs.request == nil {
				t.Fatalf("WriteTaskProgressLogs() = %d, %d, %v, calls=%d request=%#v", accepted, duplicates, err, logs.calls, logs.request)
			}
			if tasks.ctx != ctx || tasks.ctx.Value(contextKey{}) != "caller" {
				t.Fatal("task lease lookup did not preserve caller context")
			}
		})
	}
}

func TestTaskProgressLogDataPlaneRejectsLegacyLeaseWithoutAssignmentSession(t *testing.T) {
	agentID := 17
	epoch := int64(23)
	task := &scanrepo.ScanTaskRecord{
		ID: 101, ScanID: 12, Status: string(scandomain.TaskStatusRunning), AgentID: &agentID, AssignedSessionEpoch: &epoch,
	}
	logs := &progressLogApplicationStub{}
	plane := NewTaskProgressLogDataPlane(logs, &progressTaskRepositoryStub{task: task})
	accepted, _, err := plane.WriteTaskProgressLogs(context.Background(), TaskProgressLogBatch{
		ScanID: 12, TaskID: 101, AgentID: agentID, SessionID: "session-17", SessionEpoch: epoch, RequestID: "request-1",
		Entries: []TaskProgressLogEntry{{Sequence: 1, Level: "info", Content: "started", EmittedAt: time.Now().UTC()}},
	})
	if !errors.Is(err, errTaskProgressLogSessionMismatch) || accepted != 0 || logs.calls != 0 {
		t.Fatalf("WriteTaskProgressLogs() = %d, %v, calls=%d; want session mismatch", accepted, err, logs.calls)
	}
}
