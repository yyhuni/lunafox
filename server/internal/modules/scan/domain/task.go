package domain

import "time"

type TaskStatus string

const (
	TaskStatusBlocked   TaskStatus = "blocked"
	TaskStatusPending   TaskStatus = "pending"
	TaskStatusRunning   TaskStatus = "running"
	TaskStatusSucceeded TaskStatus = "succeeded"
	TaskStatusSkipped   TaskStatus = "skipped"
	TaskStatusFailed    TaskStatus = "failed"
	TaskStatusCancelled TaskStatus = "cancelled"
)

// Planning-time skip reasons are machine values shared by persistence and
// runtime projections. Lifecycle cancellation/propagation may still retain a
// separate operational reason because it is not a planning classification.
const (
	TaskSkipReasonUserDisabled        = "user_disabled"
	TaskSkipReasonTargetNotApplicable = "target_not_applicable"
)

type ScanTask struct {
	ID           int
	ScanID       ScanID
	Name         string
	Stage        int
	Status       TaskStatus
	ErrorMessage string
	StartedAt    *time.Time
	CompletedAt  *time.Time
}

func (task *ScanTask) Succeed(completedAt time.Time) error {
	if task.Status != TaskStatusRunning {
		return ErrInvalidStatusChange
	}
	task.Status = TaskStatusSucceeded
	task.CompletedAt = &completedAt
	return nil
}

func (task *ScanTask) Fail(message string, completedAt time.Time) error {
	if message == "" {
		return ErrFailureMessageMissing
	}
	if task.Status != TaskStatusRunning {
		return ErrInvalidStatusChange
	}
	task.Status = TaskStatusFailed
	task.ErrorMessage = message
	task.CompletedAt = &completedAt
	return nil
}

func (task *ScanTask) Cancel(completedAt time.Time) error {
	if task.Status != TaskStatusPending && task.Status != TaskStatusRunning {
		return ErrInvalidStatusChange
	}
	task.Status = TaskStatusCancelled
	task.CompletedAt = &completedAt
	return nil
}

func (task *ScanTask) ApplyAgentResult(next TaskStatus, errorMessage string, completedAt time.Time) error {
	if task.Status != TaskStatusRunning {
		return ErrInvalidStatusChange
	}
	switch next {
	case TaskStatusSucceeded:
		return task.Succeed(completedAt)
	case TaskStatusFailed:
		return task.Fail(errorMessage, completedAt)
	case TaskStatusCancelled:
		return task.Cancel(completedAt)
	default:
		return ErrInvalidStatusChange
	}
}

func ParseTaskStatus(value string) (TaskStatus, bool) {
	status := TaskStatus(value)
	switch status {
	case TaskStatusBlocked, TaskStatusPending, TaskStatusRunning, TaskStatusSucceeded, TaskStatusSkipped, TaskStatusFailed, TaskStatusCancelled:
		return status, true
	default:
		return "", false
	}
}

func IsTerminalTaskStatus(status TaskStatus) bool {
	return status == TaskStatusSucceeded || status == TaskStatusSkipped || status == TaskStatusFailed || status == TaskStatusCancelled
}
