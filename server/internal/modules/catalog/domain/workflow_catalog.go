package domain

// Workflow represents a read-only workflow capability entry.
type Workflow struct {
	Name        string
	Title       string
	Description string
}

// WorkflowProfile represents a read-only workflow profile template.
type WorkflowProfile struct {
	ID            string
	Name          string
	Description   string
	WorkflowNames []string
	Configuration string
}
