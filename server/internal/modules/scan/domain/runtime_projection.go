package domain

import "time"

// ScanTaskRecord is the runtime task projection used by task runtime use-cases.
type ScanTaskRecord struct {
	ID                       int
	ScanID                   int
	ScanWorkflowStageOrder   int
	ScanWorkflowID           string
	ScanWorkflowStageID      string
	ScanWorkflowStepID       string
	EngineID                 string
	Status                   string
	SkipReason               string
	AgentID                  *int
	AssignedAgentID          *int
	AssignedSessionID        *string
	AssignedSessionEpoch     *int64
	AssignedRequestID        *string
	HasResolvedExecutionPlan bool
	TaskExecutionConfig      map[string]any
	Failure                  *FailureDetail
	Diagnostics              *EngineExecutionDiagnostics
	CompletedAt              *time.Time
}

// SavedExecutionPlanLease is the exact persisted task/lease snapshot used to
// authorize pre-start artifact streams. It contains no package or config
// authoring data and must never trigger plan recompilation.
type SavedExecutionPlanLease struct {
	TaskID                int
	ScanID                int
	InputSource           InputSource
	Status                string
	AgentID               *int
	AssignedSessionID     *string
	AssignedSessionEpoch  *int64
	ResolvedExecutionPlan []byte
}

// ScanTaskTargetRef is the minimal target projection required by task runtime flows.
type ScanTaskTargetRef struct {
	ID   int
	Name string
	Type string
}

// ScanTaskRuntimeScanRecord is the runtime scan projection required by task runtime flows.
type ScanTaskRuntimeScanRecord struct {
	ID       int
	TargetID int
	Status   string
	Failure  *FailureDetail
	Target   *ScanTaskTargetRef
}

// TaskProgressLogEntry is a scan-scoped progress event shared by application and repository.
// ScanID is persisted to route the event to its retention partition and bind it to its task.
type TaskProgressLogEntry struct {
	ID        int64
	ScanID    int
	TaskID    int
	RequestID string
	Sequence  int64
	Level     string
	Content   string
	EmittedAt *time.Time
	CreatedAt time.Time
}

// TaskProgressLogScanRef is the minimal scan projection required by task-progress-log flows.
type TaskProgressLogScanRef struct {
	ID     int
	Status string
}
