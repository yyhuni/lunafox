package handler

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	agentapp "github.com/yyhuni/lunafox/server/internal/modules/agent/application"
	agentdomain "github.com/yyhuni/lunafox/server/internal/modules/agent/domain"
	"github.com/yyhuni/lunafox/server/internal/modules/agent/dto"
	"github.com/yyhuni/lunafox/server/internal/modules/httpdto"
)

type agentLicenseService interface {
	Activate(ctx context.Context, key string) (agentapp.AgentLicense, error)
}

// AgentLicenseHandler exposes the authenticated current-license custom method.
type AgentLicenseHandler struct{ service agentLicenseService }

// NewAgentLicenseHandler builds the transport adapter over an application port.
func NewAgentLicenseHandler(service agentLicenseService) *AgentLicenseHandler {
	return &AgentLicenseHandler{service: service}
}

// Activate imports a signed Key and returns only its verified entitlement projection.
// POST /v1/admin/agentLicenses/current:activate
func (handler *AgentLicenseHandler) Activate(c *gin.Context) {
	// Gin matches the whole segment as a wildcard; only this fixed alias action
	// is public. Other aliases or verbs must never trigger activation.
	if c.Param("licenseAction") != "current:activate" {
		httpdto.NotFound(c, "Agent license method not found")
		return
	}
	if handler == nil || handler.service == nil {
		httpdto.InternalError(c, "Agent license service is not configured")
		return
	}
	var request dto.AgentLicenseActivationRequest
	// Bound before decoding so a very large submitted Key cannot allocate an
	// unbounded body; neither binding diagnostics nor responses echo Key bytes.
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 8192)
	decoder := json.NewDecoder(c.Request.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&request); err != nil {
		writeAgentLicenseError(c, "LICENSE_KEY_INVALID", "Authorization Key is invalid")
		return
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		writeAgentLicenseError(c, "LICENSE_KEY_INVALID", "Authorization Key is invalid")
		return
	}
	license, err := handler.service.Activate(c.Request.Context(), request.Key)
	if err != nil {
		if httpdto.WriteContextError(c, err) {
			return
		}
		switch {
		case errors.Is(err, agentdomain.ErrAgentLicenseKeyRequired):
			writeAgentLicenseError(c, "LICENSE_KEY_REQUIRED", "Authorization Key is required")
		case errors.Is(err, agentdomain.ErrAgentLicenseKeyInvalid):
			writeAgentLicenseError(c, "LICENSE_KEY_INVALID", "Authorization Key is invalid")
		case errors.Is(err, agentdomain.ErrAgentLicenseKeyExpired):
			writeAgentLicenseError(c, "LICENSE_KEY_EXPIRED", "Authorization Key has expired")
		case errors.Is(err, agentdomain.ErrAgentLicenseKeyNotYetValid):
			writeAgentLicenseError(c, "LICENSE_KEY_NOT_YET_VALID", "Authorization Key is not yet valid")
		default:
			httpdto.InternalError(c, "Failed to activate Agent license")
		}
		return
	}
	httpdto.Success(c, toAgentLicenseOutput(license))
}

func writeAgentLicenseError(c *gin.Context, reason, message string) {
	c.JSON(http.StatusBadRequest, httpdto.ErrorResponse{Error: httpdto.ErrorBody{
		Code: http.StatusBadRequest, Status: "INVALID_ARGUMENT", Message: message,
		Details: []any{map[string]string{"@type": "type.googleapis.com/google.rpc.ErrorInfo", "domain": "lunafox.agentLicense", "reason": reason}},
	}})
}

func toAgentLicenseOutput(license agentapp.AgentLicense) dto.AgentLicenseResponse {
	return dto.AgentLicenseResponse{Name: httpdto.AgentLicenseName(), State: license.State, AgentLimit: license.AgentLimit,
		Months: license.Months, ExpiresAt: license.ExpiresAt, LicenseID: license.LicenseID}
}
