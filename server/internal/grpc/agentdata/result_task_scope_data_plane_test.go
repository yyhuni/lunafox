package agentdata

import (
	"context"
	"errors"
	"testing"

	scandomain "github.com/yyhuni/lunafox/server/internal/modules/scan/domain"
	scanrepo "github.com/yyhuni/lunafox/server/internal/modules/scan/repository"
)

type resultScopeTaskRepositoryStub struct {
	scanrepo.ScanTaskRepository
	task *scanrepo.ScanTaskRecord
	err  error
	ctx  context.Context
}

func (stub *resultScopeTaskRepositoryStub) GetByID(ctx context.Context, _ int) (*scanrepo.ScanTaskRecord, error) {
	stub.ctx = ctx
	return stub.task, stub.err
}

type resultScopeTargetLookupStub struct {
	target *scanrepo.ScanTargetRecord
	err    error
	ctx    context.Context
}

func (stub *resultScopeTargetLookupStub) GetTargetRefByScanIDContext(ctx context.Context, _ int) (*scanrepo.ScanTargetRecord, error) {
	stub.ctx = ctx
	return stub.target, stub.err
}

func TestResultTaskScopeDataPlaneRequiresRunningOwnerAndCurrentEpoch(t *testing.T) {
	agentID := 17
	sessionID := "session-17"
	epoch := int64(23)
	validTask := func() *scanrepo.ScanTaskRecord {
		return &scanrepo.ScanTaskRecord{
			ID: 101, ScanID: 12, Status: string(scandomain.TaskStatusRunning),
			AgentID: &agentID, AssignedAgentID: &agentID, AssignedSessionID: &sessionID, AssignedSessionEpoch: &epoch,
		}
	}
	tests := []struct {
		name    string
		mutate  func(*scanrepo.ScanTaskRecord)
		request ResultTaskScopeRequest
		wantErr error
	}{
		{name: "current", request: ResultTaskScopeRequest{TaskID: 101, AgentID: agentID, SessionID: sessionID, SessionEpoch: epoch}},
		{name: "not running", mutate: func(task *scanrepo.ScanTaskRecord) { task.Status = string(scandomain.TaskStatusSucceeded) }, request: ResultTaskScopeRequest{TaskID: 101, AgentID: agentID, SessionID: sessionID, SessionEpoch: epoch}, wantErr: errResultTaskNotRunning},
		{name: "wrong owner", request: ResultTaskScopeRequest{TaskID: 101, AgentID: agentID + 1, SessionID: sessionID, SessionEpoch: epoch}, wantErr: errResultTaskOwnershipMismatch},
		{name: "wrong process session", request: ResultTaskScopeRequest{TaskID: 101, AgentID: agentID, SessionID: "session-other", SessionEpoch: epoch}, wantErr: errResultTaskSessionMismatch},
		{name: "stale epoch", request: ResultTaskScopeRequest{TaskID: 101, AgentID: agentID, SessionID: sessionID, SessionEpoch: epoch + 1}, wantErr: errResultTaskSessionMismatch},
		{name: "incomplete v2 tuple", mutate: func(task *scanrepo.ScanTaskRecord) { task.AssignedSessionID = nil }, request: ResultTaskScopeRequest{TaskID: 101, AgentID: agentID, SessionID: sessionID, SessionEpoch: epoch}, wantErr: errResultTaskSessionMismatch},
		{name: "v2 plan cannot use legacy fallback", mutate: func(task *scanrepo.ScanTaskRecord) {
			task.AssignedAgentID = nil
			task.AssignedSessionID = nil
			task.HasResolvedExecutionPlan = true
		}, request: ResultTaskScopeRequest{TaskID: 101, AgentID: agentID, SessionID: sessionID, SessionEpoch: epoch}, wantErr: errResultTaskSessionMismatch},
		{name: "v2 request cannot use legacy fallback", mutate: func(task *scanrepo.ScanTaskRecord) {
			task.AssignedAgentID = nil
			task.AssignedSessionID = nil
			requestID := "550e8400-e29b-41d4-a716-446655440000"
			task.AssignedRequestID = &requestID
		}, request: ResultTaskScopeRequest{TaskID: 101, AgentID: agentID, SessionID: sessionID, SessionEpoch: epoch}, wantErr: errResultTaskSessionMismatch},
		{name: "missing session", request: ResultTaskScopeRequest{TaskID: 101, AgentID: agentID, SessionEpoch: epoch}, wantErr: errResultTaskSessionMismatch},
		{name: "missing epoch", request: ResultTaskScopeRequest{TaskID: 101, AgentID: agentID, SessionID: sessionID}, wantErr: errResultTaskSessionMismatch},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			task := validTask()
			if test.mutate != nil {
				test.mutate(task)
			}
			tasks := &resultScopeTaskRepositoryStub{task: task}
			targets := &resultScopeTargetLookupStub{target: &scanrepo.ScanTargetRecord{ID: 34}}
			plane := NewResultTaskScopeDataPlane(tasks, targets)
			type contextKey struct{}
			ctx := context.WithValue(context.Background(), contextKey{}, "caller")

			scope, err := plane.GetResultTaskScope(ctx, test.request)
			if test.wantErr != nil {
				if !errors.Is(err, test.wantErr) || scope != nil {
					t.Fatalf("GetResultTaskScope() = %+v, %v; want %v", scope, err, test.wantErr)
				}
				return
			}
			if err != nil || scope == nil || scope.TaskID != 101 || scope.ScanID != 12 || scope.TargetID != 34 {
				t.Fatalf("GetResultTaskScope() = %+v, %v", scope, err)
			}
			if tasks.ctx != ctx || tasks.ctx.Value(contextKey{}) != "caller" {
				t.Fatal("task scope lookup did not preserve caller context")
			}
			if targets.ctx != ctx || targets.ctx.Value(contextKey{}) != "caller" {
				t.Fatal("target scope lookup did not preserve caller context")
			}
		})
	}
}

func TestResultTaskScopeDataPlaneRejectsLegacyLeaseWithoutAssignmentSession(t *testing.T) {
	agentID := 17
	epoch := int64(23)
	tasks := &resultScopeTaskRepositoryStub{task: &scanrepo.ScanTaskRecord{
		ID: 101, ScanID: 12, Status: string(scandomain.TaskStatusRunning), AgentID: &agentID, AssignedSessionEpoch: &epoch,
	}}
	plane := NewResultTaskScopeDataPlane(tasks, &resultScopeTargetLookupStub{target: &scanrepo.ScanTargetRecord{ID: 34}})
	scope, err := plane.GetResultTaskScope(context.Background(), ResultTaskScopeRequest{
		TaskID: 101, AgentID: agentID, SessionID: "session-17", SessionEpoch: epoch,
	})
	if !errors.Is(err, errResultTaskSessionMismatch) || scope != nil {
		t.Fatalf("GetResultTaskScope() = %+v, %v; want session mismatch", scope, err)
	}
}
