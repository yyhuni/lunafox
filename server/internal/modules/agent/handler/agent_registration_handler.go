package handler

import (
	"bytes"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"text/template"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/yyhuni/lunafox/contracts/agentinstall"
	"github.com/yyhuni/lunafox/contracts/sharedstorage"
	agentapp "github.com/yyhuni/lunafox/server/internal/modules/agent/application"
	agentdomain "github.com/yyhuni/lunafox/server/internal/modules/agent/domain"
	"github.com/yyhuni/lunafox/server/internal/modules/agent/dto"
	"github.com/yyhuni/lunafox/server/internal/modules/httpdto"
	"github.com/yyhuni/lunafox/server/internal/pkg/timeutil"
)

const (
	installScriptProfileExternal            = "external"
	installScriptProfileInternal            = "internal"
	installScriptInternalDefaultNetworkName = agentinstall.DefaultAgentDockerNetwork

	installScriptModeDeprecatedMessage  = "mode query parameter is no longer supported; use profile=internal or profile=external"
	installScriptTokenDeprecatedMessage = "token query parameter is no longer supported; use registrationToken"
	installScriptProfileRequiredMessage = "Install script profile is required"
)

// CreateRegistrationToken creates a new registration token.
// POST /v1/admin/agentRegistrationTokens
func (h *AgentHandler) CreateRegistrationToken(c *gin.Context) {
	token, err := h.facade.CreateRegistrationToken(c.Request.Context())
	if err != nil {
		httpdto.InternalError(c, "Failed to create registration token")
		return
	}

	httpdto.Created(c, dto.RegistrationTokenResponse{
		Name:      httpdto.AgentRegistrationTokenName(token.ID),
		Token:     token.Token,
		ExpiresAt: timeutil.ToUTC(token.ExpiresAt),
	})
}

// GetRegistrationToken returns one non-secret registration-token resource.
// GET /v1/admin/agentRegistrationTokens/:registrationToken
func (h *AgentHandler) GetRegistrationToken(c *gin.Context) {
	id, err := httpdto.ParseResourceIDSegment(c.Param("registrationToken"))
	if err != nil {
		httpdto.BadRequest(c, "Invalid registration token ID")
		return
	}
	resource, err := h.facade.GetRegistrationToken(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, agentapp.ErrRegistrationTokenNotFound) {
			httpdto.NotFound(c, "Registration token not found")
			return
		}
		httpdto.InternalError(c, "Failed to get registration token")
		return
	}
	agents := make([]dto.AgentResponse, 0, len(resource.Agents))
	for _, agent := range resource.Agents {
		agents = append(agents, toAgentOutput(agent, nil))
	}
	httpdto.Success(c, dto.RegistrationTokenResourceResponse{
		Name:      httpdto.AgentRegistrationTokenName(resource.ID),
		ExpiresAt: timeutil.ToUTC(resource.ExpiresAt),
		State:     string(resource.StateAt(time.Now().UTC())),
		Agents:    agents,
	})
}

// Register registers an agent using a registration token.
// POST /v1/agents:register
func (h *AgentHandler) Register(c *gin.Context) {
	var req dto.AgentRegistrationRequest
	if !httpdto.BindJSON(c, &req) {
		return
	}

	agent, err := h.facade.RegisterAgent(
		c.Request.Context(),
		req.Token,
		req.ObservedHostname,
		req.AgentVersion,
		agentdomain.AgentRegistrationOptions{
			MaxTasks:      req.MaxTasks,
			CPUThreshold:  req.CPUThreshold,
			MemThreshold:  req.MemThreshold,
			DiskThreshold: req.DiskThreshold,
		},
	)
	if err != nil {
		if errors.Is(err, agentapp.ErrRegistrationTokenInvalid) {
			httpdto.BadRequest(c, "Invalid or expired registration token")
			return
		}
		httpdto.InternalError(c, "Failed to register agent")
		return
	}

	httpdto.Created(c, dto.AgentRegistrationResponse{
		Name:                httpdto.AgentName(agent.ID),
		AgentID:             agent.ID,
		InstanceID:          agent.InstanceID,
		DisplayName:         agent.DisplayName,
		AuthenticationToken: agent.AuthenticationToken,
	})
}

// DownloadInstallScript returns an agent install script for a selected profile.
// GET /v1/agents:downloadInstallScript?registrationToken=...&profile=...
func (h *AgentHandler) DownloadInstallScript(c *gin.Context) {
	if rejectInstallScriptLegacyQuery(c) {
		return
	}
	token := strings.TrimSpace(c.Query("registrationToken"))
	if token == "" {
		httpdto.BadRequest(c, "Missing registration token")
		return
	}
	profile := strings.TrimSpace(c.Query("profile"))
	if profile == "" {
		httpdto.BadRequest(c, installScriptProfileRequiredMessage)
		return
	}
	if profile != installScriptProfileInternal && profile != installScriptProfileExternal {
		httpdto.BadRequest(c, "Invalid install script profile")
		return
	}
	h.installScriptByProfile(c, token, profile)
}

