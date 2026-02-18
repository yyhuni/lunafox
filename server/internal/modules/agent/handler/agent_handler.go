package handler

import (
	"strings"
	"text/template"

	"github.com/yyhuni/lunafox/server/internal/cache"
	agentapp "github.com/yyhuni/lunafox/server/internal/modules/agent/application"
	agentinstall "github.com/yyhuni/lunafox/server/internal/modules/agent/install"
)

// AgentHandler handles registration and admin APIs for agents.
type AgentHandler struct {
	facade               *agentapp.AgentFacade
	runtimeService       *agentapp.AgentRuntimeService
	serverVersion        string
	publicURL            string
	agentInternalURL     string
	agentImageRef        string
	workerImageRef       string
	sharedDataVolumeBind string
	workerToken          string
	heartbeatCache       cache.HeartbeatCache
}

type installTemplateData struct {
	Token                string
	RegisterURL          string
	AgentServerURL       string
	AgentImageRef        string
	WorkerImageRef       string
	SharedDataVolumeBind string
	AgentVersion         string
	WorkerToken          string
}

var agentInstallSHTemplate = template.Must(template.New("agent_install.sh").Parse(agentinstall.AgentInstallScript))

// NewAgentHandler creates a new AgentHandler.
func NewAgentHandler(
	facade *agentapp.AgentFacade,
	runtimeService *agentapp.AgentRuntimeService,
	serverVersion, publicURL, agentImageRef, workerImageRef, sharedDataVolumeBind, workerToken string,
	heartbeatCache cache.HeartbeatCache,
) *AgentHandler {
	return &AgentHandler{
		facade:               facade,
		runtimeService:       runtimeService,
		serverVersion:        strings.TrimSpace(serverVersion),
		publicURL:            strings.TrimSpace(publicURL),
		agentInternalURL:     "http://server:8080",
		agentImageRef:        strings.TrimSpace(agentImageRef),
		workerImageRef:       strings.TrimSpace(workerImageRef),
		sharedDataVolumeBind: strings.TrimSpace(sharedDataVolumeBind),
		workerToken:          strings.TrimSpace(workerToken),
		heartbeatCache:       heartbeatCache,
	}
}
