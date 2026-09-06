package application

import (
	"context"
	"errors"
	"sort"
	"testing"
	"time"

	cleanupdomain "github.com/yyhuni/lunafox/server/internal/modules/targetcleanup/domain"
)

type cleanupJobStoreStub struct {
	jobs      []cleanupdomain.CleanupJob
	failures  []cleanupFailureRecord
	deferrals []cleanupDeferralRecord
	backlog   cleanupdomain.CleanupBacklog
	listCalls int
}

type cleanupFailureRecord struct {
	jobID      int
	retryCount int
	next       time.Time
	class      string
}

type cleanupDeferralRecord struct {
	jobID int
	next  time.Time
}

func (stub *cleanupJobStoreStub) ListDue(context.Context, time.Time, int) ([]cleanupdomain.CleanupJob, error) {
	stub.listCalls++
	return append([]cleanupdomain.CleanupJob(nil), stub.jobs...), nil
}

func (stub *cleanupJobStoreStub) RecordFailure(_ context.Context, jobID, retryCount int, next time.Time, class string) error {
	stub.failures = append(stub.failures, cleanupFailureRecord{jobID: jobID, retryCount: retryCount, next: next, class: class})
	return nil
}

func (stub *cleanupJobStoreStub) Defer(_ context.Context, jobID int, next time.Time) error {
	stub.deferrals = append(stub.deferrals, cleanupDeferralRecord{jobID: jobID, next: next})
	return nil
}

func (stub *cleanupJobStoreStub) InspectBacklog(context.Context) (cleanupdomain.CleanupBacklog, error) {
	return stub.backlog, nil
}

type cleanupReconcilerStub struct {
	results map[int]TargetCleanupReconciliationResult
	errors  map[int]error
	called  []int
}

func (stub *cleanupReconcilerStub) Reconcile(_ context.Context, job cleanupdomain.CleanupJob, _ TargetCleanupRunOptions) (TargetCleanupReconciliationResult, error) {
	stub.called = append(stub.called, job.ID)
	return stub.results[job.ID], stub.errors[job.ID]
}

type cleanupObserverStub struct {
	started  []int
	finished []TargetCleanupRunEvent
	backlogs []cleanupdomain.CleanupBacklog
}

type persistedCleanupJobStoreStub struct {
	jobs []cleanupdomain.CleanupJob
}

func (stub *persistedCleanupJobStoreStub) ListDue(_ context.Context, dueAt time.Time, limit int) ([]cleanupdomain.CleanupJob, error) {
	due := make([]cleanupdomain.CleanupJob, 0, len(stub.jobs))
	for _, job := range stub.jobs {
		if job.Status == cleanupdomain.CleanupJobPending && !job.NextRetryAt.After(dueAt) {
			due = append(due, job)
		}
	}
	sort.Slice(due, func(left, right int) bool {
		if due[left].NextRetryAt.Equal(due[right].NextRetryAt) {
			return due[left].ID < due[right].ID
		}
		return due[left].NextRetryAt.Before(due[right].NextRetryAt)
	})
	if len(due) > limit {
		due = due[:limit]
	}
	return due, nil
}

func (stub *persistedCleanupJobStoreStub) RecordFailure(_ context.Context, jobID, retryCount int, nextRetryAt time.Time, failureClass string) error {
	for index := range stub.jobs {
		if stub.jobs[index].ID == jobID {
			stub.jobs[index].RetryCount = retryCount
			stub.jobs[index].NextRetryAt = nextRetryAt
			stub.jobs[index].LastError = failureClass
		}
	}
	return nil
}

func (stub *persistedCleanupJobStoreStub) Defer(_ context.Context, jobID int, nextRetryAt time.Time) error {
	for index := range stub.jobs {
		if stub.jobs[index].ID == jobID {
			stub.jobs[index].NextRetryAt = nextRetryAt
		}
	}
	return nil
}

func (stub *persistedCleanupJobStoreStub) InspectBacklog(context.Context) (cleanupdomain.CleanupBacklog, error) {
	var backlog cleanupdomain.CleanupBacklog
	for _, job := range stub.jobs {
		if job.Status != cleanupdomain.CleanupJobPending {
			continue
		}
		backlog.UnfinishedCount++
		if backlog.OldestCreatedAt == nil || job.CreatedAt.Before(*backlog.OldestCreatedAt) {
			createdAt := job.CreatedAt
			backlog.OldestCreatedAt = &createdAt
		}
	}
	return backlog, nil
}

