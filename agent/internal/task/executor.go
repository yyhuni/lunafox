package task

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/yyhuni/lunafox/agent/internal/docker"
	"github.com/yyhuni/lunafox/agent/internal/domain"
	"github.com/yyhuni/lunafox/agent/internal/logger"
	"go.uber.org/zap"
)

const (
	defaultMaxRuntime    = 7 * 24 * time.Hour
	dockerActionTimeout  = 10 * time.Second
	dockerTailLogTimeout = 5 * time.Second
)

// Executor runs tasks inside worker containers.
type Executor struct {
	docker      DockerRunner
	client      statusReporter
	counter     *Counter
	agentSocket string
	maxRuntime  time.Duration

	mu        sync.Mutex
	running   map[int]context.CancelFunc
	cancelMu  sync.Mutex
	cancelled map[int]struct{}
	wg        sync.WaitGroup
	startMu   sync.RWMutex

	stopping atomic.Bool
}

type statusReporter interface {
	UpdateStatus(ctx context.Context, taskID int, status string, failure *domain.FailureDetail) error
}

const (
	failureKindUnknown             = "unknown"
	failureKindWorkerStartFailed   = "worker_start_failed"
	failureKindContainerWaitFailed = "container_wait_failed"
	failureKindTaskTimeout         = "task_timeout"
	failureKindContainerExitFailed = "container_exit_failed"
	failureKindDecodeConfigFailed  = "decode_config_failed"
	failureKindRuntimeError        = "runtime_error"
)

type taskSessionRegistry interface {
	RegisterTaskSession(taskID int, taskToken string)
	ClearTaskSession(taskID int, taskToken string)
}

type DockerRunner interface {
	StartWorker(ctx context.Context, t *domain.Task, agentSocket, taskToken string) (string, error)
	Wait(ctx context.Context, containerID string) (int64, error)
	Stop(ctx context.Context, containerID string) error
	Remove(ctx context.Context, containerID string) error
	TailLogs(ctx context.Context, containerID string, lines int) (string, error)
}

// NewExecutor creates an Executor.
func NewExecutor(dockerClient DockerRunner, taskClient statusReporter, counter *Counter, agentSocket string) *Executor {
	return &Executor{
		docker:      dockerClient,
		client:      taskClient,
		counter:     counter,
		agentSocket: agentSocket,
		maxRuntime:  defaultMaxRuntime,
		running:     map[int]context.CancelFunc{},
		cancelled:   map[int]struct{}{},
	}
}

// Start processes tasks from the queue.
func (e *Executor) Start(ctx context.Context, tasks <-chan *domain.Task) {
	for {
		select {
		case <-ctx.Done():
			return
		case t, ok := <-tasks:
			if !ok {
				return
			}
			if t == nil {
				continue
			}
			if e.stopping.Load() {
				// During shutdown/update: keep draining queue to avoid producer blocking,
				// but do not start any new container task.
				continue
			}
			if e.isCancelled(t.ID) {
				e.reportStatus(ctx, t.ID, "cancelled", nil)
				e.clearCancelled(t.ID)
				continue
			}

			e.startMu.RLock()
			if e.stopping.Load() {
				e.startMu.RUnlock()
				// During shutdown/update: keep draining queue to avoid producer blocking,
				// but do not start any new container task.
				continue
			}
			e.wg.Add(1)
			go func(task *domain.Task) {
				defer e.wg.Done()
				e.execute(ctx, task)
			}(t)
			e.startMu.RUnlock()
		}
	}
}

// CancelTask requests cancellation of a running task.
func (e *Executor) CancelTask(taskID int) {
	e.mu.Lock()
	cancel := e.running[taskID]
	e.mu.Unlock()
	if cancel != nil {
		cancel()
	}
}

// MarkCancelled records a task as cancelled to prevent execution.
func (e *Executor) MarkCancelled(taskID int) {
	e.cancelMu.Lock()
	e.cancelled[taskID] = struct{}{}
	e.cancelMu.Unlock()
}

