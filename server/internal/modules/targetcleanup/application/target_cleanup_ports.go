package application

import (
	"context"
	"time"

	cleanupdomain "github.com/yyhuni/lunafox/server/internal/modules/targetcleanup/domain"
)

type TargetCleanupJobStore interface {
	ListDue(context.Context, time.Time, int) ([]cleanupdomain.CleanupJob, error)
	RecordFailure(context.Context, int, int, time.Time, string) error
	Defer(context.Context, int, time.Time) error
	InspectBacklog(context.Context) (cleanupdomain.CleanupBacklog, error)
}

type TargetCleanupDataStore interface {
	ConfirmTombstone(context.Context, int) error
	DeleteTargetControlPlane(context.Context, int) (organizationRelations, targetPolicies int64, err error)
	DeleteCurrentAssetBatch(context.Context, int, cleanupdomain.AssetResource, int) (int64, error)
	MarkCompletedIfClear(context.Context, int, int, time.Time) (bool, error)
}

type TargetScheduleCleaner interface {
	DeleteTargetScopedSchedules(context.Context, int) (int, error)
}

type TargetScanCancellation struct {
	ScanID                 int
	CancelledTaskCount     int
	NotificationCandidates []TargetTaskCancelNotification
}

type TargetTaskCancelNotification struct {
	TaskID  int
	AgentID int
}

type TargetScanCanceller interface {
	CancelNextActiveScan(context.Context, int, time.Time) (*TargetScanCancellation, error)
}

// TaskCancelPublisher is intentionally best effort. A false result is only a
// delivery observation after the database cancellation has committed.
type TaskCancelPublisher interface {
	TrySendTaskCancel(agentID, scanID, taskID int) bool
}

type TargetCleanupReconciler interface {
	Reconcile(context.Context, cleanupdomain.CleanupJob, TargetCleanupRunOptions) (TargetCleanupReconciliationResult, error)
}

type TargetCleanupRunObserver interface {
	RunStarted(cleanupdomain.CleanupJob, time.Time)
	RunFinished(TargetCleanupRunEvent)
	BacklogObserved(cleanupdomain.CleanupBacklog)
}

type TargetCleanupRunEvent struct {
	Job          cleanupdomain.CleanupJob
	StartedAt    time.Time
	FinishedAt   time.Time
	Outcome      string
	Result       TargetCleanupReconciliationResult
	RetryCount   int
	NextRetryAt  *time.Time
	FailureClass string
}
