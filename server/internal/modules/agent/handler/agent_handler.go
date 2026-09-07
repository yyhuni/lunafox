// Package handler provides HTTP handlers for agent registration, agent control-plane, and admin APIs.
package handler

import (
	"strings"
	"text/template"

	"github.com/yyhuni/lunafox/server/internal/cache"
	agentapp "github.com/yyhuni/lunafox/server/internal/modules/agent/application"
	agentdomain "github.com/yyhuni/lunafox/server/internal/modules/agent/domain"
	agentinstall "github.com/yyhuni/lunafox/server/internal/modules/agent/install"
)

type agentControlEventPublisher interface {
	SendConfigUpdate(agent *agentdomain.Agent)
}

// AgentHandler handles registration and admin APIs for agents.
type AgentHandler struct {
	facade                  *agentapp.AgentFacade
	controlPublisher        agentControlEventPublisher
	agentVersion            string
	publicURL               string
	agentControlInternalURL string
	agentImageRef           string
	sharedDataVolumeBind    string
	heartbeatCache          cache.HeartbeatCache
}

type installTemplateData struct {
	Token                string
	RegisterURL          string
	AgentControlURL      string
	DockerNetworkDefault string
	RequireDockerNetwork string
	LokiPushURL          string
	AgentImageRef        string
	SharedDataVolumeBind string
	AgentVersion         string
}

var agentInstallSHTemplate = template.Must(template.New("agent_install.sh").Parse(agentinstall.AgentInstallScript))

// NewAgentHandler creates a new AgentHandler.
func NewAgentHandler(
	facade *agentapp.AgentFacade,
	controlPublisher agentControlEventPublisher,
	agentVersion, publicURL, agentControlInternalURL, agentImageRef, sharedDataVolumeBind string,
	heartbeatCache cache.HeartbeatCache,
) *AgentHandler {
	agentControlInternalURL = strings.TrimSpace(agentControlInternalURL)

	return &AgentHandler{
		facade:                  facade,
		controlPublisher:        controlPublisher,
		agentVersion:            strings.TrimSpace(agentVersion),
		publicURL:               strings.TrimSpace(publicURL),
		agentControlInternalURL: agentControlInternalURL,
		agentImageRef:           strings.TrimSpace(agentImageRef),
		sharedDataVolumeBind:    strings.TrimSpace(sharedDataVolumeBind),
		heartbeatCache:          heartbeatCache,
	}
}