func (e *Executor) reportStatus(ctx context.Context, taskID int, status string, failure *domain.FailureDetail) {
	if e.client == nil {
		return
	}
	if status != "failed" {
		failure = nil
	} else if failure == nil {
		failure = &domain.FailureDetail{Kind: failureKindUnknown}
	}
	if failure != nil {
		failure = &domain.FailureDetail{Kind: failure.Kind, Message: failure.Message}
		if strings.TrimSpace(failure.Kind) == "" {
			failure.Kind = failureKindUnknown
		}
	}
	statusCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 30*time.Second)
	defer cancel()
	if err := e.client.UpdateStatus(statusCtx, taskID, status, failure); err != nil {
		logger.Log.Warn("failed to report task status",
			zap.Int("taskId", taskID),
			zap.String("status", status),
			zap.String("errorMessage", failureMessage(failure)),
			zap.String("failureKind", failureKindValue(failure)),
			zap.Error(err),
		)
	}
}

func (e *Executor) execute(ctx context.Context, t *domain.Task) {
	defer e.clearCancelled(t.ID)

	if e.counter != nil {
		e.counter.Inc()
		defer e.counter.Dec()
	}

	if e.agentSocket == "" {
		e.reportStatus(ctx, t.ID, "failed", newFailureDetail(failureKindWorkerStartFailed, "missing worker runtime socket"))
		return
	}
	if e.docker == nil {
		e.reportStatus(ctx, t.ID, "failed", newFailureDetail(failureKindWorkerStartFailed, "docker client unavailable"))
		return
	}

	runCtx, cancel := context.WithTimeout(ctx, e.maxRuntime)
	defer cancel()

	taskToken, err := generateSessionToken()
	if err != nil {
		e.reportStatus(ctx, t.ID, "failed", newFailureDetail(failureKindWorkerStartFailed, "generate task token failed"))
		return
	}
	if sessionRegistry, ok := e.client.(taskSessionRegistry); ok {
		sessionRegistry.RegisterTaskSession(t.ID, taskToken)
		defer sessionRegistry.ClearTaskSession(t.ID, taskToken)
	}

	containerID, err := e.docker.StartWorker(runCtx, t, e.agentSocket, taskToken)
	if err != nil {
		logger.Log.Warn("failed to start worker container",
			zap.Int("taskId", t.ID),
			zap.Int("scanId", t.ScanID),
			zap.String("workflow", t.WorkflowID),
			zap.String("target", t.TargetName),
			zap.Error(err),
		)
		message := docker.TruncateErrorMessage(err.Error())
		e.reportStatus(ctx, t.ID, "failed", newFailureDetail(failureKindWorkerStartFailed, message))
		return
	}
	logger.Log.Info("worker container started",
		zap.Int("taskId", t.ID),
		zap.Int("scanId", t.ScanID),
		zap.String("workflow", t.WorkflowID),
		zap.String("containerId", containerID),
	)
	defer func() {
		e.removeContainer(containerID)
	}()

	e.trackCancel(t.ID, cancel)
	defer e.clearCancel(t.ID)

	exitCode, waitErr := e.docker.Wait(runCtx, containerID)
	if waitErr != nil {
		if errors.Is(waitErr, context.DeadlineExceeded) || errors.Is(runCtx.Err(), context.DeadlineExceeded) {
			e.handleTimeout(ctx, t, containerID)
			return
		}
		if errors.Is(waitErr, context.Canceled) || errors.Is(runCtx.Err(), context.Canceled) {
			e.handleCancel(ctx, t, containerID)
			return
		}
		logger.Log.Warn("worker container wait failed",
			zap.Int("taskId", t.ID),
			zap.Int("scanId", t.ScanID),
			zap.String("containerId", containerID),
			zap.Error(waitErr),
		)
		message := docker.TruncateErrorMessage(waitErr.Error())
		e.reportStatus(ctx, t.ID, "failed", newFailureDetail(failureKindContainerWaitFailed, message))
		return
	}

	if runCtx.Err() != nil {
		if errors.Is(runCtx.Err(), context.DeadlineExceeded) {
			e.handleTimeout(ctx, t, containerID)
			return
		}
		if errors.Is(runCtx.Err(), context.Canceled) {
			e.handleCancel(ctx, t, containerID)
			return
		}
	}

	if exitCode == 0 {
		logger.Log.Info("worker container completed",
			zap.Int("taskId", t.ID),
			zap.Int("scanId", t.ScanID),
			zap.String("containerId", containerID),
		)
		e.reportStatus(ctx, t.ID, "completed", nil)
		return
	}

	logs, _ := e.tailLogs(containerID, 100)
	message := logs
	if message == "" {
		message = fmt.Sprintf("container exited with code %d", exitCode)
	}
	message = docker.TruncateErrorMessage(message)
	logger.Log.Warn("worker container exited with failure",
		zap.Int("taskId", t.ID),
		zap.Int("scanId", t.ScanID),
		zap.String("containerId", containerID),
		zap.Int64("exitCode", exitCode),
		zap.String("logExcerpt", message),
	)
	e.reportStatus(ctx, t.ID, "failed", classifyFailureDetail(message))
}

