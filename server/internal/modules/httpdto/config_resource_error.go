package httpdto

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	shared "github.com/yyhuni/lunafox/server/internal/dto"
)

const (
	configResourceUnavailableCause           = "unavailable"
	configResourceValidationUnavailableCause = "validationUnavailable"
	configResourceInternalCause              = "internal"
)

type configResourceValidationErrorView interface {
	error
	ConfigResourceValidationCause() string
	ConfigResourceValidationField() string
	ConfigResourceValidationKind() string
	ConfigResourceValidationName() string
}

// WriteConfigResourceValidationError maps the closed resource-validation view
// without importing the Scan application package into shared HTTP adapters.
func WriteConfigResourceValidationError(c *gin.Context, err error) bool {
	if c == nil || err == nil {
		return false
	}
	var view configResourceValidationErrorView
	if !errors.As(err, &view) || view == nil {
		return false
	}
	switch view.ConfigResourceValidationCause() {
	case configResourceUnavailableCause:
		shared.ErrorWithStatusAndTypedDetails(
			c,
			http.StatusBadRequest,
			"ENGINE_CONFIG_RESOURCE_UNAVAILABLE",
			"FAILED_PRECONDITION",
			"Selected Engine configuration resource is unavailable",
			configResourceMetadata(view),
		)
	case configResourceValidationUnavailableCause:
		shared.ErrorWithStatusAndTypedDetails(
			c,
			http.StatusServiceUnavailable,
			"ENGINE_CONFIG_RESOURCE_VALIDATION_UNAVAILABLE",
			"UNAVAILABLE",
			"Engine configuration resource validation could not complete",
			configResourceMetadata(view),
		)
	case configResourceInternalCause:
		shared.InternalError(c, "Engine configuration resource validation failed")
	default:
		shared.InternalError(c, "Engine configuration resource validation failed")
	}
	return true
}

func configResourceMetadata(view configResourceValidationErrorView) map[string]string {
	return map[string]string{
		"field":        view.ConfigResourceValidationField(),
		"resourceKind": view.ConfigResourceValidationKind(),
		"resourceName": view.ConfigResourceValidationName(),
	}
}
