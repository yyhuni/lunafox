package domain

// Workflow represents a read-only workflow capability entry.
type Workflow struct {
	WorkflowID  string
	DisplayName string
	Description string
	Executor    WorkflowExecutorBinding
}

type WorkflowExecutorBinding struct {
	Type string
	Ref  string
}

// WorkflowProfile represents a read-only workflow profile template.
type WorkflowProfile struct {
	ID            string
	Name          string
	Description   string
	WorkflowIDs   []string
	Configuration map[string]any
}
