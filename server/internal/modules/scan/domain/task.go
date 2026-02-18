package domain

import "time"

type TaskStatus string

const (
	TaskStatusBlocked   TaskStatus = "blocked"
	TaskStatusPending   TaskStatus = "pending"
	TaskStatusRunning   TaskStatus = "running"
	TaskStatusCompleted TaskStatus = "completed"
	TaskStatusFailed    TaskStatus = "failed"
	TaskStatusCancelled TaskStatus = "cancelled"
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

func (task *ScanTask) Complete(completedAt time.Time) error {
	if task.Status != TaskStatusRunning {
		return ErrInvalidStatusChange
	}
	task.Status = TaskStatusCompleted
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
	case TaskStatusCompleted:
		return task.Complete(completedAt)
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
	case TaskStatusBlocked, TaskStatusPending, TaskStatusRunning, TaskStatusCompleted, TaskStatusFailed, TaskStatusCancelled:
		return status, true
	default:
		return "", false
	}
}

func IsTerminalTaskStatus(status TaskStatus) bool {
	return status == TaskStatusCompleted || status == TaskStatusFailed || status == TaskStatusCancelled
}
