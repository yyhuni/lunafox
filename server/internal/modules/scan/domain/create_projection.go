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
	ID            int
	TargetID      int
	WorkflowIDs   []byte
	Configuration map[string]any
	ScanMode      string
	Status        string
	CreatedAt     time.Time
	Target        *CreateTargetRef
}

// CreateScanTask is the task projection persisted during scan creation.
type CreateScanTask struct {
	Stage          int
	WorkflowID     string
	WorkflowConfig map[string]any
	Status         string
}
