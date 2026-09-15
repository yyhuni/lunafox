package handler

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/yyhuni/lunafox/server/internal/middleware"
	"github.com/yyhuni/lunafox/server/internal/modules/httpdto"
	"github.com/yyhuni/lunafox/server/internal/modules/upgrade/application"
	"github.com/yyhuni/lunafox/server/internal/modules/upgrade/domain"
	"github.com/yyhuni/lunafox/server/internal/modules/upgrade/dto"
)

type upgradeService interface {
	CheckForUpdates(context.Context, int) (application.CheckForUpdatesResult, error)
	CreateOperation(context.Context, int, application.CreateUpgradeOperationInput) (*domain.Operation, bool, error)
	GetOperation(context.Context, int, string) (*domain.Operation, error)
	RetryOperation(context.Context, int, string, bool) (*domain.Operation, error)
}

// UpgradeHandler is the protected HTTP boundary for system upgrade actions.
// It obtains identity only from the authenticated middleware claims and never
// trusts a username or role supplied by the client.
type UpgradeHandler struct {
	service upgradeService
}

func NewUpgradeHandler(service upgradeService) *UpgradeHandler {
	if service == nil {
		panic("upgrade handler service is required")
	}
	return &UpgradeHandler{service: service}
}

// CheckForUpdates handles POST /v1/system:checkForUpdates.
func (handler *UpgradeHandler) CheckForUpdates(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		httpdto.Unauthorized(c, "Not authenticated")
		return
	}
	// Keep the action body strict even though it is currently empty. This makes
	// future additions explicit and prevents accidental image/path injection.
	var request dto.CheckForUpdatesRequest
	if !httpdto.BindJSON(c, &request) {
		return
	}
	result, err := handler.service.CheckForUpdates(c.Request.Context(), userID)
	if err != nil {
		writeUpgradeError(c, err)
		return
	}
	httpdto.Success(c, dto.NewCheckForUpdatesResponse(result))
}

// CreateOperation handles POST /v1/upgradeOperations.
func (handler *UpgradeHandler) CreateOperation(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		httpdto.Unauthorized(c, "Not authenticated")
		return
	}
	var request dto.CreateUpgradeOperationRequest
	if !httpdto.BindJSON(c, &request) {
		return
	}
	if request.HasClientImageReference() {
		writeUpgradeError(c, domain.NewManifestTargetInvalid(
			"imageRefs",
			"image references must be selected by the validated release manifest",
		))
		return
	}
	operation, created, err := handler.service.CreateOperation(c.Request.Context(), userID, application.CreateUpgradeOperationInput{
		RequestID: request.RequestID, ManifestID: request.ManifestID, ManifestDigest: request.ManifestDigest,
		Confirmed: request.Confirmed, ImageRefs: request.ImageRefs, ImageRef: request.ImageRef,
	})
	if err != nil {
		writeUpgradeError(c, err)
		return
	}
	// Replays are successful idempotent reads of the original resource. A new
	// Operation is the only case that uses 201 Created.
	if created {
		httpdto.Created(c, dto.NewUpgradeOperationResponse(operation))
		return
	}
	httpdto.Success(c, dto.NewUpgradeOperationResponse(operation))
}

// GetOperation handles GET /v1/upgradeOperations/:upgradeOperation.
func (handler *UpgradeHandler) GetOperation(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		httpdto.Unauthorized(c, "Not authenticated")
		return
	}
	operationID, ok := parseOperationSegment(c.Param("upgradeOperation"))
	if !ok {
		httpdto.ErrorWithStatus(c, http.StatusBadRequest, "INVALID_ARGUMENT", "INVALID_ARGUMENT", "upgradeOperation must be a canonical UUID")
		return
	}
	operation, err := handler.service.GetOperation(c.Request.Context(), userID, operationID)
	if err != nil {
		writeUpgradeError(c, err)
		return
	}
	httpdto.Success(c, dto.NewUpgradeOperationResponse(operation))
}

// RetryAction dispatches POST /v1/upgradeOperations/{id}:retry. Gin cannot
// register a literal custom-method sibling next to a resource wildcard, so
// the router passes the bounded action segment here.
func (handler *UpgradeHandler) RetryAction(c *gin.Context) {
	value := strings.TrimPrefix(c.Param("upgradeOperationAction"), "/")
	operationID, method, found := strings.Cut(value, ":")
	if !found || method != "retry" || strings.Contains(operationID, ":") {
		httpdto.NotFound(c, "Custom method not found")
		return
	}
	operationID, ok := parseOperationSegment(operationID)
	if !ok {
		httpdto.ErrorWithStatus(c, http.StatusBadRequest, "INVALID_ARGUMENT", "INVALID_ARGUMENT", "upgradeOperation must be a canonical UUID")
		return
	}
	userID, ok := currentUserID(c)
	if !ok {
		httpdto.Unauthorized(c, "Not authenticated")
		return
	}
	var request dto.RetryUpgradeOperationRequest
	if !httpdto.BindJSON(c, &request) {
		return
	}
	operation, err := handler.service.RetryOperation(c.Request.Context(), userID, operationID, request.Confirmed)
	if err != nil {
		writeUpgradeError(c, err)
		return
	}
	httpdto.Success(c, dto.NewUpgradeOperationResponse(operation))
}

