package dto

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	pkgvalidator "github.com/yyhuni/lunafox/server/internal/pkg/validator"
)

// BindJSON binds JSON request body and handles validation errors automatically.
// Returns true if binding succeeded, false if failed (response already sent).
func BindJSON(c *gin.Context, obj any) bool {
	contentType := c.GetHeader("Content-Type")
	if contentType == "" || !strings.HasPrefix(contentType, "application/json") {
		Error(c, http.StatusUnsupportedMediaType, "UNSUPPORTED_MEDIA_TYPE", "Content-Type must be application/json")
		return false
	}

	if err := c.ShouldBindJSON(obj); err != nil {
		if HandleBindingError(c, err) {
			return false
		}
		BadRequest(c, "Invalid request body")
		return false
	}
	return true
}

// BindQuery binds query parameters and handles validation errors automatically.
// Returns true if binding succeeded, false if failed (response already sent).
func BindQuery(c *gin.Context, obj any) bool {
	if err := c.ShouldBindQuery(obj); err != nil {
		if HandleBindingError(c, err) {
			return false
		}
		BadRequest(c, "Invalid query parameters")
		return false
	}
	return true
}

// BindURI binds URI parameters and handles validation errors automatically.
// Returns true if binding succeeded, false if failed (response already sent).
func BindURI(c *gin.Context, obj any) bool {
	if err := c.ShouldBindUri(obj); err != nil {
		if HandleBindingError(c, err) {
			return false
		}
		BadRequest(c, "Invalid URI parameters")
		return false
	}
	return true
}

// ErrorResponse represents an error response.
type ErrorResponse struct {
	Error ErrorBody `json:"error"`
}

// ErrorBody represents the error details.
type ErrorBody struct {
	Code    string        `json:"code"`
	Message string        `json:"message,omitempty"`
	Details []ErrorDetail `json:"details,omitempty"`
}

// ErrorDetail represents field-level error details.
type ErrorDetail struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// Success sends a success response (200) - returns data directly.
func Success(c *gin.Context, data any) {
	c.JSON(http.StatusOK, data)
}

// OK is an alias for Success.
func OK(c *gin.Context, data any) {
	Success(c, data)
}

// Created sends a created response (201) - returns data directly.
func Created(c *gin.Context, data any) {
	c.JSON(http.StatusCreated, data)
}

// NoContent sends a no content response (204).
func NoContent(c *gin.Context) {
	c.Status(http.StatusNoContent)
}

// Error sends an error response with code.
func Error(c *gin.Context, status int, code string, message string) {
	c.JSON(status, ErrorResponse{
		Error: ErrorBody{
			Code:    code,
			Message: message,
		},
	})
}

// ErrorWithDetails sends an error response with field details.
func ErrorWithDetails(c *gin.Context, status int, code string, message string, details []ErrorDetail) {
	c.JSON(status, ErrorResponse{
		Error: ErrorBody{
			Code:    code,
			Message: message,
			Details: details,
		},
	})
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

// HandleBindingError handles binding/validation errors from Gin.
// Returns true if error was handled, false if not a validation error.
func HandleBindingError(c *gin.Context, err error) bool {
	if fieldErrors := pkgvalidator.TranslateErrorToSlice(err); len(fieldErrors) > 0 {
		details := make([]ErrorDetail, len(fieldErrors))
		for i, fe := range fieldErrors {
			details[i] = ErrorDetail{
				Field:   fe.Field,
				Message: fe.Message,
			}
		}
		ValidationError(c, details)
		return true
	}
	return false
}
