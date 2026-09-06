package adapters

import (
	"github.com/yyhuni/lunafox/server/internal/mcp/tools"
	agentapp "github.com/yyhuni/lunafox/server/internal/modules/agent/application"
	assetapp "github.com/yyhuni/lunafox/server/internal/modules/asset/application"
	catalogapp "github.com/yyhuni/lunafox/server/internal/modules/catalog/application"
	identityapp "github.com/yyhuni/lunafox/server/internal/modules/identity/application"
	scanapp "github.com/yyhuni/lunafox/server/internal/modules/scan/application"
	securityapp "github.com/yyhuni/lunafox/server/internal/modules/security/application"
	systemapp "github.com/yyhuni/lunafox/server/internal/modules/system/application"
)

// Dependencies are application facades supplied by bootstrap. The mutation
// surface remains limited to organization and target creation.
type Dependencies struct {
	Targets         *catalogapp.TargetFacade
	Organizations   *identityapp.OrganizationFacade
	Scans           *scanapp.ScanFacade
	Websites        *assetapp.WebsiteFacade
	Subdomains      *assetapp.SubdomainFacade
	Endpoints       *assetapp.EndpointFacade
	Directories     *assetapp.DirectoryFacade
	HostPorts       *assetapp.HostPortFacade
	Vulnerabilities *securityapp.VulnerabilityFacade
	Screenshots     *assetapp.ScreenshotFacade
	Workflows       *catalogapp.ScanWorkflowManagementService
	Profiles        *catalogapp.ScanWorkflowProfileService
	Engines         *catalogapp.EngineCatalogFacade
	Wordlists       *catalogapp.WordlistFacade
	ServerLogs      *systemapp.ServerLogService
	AgentLogs       *agentapp.LokiLogQueryService
	Agents          *agentapp.AgentFacade
}

// NewReaders projects application facades onto MCP's narrow query and
// constrained-creation ports.
func NewReaders(deps Dependencies) tools.Dependencies {
	return tools.Dependencies{
		Targets:              NewTargetReader(deps.Targets),
		Scans:                NewScanReader(deps.Scans),
		Websites:             NewWebsiteReader(deps.Websites),
		Subdomains:           NewSubdomainReader(deps.Subdomains),
		Endpoints:            NewEndpointReader(deps.Endpoints),
		Directories:          NewDirectoryReader(deps.Directories),
		HostPorts:            NewHostPortReader(deps.HostPorts),
		Vulnerabilities:      NewVulnerabilityReader(deps.Vulnerabilities),
		OrganizationQueries:  NewOrganizationReader(deps.Organizations),
		TargetVulns:          NewTargetVulnerabilityReader(deps.Vulnerabilities),
		Screenshots:          NewScreenshotReader(deps.Screenshots),
		Workflows:            NewScanWorkflowReader(deps.Workflows),
		Profiles:             NewScanWorkflowProfileReader(deps.Profiles),
		Engines:              NewEngineReader(deps.Engines),
		Wordlists:            NewWordlistReader(deps.Wordlists),
		ServerLogs:           NewServerLogReader(deps.ServerLogs),
		AgentLogs:            NewAgentLogReader(deps.AgentLogs, deps.Agents),
		Organizations:        NewOrganizationWriter(deps.Organizations),
		TargetCreator:        NewTargetWriter(deps.Targets),
		VulnerabilityActions: NewVulnerabilityAction(deps.Vulnerabilities),
		ScanStarter:          NewScanStarter(deps.Scans),
		Operations:           NewOperationReader(deps.Scans),
	}
}
