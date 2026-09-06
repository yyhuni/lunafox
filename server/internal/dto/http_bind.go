package dto

import (
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	contractresults "github.com/yyhuni/lunafox/contracts/results"
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

	// encoding/json substitutes U+FFFD for malformed UTF-8 and unpaired
	// surrogate escapes. Validate the body before decoding so URL-bearing
	// requests are rejected instead of storing a transport-altered value.
	if c.Request.Body != nil {
		c.Request.Body = &jsonTextValidatingReadCloser{ReadCloser: c.Request.Body}
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

// jsonTextValidatingReadCloser rejects input whose JSON decoder would replace
// text. It preserves streaming behavior and leaves JSON syntax to Gin.
type jsonTextValidatingReadCloser struct {
	io.ReadCloser
	validator contractresults.JSONTextValidator
	err       error
}

func (reader *jsonTextValidatingReadCloser) Read(buffer []byte) (int, error) {
	if reader.err != nil {
		return 0, reader.err
	}

	count, readErr := reader.ReadCloser.Read(buffer)
	if count > 0 {
		if err := reader.validator.Write(buffer[:count]); err != nil {
			reader.err = err
			return 0, reader.err
		}
		if errors.Is(readErr, io.EOF) {
			if err := reader.validator.Finalize(); err != nil {
				reader.err = err
				return 0, reader.err
			}
		}
		return count, readErr
	}
	if errors.Is(readErr, io.EOF) {
		if err := reader.validator.Finalize(); err != nil {
			reader.err = err
			return 0, reader.err
		}
	}
	return 0, readErr
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
	if validator, ok := obj.(interface{ ValidatePageToken() error }); ok {
		if err := validator.ValidatePageToken(); err != nil {
			BadRequest(c, err.Error())
			return false
		}
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
