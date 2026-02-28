package handler

import (
	"bytes"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"text/template"

	"github.com/gin-gonic/gin"
	agentapp "github.com/yyhuni/lunafox/server/internal/modules/agent/application"
	agentdomain "github.com/yyhuni/lunafox/server/internal/modules/agent/domain"
	"github.com/yyhuni/lunafox/server/internal/modules/agent/dto"
	"github.com/yyhuni/lunafox/server/internal/pkg/timeutil"
)

const (
	installScriptProfileRemote           = "remote"
	installScriptProfileLocal            = "local"
	installScriptLocalDefaultNetworkName = "lunafox_network"

	installScriptModeDeprecatedMessage  = "mode query parameter is no longer supported; use /api/agent/install-script/local or /api/agent/install-script/remote"
	installScriptProfileRequiredMessage = "Install script profile is required; use /api/agent/install-script/local or /api/agent/install-script/remote"
)

// CreateRegistrationToken creates a new registration token.
// POST /api/admin/agents/registration-tokens
func (h *AgentHandler) CreateRegistrationToken(c *gin.Context) {
	token, err := h.facade.CreateRegistrationToken(c.Request.Context())
	if err != nil {
		dto.InternalError(c, "Failed to create registration token")
		return
	}

	dto.Created(c, dto.RegistrationTokenResponse{
		Token:     token.Token,
		ExpiresAt: timeutil.ToUTC(token.ExpiresAt),
	})
}

// Register registers an agent using a registration token.
// POST /api/agent/register
func (h *AgentHandler) Register(c *gin.Context) {
	var req dto.AgentRegistrationRequest
	if !dto.BindJSON(c, &req) {
		return
	}

	agent, err := h.facade.RegisterAgent(
		c.Request.Context(),
		req.Token,
		req.Hostname,
		req.Version,
		"",
		agentdomain.AgentRegistrationOptions{
			MaxTasks:      req.MaxTasks,
			CPUThreshold:  req.CPUThreshold,
			MemThreshold:  req.MemThreshold,
			DiskThreshold: req.DiskThreshold,
		},
	)
	if err != nil {
		if errors.Is(err, agentapp.ErrRegistrationTokenInvalid) {
			dto.BadRequest(c, "Invalid or expired registration token")
			return
		}
		dto.InternalError(c, "Failed to register agent")
		return
	}

	dto.Created(c, dto.AgentRegistrationResponse{AgentID: agent.ID, APIKey: agent.APIKey})
}

// InstallScript keeps legacy path explicit and avoids silent profile fallback.
// GET /api/agent/install-script?token=...
func (h *AgentHandler) InstallScript(c *gin.Context) {
	if rejectInstallScriptModeQuery(c) {
		return
	}
	dto.BadRequest(c, installScriptProfileRequiredMessage)
}

// InstallScriptLocal returns install script for local profile.
// GET /api/agent/install-script/local?token=...
func (h *AgentHandler) InstallScriptLocal(c *gin.Context) {
	if rejectInstallScriptModeQuery(c) {
		return
	}
	h.installScriptByProfile(c, installScriptProfileLocal)
}

// InstallScriptRemote returns install script for remote profile.
// GET /api/agent/install-script/remote?token=...
func (h *AgentHandler) InstallScriptRemote(c *gin.Context) {
	if rejectInstallScriptModeQuery(c) {
		return
	}
	h.installScriptByProfile(c, installScriptProfileRemote)
}

func (h *AgentHandler) installScriptByProfile(c *gin.Context, profile string) {
	token := strings.TrimSpace(c.Query("token"))
	if token == "" {
		dto.BadRequest(c, "Missing registration token")
		return
	}
	if h.facade != nil {
		if err := h.facade.ValidateRegistrationToken(c.Request.Context(), token); err != nil {
			if errors.Is(err, agentapp.ErrRegistrationTokenInvalid) {
				dto.BadRequest(c, "Invalid or expired registration token")
				return
			}
			dto.InternalError(c, "Failed to validate registration token")
			return
		}
	}

	publicURL, err := validateInstallScriptPublicURL(h.publicURL)
	if err != nil {
		dto.InternalError(c, err.Error())
		return
	}
	lokiPushURL, err := buildLokiPushURL(publicURL)
	if err != nil {
		dto.InternalError(c, err.Error())
		return
	}
	// REGISTER_URL always goes through PUBLIC_URL (host-side registration request),
	// while runtime endpoints depend on selected install profile.
	registerURL := publicURL
	runtimeGRPCURL := publicURL
	dockerNetworkDefault := "off"
	requireDockerNetwork := "0"
	if profile == installScriptProfileLocal {
		if strings.TrimSpace(h.runtimeInternalURL) == "" {
			dto.InternalError(c, "Runtime internal URL is not configured")
			return
		}
		runtimeGRPCURL = h.runtimeInternalURL
		dockerNetworkDefault = installScriptLocalDefaultNetworkName
		requireDockerNetwork = "1"
	}

	version := h.serverVersion
	if strings.TrimSpace(version) == "" {
		dto.InternalError(c, "Agent version is not configured")
		return
	}
	// Fail fast on missing runtime contracts so the generated script never carries
	// ambiguous defaults.
	agentImageRef := h.agentImageRef
	if agentImageRef == "" {
		dto.InternalError(c, "Agent image ref is not configured")
		return
	}

	workerImageRef := h.workerImageRef
	if workerImageRef == "" {
		dto.InternalError(c, "Worker image ref is not configured")
		return
	}
	sharedDataVolumeBind := h.sharedDataVolumeBind
	if sharedDataVolumeBind == "" {
		dto.InternalError(c, "Shared data volume bind is not configured")
		return
	}

	script, err := renderInstallScript(agentInstallSHTemplate, installTemplateData{
		Token:                token,
		RegisterURL:          registerURL,
		RuntimeGRPCURL:       runtimeGRPCURL,
		DockerNetworkDefault: dockerNetworkDefault,
		RequireDockerNetwork: requireDockerNetwork,
		LokiPushURL:          lokiPushURL,
		AgentImageRef:        agentImageRef,
		WorkerImageRef:       workerImageRef,
		SharedDataVolumeBind: sharedDataVolumeBind,
		AgentVersion:         version,
	})
	if err != nil {
		dto.InternalError(c, "Failed to build install script")
		return
	}

	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%q", "install.sh"))
	c.Data(http.StatusOK, "text/plain; charset=utf-8", []byte(script))
}

func rejectInstallScriptModeQuery(c *gin.Context) bool {
	if strings.TrimSpace(c.Query("mode")) == "" {
		return false
	}
	dto.BadRequest(c, installScriptModeDeprecatedMessage)
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
