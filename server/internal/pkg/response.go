package pkg

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// ErrorResponse represents an AIP-compatible HTTP error body.
type ErrorResponse struct {
	Error ErrorBody `json:"error"`
}

type ErrorBody struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Status  string      `json:"status"`
	Details []ErrorInfo `json:"details"`
}

type ErrorInfo struct {
	Type     string            `json:"@type,omitempty"`
	Reason   string            `json:"reason"`
	Domain   string            `json:"domain"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

// Meta carries canonical list pagination metadata.
type Meta struct {
	NextPageToken string `json:"nextPageToken,omitempty"`
	TotalSize     int64  `json:"totalSize,omitempty"`
}

// OK sends a successful response with data.
func OK(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, data)
}

// OKWithMeta sends a successful canonical list response.
func OKWithMeta(c *gin.Context, data interface{}, meta *Meta) {
	response := gin.H{"results": data}
	if meta != nil {
		if meta.NextPageToken != "" {
			response["nextPageToken"] = meta.NextPageToken
		}
		if meta.TotalSize > 0 {
			response["totalSize"] = meta.TotalSize
		}
	}
	c.JSON(http.StatusOK, response)
}

// Created sends a 201 Created response.
func Created(c *gin.Context, data interface{}) {
	c.JSON(http.StatusCreated, data)
}

// NoContent sends a 204 No Content response.
func NoContent(c *gin.Context) {
	c.Status(http.StatusNoContent)
}

// BadRequest sends a 400 Bad Request response.
func BadRequest(c *gin.Context, message string) {
	errorJSON(c, http.StatusBadRequest, "BAD_REQUEST", message, nil)
}

// Unauthorized sends a 401 Unauthorized response.
func Unauthorized(c *gin.Context, message string) {
	errorJSON(c, http.StatusUnauthorized, "UNAUTHORIZED", message, nil)
}

// Forbidden sends a 403 Forbidden response.
func Forbidden(c *gin.Context, message string) {
	errorJSON(c, http.StatusForbidden, "FORBIDDEN", message, nil)
}

// NotFound sends a 404 Not Found response.
func NotFound(c *gin.Context, message string) {
	errorJSON(c, http.StatusNotFound, "NOT_FOUND", message, nil)
}

// InternalError sends a 500 Internal Server Error response.
func InternalError(c *gin.Context, message string) {
	errorJSON(c, http.StatusInternalServerError, "INTERNAL_ERROR", message, nil)
}

// ValidationError sends a 422 Unprocessable Entity response.
func ValidationError(c *gin.Context, message string, details string) {
	metadata := map[string]string{}
	if details != "" {
		metadata["details"] = details
	}
	errorJSON(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", message, metadata)
}

func errorJSON(c *gin.Context, status int, reason string, message string, metadata map[string]string) {
	c.JSON(status, ErrorResponse{
		Error: ErrorBody{
			Code:    status,
			Message: message,
			Status:  canonicalStatus(status),
			Details: []ErrorInfo{{
				Type:     "type.googleapis.com/google.rpc.ErrorInfo",
				Reason:   canonicalReason(reason),
				Domain:   "lunafox",
				Metadata: metadata,
			}},
		},
	})
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
	case http.StatusBadRequest, http.StatusUnprocessableEntity:
		return "INVALID_ARGUMENT"
	case http.StatusUnauthorized:
		return "UNAUTHENTICATED"
	case http.StatusForbidden:
		return "PERMISSION_DENIED"
	case http.StatusNotFound:
		return "NOT_FOUND"
	case http.StatusConflict:
		return "ALREADY_EXISTS"
	default:
		if status >= 500 {
			return "INTERNAL"
		}
		return "UNKNOWN"
	}
}

// NextPageToken encodes the next internal page for legacy offset-backed stores.
func NextPageToken(page int) string {
	if page <= 0 {
		return ""
	}
	return base64.RawURLEncoding.EncodeToString([]byte(fmt.Sprintf("page:%d", page)))
}

// CalculateTotalPages remains for non-contract internal calculations.
func CalculateTotalPages(totalCount int64, pageSize int) int {
	if pageSize <= 0 {
		return 0
	}
	pages := int(totalCount) / pageSize
	if int(totalCount)%pageSize > 0 {
		pages++
	}
	return pages
}