func currentUserID(c *gin.Context) (int, bool) {
	claims, ok := middleware.GetUserClaims(c)
	if !ok || claims == nil || claims.UserID <= 0 {
		return 0, false
	}
	return claims.UserID, true
}

func parseOperationSegment(value string) (string, bool) {
	parsed, err := uuid.Parse(strings.TrimSpace(value))
	if err != nil || parsed.String() != value {
		return "", false
	}
	return parsed.String(), true
}

func writeUpgradeError(c *gin.Context, err error) {
	if err == nil {
		httpdto.InternalError(c, "Upgrade operation failed")
		return
	}
	if httpdto.WriteContextError(c, err) {
		return
	}
	if diagnostic, ok := domain.DiagnosticOf(err); ok {
		status := upgradeDiagnosticHTTPStatus(diagnostic.Code)
		metadata := map[string]string{}
		if diagnostic.Stage != "" {
			metadata["stage"] = diagnostic.Stage
		}
		if diagnostic.Field != "" {
			metadata["field"] = diagnostic.Field
		}
		if diagnostic.Code == domain.ErrorCodeUpgradeNoUpdateAvailable || diagnostic.Code == domain.ErrorCodeReleaseCompatibilityUnsupported {
			httpdto.ErrorWithStatusAndTypedDetails(c, status, string(diagnostic.Code), "FAILED_PRECONDITION", diagnostic.Reason, metadata, diagnostic)
		} else {
			httpdto.ErrorWithTypedDetails(c, status, string(diagnostic.Code), diagnostic.Reason, metadata, diagnostic)
		}
		return
	}
	switch {
	case errors.Is(err, domain.ErrUpgradeUnauthorized):
		httpdto.ErrorWithStatus(c, http.StatusForbidden, string(domain.ErrorCodeUpgradeUnauthorized), "PERMISSION_DENIED", "An active superuser is required")
	case errors.Is(err, domain.ErrUpgradeAlreadyRunning):
		httpdto.ErrorWithStatus(c, http.StatusConflict, string(domain.ErrorCodeUpgradeAlreadyRunning), "ABORTED", "Another upgrade operation is already running")
	case errors.Is(err, domain.ErrUpgradeRequestConflict):
		httpdto.ErrorWithStatus(c, http.StatusConflict, string(domain.ErrorCodeUpgradeRequestConflict), "ALREADY_EXISTS", "requestId is already bound to another upgrade target")
	case errors.Is(err, domain.ErrUpgradeTransitionConflict):
		httpdto.ErrorWithStatus(c, http.StatusConflict, string(domain.ErrorCodeUpgradeTransitionConflict), "ABORTED", "The upgrade operation changed concurrently; retry with the latest state")
	case errors.Is(err, domain.ErrUpgradeConfirmationRequired):
		httpdto.ErrorWithTypedDetails(c, http.StatusBadRequest, string(domain.ErrorCodeUpgradeConfirmationRequired), "Administrator confirmation is required", nil, domain.Diagnostic{Code: domain.ErrorCodeUpgradeConfirmationRequired, Field: "confirmed", Reason: "confirmed must be true"})
	case errors.Is(err, domain.ErrUpgradeHostUnavailable):
		httpdto.ErrorWithStatus(c, http.StatusServiceUnavailable, string(domain.ErrorCodeUpgradeHostUnavailable), "UNAVAILABLE", "The host upgrader is unavailable")
	case errors.Is(err, domain.ErrUpgradeRetryNotAllowed):
		httpdto.ErrorWithStatus(c, http.StatusConflict, string(domain.ErrorCodeUpgradeRetryNotAllowed), "FAILED_PRECONDITION", "The upgrade operation cannot be retried in its current state")
	case errors.Is(err, domain.ErrUpgradeNoUpdateAvailable):
		httpdto.ErrorWithStatus(c, http.StatusBadRequest, string(domain.ErrorCodeUpgradeNoUpdateAvailable), "FAILED_PRECONDITION", "The target release is already current; no update is available")
	case errors.Is(err, domain.ErrUpgradeNotFound):
		httpdto.NotFound(c, "Upgrade operation not found")
	case errors.Is(err, domain.ErrReleaseManifestTargetMismatch):
		httpdto.ErrorWithStatus(c, http.StatusBadRequest, string(domain.ErrorCodeReleaseManifestTargetMismatch), "INVALID_ARGUMENT", err.Error())
	case errors.Is(err, domain.ErrReleaseManifestTargetInvalid):
		httpdto.ErrorWithStatus(c, http.StatusBadRequest, string(domain.ErrorCodeReleaseManifestTargetInvalid), "INVALID_ARGUMENT", err.Error())
	default:
		httpdto.InternalError(c, "Upgrade operation failed")
	}
}

func upgradeDiagnosticHTTPStatus(code domain.ErrorCode) int {
	switch code {
	case domain.ErrorCodeUpgradeUnauthorized:
		return http.StatusForbidden
	case domain.ErrorCodeUpgradeAlreadyRunning, domain.ErrorCodeUpgradeRequestConflict, domain.ErrorCodeUpgradeTransitionConflict:
		return http.StatusConflict
	case domain.ErrorCodeUpgradeHostUnavailable:
		return http.StatusServiceUnavailable
	case domain.ErrorCodeUpgradeNotFound:
		return http.StatusNotFound
	default:
		return http.StatusBadRequest
	}
}
