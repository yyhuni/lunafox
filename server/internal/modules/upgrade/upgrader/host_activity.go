package upgrader

import (
	"context"
	"errors"
	"time"

	"github.com/yyhuni/lunafox/server/internal/modules/upgrade/domain"
)

const hostActivityHeartbeatInterval = 15 * time.Second

var errHostActionValidationFailed = errors.New("host action validation failed")

// runControlledHostAction adds bounded observation around one fixed host
// action. Journal writes are deliberately best effort: they can make a wait
// visible, but must never alter Compose execution, checkpoints, or outcomes.
func (executor *ComposeExecutor) runControlledHostAction(ctx context.Context, store *JournalStore, request Request, stage Stage, action domain.HostAction, run func() error) error {
	startedAt := executor.hostActivityTimestamp()
	activity := domain.HostActivity{Action: action, StartedAt: startedAt, LastHeartbeatAt: startedAt}
	tryAppendHostActionBoundary(store, request, stage, ProgressHostActionStarted, action, "", startedAt)
	if store != nil {
		_, _ = store.SetHostActivity(request.OperationID, request.ManifestDigest, activity)
	}

	done := make(chan struct{})
	stopped := make(chan struct{})
	ticker := time.NewTicker(executor.hostActivityInterval())
	go func() {
		defer close(stopped)
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				if store == nil {
					continue
				}
				heartbeat := domain.HostActivity{Action: action, StartedAt: startedAt, LastHeartbeatAt: executor.hostActivityTimestamp()}
				_, _ = store.SetHostActivity(request.OperationID, request.ManifestDigest, heartbeat)
			}
		}
	}()

	defer func() {
		ticker.Stop()
		close(done)
		<-stopped
		if store != nil {
			_, _ = store.ClearHostActivity(request.OperationID, request.ManifestDigest, &activity)
		}
	}()

	err := run()
	if err != nil {
		tryAppendHostActionBoundary(store, request, stage, ProgressHostActionFailed, action, hostActionFailureReason(err), executor.hostActivityTimestamp())
		return err
	}
	tryAppendHostActionBoundary(store, request, stage, ProgressHostActionCompleted, action, "", executor.hostActivityTimestamp())
	return nil
}

func (executor *ComposeExecutor) hostActivityTimestamp() time.Time {
	if executor != nil && executor.hostActivityNow != nil {
		return executor.hostActivityNow().UTC()
	}
	return nowUTC()
}

func (executor *ComposeExecutor) hostActivityInterval() time.Duration {
	if executor != nil && executor.hostActivityHeartbeatInterval > 0 {
		return min(executor.hostActivityHeartbeatInterval, hostActivityHeartbeatInterval)
	}
	return hostActivityHeartbeatInterval
}

func hostActionFailureReason(err error) domain.HostActionFailureReason {
	if errors.Is(err, errHostActionValidationFailed) {
		return domain.HostActionFailureValidationFailed
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return domain.HostActionFailureCommandCancelled
	}
	return domain.HostActionFailureCommandFailed
}

func tryAppendHostActionBoundary(store *JournalStore, request Request, stage Stage, messageKey string, action domain.HostAction, reason domain.HostActionFailureReason, at time.Time) {
	if store == nil {
		return
	}
	event, err := CatalogHostActionProgressEvent(stage, messageKey, action, reason, at)
	if err != nil {
		return
	}
	_, _ = store.AppendProgress(request.OperationID, request.ManifestDigest, event)
}

func tryAppendHostActionLocalFailure(store *JournalStore, request Request, stage Stage, action domain.HostAction) {
	tryAppendHostActionBoundary(store, request, stage, ProgressHostActionFailed, action, domain.HostActionFailureLocalActionFailed, nowUTC())
}
