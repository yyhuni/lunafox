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
	runtimeInternalURL   string
	agentImageRef        string
	workerImageRef       string
	sharedDataVolumeBind string
	heartbeatCache       cache.HeartbeatCache
}

type installTemplateData struct {
	Token                string
	RegisterURL          string
	RuntimeGRPCURL       string
	DockerNetworkDefault string
	RequireDockerNetwork string
	LokiPushURL          string
	AgentImageRef        string
	WorkerImageRef       string
	SharedDataVolumeBind string
	AgentVersion         string
}

var agentInstallSHTemplate = template.Must(template.New("agent_install.sh").Parse(agentinstall.AgentInstallScript))

// NewAgentHandler creates a new AgentHandler.
func NewAgentHandler(
	facade *agentapp.AgentFacade,
	runtimeService *agentapp.AgentRuntimeService,
	serverVersion, publicURL, runtimeInternalURL, agentImageRef, workerImageRef, sharedDataVolumeBind string,
	heartbeatCache cache.HeartbeatCache,
) *AgentHandler {
	runtimeInternalURL = strings.TrimSpace(runtimeInternalURL)

	return &AgentHandler{
		facade:               facade,
		runtimeService:       runtimeService,
		serverVersion:        strings.TrimSpace(serverVersion),
		publicURL:            strings.TrimSpace(publicURL),
		runtimeInternalURL:   runtimeInternalURL,
		agentImageRef:        strings.TrimSpace(agentImageRef),
		workerImageRef:       strings.TrimSpace(workerImageRef),
		sharedDataVolumeBind: strings.TrimSpace(sharedDataVolumeBind),
		heartbeatCache:       heartbeatCache,
	}
}
