package application

import (
	"context"
	"fmt"
	"sync"
	"time"

	scanapp "github.com/yyhuni/lunafox/server/internal/modules/scan/application"
	"go.uber.org/zap"
)

const (
	schedulerIdleInterval = time.Minute
	handoffDeadline       = 5 * time.Minute
)

type HandoffDispatcher interface {
	Dispatch(ctx context.Context, input FrozenDispatchInput) (*scanapp.BatchScanResult, error)
}

type SchedulerPassResult struct {
	Materialized    int
	Attempted       bool
	PreAttemptError bool
	Err             error
}

func (result SchedulerPassResult) Progressed() bool {
	return result.Materialized > 0 || result.Attempted
}

type SchedulerController struct {
	repository ExecutionRepository
	dispatcher HandoffDispatcher
	clock      RuntimeClock
	waiter     RuntimeWaiter
	logger     EventLogger
	startOnce  sync.Once
	done       chan struct{}
}

func NewSchedulerController(repository ExecutionRepository, dispatcher HandoffDispatcher) *SchedulerController {
	return &SchedulerController{
		repository: repository,
		dispatcher: dispatcher,
		clock:      systemRuntimeClock{},
		waiter:     systemRuntimeWaiter{},
		logger:     packageEventLogger{},
		done:       make(chan struct{}),
	}
}

func (controller *SchedulerController) WithRuntime(clock RuntimeClock, waiter RuntimeWaiter, logger EventLogger) *SchedulerController {
	if controller == nil {
		return nil
	}
	if clock != nil {
		controller.clock = clock
	}
	if waiter != nil {
		controller.waiter = waiter
	}
	if logger != nil {
		controller.logger = logger
	}
	return controller
}

func (controller *SchedulerController) Start(ctx context.Context) {
	if controller == nil {
		return
	}
	controller.startOnce.Do(func() {
		go controller.run(ctx)
	})
}

func (controller *SchedulerController) Done() <-chan struct{} {
	if controller == nil {
		closed := make(chan struct{})
		close(closed)
		return closed
	}
	return controller.done
}

func (controller *SchedulerController) run(ctx context.Context) {
	defer close(controller.done)
	for ctx.Err() == nil {
		result := controller.RunPass(ctx)
		if result.Err != nil {
			controller.logger.Error("Scheduled scan controller pass failed",
				zap.Int("scheduled_scan.materialized_count", result.Materialized),
				zap.Bool("scheduled_scan.attempted", result.Attempted),
				zap.String("error_kind", runtimeErrorKind(result.Err)),
			)
		}
		if result.Progressed() && !result.PreAttemptError {
			continue
		}
		if err := controller.waiter.Wait(ctx, schedulerIdleInterval); err != nil {
			return
		}
	}
}

func (controller *SchedulerController) RunPass(ctx context.Context) SchedulerPassResult {
	if controller == nil || controller.repository == nil || controller.dispatcher == nil || controller.clock == nil {
		return SchedulerPassResult{PreAttemptError: true, Err: fmt.Errorf("scheduled scan controller is not configured")}
	}
	evaluationAt := controller.clock.Now().UTC()
	dueSchedules, err := controller.repository.ListDueSchedules(ctx, evaluationAt)
	if err != nil {
		return SchedulerPassResult{PreAttemptError: true, Err: err}
	}
	result := SchedulerPassResult{}
	for _, due := range dueSchedules {
		committed, materializeErr := controller.repository.MaterializeDue(ctx, due.ID, evaluationAt)
		if committed {
			result.Materialized++
		}
		if materializeErr != nil {
			result.PreAttemptError = true
			result.Err = materializeErr
			return result
		}
	}

	candidate, err := controller.repository.SelectAttemptCandidate(ctx)
	if err != nil {
		result.PreAttemptError = true
		result.Err = err
		return result
	}
	if candidate == nil {
		return result
	}
	attemptedAt := controller.clock.Now().UTC()
	frozen, err := controller.repository.StartAttempt(ctx, *candidate, attemptedAt)
	if err != nil {
		result.PreAttemptError = true
		result.Err = err
		return result
	}
	if frozen == nil {
		return result
	}
	result.Attempted = true

	attemptCtx, cancel := context.WithTimeout(ctx, handoffDeadline)
	batchResult, dispatchErr := controller.dispatcher.Dispatch(attemptCtx, *frozen)
	cancel()
	outcome := ClassifyHandoff(batchResult, dispatchErr, frozen.TargetScoped)
	recordedAt := controller.clock.Now().UTC()
	recorded, recordErr := controller.repository.RecordOutcome(ctx, frozen.OccurrenceID, outcome, recordedAt)
	duration := recordedAt.Sub(attemptedAt)
	if duration < 0 {
		duration = 0
	}
	if recordErr != nil {
		result.Err = fmt.Errorf("record scheduled scan handoff outcome: %w", recordErr)
		return result
	}
	if !recorded {
		return result
	}
	fields := []zap.Field{
		zap.Int("scheduled_scan.id", frozen.ScheduledScanID),
		zap.Int64("scheduled_scan.occurrence.id", frozen.OccurrenceID),
		zap.Time("scheduled_scan.scheduled_for", frozen.ScheduledFor),
		zap.String("outcome", string(outcome.Kind)),
		zap.Duration("duration", duration),
	}
	if outcome.Completed() {
		controller.logger.Info("Scheduled scan handoff completed", fields...)
	} else {
		controller.logger.Error("Scheduled scan handoff did not complete", fields...)
	}
	return result
}
