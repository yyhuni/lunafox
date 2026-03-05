package domain

import "time"

// TaskRecord is the runtime task projection used by task runtime use-cases.
type TaskRecord struct {
	ID                 int
	ScanID             int
	Stage              int
	WorkflowID         string
	Status             string
	AgentID            *int
	WorkflowConfigYAML string
}

// TaskTargetRef is the minimal target projection required by task runtime flows.
type TaskTargetRef struct {
	ID   int
	Name string
	Type string
}

// TaskScanRecord is the runtime scan projection required by task runtime flows.
type TaskScanRecord struct {
	ID                int
	TargetID          int
	Status            string
	YAMLConfiguration string
	Target            *TaskTargetRef
}

// ScanLogEntry is the scan log projection shared by application and repository.
type ScanLogEntry struct {
	ID        int64
	ScanID    int
	Level     string
	Content   string
	CreatedAt time.Time
}

// ScanLogScanRef is the minimal scan projection required by scan-log flows.
type ScanLogScanRef struct {
	ID     int
	Status string
}