type cleanupReconcilerFunc func(context.Context, cleanupdomain.CleanupJob, TargetCleanupRunOptions) (TargetCleanupReconciliationResult, error)

func (function cleanupReconcilerFunc) Reconcile(ctx context.Context, job cleanupdomain.CleanupJob, options TargetCleanupRunOptions) (TargetCleanupReconciliationResult, error) {
	return function(ctx, job, options)
}

func (stub *cleanupObserverStub) RunStarted(job cleanupdomain.CleanupJob, _ time.Time) {
	stub.started = append(stub.started, job.ID)
}

func (stub *cleanupObserverStub) RunFinished(event TargetCleanupRunEvent) {
	stub.finished = append(stub.finished, event)
}

func (stub *cleanupObserverStub) BacklogObserved(backlog cleanupdomain.CleanupBacklog) {
	stub.backlogs = append(stub.backlogs, backlog)
}

func TestTargetCleanupRunnerFailureRemainsRetryableAndDoesNotBlockOtherDueJobs(t *testing.T) {
	now := time.Date(2026, 8, 6, 12, 0, 0, 0, time.UTC)
	jobs := &cleanupJobStoreStub{jobs: []cleanupdomain.CleanupJob{{ID: 1, TargetID: 1, RetryCount: 0}, {ID: 2, TargetID: 2, RetryCount: 7}}}
	reconciler := &cleanupReconcilerStub{
		results: map[int]TargetCleanupReconciliationResult{2: {Completed: true}},
		errors:  map[int]error{1: errors.New("database lock timeout")},
	}
	observer := &cleanupObserverStub{}
	runner, err := NewTargetCleanupRunner(jobs, reconciler, TargetCleanupRunOptions{AssetBatchSize: 1, MaxAssetBatches: 1, MaxRunDuration: time.Minute}, observer)
	if err != nil {
		t.Fatalf("NewTargetCleanupRunner(): %v", err)
	}
	runner.WithClock(func() time.Time { return now })

	if err := runner.RunPass(context.Background()); err != nil {
		t.Fatalf("RunPass(): %v", err)
	}
	if len(reconciler.called) != 2 || reconciler.called[0] != 1 || reconciler.called[1] != 2 {
		t.Fatalf("due jobs were not isolated: %+v", reconciler.called)
	}
	if len(jobs.failures) != 1 || jobs.failures[0].retryCount != 1 || jobs.failures[0].class != "database_lock" || !jobs.failures[0].next.After(now) {
		t.Fatalf("retry record = %+v", jobs.failures)
	}
	if len(observer.finished) != 2 || observer.finished[0].Outcome != "failed" || observer.finished[1].Outcome != "completed" {
		t.Fatalf("run outcomes = %+v", observer.finished)
	}
}

func TestTargetCleanupRunnerDeferralDoesNotIncrementErrorState(t *testing.T) {
	now := time.Date(2026, 8, 6, 12, 0, 0, 0, time.UTC)
	jobs := &cleanupJobStoreStub{jobs: []cleanupdomain.CleanupJob{{ID: 1, TargetID: 1, RetryCount: 4}}}
	reconciler := &cleanupReconcilerStub{results: map[int]TargetCleanupReconciliationResult{1: {Deferred: true}}}
	runner, err := NewTargetCleanupRunner(jobs, reconciler, TargetCleanupRunOptions{AssetBatchSize: 1, MaxAssetBatches: 1, MaxRunDuration: time.Minute}, nil)
	if err != nil {
		t.Fatalf("NewTargetCleanupRunner(): %v", err)
	}
	runner.WithClock(func() time.Time { return now })

	if err := runner.RunPass(context.Background()); err != nil {
		t.Fatalf("RunPass(): %v", err)
	}
	if len(jobs.failures) != 0 || len(jobs.deferrals) != 1 || jobs.deferrals[0].jobID != 1 || !jobs.deferrals[0].next.After(now) {
		t.Fatalf("deferred run changed retry state: failures=%+v deferrals=%+v", jobs.failures, jobs.deferrals)
	}
}

func TestTargetCleanupRetryDelayIsCapped(t *testing.T) {
	if got := targetCleanupRetryDelay(100); got != targetCleanupRetryCap {
		t.Fatalf("targetCleanupRetryDelay(100) = %s, want %s", got, targetCleanupRetryCap)
	}
}