func (e *Executor) handleCancel(ctx context.Context, t *domain.Task, containerID string) {
	e.stopContainer(containerID)
	logger.Log.Info("worker container cancelled",
		zap.Int("taskId", t.ID),
		zap.Int("scanId", t.ScanID),
		zap.String("containerId", containerID),
	)
	e.reportStatus(ctx, t.ID, "cancelled", nil)
}

func (e *Executor) handleTimeout(ctx context.Context, t *domain.Task, containerID string) {
	e.stopContainer(containerID)
	logger.Log.Warn("worker container timed out",
		zap.Int("taskId", t.ID),
		zap.Int("scanId", t.ScanID),
		zap.String("containerId", containerID),
	)
	message := docker.TruncateErrorMessage("task timed out")
	e.reportStatus(ctx, t.ID, "failed", newFailureDetail(failureKindTaskTimeout, message))
}

func classifyFailureDetail(message string) *domain.FailureDetail {
	trimmed := strings.TrimSpace(strings.ToLower(message))
	kind := failureKindContainerExitFailed
	if strings.Contains(trimmed, "decode workflow config") {
		kind = failureKindDecodeConfigFailed
	}
	return newFailureDetail(kind, message)
}

func newFailureDetail(kind, message string) *domain.FailureDetail {
	return &domain.FailureDetail{Kind: kind, Message: message}
}

func failureKindValue(failure *domain.FailureDetail) string {
	if failure == nil {
		return ""
	}
	return failure.Kind
}

func failureMessage(failure *domain.FailureDetail) string {
	if failure == nil {
		return ""
	}
	return failure.Message
}

func (e *Executor) trackCancel(taskID int, cancel context.CancelFunc) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.running[taskID] = cancel
}

func (e *Executor) clearCancel(taskID int) {
	e.mu.Lock()
	defer e.mu.Unlock()
	delete(e.running, taskID)
}

func (e *Executor) isCancelled(taskID int) bool {
	e.cancelMu.Lock()
	defer e.cancelMu.Unlock()
	_, ok := e.cancelled[taskID]
	return ok
}

func (e *Executor) clearCancelled(taskID int) {
	e.cancelMu.Lock()
	delete(e.cancelled, taskID)
	e.cancelMu.Unlock()
}

func (e *Executor) stopContainer(containerID string) {
	ctx, cancel := context.WithTimeout(context.Background(), dockerActionTimeout)
	defer cancel()
	if err := e.docker.Stop(ctx, containerID); err != nil {
		logger.Log.Warn("failed to stop worker container",
			zap.String("containerId", containerID),
			zap.Error(err),
		)
	}
}

func (e *Executor) removeContainer(containerID string) {
	ctx, cancel := context.WithTimeout(context.Background(), dockerActionTimeout)
	defer cancel()
	if err := e.docker.Remove(ctx, containerID); err != nil {
		logger.Log.Warn("failed to remove worker container",
			zap.String("containerId", containerID),
			zap.Error(err),
		)
	}
}

func (e *Executor) tailLogs(containerID string, lines int) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), dockerTailLogTimeout)
	defer cancel()
	return e.docker.TailLogs(ctx, containerID, lines)
}

// CancelAll requests cancellation for all running tasks.
func (e *Executor) CancelAll() {
	e.mu.Lock()
	cancels := make([]context.CancelFunc, 0, len(e.running))
	for _, cancel := range e.running {
		cancels = append(cancels, cancel)
	}
	e.mu.Unlock()

	for _, cancel := range cancels {
		cancel()
	}
}

// Shutdown cancels running tasks and waits for completion.
func (e *Executor) Shutdown(ctx context.Context) error {
	// Shutdown ordering is strict: block new starts, cancel in-flight work,
	// then wait for all execute goroutines to return.
	e.startMu.Lock()
	defer e.startMu.Unlock()

	e.stopping.Store(true)
	e.CancelAll()

	done := make(chan struct{})
	go func() {
		e.wg.Wait()
		close(done)
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-done:
		return nil
	}
}

func generateSessionToken() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}
