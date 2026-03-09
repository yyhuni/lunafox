package task

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/yyhuni/lunafox/agent/internal/domain"
)

type fakeReporter struct {
	status      string
	msg         string
	failureKind string
}

type fakeDockerRunner struct {
	startWorkerFn func(ctx context.Context, t *domain.Task, agentSocket, taskToken string) (string, error)
	waitFn        func(ctx context.Context, containerID string) (int64, error)
	stopFn        func(ctx context.Context, containerID string) error
	removeFn      func(ctx context.Context, containerID string) error
	tailLogsFn    func(ctx context.Context, containerID string, lines int) (string, error)
}

func (fake *fakeDockerRunner) StartWorker(ctx context.Context, t *domain.Task, agentSocket, taskToken string) (string, error) {
	if fake.startWorkerFn == nil {
		return "container-1", nil
	}
	return fake.startWorkerFn(ctx, t, agentSocket, taskToken)
}

func (fake *fakeDockerRunner) Wait(ctx context.Context, containerID string) (int64, error) {
	if fake.waitFn == nil {
		return 0, nil
	}
	return fake.waitFn(ctx, containerID)
}

func (fake *fakeDockerRunner) Stop(ctx context.Context, containerID string) error {
	if fake.stopFn == nil {
		return nil
	}
	return fake.stopFn(ctx, containerID)
}

func (fake *fakeDockerRunner) Remove(ctx context.Context, containerID string) error {
	if fake.removeFn == nil {
		return nil
	}
	return fake.removeFn(ctx, containerID)
}

func (fake *fakeDockerRunner) TailLogs(ctx context.Context, containerID string, lines int) (string, error) {
	if fake.tailLogsFn == nil {
		return "", nil
	}
	return fake.tailLogsFn(ctx, containerID, lines)
}

func (f *fakeReporter) UpdateStatus(ctx context.Context, taskID int, status, errorMessage, failureKind string) error {
	f.status = status
	f.msg = errorMessage
	f.failureKind = failureKind
	return nil
}

func TestExecutorMissingWorkerRuntimeSocket(t *testing.T) {
	reporter := &fakeReporter{}
	exec := &Executor{
		client: reporter,
	}

	exec.execute(context.Background(), &domain.Task{ID: 1})
	if reporter.status != "failed" {
		t.Fatalf("expected failed status, got %s", reporter.status)
	}
	if reporter.msg == "" {
		t.Fatalf("expected error message")
	}
	if reporter.failureKind != "worker_start_failed" {
		t.Fatalf("expected worker_start_failed, got %q", reporter.failureKind)
	}
}

func TestExecutorDockerUnavailable(t *testing.T) {
	reporter := &fakeReporter{}
	exec := &Executor{
		client:      reporter,
		agentSocket: "/run/lunafox/worker-runtime.sock",
	}

	exec.execute(context.Background(), &domain.Task{ID: 2})
	if reporter.status != "failed" {
		t.Fatalf("expected failed status, got %s", reporter.status)
	}
	if reporter.msg == "" {
		t.Fatalf("expected error message")
	}
	if reporter.failureKind != "worker_start_failed" {
		t.Fatalf("expected worker_start_failed, got %q", reporter.failureKind)
	}
}

func TestExecutorCancelAll(t *testing.T) {
	exec := &Executor{
		running: map[int]context.CancelFunc{},
	}
	calls := 0
	exec.running[1] = func() { calls++ }
	exec.running[2] = func() { calls++ }

	exec.CancelAll()
	if calls != 2 {
		t.Fatalf("expected cancel calls, got %d", calls)
	}
}

func TestExecutorShutdownWaits(t *testing.T) {
	exec := &Executor{
		running: map[int]context.CancelFunc{},
	}
	calls := 0
	exec.running[1] = func() { calls++ }

	exec.wg.Add(1)
	go func() {
		time.Sleep(10 * time.Millisecond)
		exec.wg.Done()
	}()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	if err := exec.Shutdown(ctx); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if calls != 1 {
		t.Fatalf("expected cancel call")
	}
}

func TestExecutorShutdownTimeout(t *testing.T) {
	exec := &Executor{
		running: map[int]context.CancelFunc{},
	}
	exec.wg.Add(1)
	defer exec.wg.Done()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	if err := exec.Shutdown(ctx); err == nil {
		t.Fatalf("expected timeout error")
	}
}

