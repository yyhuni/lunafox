package installapp

import "time"

type InstallState string

const (
	StateIdle      InstallState = "idle"
	StateRunning   InstallState = "running"
	StateSucceeded InstallState = "succeeded"
	StateFailed    InstallState = "failed"
	StateCancelled InstallState = "cancelled"
)

type InstallEventType string

const (
	EventSnapshot InstallEventType = "snapshot"
	EventLog      InstallEventType = "log"
	EventState    InstallEventType = "state"
	EventDone     InstallEventType = "done"
)

type InstallEvent struct {
	ID        int64            `json:"id"`
	JobID     string           `json:"jobId"`
	Type      InstallEventType `json:"type"`
	Timestamp string           `json:"timestamp"`
	Data      any              `json:"data"`
}

type InstallStateSnapshot struct {
	JobID       string `json:"jobId"`
	State       string `json:"state"`
	StartedAt   string `json:"startedAt,omitempty"`
	FinishedAt  string `json:"finishedAt,omitempty"`
	Error       string `json:"error,omitempty"`
	Logs        string `json:"logs"`
	CurrentStep int    `json:"currentStep"`
	TotalSteps  int    `json:"totalSteps"`
	StepTitle   string `json:"stepTitle,omitempty"`
}

type LogEvent struct {
	Message string `json:"message"`
}

type StateEvent struct {
	State       string `json:"state"`
	StartedAt   string `json:"startedAt,omitempty"`
	FinishedAt  string `json:"finishedAt,omitempty"`
	Error       string `json:"error,omitempty"`
	CurrentStep int    `json:"currentStep"`
	TotalSteps  int    `json:"totalSteps"`
	StepTitle   string `json:"stepTitle,omitempty"`
}

func nowRFC3339() string {
	return time.Now().UTC().Format(time.RFC3339)
}
