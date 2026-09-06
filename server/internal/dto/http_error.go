package dto

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	workflowconfig "github.com/yyhuni/lunafox/contracts/scanworkflow/configuration"
)

// ErrorResponse represents an error response.
type ErrorResponse struct {
	Error ErrorBody `json:"error"`
}

// ErrorBody represents the error details.
type ErrorBody struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Status  string `json:"status"`
	Details []any  `json:"details"`
}

// ErrorDetail represents field-level error details.
type ErrorDetail struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

type ErrorInfo struct {
	Type     string            `json:"@type,omitempty"`
	Reason   string            `json:"reason"`
	Domain   string            `json:"domain"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

// WorkflowFieldViolation is the stable field-level diagnostic for a dynamic
// Workflow Step envelope. It intentionally mirrors google.rpc.BadRequest's
// JSON shape while keeping the stable reason alongside the dynamic path.
type WorkflowFieldViolation struct {
	Field       string `json:"field"`
	Description string `json:"description"`
	Reason      string `json:"reason"`
}

type workflowBadRequestDetail struct {
	Type            string                   `json:"@type"`
	FieldViolations []WorkflowFieldViolation `json:"fieldViolations"`
}

// WriteWorkflowConfigurationError maps a typed shared decoder failure to the
// canonical HTTP contract. It returns false when err is not such a failure so
// callers can continue their ordinary error mapping.
func WriteWorkflowConfigurationError(c *gin.Context, err error) bool {
	if c == nil || err == nil {
		return false
	}
	var validationErr *workflowconfig.ValidationError
	if !errors.As(err, &validationErr) || validationErr == nil {
		return false
	}
	violation := validationErr.Violation
	detail := workflowBadRequestDetail{
		Type: "type.googleapis.com/google.rpc.BadRequest",
		FieldViolations: []WorkflowFieldViolation{{
			Field:       violation.Path,
			Description: violation.Message,
			Reason:      string(violation.Reason),
		}},
	}
	ErrorWithTypedDetails(c, http.StatusBadRequest, "WORKFLOW_CONFIGURATION_INVALID", "Workflow configuration is invalid", nil, detail)
	return true
}

// WriteContextError writes the canonical synchronous-operation response for a
// wrapped caller cancellation or deadline. It returns false for all other
// errors so handlers can continue their ordinary business-error mapping.
func WriteContextError(c *gin.Context, err error) bool {
	switch {
	case errors.Is(err, context.Canceled):
		ErrorWithStatus(c, 499, "CANCELLED", "CANCELLED", "Request was cancelled")
		return true
	case errors.Is(err, context.DeadlineExceeded):
		ErrorWithStatus(c, http.StatusGatewayTimeout, "DEADLINE_EXCEEDED", "DEADLINE_EXCEEDED", "Request deadline exceeded")
		return true
	default:
		return false
	}
}

// Error sends an error response with code.
func Error(c *gin.Context, status int, code string, message string) {
	c.JSON(status, ErrorResponse{
		Error: ErrorBody{
			Code:    status,
			Message: message,
			Status:  canonicalStatus(status),
			Details: []any{newErrorInfo(code, nil)},
		},
	})
}

// ErrorWithStatus is for protocol cases where one HTTP status has multiple
// canonical RPC meanings, such as 409 ABORTED versus ALREADY_EXISTS.
func ErrorWithStatus(c *gin.Context, status int, code, canonicalStatus, message string) {
	ErrorWithStatusAndTypedDetails(c, status, code, canonicalStatus, message, nil)
}

// ErrorWithDetails sends an error response with field details.
func ErrorWithDetails(c *gin.Context, status int, code string, message string, details []ErrorDetail) {
	structuredDetails := []any{newErrorInfo(code, nil)}
	for _, detail := range details {
		structuredDetails = append(structuredDetails, detail)
	}
	c.JSON(status, ErrorResponse{
		Error: ErrorBody{
			Code:    status,
			Message: message,
			Status:  canonicalStatus(status),
			Details: structuredDetails,
		},
	})
}

// ErrorWithContract sends an error response for the standardized task execution error contract.
func ErrorWithContract(c *gin.Context, status int, code, stage, field, message string) {
	c.JSON(status, ErrorResponse{
		Error: ErrorBody{
			Code:    status,
			Message: message,
			Status:  canonicalStatus(status),
			Details: []any{newErrorInfo(code, map[string]string{"stage": stage, "field": field})},
		},
	})
}

// ErrorWithTypedDetails emits the standard ErrorInfo plus caller-owned typed
// diagnostics. It is used where a protocol boundary needs location metadata
// that generic field/message details cannot represent, such as file imports.
func ErrorWithTypedDetails(c *gin.Context, status int, code, message string, metadata map[string]string, details ...any) {
	ErrorWithStatusAndTypedDetails(c, status, code, canonicalStatus(status), message, metadata, details...)
}

// ErrorWithStatusAndTypedDetails emits a caller-selected canonical RPC status
// plus one standard ErrorInfo and optional typed diagnostics.
func ErrorWithStatusAndTypedDetails(c *gin.Context, status int, code, rpcStatus, message string, metadata map[string]string, details ...any) {
	structuredDetails := make([]any, 0, len(details)+1)
	structuredDetails = append(structuredDetails, newErrorInfo(code, metadata))
	structuredDetails = append(structuredDetails, details...)
	c.JSON(status, ErrorResponse{
		Error: ErrorBody{
			Code:    status,
			Message: message,
			Status:  rpcStatus,
			Details: structuredDetails,
		},
	})
}

func newErrorInfo(reason string, metadata map[string]string) ErrorInfo {
	if reason == "" {
		reason = "UNKNOWN"
	}
	return ErrorInfo{
		Type:     "type.googleapis.com/google.rpc.ErrorInfo",
		Reason:   canonicalReason(reason),
		Domain:   "lunafox",
		Metadata: metadata,
	}
}

func canonicalReason(reason string) string {
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return "UNKNOWN"
	}
	reason = strings.ToUpper(reason)
	reason = strings.ReplaceAll(reason, "-", "_")
	reason = strings.ReplaceAll(reason, " ", "_")
	return strings.Trim(reason, "_")
}

func canonicalStatus(status int) string {
	switch status {
	case http.StatusBadRequest, http.StatusUnsupportedMediaType:
		return "INVALID_ARGUMENT"
	case http.StatusUnauthorized:
		return "UNAUTHENTICATED"
	case http.StatusForbidden:
		return "PERMISSION_DENIED"
	case http.StatusNotFound:
		return "NOT_FOUND"
	case http.StatusConflict:
		return "ALREADY_EXISTS"
	case http.StatusRequestEntityTooLarge, http.StatusTooManyRequests:
		return "RESOURCE_EXHAUSTED"
	case http.StatusNotImplemented:
		return "UNIMPLEMENTED"
	case http.StatusServiceUnavailable:
		return "UNAVAILABLE"
	case http.StatusGatewayTimeout:
		return "DEADLINE_EXCEEDED"
	default:
		if status >= 500 {
			return "INTERNAL"
		}
		return "UNKNOWN"
	}
}

// BadRequest sends a bad request error (400).
func BadRequest(c *gin.Context, message string) {
	Error(c, http.StatusBadRequest, "BAD_REQUEST", message)
}

// Unauthorized sends an unauthorized error (401).
func Unauthorized(c *gin.Context, message string) {
	Error(c, http.StatusUnauthorized, "UNAUTHORIZED", message)
}

// Forbidden sends a forbidden error (403).
func Forbidden(c *gin.Context, message string) {
	Error(c, http.StatusForbidden, "FORBIDDEN", message)
}

// NotFound sends a not found error (404).
func NotFound(c *gin.Context, message string) {
	Error(c, http.StatusNotFound, "NOT_FOUND", message)
}

// Conflict sends a conflict error (409).
func Conflict(c *gin.Context, message string) {
	Error(c, http.StatusConflict, "CONFLICT", message)
}

// InternalError sends an internal server error (500).
func InternalError(c *gin.Context, message string) {
	Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", message)
}

// ValidationError sends a validation error with field details (400).
func ValidationError(c *gin.Context, details []ErrorDetail) {
	ErrorWithDetails(c, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid input data", details)
}
