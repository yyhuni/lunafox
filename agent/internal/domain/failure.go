package domain

// FailureDetail is the structured failure payload reported for task execution failures.
type FailureDetail struct {
	Kind    string
	Message string
}
