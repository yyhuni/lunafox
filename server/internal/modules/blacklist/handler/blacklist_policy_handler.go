package handler

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/yyhuni/lunafox/contracts/resourcenames"
	blacklistapp "github.com/yyhuni/lunafox/server/internal/modules/blacklist/application"
	"github.com/yyhuni/lunafox/server/internal/modules/blacklist/dto"
	"github.com/yyhuni/lunafox/server/internal/modules/httpdto"
)

var errBlacklistPolicyHandlerDependency = errors.New("blacklist policy handler service is required")

type blacklistPolicyService interface {
	GetGlobal(context.Context) (*blacklistapp.BlacklistPolicy, error)
	GetTarget(context.Context, int) (*blacklistapp.BlacklistPolicy, error)
	ReplaceGlobal(context.Context, blacklistapp.ReplaceBlacklistPolicyInput) (*blacklistapp.BlacklistPolicy, error)
	ReplaceTarget(context.Context, int, blacklistapp.ReplaceBlacklistPolicyInput) (*blacklistapp.BlacklistPolicy, error)
}

// BlacklistPolicyHandler maps the two singleton BlacklistPolicy resources to
// the protected HTTP boundary.
type BlacklistPolicyHandler struct {
	service blacklistPolicyService
}

// NewBlacklistPolicyHandler creates a policy HTTP handler with its required
// application boundary.
func NewBlacklistPolicyHandler(service blacklistPolicyService) (*BlacklistPolicyHandler, error) {
	if service == nil {
		return nil, errBlacklistPolicyHandlerDependency
	}
	return &BlacklistPolicyHandler{service: service}, nil
}

// GetGlobal returns the global BlacklistPolicy singleton.
// GET /v1/blacklistPolicy
func (handler *BlacklistPolicyHandler) GetGlobal(c *gin.Context) {
	policy, err := handler.service.GetGlobal(c.Request.Context())
	if err != nil {
		handler.writeError(c, err)
		return
	}
	if policy == nil {
		httpdto.InternalError(c, "blacklist policy data is unavailable")
		return
	}
	httpdto.Success(c, toBlacklistPolicyOutput(policy))
}

// PatchGlobal fully replaces the global BlacklistPolicy singleton.
// PATCH /v1/blacklistPolicy?updateMask=patterns
func (handler *BlacklistPolicyHandler) PatchGlobal(c *gin.Context) {
	if !validateBlacklistPolicyUpdateMask(c) {
		return
	}
	var request dto.UpdateBlacklistPolicyRequest
	if !httpdto.BindJSON(c, &request) {
		return
	}
	if !validateBlacklistPolicyRequest(c, request, resourcenames.BlacklistPolicy()) {
		return
	}
	policy, err := handler.service.ReplaceGlobal(c.Request.Context(), toReplaceBlacklistPolicyInput(request))
	if err != nil {
		handler.writeError(c, err)
		return
	}
	if policy == nil {
		httpdto.InternalError(c, "blacklist policy data is unavailable")
		return
	}
	httpdto.Success(c, toBlacklistPolicyOutput(policy))
}

// GetTarget returns one Target-local BlacklistPolicy singleton.
// GET /v1/targets/:target/blacklistPolicy
func (handler *BlacklistPolicyHandler) GetTarget(c *gin.Context) {
	targetID, ok := parseBlacklistPolicyTargetID(c)
	if !ok {
		return
	}
	policy, err := handler.service.GetTarget(c.Request.Context(), targetID)
	if err != nil {
		handler.writeError(c, err)
		return
	}
	if policy == nil {
		httpdto.InternalError(c, "blacklist policy data is unavailable")
		return
	}
	httpdto.Success(c, toBlacklistPolicyOutput(policy))
}

// PatchTarget fully replaces one Target-local BlacklistPolicy singleton.
// PATCH /v1/targets/:target/blacklistPolicy?updateMask=patterns
func (handler *BlacklistPolicyHandler) PatchTarget(c *gin.Context) {
	targetID, ok := parseBlacklistPolicyTargetID(c)
	if !ok {
		return
	}
	if !validateBlacklistPolicyUpdateMask(c) {
		return
	}
	var request dto.UpdateBlacklistPolicyRequest
	if !httpdto.BindJSON(c, &request) {
		return
	}
	if !validateBlacklistPolicyRequest(c, request, resourcenames.TargetBlacklistPolicy(targetID)) {
		return
	}
	policy, err := handler.service.ReplaceTarget(c.Request.Context(), targetID, toReplaceBlacklistPolicyInput(request))
	if err != nil {
		handler.writeError(c, err)
		return
	}
	if policy == nil {
		httpdto.InternalError(c, "blacklist policy data is unavailable")
		return
	}
	httpdto.Success(c, toBlacklistPolicyOutput(policy))
}

func parseBlacklistPolicyTargetID(c *gin.Context) (int, bool) {
	targetID, err := httpdto.ParseResourceIDSegment(c.Param("target"))
	if err != nil {
		httpdto.Error(c, http.StatusBadRequest, "INVALID_ARGUMENT", "target must be a positive resource ID")
		return 0, false
	}
	return targetID, true
}

func validateBlacklistPolicyUpdateMask(c *gin.Context) bool {
	query := c.Request.URL.Query()
	values, ok := query["updateMask"]
	if !ok || len(values) != 1 || values[0] != "patterns" || len(query) != 1 {
		httpdto.Error(c, http.StatusBadRequest, "INVALID_ARGUMENT", "updateMask must be exactly patterns")
		return false
	}
	return true
}

func validateBlacklistPolicyRequest(c *gin.Context, request dto.UpdateBlacklistPolicyRequest, expectedName string) bool {
	if strings.TrimSpace(request.Name) != expectedName {
		httpdto.Error(c, http.StatusBadRequest, "INVALID_ARGUMENT", "name must match the resource path")
		return false
	}
	if !request.PatternsPresent() || request.PatternsNull() {
		httpdto.Error(c, http.StatusBadRequest, "INVALID_ARGUMENT", "patterns must be a non-null array")
		return false
	}
	if strings.TrimSpace(request.ETag) == "" {
		httpdto.Error(c, http.StatusBadRequest, "INVALID_ARGUMENT", "etag is required")
		return false
	}
	return true
}

func (handler *BlacklistPolicyHandler) writeError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, blacklistapp.ErrBlacklistPolicyInvalidArgument):
		httpdto.Error(c, http.StatusBadRequest, "INVALID_ARGUMENT", "invalid blacklist policy")
	case errors.Is(err, blacklistapp.ErrBlacklistPolicyNotFound):
		httpdto.NotFound(c, "blacklist policy not found")
	case errors.Is(err, blacklistapp.ErrBlacklistPolicyConflict):
		httpdto.ErrorWithStatus(c, http.StatusConflict, "ABORTED", "ABORTED", "blacklist policy was modified; reload before saving")
	case errors.Is(err, blacklistapp.ErrBlacklistPolicyDataIntegrity):
		httpdto.InternalError(c, "blacklist policy data is unavailable")
	default:
		httpdto.InternalError(c, "blacklist policy operation failed")
	}
}
