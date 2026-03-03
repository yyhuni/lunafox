package application

import (
	"errors"
	"fmt"
	"strings"
)

const (
	WorkflowErrorCodeSchemaInvalid             = "SCHEMA_INVALID"
	WorkflowErrorCodeWorkflowConfigInvalid     = "WORKFLOW_CONFIG_INVALID"
	WorkflowErrorCodeWorkflowPrereqMissing     = "WORKFLOW_PREREQ_MISSING"
	WorkflowErrorCodeWorkerVersionIncompatible = "WORKER_VERSION_INCOMPATIBLE"
)

const (
	WorkflowErrorStageServerSchemaGate       = "server_schema_gate"
	WorkflowErrorStageWorkerValidate         = "worker_validate"
	WorkflowErrorStageWorkerPrereq           = "worker_prereq"
	WorkflowErrorStageSchedulerCompatibility = "scheduler_compatibility_gate"
)

// WorkflowError is the standardized validation/compatibility error contract.
// It is designed to be mapped to both HTTP and gRPC responses.
type WorkflowError struct {
	Code    string
	Stage   string
	Field   string
	Message string
	cause   error
}

func (err *WorkflowError) Error() string {
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
		return "workflow error"
	}
	return strings.Join(parts, ": ")
}

func (err *WorkflowError) Unwrap() error {
	if err == nil {
		return nil
	}
	return err.cause
}

func NewWorkflowError(code, stage, field, message string, cause error) *WorkflowError {
	return &WorkflowError{
		Code:    strings.TrimSpace(code),
		Stage:   strings.TrimSpace(stage),
		Field:   strings.TrimSpace(field),
		Message: strings.TrimSpace(message),
		cause:   cause,
	}
}

func AsWorkflowError(err error) (*WorkflowError, bool) {
	var workflowErr *WorkflowError
	if !errors.As(err, &workflowErr) {
		return nil, false
	}
	return workflowErr, true
}

func WrapSchemaInvalid(field, message string, cause error) error {
	if cause == nil {
		cause = ErrCreateInvalidConfig
	} else if !errors.Is(cause, ErrCreateInvalidConfig) {
		cause = fmt.Errorf("%w: %v", ErrCreateInvalidConfig, cause)
	}
	return NewWorkflowError(
		WorkflowErrorCodeSchemaInvalid,
		WorkflowErrorStageServerSchemaGate,
		field,
		message,
		cause,
	)
}