func (h *AgentHandler) installScriptByProfile(c *gin.Context, token string, profile string) {
	if h.facade != nil {
		if err := h.facade.ValidateRegistrationToken(c.Request.Context(), token); err != nil {
			if errors.Is(err, agentapp.ErrRegistrationTokenInvalid) {
				httpdto.BadRequest(c, "Invalid or expired registration token")
				return
			}
			httpdto.InternalError(c, "Failed to validate registration token")
			return
		}
	}

	publicURL, err := validateInstallScriptPublicURL(h.publicURL)
	if err != nil {
		httpdto.InternalError(c, err.Error())
		return
	}
	lokiPushURL, err := buildLokiPushURL(publicURL)
	if err != nil {
		httpdto.InternalError(c, err.Error())
		return
	}
	// REGISTER_URL always goes through PUBLIC_URL (host-side registration request),
	// while runtime endpoints depend on selected install profile.
	registerURL := publicURL
	runtimeGRPCURL := publicURL
	dockerNetworkDefault := "off"
	requireDockerNetwork := "0"
	if profile == installScriptProfileInternal {
		if strings.TrimSpace(h.agentControlInternalURL) == "" {
			httpdto.InternalError(c, "Agent control internal URL is not configured")
			return
		}
		runtimeGRPCURL = h.agentControlInternalURL
		dockerNetworkDefault = installScriptInternalDefaultNetworkName
		requireDockerNetwork = "1"
	}

	agentVersion := h.agentVersion
	if strings.TrimSpace(agentVersion) == "" {
		httpdto.InternalError(c, "Agent version is not configured")
		return
	}
	// Fail fast on missing runtime contracts so the generated script never carries
	// ambiguous defaults.
	agentImageRef := h.agentImageRef
	if agentImageRef == "" {
		httpdto.InternalError(c, "Agent image ref is not configured")
		return
	}

	sharedDataVolumeBind := h.sharedDataVolumeBind
	if sharedDataVolumeBind == "" {
		httpdto.InternalError(c, "Shared data volume bind is not configured")
		return
	}
	if _, err := sharedstorage.ParseSharedDataVolumeBind(sharedDataVolumeBind); err != nil {
		httpdto.InternalError(c, "Shared data volume bind is invalid")
		return
	}

	script, err := renderInstallScript(agentInstallSHTemplate, installTemplateData{
		Token:                token,
		RegisterURL:          registerURL,
		AgentControlURL:      runtimeGRPCURL,
		DockerNetworkDefault: dockerNetworkDefault,
		RequireDockerNetwork: requireDockerNetwork,
		LokiPushURL:          lokiPushURL,
		AgentImageRef:        agentImageRef,
		SharedDataVolumeBind: sharedDataVolumeBind,
		AgentVersion:         agentVersion,
	})
	if err != nil {
		httpdto.InternalError(c, "Failed to build install script")
		return
	}

	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%q", "install.sh"))
	c.Data(http.StatusOK, "text/plain; charset=utf-8", []byte(script))
}

func rejectInstallScriptLegacyQuery(c *gin.Context) bool {
	if strings.TrimSpace(c.Query("mode")) == "" {
		if strings.TrimSpace(c.Query("token")) == "" {
			return false
		}
		httpdto.BadRequest(c, installScriptTokenDeprecatedMessage)
		return true
	}
	httpdto.BadRequest(c, installScriptModeDeprecatedMessage)
	return true
}

func renderInstallScript(tpl *template.Template, data installTemplateData) (string, error) {
	var buf bytes.Buffer
	if err := tpl.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func validateInstallScriptPublicURL(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", fmt.Errorf("PUBLIC_URL is required for install script generation")
	}
	parsed, err := url.Parse(trimmed)
	if err != nil {
		return "", fmt.Errorf("PUBLIC_URL is invalid: %w", err)
	}
	if parsed.Scheme != "https" {
		return "", fmt.Errorf("PUBLIC_URL must use https scheme")
	}
	if parsed.Host == "" {
		return "", fmt.Errorf("PUBLIC_URL host is required")
	}
	return strings.TrimRight(trimmed, "/"), nil
}

func buildLokiPushURL(publicURL string) (string, error) {
	trimmed := strings.TrimSpace(publicURL)
	if trimmed == "" {
		return "", fmt.Errorf("PUBLIC_URL is required for loki push url generation")
	}
	parsed, err := url.Parse(trimmed)
	if err != nil {
		return "", fmt.Errorf("PUBLIC_URL is invalid: %w", err)
	}
	if parsed.Scheme != "https" {
		return "", fmt.Errorf("PUBLIC_URL must use https scheme")
	}
	if parsed.Host == "" {
		return "", fmt.Errorf("PUBLIC_URL host is required")
	}
	return strings.TrimRight(trimmed, "/") + "/loki/api/v1/push", nil
}