func TestTargetCleanupRunnerRepeatedFailuresRemainPendingForAnotherRun(t *testing.T) {
	now := time.Date(2026, 8, 6, 12, 0, 0, 0, time.UTC)
	jobs := &cleanupJobStoreStub{jobs: []cleanupdomain.CleanupJob{{ID: 1, TargetID: 1, RetryCount: 0}}}
	reconciler := &cleanupReconcilerStub{errors: map[int]error{1: errors.New("database unavailable")}}
	runner, err := NewTargetCleanupRunner(jobs, reconciler, TargetCleanupRunOptions{AssetBatchSize: 1, MaxAssetBatches: 1, MaxRunDuration: time.Minute}, nil)
	if err != nil {
		t.Fatalf("NewTargetCleanupRunner(): %v", err)
	}
	runner.WithClock(func() time.Time { return now })

	if err := runner.RunPass(context.Background()); err != nil {
		t.Fatalf("first RunPass(): %v", err)
	}
	jobs.jobs[0].RetryCount = 1
	if err := runner.RunPass(context.Background()); err != nil {
		t.Fatalf("second RunPass(): %v", err)
	}
	if len(jobs.failures) != 2 || jobs.failures[0].retryCount != 1 || jobs.failures[1].retryCount != 2 {
		t.Fatalf("repeated failures became non-retryable: %+v", jobs.failures)
	}
}

func TestTargetCleanupRunnerRestartDiscoversPersistedDeferredJob(t *testing.T) {
	now := time.Date(2026, 8, 6, 12, 0, 0, 0, time.UTC)
	jobs := &persistedCleanupJobStoreStub{jobs: []cleanupdomain.CleanupJob{{
		ID: 1, TargetID: 7, Status: cleanupdomain.CleanupJobPending, NextRetryAt: now, CreatedAt: now,
	}}}
	options := TargetCleanupRunOptions{AssetBatchSize: 1, MaxAssetBatches: 1, MaxRunDuration: time.Minute}

	first, err := NewTargetCleanupRunner(jobs, cleanupReconcilerFunc(func(context.Context, cleanupdomain.CleanupJob, TargetCleanupRunOptions) (TargetCleanupReconciliationResult, error) {
		return TargetCleanupReconciliationResult{Deferred: true}, nil
	}), options, nil)
	if err != nil {
		t.Fatalf("NewTargetCleanupRunner(first): %v", err)
	}
	first.WithClock(func() time.Time { return now })
	if err := first.RunPass(context.Background()); err != nil {
		t.Fatalf("first RunPass(): %v", err)
	}
	if !jobs.jobs[0].NextRetryAt.After(now) || jobs.jobs[0].RetryCount != 0 {
		t.Fatalf("first Runner did not persist a non-error deferral: %+v", jobs.jobs[0])
	}

	restartedAt := jobs.jobs[0].NextRetryAt.Add(time.Second)
	completedCalls := 0
	second, err := NewTargetCleanupRunner(jobs, cleanupReconcilerFunc(func(_ context.Context, job cleanupdomain.CleanupJob, _ TargetCleanupRunOptions) (TargetCleanupReconciliationResult, error) {
		completedCalls++
		if job.ID != 1 || job.TargetID != 7 {
			t.Fatalf("restarted Runner received wrong persisted Job: %+v", job)
		}
		jobs.jobs[0].Status = cleanupdomain.CleanupJobCompleted
		return TargetCleanupReconciliationResult{Completed: true}, nil
	}), options, nil)
	if err != nil {
		t.Fatalf("NewTargetCleanupRunner(second): %v", err)
	}
	second.WithClock(func() time.Time { return restartedAt })
	if err := second.RunPass(context.Background()); err != nil {
		t.Fatalf("restarted RunPass(): %v", err)
	}
	if completedCalls != 1 || jobs.jobs[0].Status != cleanupdomain.CleanupJobCompleted {
		t.Fatalf("restart did not resume persistent database truth: calls=%d job=%+v", completedCalls, jobs.jobs[0])
	}

	thirdCalls := 0
	third, err := NewTargetCleanupRunner(jobs, cleanupReconcilerFunc(func(context.Context, cleanupdomain.CleanupJob, TargetCleanupRunOptions) (TargetCleanupReconciliationResult, error) {
		thirdCalls++
		return TargetCleanupReconciliationResult{}, nil
	}), options, nil)
	if err != nil {
		t.Fatalf("NewTargetCleanupRunner(third): %v", err)
	}
	third.WithClock(func() time.Time { return restartedAt.Add(time.Hour) })
	if err := third.RunPass(context.Background()); err != nil {
		t.Fatalf("completed RunPass(): %v", err)
	}
	if thirdCalls != 0 {
		t.Fatalf("completed Job was rediscovered as due: calls=%d", thirdCalls)
	}
}
