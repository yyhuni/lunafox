package tools

import "context"

// TargetQuery is the allowlisted MCP query projection for targets.
type TargetQuery struct {
	PageSize  int
	PageToken string
	Name      string
	Filter    string
	Type      string
	OrderBy   string
}

type ScanQuery struct {
	PageSize  int
	PageToken string
	TargetID  int
	Status    string
	OrderBy   string
}

type AssetQuery struct {
	PageSize    int
	PageToken   string
	OrderBy     string
	URL         string
	DNSName     string
	Host        string
	IP          string
	Port        *int
	StatusCode  *int
	Webserver   string
	ContentType string
	Tech        string
	Vhost       *bool
}

type VulnerabilityQuery struct {
	PageSize  int
	PageToken string
	OrderBy   string
	URL       string
	Severity  string
	Source    string
	VulnType  string
	Reviewed  *bool
}

type TargetReader interface {
	List(context.Context, TargetQuery) (Page[TargetRecord], error)
	Get(context.Context, int) (TargetRecord, error)
}

type ScanReader interface {
	List(context.Context, ScanQuery) (Page[ScanRecord], error)
	Get(context.Context, int) (ScanRecord, error)
}

type WebsiteReader interface {
	List(context.Context, int, AssetQuery) (Page[WebsiteRecord], error)
	Get(context.Context, int) (WebsiteRecord, error)
}

type SubdomainReader interface {
	List(context.Context, int, AssetQuery) (Page[SubdomainRecord], error)
}

type EndpointReader interface {
	List(context.Context, int, AssetQuery) (Page[EndpointRecord], error)
	Get(context.Context, int) (EndpointRecord, error)
}

type DirectoryReader interface {
	List(context.Context, int, AssetQuery) (Page[DirectoryRecord], error)
}

type HostPortReader interface {
	List(context.Context, int, AssetQuery) (Page[HostPortRecord], error)
}

type VulnerabilityReader interface {
	List(context.Context, VulnerabilityQuery) (Page[VulnerabilityRecord], error)
	Get(context.Context, int) (VulnerabilityRecord, error)
}

// Dependencies are the application ports supplied by bootstrap. Mutation
// ports are deliberately limited to the two approved constrained commands.
type Dependencies struct {
	Targets         TargetReader
	Scans           ScanReader
	Websites        WebsiteReader
	Subdomains      SubdomainReader
	Endpoints       EndpointReader
	Directories     DirectoryReader
	HostPorts       HostPortReader
	Vulnerabilities VulnerabilityReader
	// Organizations is retained for the existing create_organization port.
	Organizations        OrganizationCreator
	OrganizationQueries  OrganizationReader
	TargetVulns          TargetVulnerabilityReader
	Screenshots          ScreenshotReader
	Workflows            ScanWorkflowReader
	Profiles             ScanWorkflowProfileReader
	Engines              EngineReader
	Wordlists            WordlistReader
	ServerLogs           ServerLogReader
	AgentLogs            AgentLogReader
	VulnerabilityActions VulnerabilityAction
	ScanStarter          ScanStarter
	Operations           OperationReader
	TargetCreator        TargetBatchCreator
}
