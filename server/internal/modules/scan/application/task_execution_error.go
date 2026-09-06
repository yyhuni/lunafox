package application

import (
	"errors"
	"strings"
)

const (
	TaskExecutionErrorCodeSchemaInvalid                 = "SCHEMA_INVALID"
	TaskExecutionErrorCodeTaskExecutionConfigInvalid    = "TASK_EXECUTION_CONFIG_INVALID"
	TaskExecutionErrorCodeOperationRuntimePrereqMissing = "OPERATION_RUNTIME_PREREQ_MISSING"
)

const (
	TaskExecutionErrorStageServerSchemaGate = "server_schema_gate"
)

// TaskExecutionError is the standardized task-execution/runtime-compatibility error contract.
// It is designed to be mapped to both HTTP and gRPC responses.
type TaskExecutionError struct {
	Code    string
	Stage   string
	Field   string
	Message string
	cause   error
}

func (err *TaskExecutionError) Error() string {
	if err == nil {
		return ""
	}
	parts := []string{}
	if strings.TrimSpace(err.Code) != "" {
		parts = append(parts, err.Code)
	}
	if strings.TrimSpace(err.Stage) != "" {
		parts = append(parts, err.Stage)
	}
	if strings.TrimSpace(err.Field) != "" {
		parts = append(parts, err.Field)
	}
	if strings.TrimSpace(err.Message) != "" {
		parts = append(parts, err.Message)
	}
	if len(parts) == 0 {
		return "task execution error"
	}
	return strings.Join(parts, ": ")
}

func (err *TaskExecutionError) Unwrap() error {
	if err == nil {
		return nil
	}
	return err.cause
}

func NewTaskExecutionError(code, stage, field, message string, cause error) *TaskExecutionError {
	return &TaskExecutionError{
		Code:    strings.TrimSpace(code),
		Stage:   strings.TrimSpace(stage),
		Field:   strings.TrimSpace(field),
		Message: strings.TrimSpace(message),
		cause:   cause,
	}
}

func AsTaskExecutionError(err error) (*TaskExecutionError, bool) {
	var taskExecutionErr *TaskExecutionError
	if !errors.As(err, &taskExecutionErr) {
		return nil, false
	}
	return taskExecutionErr, true
}

func WrapSchemaInvalid(field, message string, cause error) error {
	if cause == nil {
		cause = ErrCreateInvalidConfig
	} else if !errors.Is(cause, ErrCreateInvalidConfig) {
		// Keep both the legacy sentinel and the typed decoder violation in the
		// unwrap chain so handlers can map the stable field contract without
		// parsing the human-readable task error.
		cause = errors.Join(ErrCreateInvalidConfig, cause)
	}
	return NewTaskExecutionError(
		TaskExecutionErrorCodeSchemaInvalid,
		TaskExecutionErrorStageServerSchemaGate,
		field,
		message,
		cause,
	)
}
