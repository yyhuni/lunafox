package domain

import "time"

// CreateTargetRef is the target projection used during scan creation.
type CreateTargetRef struct {
	ID            int
	Name          string
	Type          string
	CreatedAt     time.Time
	LastScannedAt *time.Time
	DeletedAt     *time.Time
}

// CreateScan is the write-model projection used by scan create use-cases.
type CreateScan struct {
	ID             int
	TargetID       int
	ScanWorkflowID string
	Configuration  map[string]any
	InputSource    InputSource
	TriggerType    ScanTriggerType
	AssignmentMode string
	AgentID        *int
	Status         string
	CreatedAt      time.Time
	Target         *CreateTargetRef
	ScanTasks      []CreateScanTask
}

// CreateScanTask is the engine execution intent for one workflow step.
type CreateScanTask struct {
	ID                  int
	StageOrder          int
	StageID             string
	StepOrder           int
	StepID              string
	EngineID            string
	EngineConfig        map[string]any
	TaskExecutionConfig map[string]any
	Status              string
	SkipReason          string
	// ResolvedExecutionPlan is deterministic protobuf bytes owned by the
	// Server v2 execution lane. Empty is required for planning-time skipped.
	ResolvedExecutionPlan []byte
}

// CreateScanTaskFinalizer runs after the transaction has allocated the scan
// and task identities. It may only fill the saved plan/skipped outcome; any
// error aborts the surrounding scan transaction.
type CreateScanTaskFinalizer func(scanID, taskID int, task *CreateScanTask) error