func TestExecutorShutdownPreventsLateStartAfterDrainRace(t *testing.T) {
	started := make(chan int, 1)

	fakeDocker := &fakeDockerRunner{
		startWorkerFn: func(ctx context.Context, t *domain.Task, agentSocket, taskToken string) (string, error) {
			started <- t.ID
			return "container-1", nil
		},
		waitFn: func(ctx context.Context, containerID string) (int64, error) {
			return 0, nil
		},
	}

	exec := NewExecutor(fakeDocker, nil, nil, "/run/lunafox/worker-runtime.sock")
	runCtx, runCancel := context.WithCancel(context.Background())
	defer runCancel()

	tasks := make(chan *domain.Task)
	go exec.Start(runCtx, tasks)

	// Hold cancelMu so Start() gets past the first stopping check, then blocks
	// before wg.Add(1). This reproduces the Add/Wait race window deterministically.
	exec.cancelMu.Lock()
	locked := true
	defer func() {
		if locked {
			exec.cancelMu.Unlock()
		}
	}()

	sent := make(chan struct{})
	go func() {
		tasks <- &domain.Task{ID: 42}
		close(sent)
	}()

	select {
	case <-sent:
	case <-time.After(time.Second):
		t.Fatalf("timed out sending task to executor")
	}

	shutdownDone := make(chan error, 1)
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		shutdownDone <- exec.Shutdown(ctx)
	}()

	select {
	case err := <-shutdownDone:
		if err != nil {
			t.Fatalf("unexpected shutdown error: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatalf("shutdown did not finish in time")
	}

	exec.cancelMu.Unlock()
	locked = false

	select {
	case taskID := <-started:
		t.Fatalf("task %d started after shutdown", taskID)
	case <-time.After(80 * time.Millisecond):
	}
}

func TestExecutorFailurePathUsesTimeoutContexts(t *testing.T) {
	reporter := &fakeReporter{}
	tailLogsHasDeadline := false
	removeHasDeadline := false

	fakeDocker := &fakeDockerRunner{
		startWorkerFn: func(ctx context.Context, t *domain.Task, agentSocket, taskToken string) (string, error) {
			return "container-1", nil
		},
		waitFn: func(ctx context.Context, containerID string) (int64, error) {
			return 1, nil
		},
		tailLogsFn: func(ctx context.Context, containerID string, lines int) (string, error) {
			_, tailLogsHasDeadline = ctx.Deadline()
			return "tool failed", nil
		},
		removeFn: func(ctx context.Context, containerID string) error {
			_, removeHasDeadline = ctx.Deadline()
			return nil
		},
	}

	exec := NewExecutor(fakeDocker, reporter, nil, "/run/lunafox/worker-runtime.sock")
	exec.execute(context.Background(), &domain.Task{ID: 10, ScanID: 20})

	if reporter.status != "failed" {
		t.Fatalf("expected failed status, got %s", reporter.status)
	}
	if !tailLogsHasDeadline {
		t.Fatalf("expected tail logs context to have deadline")
	}
	if !removeHasDeadline {
		t.Fatalf("expected remove context to have deadline")
	}
	if reporter.failureKind != "container_exit_failed" {
		t.Fatalf("expected container_exit_failed, got %q", reporter.failureKind)
	}
}

func TestExecutorWaitFailureUsesContainerWaitFailed(t *testing.T) {
	reporter := &fakeReporter{}
	fakeDocker := &fakeDockerRunner{
		startWorkerFn: func(ctx context.Context, t *domain.Task, agentSocket, taskToken string) (string, error) {
			return "container-1", nil
		},
		waitFn: func(ctx context.Context, containerID string) (int64, error) {
			return 0, errors.New("wait failed")
		},
	}

	exec := NewExecutor(fakeDocker, reporter, nil, "/run/lunafox/worker-runtime.sock")
	exec.execute(context.Background(), &domain.Task{ID: 12, ScanID: 23})

	if reporter.status != "failed" {
		t.Fatalf("expected failed status, got %s", reporter.status)
	}
	if reporter.failureKind != "container_wait_failed" {
		t.Fatalf("expected container_wait_failed, got %q", reporter.failureKind)
	}
}

func TestExecutorExitFailureClassifiesDecodeConfigFailure(t *testing.T) {
	reporter := &fakeReporter{}
	fakeDocker := &fakeDockerRunner{
		startWorkerFn: func(ctx context.Context, t *domain.Task, agentSocket, taskToken string) (string, error) {
			return "container-1", nil
		},
		waitFn: func(ctx context.Context, containerID string) (int64, error) {
			return 1, nil
		},
		tailLogsFn: func(ctx context.Context, containerID string, lines int) (string, error) {
			return "decode workflow config subdomain_discovery: invalid config", nil
		},
	}

	exec := NewExecutor(fakeDocker, reporter, nil, "/run/lunafox/worker-runtime.sock")
	exec.execute(context.Background(), &domain.Task{ID: 11, ScanID: 22})

	if reporter.status != "failed" {
		t.Fatalf("expected failed status, got %s", reporter.status)
	}
	if reporter.failureKind != "decode_config_failed" {
		t.Fatalf("expected decode_config_failed, got %q", reporter.failureKind)
	}
}

func TestExecutorHandleTimeoutUsesDeadlineOnStop(t *testing.T) {
	reporter := &fakeReporter{}
	stopHasDeadline := false

	fakeDocker := &fakeDockerRunner{
		stopFn: func(ctx context.Context, containerID string) error {
			_, stopHasDeadline = ctx.Deadline()
			return nil
		},
	}

	exec := NewExecutor(fakeDocker, reporter, nil, "/run/lunafox/worker-runtime.sock")
	exec.handleTimeout(context.Background(), &domain.Task{ID: 1, ScanID: 2}, "container-1")

	if !stopHasDeadline {
		t.Fatalf("expected stop context to have deadline")
	}
	if reporter.status != "failed" {
		t.Fatalf("expected failed status, got %s", reporter.status)
	}
	if reporter.failureKind != "task_timeout" {
		t.Fatalf("expected task_timeout, got %q", reporter.failureKind)
	}
}

func TestExecutorInjectsWorkerRuntimeSocketAndTaskToken(t *testing.T) {
	reporter := &fakeReporter{}
	var (
		gotAgentSocket string
		gotTaskToken   string
	)

	fakeDocker := &fakeDockerRunner{
		startWorkerFn: func(ctx context.Context, t *domain.Task, agentSocket, taskToken string) (string, error) {
			gotAgentSocket = agentSocket
			gotTaskToken = taskToken
			return "container-1", nil
		},
		waitFn: func(ctx context.Context, containerID string) (int64, error) {
			return 0, nil
		},
	}

	socketPath := "/run/lunafox/worker-runtime.sock"
	exec := NewExecutor(fakeDocker, reporter, nil, socketPath)
	exec.execute(context.Background(), &domain.Task{ID: 9, ScanID: 10})

	if gotAgentSocket != socketPath {
		t.Fatalf("expected injected agent socket %q, got %q", socketPath, gotAgentSocket)
	}
	if gotTaskToken == "" {
		t.Fatalf("expected non-empty task token")
	}
}
