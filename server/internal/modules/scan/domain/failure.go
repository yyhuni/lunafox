package domain

// FailureDetail is the structured machine/human-readable failure payload shared by task and scan projections.
type FailureDetail struct {
	Kind    string
	Message string
}
